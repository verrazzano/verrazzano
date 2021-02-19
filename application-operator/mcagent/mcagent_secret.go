// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Synchronize MultiClusterSecret objects to the local cluster
func (s *Syncer) syncMCSecretObjects() error {
	// Get all the MultiClusterSecret objects from the admin cluster
	allMCSecrets := clustersv1alpha1.MultiClusterSecretList{}
	listOptions := &client.ListOptions{}
	err := s.AdminClient.List(s.Context, &allMCSecrets, listOptions)
	if err != nil {
		return client.IgnoreNotFound(err)
	}

	// Write each of the records that are targeted to this cluster
	for _, mcSecret := range allMCSecrets.Items {
		if s.isThisCluster(mcSecret.Spec.Placement) {
			_, err := s.createOrUpdateMCSecret(mcSecret)
			if err != nil {
				s.Log.Error(err, "Error syncing object",
					"MultiClusterSecret",
					types.NamespacedName{Namespace: mcSecret.Namespace, Name: mcSecret.Name})
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

// mutateMCSecret mutates the MultiClusterSecret to reflect the contents of the parent MultiClusterSecret
func mutateMCSecret(mcSecret clustersv1alpha1.MultiClusterSecret, mcSecretNew *clustersv1alpha1.MultiClusterSecret) {
	mcSecretNew.Spec.Placement = mcSecret.Spec.Placement
	mcSecretNew.Spec.Template = mcSecret.Spec.Template
	mcSecretNew.Labels = mcSecret.Labels
}
