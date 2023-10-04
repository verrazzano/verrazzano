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
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/cluster-api/api/v1beta1"
)

const (
	importedProviderDisplayName = "Imported"
	ocneProviderDisplayName     = "Oracle OCNE on OCI"
	okeProviderDisplayName      = "Oracle OKE"
)

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
	capiCluster, err := capi.GetCluster(context.TODO(), r.Client, clusterNamespacedName)
	if err != nil {
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
func (r *VerrazzanoManagedClusterReconciler) getCAPIProviderDisplayString(capiCluster *v1beta1.Cluster) (string, error) {
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
		return r.getCAPIProviderDisplayStringClusterClass(clusterClass), nil
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
func (r *VerrazzanoManagedClusterReconciler) getCAPIProviderDisplayStringClusterClass(clusterClass *v1beta1.ClusterClass) string {
	clusterClassNamespacedName := types.NamespacedName{
		Name:      clusterClass.GetName(),
		Namespace: clusterClass.GetNamespace(),
	}

	// Get infrastructure provider
	infraProvider, found, err := unstructured.NestedString(clusterClass.Object, "spec", "infrastructure", "ref", "kind")
	if !found {
		r.log.Progressf("could not find spec.infrastructure.ref.kind field inside cluster %s: %v", clusterClassNamespacedName, err)
		return ""
	}
	if err != nil {
		r.log.Progressf("error while looking for spec.infrastructure.ref.kind field for cluster %s: %v", clusterClass, err)
		return ""
	}

	// Get control plane provider
	cpProvider, found, err := unstructured.NestedString(clusterClass.Object, "spec", "controlPlane", "ref", "kind")
	if !found {
		r.log.Progressf("could not find spec.controlPlane.ref.kind field inside cluster %s: %v", clusterClassNamespacedName, err)
		return ""
	}
	if err != nil {
		r.log.Progressf("error while looking for spec.controlPlane.ref.kind field for cluster %s: %v", clusterClassNamespacedName, err)
		return ""
	}

	// Remove the "Template" suffix from the provider names
	infraProvider = strings.TrimSuffix(infraProvider, "Template")
	cpProvider = strings.TrimSuffix(cpProvider, "Template")

	return r.formProviderDisplayString(infraProvider, cpProvider)
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
	cluster, err := capi.GetCluster(context.TODO(), r.Client, clusterNamespacedName)
	if err != nil {
		if errors.IsNotFound(err) {
			return "", nil
		}
		return "", err
	}

	// Get the state
	phase, found, err := unstructured.NestedString(cluster.Object, "status", "phase")
	if !found {
		r.log.Progressf("could not find status.phase field inside cluster %s: %v", clusterNamespacedName, err)
		return "", nil
	}
	if err != nil {
		r.log.Progressf("error while looking for status.phase field for cluster %s: %v", clusterNamespacedName, err)
		return "", nil
	}

	// Validate that the CAPI Phase is a proper StateType for the VMC
	switch state := clustersv1alpha1.StateType(phase); state {
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
