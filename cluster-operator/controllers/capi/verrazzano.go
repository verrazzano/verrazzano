// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package capi

import (
	"context"
	"fmt"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/pkg/constants"
	vzctrl "github.com/verrazzano/verrazzano/pkg/controller"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type VerrazzanoRegistration struct {
	client.Client
	Log *zap.SugaredLogger
}

func (v *VerrazzanoRegistration) doReconcile(ctx context.Context, cluster *unstructured.Unstructured) (ctrl.Result, error) {
	workloadClient, err := getWorkloadClusterClient(v.Client, v.Log, cluster)
	if err != nil {
		return ctrl.Result{}, err
	}

	// ensure Verrazzano is installed and ready in workload cluster
	ready, err := v.isVerrazzanoReady(ctx, workloadClient)
	if !ready {
		return vzctrl.ShortRequeue(), err
	}

	// if verrazzano-tls-ca exists, the cluster is untrusted
	err = workloadClient.Get(ctx, types.NamespacedName{Name: constants.PrivateCABundle,
		Namespace: constants.VerrazzanoSystemNamespace}, &v1.Secret{})
	if err != nil {
		if errors.IsNotFound(err) {
			// need to create a CA secret in admin cluster

			// get the workload cluster API CA cert
			caCrt, err := v.getWorkloadClusterCACert(workloadClient)
			if err != nil {
				return ctrl.Result{}, err
			}
			// persist the workload API certificate on the admin cluster
			adminWorkloadCertSecret := &v1.Secret{ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("ca-secret-%s", cluster.GetName()),
				Namespace: constants.VerrazzanoMultiClusterNamespace}}
			if _, err := ctrl.CreateOrUpdate(context.TODO(), workloadClient, adminWorkloadCertSecret, func() error {
				if len(adminWorkloadCertSecret.Data) == 0 {
					adminWorkloadCertSecret.Data = make(map[string][]byte)
				}
				adminWorkloadCertSecret.Data["cacrt"] = caCrt

				return nil
			}); err != nil {
				return ctrl.Result{}, err
			}
		}
		return vzctrl.ShortRequeue(), err
	}
	// obtain the API endpoint IP address for the admin cluster
	err = v.createAdminAccessConfigMap(ctx)
	if err != nil {
		return ctrl.Result{}, err
	}

	// create the VMC if it does not exist
	vmc, err := v.createWorkloadClusterVMC(ctx, cluster)
	if err != nil {
		return ctrl.Result{}, err
	}

	// wait for VMC status to indicate the VMC is ready
	existingVMC := &clustersv1alpha1.VerrazzanoManagedCluster{}
	err = v.Get(ctx, types.NamespacedName{Namespace: vmc.Namespace, Name: vmc.Name}, existingVMC)
	if err != nil {
		return ctrl.Result{}, err
	}
	for _, condition := range existingVMC.Status.Conditions {
		if condition.Type == clustersv1alpha1.ConditionReady && condition.Status != v1.ConditionTrue {
			return vzctrl.ShortRequeue(), nil
		}
	}

	manifest, err := v.getClusterManifest(workloadClient, cluster)
	if err != nil {
		return ctrl.Result{}, err
	}

	// apply the manifest to workload cluster
	yamlApplier := k8sutil.NewYAMLApplier(workloadClient, "")
	err = yamlApplier.ApplyS(string(manifest))
	if err != nil {
		v.Log.Infof("Failed applying cluster manifest to workload cluster %s", cluster.GetName())
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (v *VerrazzanoRegistration) createWorkloadClusterVMC(ctx context.Context, cluster *unstructured.Unstructured) (*clustersv1alpha1.VerrazzanoManagedCluster, error) {
	vmc := &clustersv1alpha1.VerrazzanoManagedCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.GetName(),
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
		Spec: clustersv1alpha1.VerrazzanoManagedClusterSpec{
			CASecret:    fmt.Sprintf("ca-secret-%s", cluster.GetName()),
			Description: fmt.Sprintf("%s VerrazzanoManagedCluster Resource", cluster.GetName()),
		},
	}
	if err := v.Create(ctx, vmc); err != nil {
		if !errors.IsAlreadyExists(err) {
			v.Log.Errorf("Unable to create VMC with name %s: %v", cluster.GetName(), err)
			return nil, err
		}
		v.Log.Debugf("VMC %s already exists", cluster.GetName())
	} else {
		v.Log.Infof("Created VMC for cluster %s", cluster.GetName())
	}
	return vmc, nil
}

func (v *VerrazzanoRegistration) createAdminAccessConfigMap(ctx context.Context) error {
	ep := &v1.Endpoints{}
	if err := v.Get(ctx, types.NamespacedName{Name: "kubernetes"}, ep); err != nil {
		return err
	}
	apiServerIP := ep.Subsets[0].Addresses[0].IP

	// create the admin server IP config map
	cm := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "verrazzano-admin-cluster",
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
	}
	if _, err := ctrl.CreateOrUpdate(ctx, v.Client, cm, func() error {
		if cm.Data == nil {
			cm.Data = make(map[string]string)
		}
		cm.Data["server"] = apiServerIP

		return nil
	}); err != nil {
		v.Log.Errorf("Failed to create the Verrazzano admin cluster config map: %v", err)
		return err
	}
	return nil
}

