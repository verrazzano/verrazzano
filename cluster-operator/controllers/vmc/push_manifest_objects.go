// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vmc

import (
	"context"
	"fmt"
	clusterapi "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	internalcapi "github.com/verrazzano/verrazzano/cluster-operator/internal/capi"
	constants2 "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/rancherutil"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// pushManifestObjects applies the Verrazzano manifest objects to the managed cluster.
// To access the managed cluster, we are taking advantage of the Rancher proxy or CAPI based access.
func (r *VerrazzanoManagedClusterReconciler) pushManifestObjects(ctx context.Context, rancherEnabled bool, vmc *clusterapi.VerrazzanoManagedCluster) (bool, error) {
	if rancherEnabled {
		clusterID := vmc.Status.RancherRegistration.ClusterID
		if len(clusterID) == 0 {
			r.log.Progressf("Waiting to push manifest objects, Rancher ClusterID not found in the VMC %s/%s status", vmc.GetNamespace(), vmc.GetName())
			return false, nil
		}
		rc, err := rancherutil.NewVerrazzanoClusterRancherConfig(r.Client, r.RancherIngressHost, r.log)
		if err != nil || rc == nil {
			return false, err
		}

		// If the managed cluster is not active, we should not attempt to push resources
		isActive, err := isManagedClusterActiveInRancher(rc, clusterID, r.log)
		if err != nil {
			return false, err
		}

		vsNamespaceCreated, _ := isNamespaceCreated(vmc, r, clusterID, constants.VerrazzanoSystemNamespace)
		if isActive && vsNamespaceCreated {
			// Create or Update the agent and registration secrets
			agentSecret := corev1.Secret{}
			agentSecret.Namespace = constants.VerrazzanoSystemNamespace
			agentSecret.Name = constants.MCAgentSecret
			regSecret := corev1.Secret{}
			regSecret.Namespace = constants.VerrazzanoSystemNamespace
			regSecret.Name = constants.MCRegistrationSecret
			agentOperation, err := createOrUpdateSecretRancherProxy(&agentSecret, rc, clusterID, func() error {
				existingAgentSec, err := r.getSecret(vmc.Namespace, GetAgentSecretName(vmc.Name), true)
				if err != nil {
					return err
				}
				agentSecret.Data = existingAgentSec.Data
				return nil
			}, r.log)
			if err != nil {
				return false, err
			}
			regOperation, err := createOrUpdateSecretRancherProxy(&regSecret, rc, clusterID, func() error {
				existingRegSecret, err := r.getSecret(vmc.Namespace, GetRegistrationSecretName(vmc.Name), true)
				if err != nil {
					return err
				}
				regSecret.Data = existingRegSecret.Data
				return nil
			}, r.log)
			if err != nil {
				return false, err
			}
			agentModified := agentOperation != controllerutil.OperationResultNone
			regModified := regOperation != controllerutil.OperationResultNone
			return agentModified || regModified, nil
		}
	}
	if vmc.Status.ClusterRef != nil {
		if vmc.Status.RancherRegistration.Status != clusterapi.RegistrationApplied {
			cluster := &unstructured.Unstructured{}
			cluster.SetGroupVersionKind(internalcapi.GVKCAPICluster)
			err := r.Get(context.TODO(), types.NamespacedName{Namespace: vmc.Status.ClusterRef.Namespace, Name: vmc.Status.ClusterRef.Name}, cluster)
			if err != nil && !errors.IsNotFound(err) {
				return false, err
			}
			manifest, err := r.getClusterManifest(cluster)
			if err != nil {
				return false, err
			}
			// register the cluster if Verrazzano installed on workload cluster
			workloadClient, err := r.getWorkloadClusterClient(cluster)
			if err != nil {
				r.log.Errorf("Error getting workload cluster %s client: %v", cluster.GetName(), err)
				return false, err
			}

			// apply the manifest to workload cluster
			yamlApplier := k8sutil.NewYAMLApplier(workloadClient, "")
			err = yamlApplier.ApplyS(string(manifest))
			if err != nil {
				r.log.Errorf("Failed applying cluster manifest to workload cluster %s: %v", cluster.GetName(), err)
				return false, err
			}
			r.log.Infof("Registration manifest applied to cluster %s", cluster.GetName())

			// update the registration status if Rancher is enabled since repeated application of the manifest will
			// trigger connection issues
			if rancherEnabled {
				existingVMC := &clusterapi.VerrazzanoManagedCluster{}
				err = r.Get(context.TODO(), types.NamespacedName{Namespace: vmc.Namespace, Name: vmc.Name}, existingVMC)
				if err != nil {
					return false, err
				}
				existingVMC.Status.RancherRegistration.Status = clusterapi.RegistrationApplied
				err = r.Status().Update(ctx, existingVMC)
				if err != nil {
					r.log.Errorf("Error updating VMC status for cluster %s: %v", cluster.GetName(), err)
					return false, err
				}
				vmc = existingVMC

				// get and label the cattle-system namespace
				ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: common.CattleSystem}}
				if _, err := ctrl.CreateOrUpdate(context.TODO(), workloadClient, ns, func() error {
					if ns.Labels == nil {
						ns.Labels = make(map[string]string)
					}
					ns.Labels[constants2.LabelVerrazzanoNamespace] = common.CattleSystem
					return nil
				}); err != nil {
					return false, err
				}
			}
			return true, nil
		}

	}

	return false, nil
}

// getClusterManifest retrieves the registration manifest for the workload cluster
func (r *VerrazzanoManagedClusterReconciler) getClusterManifest(cluster *unstructured.Unstructured) ([]byte, error) {
	// retrieve the manifest for the workload cluster
	manifestSecret := &corev1.Secret{}
	err := r.Get(context.TODO(), types.NamespacedName{
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
