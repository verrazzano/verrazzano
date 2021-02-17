// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Check Project resources in the verrazzano-mc namespace on the admin cluster
// and create namespaces specified in the Project resources in the local cluster
func (s *Syncer) syncVerrazzanoProjects() error {
	// Get all the Project objects from the admin cluster
	allProjects := &clustersv1alpha1.VerrazzanoProjectList{}
	err := s.AdminClient.List(s.Context, allProjects)
	if err != nil {
		return client.IgnoreNotFound(err)
	}

	// Write each of the records in verrazzano-mc namespace
	for _, vp := range allProjects.Items {
		if vp.Namespace == constants.VerrazzanoMultiClusterNamespace {
			_, err := s.createOrUpdateVerrazzanoProject(vp)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Create or update a VerrazzanoProject
func (s *Syncer) createOrUpdateVerrazzanoProject(vp clustersv1alpha1.VerrazzanoProject) (controllerutil.OperationResult, error) {
	var vpNew clustersv1alpha1.VerrazzanoProject
	vpNew.Namespace = vp.Namespace
	vpNew.Name = vp.Name

	// Create or update on the local cluster
	return controllerutil.CreateOrUpdate(s.Context, s.LocalClient, &vpNew, func() error {
		vpNew.Spec.Namespaces = vp.Spec.Namespaces
		return nil
	})
}