func (v *VerrazzanoRegistration) getWorkloadClusterCACert(workloadClient client.Client) ([]byte, error) {
	caCrtSecret := &v1.Secret{}
	err := workloadClient.Get(context.TODO(), types.NamespacedName{
		Name:      constants.VerrazzanoIngressTLSSecret,
		Namespace: constants.VerrazzanoSystemNamespace},
		caCrtSecret)
	if err != nil {
		return nil, err
	}
	caCrt, ok := caCrtSecret.Data["ca.crt"]
	if !ok {
		return nil, fmt.Errorf("Workload cluster CA certificate not found in verrazzano-tls secret")
	}
	return caCrt, nil
}

func (v *VerrazzanoRegistration) isVerrazzanoReady(ctx context.Context, workloadClient client.Client) (bool, error) {
	if err := workloadClient.Get(ctx,
		types.NamespacedName{Name: constants.Verrazzano, Namespace: constants.VerrazzanoSystemNamespace},
		&v1.Secret{}); err != nil {
		if !errors.IsNotFound(err) {
			v.Log.Debugf("Failed to retrieve verrazzano secret: %v", err)
			return false, err
		}
		v.Log.Debugf("Verrazzano secret not found")
		return false, nil
	}
	return true, nil
}

func (v *VerrazzanoRegistration) getClusterManifest(workloadClient client.Client, cluster *unstructured.Unstructured) ([]byte, error) {
	// retrieve the manifest for the workload cluster
	manifestSecret := &v1.Secret{}
	err := workloadClient.Get(context.TODO(), types.NamespacedName{
		Name:      fmt.Sprintf("verrazzano-cluster-%s-manifest", cluster.GetName()),
		Namespace: constants.VerrazzanoMultiClusterNamespace},
		manifestSecret)
	if err != nil {
		return nil, err
	}
	manifest, ok := manifestSecret.Data["yaml"]
	if !ok {
		return nil, fmt.Errorf("Error retrieving cluster manifest for %s", cluster.GetName())
	}
	return manifest, nil
}

// UnregisterRancherCluster performs the operations required to de-register the cluster from Rancher
func UnregisterVerrazzanoCluster(ctx context.Context, v *VerrazzanoRegistration, cluster *unstructured.Unstructured) error {
	workloadClient, err := getWorkloadClusterClient(v.Client, v.Log, cluster)
	if err != nil {
		return err
	}
	// get the list of cluster related resources
	manifest, err := v.getClusterManifest(workloadClient, cluster)
	if err != nil {
		return err
	}

	// remove the VMC
	vmc := &clustersv1alpha1.VerrazzanoManagedCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.GetName(),
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
	}
	err = v.Delete(ctx, vmc)
	if err != nil {
		if errors.IsNotFound(err) {
			v.Log.Infof("VMC for cluster %s not found - nothing to do", cluster.GetName())
		}
		return err
	}

	// remove the resources from workload cluster
	yamlApplier := k8sutil.NewYAMLApplier(workloadClient, "")
	err = yamlApplier.DeleteS(string(manifest))
	if err != nil {
		v.Log.Infof("Failed deleting resources of cluster manifest from workload cluster %s", cluster.GetName())
		return err
	}

	return nil
}
