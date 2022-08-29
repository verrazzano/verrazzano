// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	"fmt"

	"github.com/verrazzano/verrazzano/application-operator/constants"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Synchronize MultiClusterComponent objects to the local cluster
func (s *Syncer) syncMCComponentObjects(namespace string) error {
	// Get all the MultiClusterComponent objects from the admin cluster
	allAdminMCComponents := clustersv1alpha1.MultiClusterComponentList{}
	listOptions := &client.ListOptions{Namespace: namespace}
	err := s.AdminClient.List(s.Context, &allAdminMCComponents, listOptions)
	// When placements are changed a forbidden error can be returned.  In this case,
	// we want to fall through and delete orphaned resources.
	if err != nil && !apierrors.IsNotFound(err) && !apierrors.IsForbidden(err) {
		return err
	}

	// Write each of the records that are targeted to this cluster
	for _, mcComponent := range allAdminMCComponents.Items {
		if s.isThisCluster(mcComponent.Spec.Placement) {
			_, err := s.createOrUpdateMCComponent(mcComponent)
			if err != nil {
				s.Log.Errorw(fmt.Sprintf("Failed syncing object: %v", err),
					"MultiClusterComponent",
					types.NamespacedName{Namespace: mcComponent.Namespace, Name: mcComponent.Name})
			}
		}
	}

	// Delete orphaned MultiClusterComponent resources.
	// Get the list of MultiClusterComponent resources on the
	// local cluster and compare to the list received from the admin cluster.
	// The admin cluster is the source of truth.
	allLocalMCComponents := clustersv1alpha1.MultiClusterComponentList{}
	err = s.LocalClient.List(s.Context, &allLocalMCComponents, listOptions)
	if err != nil {
		s.Log.Errorf("Failed to list MultiClusterComponent on local cluster: %v", err)
		return nil
	}
	for i, mcComponent := range allLocalMCComponents.Items {
		// Delete each MultiClusterComponent object that is not on the admin cluster or no longer placed on this cluster
		if !s.componentPlacedOnCluster(&allAdminMCComponents, mcComponent.Name, mcComponent.Namespace) {
			err := s.LocalClient.Delete(s.Context, &allLocalMCComponents.Items[i])
			if err != nil {
				s.Log.Errorf("Failed to delete MultiClusterComponent with name %q and namespace %q: %v", mcComponent.Name, mcComponent.Namespace, err)
			}
		}
	}

	return nil
}

// Create or update a MultiClusterComponent
func (s *Syncer) createOrUpdateMCComponent(mcComponent clustersv1alpha1.MultiClusterComponent) (controllerutil.OperationResult, error) {
	var mcComponentNew clustersv1alpha1.MultiClusterComponent
	mcComponentNew.Namespace = mcComponent.Namespace
	mcComponentNew.Name = mcComponent.Name

	// Create or update on the local cluster
	return controllerutil.CreateOrUpdate(s.Context, s.LocalClient, &mcComponentNew, func() error {
		mutateMCComponent(mcComponent, &mcComponentNew)
		return nil
	})
}

func (s *Syncer) updateMultiClusterComponentStatus(name types.NamespacedName, newCond clustersv1alpha1.Condition, newClusterStatus clustersv1alpha1.ClusterLevelStatus) error {
	var fetched clustersv1alpha1.MultiClusterComponent
	err := s.AdminClient.Get(s.Context, name, &fetched)
	if err != nil {
		return err
	}
	fetched.Status.Conditions = append(fetched.Status.Conditions, newCond)
	clusters.SetClusterLevelStatus(&fetched.Status, newClusterStatus)
	return s.AdminClient.Status().Update(s.Context, &fetched)
}

// mutateMCComponent mutates the MultiClusterComponent to reflect the contents of the parent MultiClusterComponent
func mutateMCComponent(mcComponent clustersv1alpha1.MultiClusterComponent, mcComponentNew *clustersv1alpha1.MultiClusterComponent) {
	mcComponentNew.Spec.Placement = mcComponent.Spec.Placement
	mcComponentNew.Spec.Template = mcComponent.Spec.Template
	mcComponentNew.Labels = mcComponent.Labels
	// Mark the MC component we synced from Admin cluster with verrazzano-managed=true, to
	// distinguish from any (though unlikely) that the user might have created on managed cluster
	if mcComponentNew.Labels == nil {
		mcComponentNew.Labels = map[string]string{}
	}
	mcComponentNew.Labels[vzconst.VerrazzanoManagedLabelKey] = constants.LabelVerrazzanoManagedDefault

}

// componentPlacedOnCluster returns boolean indicating if the list contains the object with the specified name and namespace
func (s *Syncer) componentPlacedOnCluster(mcAdminList *clustersv1alpha1.MultiClusterComponentList, name string, namespace string) bool {
	for _, item := range mcAdminList.Items {
		if item.Name == name && item.Namespace == namespace {
			return s.isThisCluster(item.Spec.Placement)
		}
	}
	return false
}
