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

// Synchronize MultiClusterSecret objects to the local cluster
func (s *Syncer) syncMCSecretObjects(namespace string) error {
	// Get all the MultiClusterSecret objects from the admin cluster
	allAdminMCSecrets := clustersv1alpha1.MultiClusterSecretList{}
	listOptions := &client.ListOptions{Namespace: namespace}
	err := s.AdminClient.List(s.Context, &allAdminMCSecrets, listOptions)
	if err != nil {
		return client.IgnoreNotFound(err)
	}

	// Write each of the records that are targeted to this cluster
	for _, mcSecret := range allAdminMCSecrets.Items {
		if s.isThisCluster(mcSecret.Spec.Placement) {
			_, err := s.createOrUpdateMCSecret(mcSecret)
			if err != nil {
				s.Log.Error(err, "Error syncing object",
					"MultiClusterSecret",
					types.NamespacedName{Namespace: mcSecret.Namespace, Name: mcSecret.Name})
			}
		}
	}

	// Delete orphaned or no longer placed MultiClusterSecret resources.
	// Get the list of MultiClusterSecret resources on the
	// local cluster and compare to the list received from the admin cluster.
	// The admin cluster is the source of truth.
	allLocalMCSecrets := clustersv1alpha1.MultiClusterSecretList{}
	err = s.LocalClient.List(s.Context, &allLocalMCSecrets, listOptions)
	if err != nil {
		s.Log.Error(err, "failed to list MultiClusterSecret on local cluster")
		return nil
	}
	for _, mcSecret := range allLocalMCSecrets.Items {
		// Delete each MultiClusterSecret object that is not on the admin cluster or no longer placed on this cluster
		if !s.secretPlacedOnCluster(&allAdminMCSecrets, mcSecret.Name, mcSecret.Namespace) {
			err := s.LocalClient.Delete(s.Context, &mcSecret)
			if err != nil {
				s.Log.Error(err, fmt.Sprintf("failed to delete MultiClusterSecret with name %q and namespace %q", mcSecret.Name, mcSecret.Namespace))
			}
		}
	}

	return nil
}

// Create or update a MultiClusterSecret
func (s *Syncer) createOrUpdateMCSecret(mcSecret clustersv1alpha1.MultiClusterSecret) (controllerutil.OperationResult, error) {
	var mcSecretNew clustersv1alpha1.MultiClusterSecret
	mcSecretNew.Namespace = mcSecret.Namespace
	mcSecretNew.Name = mcSecret.Name

	// Create or update on the local cluster
	return controllerutil.CreateOrUpdate(s.Context, s.LocalClient, &mcSecretNew, func() error {
		mutateMCSecret(mcSecret, &mcSecretNew)
		return nil
	})
}

func (s *Syncer) updateMultiClusterSecretStatus(name types.NamespacedName, newCond clustersv1alpha1.Condition, newClusterStatus clustersv1alpha1.ClusterLevelStatus) error {
	fetched := clustersv1alpha1.MultiClusterSecret{}
	err := s.AdminClient.Get(s.Context, name, &fetched)
	if err != nil {
		return err
	}
	fetched.Status.Conditions = append(fetched.Status.Conditions, newCond)
	clusters.SetClusterLevelStatus(&fetched.Status, newClusterStatus)
	return s.AdminClient.Status().Update(s.Context, &fetched)
}

// mutateMCSecret mutates the MultiClusterSecret to reflect the contents of the parent MultiClusterSecret
func mutateMCSecret(mcSecret clustersv1alpha1.MultiClusterSecret, mcSecretNew *clustersv1alpha1.MultiClusterSecret) {
	mcSecretNew.Spec.Placement = mcSecret.Spec.Placement
	mcSecretNew.Spec.Template = mcSecret.Spec.Template
	mcSecretNew.Labels = mcSecret.Labels
}

// secretPlacedOnCluster returns boolean indicating if the list contains the object with the specified name and namespace
// and indicates the object is placed on the local cluster
func (s *Syncer) secretPlacedOnCluster(mcAdminList *clustersv1alpha1.MultiClusterSecretList, name string, namespace string) bool {
	for _, item := range mcAdminList.Items {
		if item.Name == name && item.Namespace == namespace {
			return s.isThisCluster(item.Spec.Placement)
		}
	}
	return false
}
