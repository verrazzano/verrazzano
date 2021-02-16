// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Synchronize MultiClusterLoggingScope objects to the local cluster
func (s *Syncer) syncMCLoggingScopeObjects() error {
	// Get all the MultiClusterLoggingScope objects from the admin cluster
	allMCLoggingScopes := clustersv1alpha1.MultiClusterLoggingScopeList{}
	err := s.AdminClient.List(s.Context, &allMCLoggingScopes)
	if err != nil {
		return client.IgnoreNotFound(err)
	}

	// Write each of the records that are targeted to this cluster
	for _, mcLoggingScope := range allMCLoggingScopes.Items {
		if s.isThisCluster(mcLoggingScope.Spec.Placement) {
			_, err := s.createOrUpdateMCLoggingScope(mcLoggingScope)
			if err != nil {
				return err
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
	mcLoggingScopeNew.Labels = mcLoggingScope.Labels

	// Create or update on the local cluster
	return controllerutil.CreateOrUpdate(s.Context, s.MCClient, &mcLoggingScopeNew, func() error {
		mutateMCLoggingScope(mcLoggingScope, &mcLoggingScopeNew)
		return nil
	})
}

// mutateMCLoggingScope mutates the MultiClusterLoggingScope to reflect the contents of the parent MultiClusterLoggingScope
func mutateMCLoggingScope(mcLoggingScope clustersv1alpha1.MultiClusterLoggingScope, mcLoggingScopeNew *clustersv1alpha1.MultiClusterLoggingScope) {
	mcLoggingScopeNew.Spec.Placement = mcLoggingScope.Spec.Placement
	mcLoggingScopeNew.Spec.Template = mcLoggingScope.Spec.Template
}
