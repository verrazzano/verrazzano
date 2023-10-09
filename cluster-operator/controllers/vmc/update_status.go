// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vmc

import (
	"context"
	"fmt"
	"strings"

	clustersv1alpha1 "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/cluster-operator/internal/capi"
	vzconstants "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	capiv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	importedProviderDisplayName = "Imported"
	ocneProviderDisplayName     = "Oracle OCNE on OCI"
	okeProviderDisplayName      = "Oracle OKE"
)

// for unit testing
var getCAPIClientFunc = capi.GetClusterClient

// updateStatus updates the status of the VMC in the cluster, with all provided conditions, after setting the vmc.Status.State field for the cluster
func (r *VerrazzanoManagedClusterReconciler) updateStatus(ctx context.Context, vmc *clustersv1alpha1.VerrazzanoManagedCluster) error {
	// Update the VMC's status.state
	if err := r.updateState(vmc); err != nil {
		return err
	}

	// Update the VMC's status.imported
	imported := vmc.Status.ClusterRef == nil
	vmc.Status.Imported = &imported

	// Update the VMC's status.provider
	if err := r.updateProvider(vmc); err != nil {
		return err
	}

	// Conditionally update the VMC's Kubernetes version
	if updateNeeded, err := r.shouldUpdateK8sVersion(vmc); err != nil {
		return err
	} else if updateNeeded {
		if err := r.updateK8sVersionUsingCAPI(vmc); err != nil {
			return err
		}
	}

	// Fetch the existing VMC to avoid conflicts in the status update
	existingVMC := &clustersv1alpha1.VerrazzanoManagedCluster{}
	err := r.Get(context.TODO(), types.NamespacedName{Namespace: vmc.Namespace, Name: vmc.Name}, existingVMC)
	if err != nil {
		return err
	}

	// Replace the existing status conditions and state with the conditions generated from this reconcile
	for _, genCondition := range vmc.Status.Conditions {
		r.setStatusCondition(existingVMC, genCondition, genCondition.Type == clustersv1alpha1.ConditionManifestPushed)
	}
	existingVMC.Status.State = vmc.Status.State
	existingVMC.Status.ArgoCDRegistration = vmc.Status.ArgoCDRegistration
	existingVMC.Status.Imported = vmc.Status.Imported
	existingVMC.Status.Provider = vmc.Status.Provider
	existingVMC.Status.Kubernetes.Version = vmc.Status.Kubernetes.Version

	r.log.Debugf("Updating Status of VMC %s: %v", vmc.Name, vmc.Status.Conditions)
	return r.Status().Update(ctx, existingVMC)
}

// updateProvider sets the VMC's status.provider field
func (r *VerrazzanoManagedClusterReconciler) updateProvider(vmc *clustersv1alpha1.VerrazzanoManagedCluster) error {
	// This VMC represents an imported cluster.
	if vmc.Status.ClusterRef == nil {
		vmc.Status.Provider = importedProviderDisplayName
		return nil
	}

	// This VMC represents a CAPI cluster. Get the provider and update the VMC.
	clusterNamespacedName := types.NamespacedName{
		Name:      vmc.Status.ClusterRef.Name,
		Namespace: vmc.Status.ClusterRef.Namespace,
	}
	capiCluster := &capiv1beta1.Cluster{}
	if err := r.Client.Get(context.TODO(), clusterNamespacedName, capiCluster); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	provider, err := r.getCAPIProviderDisplayString(capiCluster)
	if err != nil {
		return err
	}
	vmc.Status.Provider = provider
	return nil
}

// getCAPIProviderDisplayString returns the string to populate the VMC's status.provider field, based on information taken from the
// provided CAPI Cluster.
func (r *VerrazzanoManagedClusterReconciler) getCAPIProviderDisplayString(capiCluster *capiv1beta1.Cluster) (string, error) {
	// If this CAPI Cluster was created using ClusterClass, then parse capiCluster differently.
	if capiCluster.Spec.Topology != nil {
		clusterClass, err := capi.GetClusterClassFromCluster(context.TODO(), r.Client, capiCluster)
		if err != nil {
			if errors.IsNotFound(err) {
				r.log.Progressf("could not find ClusterClass %s/%s: %v", clusterClass.GetNamespace(), clusterClass.GetName(), err)
				return "", nil
			}
			return "", err
		}
		return r.getCAPIProviderDisplayStringClusterClass(clusterClass)
	}

	// This cluster does not use ClusterClass.
	// Get infrastructure provider
	if capiCluster.Spec.InfrastructureRef == nil {
		return "", fmt.Errorf("clusterAPI cluster %s/%s has an unset spec.infrastructureRef field", capiCluster.Namespace, capiCluster.Name)
	}
	infraProvider := capiCluster.Spec.InfrastructureRef.Kind
	if infraProvider == "" {
		return "", fmt.Errorf("clusterAPI cluster %s/%s has an empty infrastructure provider", capiCluster.Namespace, capiCluster.Name)
	}

	// Get control plane provider
	if capiCluster.Spec.ControlPlaneRef == nil {
		return "", fmt.Errorf("clusterAPI cluster %s/%s has an unset spec.controlPlaneRef field", capiCluster.Namespace, capiCluster.Name)
	}
	cpProvider := capiCluster.Spec.ControlPlaneRef.Kind
	if cpProvider == "" {
		return "", fmt.Errorf("clusterAPI cluster %s/%s has an empty control plane provider", capiCluster.Namespace, capiCluster.Name)
	}

	return r.formProviderDisplayString(infraProvider, cpProvider), nil
}

