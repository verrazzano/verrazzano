// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package capi

import (
	"context"
	"fmt"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/cluster-operator/controllers/rancher"
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

type VerrazzanoReconcileFnType func(ctx context.Context, cluster *unstructured.Unstructured, r *CAPIClusterReconciler) (ctrl.Result, error)

var verrazzanoReconcileFn VerrazzanoReconcileFnType = doVerrazzanoReconcile

func SetVerrazzanoReconcileFunction(f VerrazzanoReconcileFnType) {
	verrazzanoReconcileFn = f
}

func SetDefaultVerrazzanoReconcileFunction() {
	verrazzanoReconcileFn = doVerrazzanoReconcile
}

// doVerrazzanoReconcile performs the reconciliation of the CAPI cluster to register it with Verrazzano
func doVerrazzanoReconcile(ctx context.Context, cluster *unstructured.Unstructured, r *CAPIClusterReconciler) (ctrl.Result, error) {
	v := r.VerrazzanoRegistrar
	v.Log.Debugf("Registering cluster %s with Verrazzano", cluster.GetName())

	// register the cluster if Verrazzano installed on workload cluster
	workloadClient, err := getWorkloadClusterClient(v.Client, v.Log, cluster)
	if err != nil {
		v.Log.Errorf("Error getting workload cluster %s client: %v", cluster.GetName(), err)
		return ctrl.Result{}, err
	}

	// ensure Verrazzano is installed and ready in workload cluster
	ready, err := v.isVerrazzanoReady(ctx, workloadClient)
	if err != nil {
		return ctrl.Result{}, err
	}
	if !ready {
		return vzctrl.LongRequeue(), fmt.Errorf("Verrazzano not installed or not ready in cluster %s", cluster.GetName())
	}

	vmc := &clustersv1alpha1.VerrazzanoManagedCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.GetName(),
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
	}
	// if verrazzano-tls-ca exists, the cluster is untrusted
	err = workloadClient.Get(ctx, types.NamespacedName{Name: constants.PrivateCABundle,
		Namespace: constants.VerrazzanoSystemNamespace}, &v1.Secret{})
	if err != nil {
		if errors.IsNotFound(err) {
			// cluster is trusted
			v.Log.Infof("Cluster %s is using trusted certs", cluster.GetName())
		} else {
			// unexpected error
			return ctrl.Result{}, err
		}
	} else {
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
		if _, err := ctrl.CreateOrUpdate(context.TODO(), v.Client, adminWorkloadCertSecret, func() error {
			if len(adminWorkloadCertSecret.Data) == 0 {
				adminWorkloadCertSecret.Data = make(map[string][]byte)
			}
			adminWorkloadCertSecret.Data["cacrt"] = caCrt

			return nil
		}); err != nil {
			return ctrl.Result{}, err
		}

		if _, err := r.createOrUpdateWorkloadClusterVMC(ctx, cluster, vmc, func() error {
			if vmc.Labels == nil {
				vmc.Labels = make(map[string]string)
			}
			vmc.Labels[rancher.CreatedByLabel] = rancher.CreatedByVerrazzano
			vmc.Spec = clustersv1alpha1.VerrazzanoManagedClusterSpec{
				CASecret: fmt.Sprintf("ca-secret-%s", cluster.GetName()),
			}
			return nil
		}); err != nil {
			return ctrl.Result{}, err
		}
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

	manifest, err := v.getClusterManifest(cluster)
	if err != nil {
		return ctrl.Result{}, err
	}

	// apply the manifest to workload cluster
	yamlApplier := k8sutil.NewYAMLApplier(workloadClient, "")
	err = yamlApplier.ApplyS(string(manifest))
	if err != nil {
		v.Log.Errorf("Failed applying cluster manifest to workload cluster %s", cluster.GetName())
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// getWorkloadClusterCACert retrieves the API endpoint CA certificate from the workload cluster
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

// isVerrazzanoReady checks to see whether the Verrazzano resource on the workload cluster is ready
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

// getClusterManifest retrieves the registration manifest for the workload cluster
func (v *VerrazzanoRegistration) getClusterManifest(cluster *unstructured.Unstructured) ([]byte, error) {
	// retrieve the manifest for the workload cluster
	manifestSecret := &v1.Secret{}
	err := v.Get(context.TODO(), types.NamespacedName{
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

// UnregisterVerrazzanoCluster performs the operations required to de-register the cluster from Verrazzano
func UnregisterVerrazzanoCluster(ctx context.Context, v *VerrazzanoRegistration, cluster *unstructured.Unstructured) error {
	workloadClient, err := getWorkloadClusterClient(v.Client, v.Log, cluster)
	if err != nil {
		return err
	}
	// get the list of cluster related resources
	manifest, err := v.getClusterManifest(cluster)
	if err != nil {
		return err
	}

	// remove the resources from workload cluster
	yamlApplier := k8sutil.NewYAMLApplier(workloadClient, "")
	err = yamlApplier.DeleteS(string(manifest))
	if err != nil {
		v.Log.Errorf("Failed deleting resources of cluster manifest from workload cluster %s", cluster.GetName())
		return err
	}

	return nil
}
