// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	k8score "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const ADMIN_PROJECTS_NAMESPACE = "verrazzano-mc"

// Check Project resources in the verrazzano-mc namespace on the admin cluster
// and create namespaces specified in the Project resources in the local cluster
func (s *Syncer) syncVerrazzanoProjects() error {
	// Get all the Project objects from the admin cluster
	allProjects := &clustersv1alpha1.VerrazzanoProjectList{}
	err := s.AdminClient.List(s.Context, allProjects)
	if err != nil {
		return client.IgnoreNotFound(err)
	}

	// Create namespaces specified in the Project resources in the local cluster
	for _, project := range allProjects.Items {
		if project.Namespace == ADMIN_PROJECTS_NAMESPACE {
			for _, namespace := range project.Spec.Namespaces {
				nsSpec := &k8score.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
				s.Log.Info("Creating/Updating namespace", namespace)
				_, err = controllerutil.CreateOrUpdate(s.Context, s.MCClient, nsSpec, func() error {
					return nil
				})
				if err != nil {
					s.Log.Error(err, "Error creatig/updating namespace", namespace)
				}
			}
		}
	}

	return nil
}
