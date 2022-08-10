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

// Synchronize MultiClusterConfigMap objects to the local cluster
func (s *Syncer) syncMCConfigMapObjects(namespace string) error {
	// Get all the MultiClusterConfigMap objects from the admin cluster
	allAdminMCConfigMaps := clustersv1alpha1.MultiClusterConfigMapList{}
	listOptions := &client.ListOptions{Namespace: namespace}
	err := s.AdminClient.List(s.Context, &allAdminMCConfigMaps, listOptions)
	// When placements are changed a forbidden error can be returned.  In this case,
	// we want to fall through and delete orphaned resources.
	if err != nil && !apierrors.IsNotFound(err) && !apierrors.IsForbidden(err) {
		return err
	}

	// Write each of the records that are targeted to this cluster
	for _, mcConfigMap := range allAdminMCConfigMaps.Items {
		if s.isThisCluster(mcConfigMap.Spec.Placement) {
			_, err := s.createOrUpdateMCConfigMap(mcConfigMap)
			if err != nil {
				s.Log.Errorw(fmt.Sprintf("Failed syncing object: %v", err),
					"MultiClusterConfigMap",
					types.NamespacedName{Namespace: mcConfigMap.Namespace, Name: mcConfigMap.Name})
			}
		}
	}

	// Delete orphaned MultiClusterConfigMap resources.
	// Get the list of MultiClusterConfigMap resources on the
	// local cluster and compare to the list received from the admin cluster.
	// The admin cluster is the source of truth.
	allLocalMCConfigMaps := clustersv1alpha1.MultiClusterConfigMapList{}
	err = s.LocalClient.List(s.Context, &allLocalMCConfigMaps, listOptions)
	if err != nil {
		s.Log.Errorf("Failed to list MultiClusterConfigMap on local cluster: %v", err)
		return nil
	}
	for i, mcConfigMap := range allLocalMCConfigMaps.Items {
		// Delete each MultiClusterConfigMap object that is not on the admin cluster or no longer placed on this cluster
		if !s.configMapPlacedOnCluster(&allAdminMCConfigMaps, mcConfigMap.Name, mcConfigMap.Namespace) {
			err := s.LocalClient.Delete(s.Context, &allLocalMCConfigMaps.Items[i])
			if err != nil {
				s.Log.Errorf("Failed to delete MultiClusterConfigMap with name %q and namespace %q: %v", mcConfigMap.Name, mcConfigMap.Namespace, err)
			}
		}
	}

	return nil
}

// Create or update a MultiClusterConfigMap
func (s *Syncer) createOrUpdateMCConfigMap(mcConfigMap clustersv1alpha1.MultiClusterConfigMap) (controllerutil.OperationResult, error) {
	var mcConfigMapNew clustersv1alpha1.MultiClusterConfigMap
	mcConfigMapNew.Namespace = mcConfigMap.Namespace
	mcConfigMapNew.Name = mcConfigMap.Name

	// Create or update on the local cluster
	return controllerutil.CreateOrUpdate(s.Context, s.LocalClient, &mcConfigMapNew, func() error {
		mutateMCConfigMap(mcConfigMap, &mcConfigMapNew)
		return nil
	})
}

func (s *Syncer) updateMultiClusterConfigMapStatus(name types.NamespacedName, newCond clustersv1alpha1.Condition, newClusterStatus clustersv1alpha1.ClusterLevelStatus) error {
	var fetched clustersv1alpha1.MultiClusterConfigMap
	err := s.AdminClient.Get(s.Context, name, &fetched)
	if err != nil {
		return err
	}
	fetched.Status.Conditions = append(fetched.Status.Conditions, newCond)
	clusters.SetClusterLevelStatus(&fetched.Status, newClusterStatus)
	return s.AdminClient.Status().Update(s.Context, &fetched)
}

// mutateMCConfigMap mutates the MultiClusterConfigMap to reflect the contents of the parent MultiClusterConfigMap
func mutateMCConfigMap(mcConfigMap clustersv1alpha1.MultiClusterConfigMap, mcConfigMapNew *clustersv1alpha1.MultiClusterConfigMap) {
	mcConfigMapNew.Spec.Placement = mcConfigMap.Spec.Placement
	mcConfigMapNew.Spec.Template = mcConfigMap.Spec.Template
	mcConfigMapNew.Labels = mcConfigMap.Labels
	// Mark the MC ConfigMap we synced from Admin cluster with verrazzano-managed=true, to
	// distinguish from any (though unlikely) that the user might have created on managed cluster
	if mcConfigMapNew.Labels == nil {
		mcConfigMapNew.Labels = map[string]string{}
	}
	mcConfigMapNew.Labels[vzconst.VerrazzanoManagedLabelKey] = constants.LabelVerrazzanoManagedDefault

}

// configMapPlacedOnCluster returns boolean indicating if the list contains the object with the specified name and namespace
func (s *Syncer) configMapPlacedOnCluster(mcAdminList *clustersv1alpha1.MultiClusterConfigMapList, name string, namespace string) bool {
	for _, item := range mcAdminList.Items {
		if item.Name == name && item.Namespace == namespace {
			return s.isThisCluster(item.Spec.Placement)
		}
	}
	return false
}
