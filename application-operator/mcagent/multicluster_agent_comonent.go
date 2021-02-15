// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Synchronize MultiClusterComponent objects to the local cluster
func (s *Syncer) syncMCComponentObjects() error {
	// Get all the MultiClusterComponent objects from the admin cluster
	allMCComponents := &clustersv1alpha1.MultiClusterComponentList{}
	err := s.AdminClient.List(s.Context, allMCComponents)
	if err != nil {
		return client.IgnoreNotFound(err)
	}

	// Write each of the records that are targeted to this cluster
	for _, mcComponent := range allMCComponents.Items {
		if isThisCluster(s.ClusterName, mcComponent.Spec.Placement) {
			_, err := s.createOrUpdateMCComponent(mcComponent)
			if err != nil {
				return err
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
	mcComponentNew.Labels = mcComponent.Labels

	// Create or update on the local cluster
	return controllerutil.CreateOrUpdate(s.Context, s.MCClient, &mcComponentNew, func() error {
		mutateMCComponent(mcComponent, &mcComponentNew)
		return nil
	})
}

// mutateMCComponent mutates the MultiClusterComponent to reflect the contents of the parent MultiClusterComponent
func mutateMCComponent(mcComponent clustersv1alpha1.MultiClusterComponent, mcComponentNew *clustersv1alpha1.MultiClusterComponent) {
	mcComponentNew.Spec.Placement = mcComponent.Spec.Placement
	mcComponentNew.Spec.Template = mcComponent.Spec.Template
}
