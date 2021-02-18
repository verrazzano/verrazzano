// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	"fmt"

	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Synchronize MultiClusterComponent objects to the local cluster
func (s *Syncer) syncMCComponentObjects() error {
	// Get all the MultiClusterComponent objects from the admin cluster
	allAdminMCComponents := clustersv1alpha1.MultiClusterComponentList{}
	err := s.AdminClient.List(s.Context, &allAdminMCComponents)
	if err != nil {
		return client.IgnoreNotFound(err)
	}

	// Write each of the records that are targeted to this cluster
	for _, mcComponent := range allAdminMCComponents.Items {
		if s.isThisCluster(mcComponent.Spec.Placement) {
			_, err := s.createOrUpdateMCComponent(mcComponent)
			s.Log.Error(err, "Error syncing MultiClusterComponent object",
				types.NamespacedName{Namespace: mcComponent.Namespace, Name: mcComponent.Name})
		}
	}

	// Delete orphaned MultiClusterComponent resources.
	// Get the list of MultiClusterComponent resources on the
	// local cluster and compare to the list received from the admin cluster.
	// The admin cluster is the source of truth.
	allLocalMCComponents := clustersv1alpha1.MultiClusterComponentList{}
	err = s.LocalClient.List(s.Context, &allLocalMCComponents)
	if err != nil {
		s.Log.Error(err, "failed to list MultiClusterComponent on local cluster")
		return nil
	}
	for _, mcComponent := range allLocalMCComponents.Items {
		// Delete each MultiClusterComponent object that is not on the admin cluster
		if !componentListContains(&allAdminMCComponents, mcComponent.Name, mcComponent.Namespace) {
			err := s.LocalClient.Delete(s.Context, &mcComponent)
			if err != nil {
				s.Log.Error(err, fmt.Sprintf("failed to delete MultiClusterComponent with name %q and namespace %q", mcComponent.Name, mcComponent.Namespace))
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

// mutateMCComponent mutates the MultiClusterComponent to reflect the contents of the parent MultiClusterComponent
func mutateMCComponent(mcComponent clustersv1alpha1.MultiClusterComponent, mcComponentNew *clustersv1alpha1.MultiClusterComponent) {
	mcComponentNew.Spec.Placement = mcComponent.Spec.Placement
	mcComponentNew.Spec.Template = mcComponent.Spec.Template
	mcComponentNew.Labels = mcComponent.Labels
}

// componentListContains returns boolean indicating if the list contains the object with the specified name and namespace
func componentListContains(mcAdminList *clustersv1alpha1.MultiClusterComponentList, name string, namespace string) bool {
	for _, item := range mcAdminList.Items {
		if item.Name == name && item.Namespace == namespace {
			return true
		}
	}
	return false
}
