// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	"fmt"

	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Synchronize MultiClusterLoggingScope objects to the local cluster
func (s *Syncer) syncMCLoggingScopeObjects(namespace string) error {
	// Get all the MultiClusterLoggingScope objects from the admin cluster
	allAdminMCLoggingScopes := clustersv1alpha1.MultiClusterLoggingScopeList{}
	listOptions := &client.ListOptions{Namespace: namespace}
	err := s.AdminClient.List(s.Context, &allAdminMCLoggingScopes, listOptions)
	if err != nil {
		return client.IgnoreNotFound(err)
	}

	// Write each of the records that are targeted to this cluster
	for _, mcLoggingScope := range allAdminMCLoggingScopes.Items {
		if s.isThisCluster(mcLoggingScope.Spec.Placement) {
			_, err := s.createOrUpdateMCLoggingScope(mcLoggingScope)
			if err != nil {
				s.Log.Error(err, "Error syncing object",
					"MultiClusterLoggingScope",
					types.NamespacedName{Namespace: mcLoggingScope.Namespace, Name: mcLoggingScope.Name})
			}
		}
	}

	// Delete orphaned MultiClusterLoggingScope resources.
	// Get the list of MultiClusterLoggingScope resources on the
	// local cluster and compare to the list received from the admin cluster.
	// The admin cluster is the source of truth.
	allLocalMCLoggingScopes := clustersv1alpha1.MultiClusterLoggingScopeList{}
	err = s.LocalClient.List(s.Context, &allLocalMCLoggingScopes, listOptions)
	if err != nil {
		s.Log.Error(err, "failed to list MultiClusterLoggingScope on local cluster")
		return nil
	}
	for _, mcLoggingScope := range allLocalMCLoggingScopes.Items {
		// Delete each MultiClusterLoggingScope object that is not on the admin cluster
		if !s.loggingScopePlacedOnCluster(&allAdminMCLoggingScopes, mcLoggingScope.Name, mcLoggingScope.Namespace) {
			err := s.LocalClient.Delete(s.Context, &mcLoggingScope)
			if err != nil {
				s.Log.Error(err, fmt.Sprintf("failed to delete MultiClusterLoggingScope with name %q and namespace %q", mcLoggingScope.Name, mcLoggingScope.Namespace))
			}
		}
	}

	return nil
}

// Create or update a MultiClusterLoggingScope
func (s *Syncer) createOrUpdateMCLoggingScope(mcLoggingScope clustersv1alpha1.MultiClusterLoggingScope) (controllerutil.OperationResult, error) {
	var mcLoggingScopeNew clustersv1alpha1.MultiClusterLoggingScope
	mcLoggingScopeNew.Namespace = mcLoggingScope.Namespace
	mcLoggingScopeNew.Name = mcLoggingScope.Name

	// Create or update on the local cluster
	return controllerutil.CreateOrUpdate(s.Context, s.LocalClient, &mcLoggingScopeNew, func() error {
		mutateMCLoggingScope(mcLoggingScope, &mcLoggingScopeNew)
		return nil
	})
}

func (s *Syncer) updateMultiClusterLoggingScopeStatus(name types.NamespacedName, newCond clustersv1alpha1.Condition, newClusterStatus clustersv1alpha1.ClusterLevelStatus) error {
	var fetched clustersv1alpha1.MultiClusterLoggingScope
	err := s.AdminClient.Get(s.Context, name, &fetched)
	if err != nil {
		return err
	}
	fetched.Status.Conditions = append(fetched.Status.Conditions, newCond)
	clusters.SetClusterLevelStatus(&fetched.Status, newClusterStatus)
	return s.AdminClient.Status().Update(s.Context, &fetched)
}

// mutateMCLoggingScope mutates the MultiClusterLoggingScope to reflect the contents of the parent MultiClusterLoggingScope
func mutateMCLoggingScope(mcLoggingScope clustersv1alpha1.MultiClusterLoggingScope, mcLoggingScopeNew *clustersv1alpha1.MultiClusterLoggingScope) {
	mcLoggingScopeNew.Spec.Placement = mcLoggingScope.Spec.Placement
	mcLoggingScopeNew.Spec.Template = mcLoggingScope.Spec.Template
	mcLoggingScopeNew.Labels = mcLoggingScope.Labels
}

// loggingScopePlacedOnCluster returns boolean indicating if the list contains the object with the specified name and namespace
func (s *Syncer) loggingScopePlacedOnCluster(mcAdminList *clustersv1alpha1.MultiClusterLoggingScopeList, name string, namespace string) bool {
	for _, item := range mcAdminList.Items {
		if item.Name == name && item.Namespace == namespace {
			return s.isThisCluster(item.Spec.Placement)
		}
	}
	return false
}