// getCAPIProviderDisplayStringClusterClass returns the string to populate the VMC's status.provider field, given the ClusterClass
// associated with this managed cluster.
func (r *VerrazzanoManagedClusterReconciler) getCAPIProviderDisplayStringClusterClass(clusterClass *capiv1beta1.ClusterClass) (string, error) {
	// Get infrastructure provider
	if clusterClass.Spec.Infrastructure.Ref == nil {
		return "", fmt.Errorf("cluster class %s/%s has an unset spec.infrastructure.ref field", clusterClass.Namespace, clusterClass.Name)
	}
	infraProvider := clusterClass.Spec.Infrastructure.Ref.Kind
	if infraProvider == "" {
		return "", fmt.Errorf("cluster class %s/%s has an empty infrastructure provider", clusterClass.Namespace, clusterClass.Name)
	}

	// Get control plane provider
	if clusterClass.Spec.ControlPlane.Ref == nil {
		return "", fmt.Errorf("cluster class %s/%s has an unset spec.controlPlane.ref field", clusterClass.Namespace, clusterClass.Name)
	}
	cpProvider := clusterClass.Spec.ControlPlane.Ref.Kind
	if cpProvider == "" {
		return "", fmt.Errorf("cluster class %s/%s has an empty control plane provider", clusterClass.Namespace, clusterClass.Name)
	}

	// Remove the "Template" suffix from the provider names
	infraProvider = strings.TrimSuffix(infraProvider, "Template")
	cpProvider = strings.TrimSuffix(cpProvider, "Template")

	return r.formProviderDisplayString(infraProvider, cpProvider), nil
}

// formProviderDisplayString forms the display string for the VMC's status.provider field, given the infrastructure and
// control plane provider strings
func (r *VerrazzanoManagedClusterReconciler) formProviderDisplayString(infraProvider, cpProvider string) string {
	// Use specialized strings for OKE and OCNE special cases
	if infraProvider == capi.OCNEInfrastructureProvider && cpProvider == capi.OCNEControlPlaneProvider {
		return ocneProviderDisplayName
	} else if infraProvider == capi.OKEInfrastructureProvider && cpProvider == capi.OKEControlPlaneProvider {
		return okeProviderDisplayName
	}
	// Otherwise, return this generic format for the provider display string
	provider := fmt.Sprintf("%s on %s Infrastructure", cpProvider, infraProvider)
	return provider
}

// updateState sets the vmc.Status.State for the given VMC.
// The state field functions differently according to whether this VMC references an underlying ClusterAPI cluster.
func (r *VerrazzanoManagedClusterReconciler) updateState(vmc *clustersv1alpha1.VerrazzanoManagedCluster) error {
	// If there is no underlying CAPI cluster, set the state field based on the lastAgentConnectTime
	if vmc.Status.ClusterRef == nil {
		r.updateStateFromLastAgentConnectTime(vmc)
		return nil
	}

	// If there is an underlying CAPI cluster, set the state field according to the phase of the CAPI cluster.
	capiClusterPhase, err := r.getCAPIClusterPhase(vmc.Status.ClusterRef)
	if err != nil {
		return err
	}
	if capiClusterPhase != "" {
		vmc.Status.State = capiClusterPhase
	}
	return nil
}

// updateStateFromLastAgentConnectTime sets the vmc.Status.State according to the lastAgentConnectTime,
// setting possible values of Active, Inactive, or Pending.
func (r *VerrazzanoManagedClusterReconciler) updateStateFromLastAgentConnectTime(vmc *clustersv1alpha1.VerrazzanoManagedCluster) {
	if vmc.Status.LastAgentConnectTime != nil {
		currentTime := metav1.Now()
		// Using the current plus added time to find the difference with lastAgentConnectTime to validate
		// if it exceeds the max allowed time before changing the state of the vmc resource.
		maxPollingTime := currentTime.Add(vzconstants.VMCAgentPollingTimeInterval * vzconstants.MaxTimesVMCAgentPollingTime)
		timeDiff := maxPollingTime.Sub(vmc.Status.LastAgentConnectTime.Time)
		if int(timeDiff.Minutes()) > vzconstants.MaxTimesVMCAgentPollingTime {
			vmc.Status.State = clustersv1alpha1.StateInactive
		} else if vmc.Status.State == "" {
			vmc.Status.State = clustersv1alpha1.StatePending
		} else {
			vmc.Status.State = clustersv1alpha1.StateActive
		}
	}
}

