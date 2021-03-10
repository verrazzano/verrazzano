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

// Synchronize MultiClusterConfigMap objects to the local cluster
func (s *Syncer) syncMCConfigMapObjects(namespace string) error {
	// Get all the MultiClusterConfigMap objects from the admin cluster
	allAdminMCConfigMaps := clustersv1alpha1.MultiClusterConfigMapList{}
	listOptions := &client.ListOptions{Namespace: namespace}
	err := s.AdminClient.List(s.Context, &allAdminMCConfigMaps, listOptions)
	if err != nil {
		return client.IgnoreNotFound(err)
	}

	// Write each of the records that are targeted to this cluster
	for _, mcConfigMap := range allAdminMCConfigMaps.Items {
		if s.isThisCluster(mcConfigMap.Spec.Placement) {
			_, err := s.createOrUpdateMCConfigMap(mcConfigMap)
			if err != nil {
				s.Log.Error(err, "Error syncing object",
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
		s.Log.Error(err, "failed to list MultiClusterConfigMap on local cluster")
		return nil
	}
	for _, mcConfigMap := range allLocalMCConfigMaps.Items {
		// Delete each MultiClusterConfigMap object that is not on the admin cluster
		if !configMapListContains(&allAdminMCConfigMaps, mcConfigMap.Name, mcConfigMap.Namespace) {
			err := s.LocalClient.Delete(s.Context, &mcConfigMap)
			if err != nil {
				s.Log.Error(err, fmt.Sprintf("failed to delete MultiClusterConfigMap with name %q and namespace %q", mcConfigMap.Name, mcConfigMap.Namespace))
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
		fetched.Status.Conditions = append(fetched.Status.Conditions, newCond)
		fetched.Status.Clusters = append(fetched.Status.Clusters, newClusterStatus)
		err = s.AdminClient.Status().Update(s.Context, &fetched)
	}
	return err
}

// mutateMCConfigMap mutates the MultiClusterConfigMap to reflect the contents of the parent MultiClusterConfigMap
func mutateMCConfigMap(mcConfigMap clustersv1alpha1.MultiClusterConfigMap, mcConfigMapNew *clustersv1alpha1.MultiClusterConfigMap) {
	mcConfigMapNew.Spec.Placement = mcConfigMap.Spec.Placement
	mcConfigMapNew.Spec.Template = mcConfigMap.Spec.Template
	mcConfigMapNew.Labels = mcConfigMap.Labels
}

// configMapListContains returns boolean indicating if the list contains the object with the specified name and namespace
func configMapListContains(mcAdminList *clustersv1alpha1.MultiClusterConfigMapList, name string, namespace string) bool {
	for _, item := range mcAdminList.Items {
		if item.Name == name && item.Namespace == namespace {
			return true
		}
	}
	return false
}
