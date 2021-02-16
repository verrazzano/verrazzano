// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Synchronize MultiClusterConfigMap objects to the local cluster
func (s *Syncer) syncMCConfigMapObjects() error {
	// Get all the MultiClusterConfigMap objects from the admin cluster
	allMCConfigMaps := clustersv1alpha1.MultiClusterConfigMapList{}
	err := s.AdminClient.List(s.Context, &allMCConfigMaps)
	if err != nil {
		return client.IgnoreNotFound(err)
	}

	// Write each of the records that are targeted to this cluster
	for _, mcConfigMap := range allMCConfigMaps.Items {
		if s.isThisCluster(mcConfigMap.Spec.Placement) {
			_, err := s.createOrUpdateMCConfigMap(mcConfigMap)
			if err != nil {
				return err
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
	mcConfigMapNew.Labels = mcConfigMap.Labels

	// Create or update on the local cluster
	return controllerutil.CreateOrUpdate(s.Context, s.MCClient, &mcConfigMapNew, func() error {
		mutateMCConfigMap(mcConfigMap, &mcConfigMapNew)
		return nil
	})
}

// mutateMCConfigMap mutates the MultiClusterConfigMap to reflect the contents of the parent MultiClusterConfigMap
func mutateMCConfigMap(mcConfigMap clustersv1alpha1.MultiClusterConfigMap, mcConfigMapNew *clustersv1alpha1.MultiClusterConfigMap) {
	mcConfigMapNew.Spec.Placement = mcConfigMap.Spec.Placement
	mcConfigMapNew.Spec.Template = mcConfigMap.Spec.Template
}