// getCAPIClusterPhase returns the phase reported by the CAPI Cluster CR which is referenced by clusterRef.
func (r *VerrazzanoManagedClusterReconciler) getCAPIClusterPhase(clusterRef *clustersv1alpha1.ClusterReference) (clustersv1alpha1.StateType, error) {
	// Get the CAPI Cluster CR
	clusterNamespacedName := types.NamespacedName{
		Name:      clusterRef.Name,
		Namespace: clusterRef.Namespace,
	}
	cluster := &capiv1beta1.Cluster{}
	if err := r.Client.Get(context.TODO(), clusterNamespacedName, cluster); err != nil {
		if errors.IsNotFound(err) {
			return "", nil
		}
		return "", err
	}

	// Validate that the CAPI Phase is a proper StateType for the VMC
	switch state := clustersv1alpha1.StateType(cluster.Status.Phase); state {
	case clustersv1alpha1.StatePending,
		clustersv1alpha1.StateProvisioning,
		clustersv1alpha1.StateProvisioned,
		clustersv1alpha1.StateDeleting,
		clustersv1alpha1.StateUnknown,
		clustersv1alpha1.StateFailed:
		return state, nil
	default:
		r.log.Progressf("retrieved an invalid ClusterAPI Cluster phase of %s", state)
		return clustersv1alpha1.StateUnknown, nil
	}
}

// shouldUpdateK8sVersion determines if this VMC reconciler should update the VMC's Kubernetes version.
func (r *VerrazzanoManagedClusterReconciler) shouldUpdateK8sVersion(vmc *clustersv1alpha1.VerrazzanoManagedCluster) (bool, error) {
	// The VMC controller cannot update the Kubernetes version if this is not a CAPI cluster.
	if vmc.Status.ClusterRef == nil {
		return false, nil
	}

	// If Verrazzano is installed on the workload cluster, then let the verrazzano cluster agent handle updating the K8s version.
	capiClusterName := types.NamespacedName{Name: vmc.Status.ClusterRef.Name, Namespace: vmc.Status.ClusterRef.Namespace}
	capiClient, err := getCAPIClientFunc(context.TODO(), r.Client, capiClusterName, r.Scheme)
	if err != nil {
		return false, fmt.Errorf("failed to get client for ClusterAPI cluster %s: %v", capiClusterName, err)
	}
	vzList := &v1beta1.VerrazzanoList{}
	if err = capiClient.List(context.TODO(), vzList, &clipkg.ListOptions{}); err != nil {
		vzGroupVersionResource := schema.GroupVersionResource{
			Group:    v1beta1.SchemeGroupVersion.Group,
			Version:  v1beta1.SchemeGroupVersion.Version,
			Resource: "verrazzanos",
		}
		_, gvkErr := capiClient.RESTMapper().KindFor(vzGroupVersionResource)
		if errors.IsNotFound(err) || gvkErr != nil {
			return true, nil
		}
		return false, fmt.Errorf("error listing verrazzanos in ClusterAPI cluster %s: %v", capiClusterName, err)
	}
	if len(vzList.Items) > 0 {
		return false, nil
	}
	return true, nil
}

// updateK8sVersionUsingCAPI updates the VMC's status.kubernetes.version field, retrieving the version from ClusterAPI CRs
func (r *VerrazzanoManagedClusterReconciler) updateK8sVersionUsingCAPI(vmc *clustersv1alpha1.VerrazzanoManagedCluster) error {
	// Get the CAPI Cluster CR
	clusterNamespacedName := types.NamespacedName{
		Name:      vmc.Status.ClusterRef.Name,
		Namespace: vmc.Status.ClusterRef.Namespace,
	}
	cluster := &capiv1beta1.Cluster{}
	if err := r.Client.Get(context.TODO(), clusterNamespacedName, cluster); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}

	// Get control plane ref
	cpKind := cluster.Spec.ControlPlaneRef.Kind
	cpAPIVersion := cluster.Spec.ControlPlaneRef.APIVersion
	cpList := &unstructured.UnstructuredList{}
	cpList.SetAPIVersion(cpAPIVersion)
	cpList.SetKind(cpKind)
	if err := r.List(context.TODO(), cpList, clipkg.InNamespace(clusterNamespacedName.Namespace)); err != nil {
		return fmt.Errorf("error listing control plane objects: %v", err)
	}
	if len(cpList.Items) < 1 {
		return fmt.Errorf("failed to find %s objects", cpKind)
	}
	k8sVersion, found, err := unstructured.NestedString(cpList.Items[0].Object, "status", "version")
	if !found {
		return fmt.Errorf("could not find status.version field in %s object", cpKind)
	} else if err != nil {
		return fmt.Errorf("error accessing status.version field in %s object: %v", cpKind, err)
	}

	// Set the K8s version in the VMC
	vmc.Status.Kubernetes.Version = k8sVersion
	return nil
}
