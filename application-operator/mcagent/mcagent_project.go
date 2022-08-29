// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	"fmt"

	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Check Project resources in the verrazzano-mc namespace on the admin cluster
// and create namespaces specified in the Project resources in the local cluster
func (s *Syncer) syncVerrazzanoProjects() error {
	// Get all the Project objects from the admin cluster
	allAdminProjects := &clustersv1alpha1.VerrazzanoProjectList{}
	listOptions := &client.ListOptions{Namespace: constants.VerrazzanoMultiClusterNamespace}
	err := s.AdminClient.List(s.Context, allAdminProjects, listOptions)
	if err != nil {
		return client.IgnoreNotFound(err)
	}

	// Write each of the records in verrazzano-mc namespace
	for _, vp := range allAdminProjects.Items {
		if vp.Namespace == constants.VerrazzanoMultiClusterNamespace {
			if s.isThisCluster(vp.Spec.Placement) {
				_, err := s.createOrUpdateVerrazzanoProject(vp)
				if err != nil {
					s.Log.Errorw(fmt.Sprintf("Failed syncing object: %v", err),
						"VerrazzanoProject",
						types.NamespacedName{Namespace: vp.Namespace, Name: vp.Name})
				}
			} else {
				// Remove the VerrazzanoProject resource if it is on the local cluster but no longer
				// contains placements for this cluster.
				vpLocal := clustersv1alpha1.VerrazzanoProject{}
				err := s.LocalClient.Get(s.Context, types.NamespacedName{Namespace: vp.Namespace, Name: vp.Name}, &vpLocal)
				if err == nil {
					s.Log.Debugf("Deleting VerrazzanoProject %q from namespace %q because it is no longer targetted at this cluster", vp.Name, vp.Namespace)
					err2 := s.LocalClient.Delete(s.Context, &vpLocal)
					if err2 != nil {
						s.Log.Errorf("failed to delete VerrazzanoProject with name %q and namespace %q: %v", vp.Name, vp.Namespace, err2)
					}
				}
			}
		}
	}

	// Delete orphaned VerrazzanoProject resources.
	// Get the list of VerrazzanoProject resources on the
	// local cluster and compare to the list received from the admin cluster.
	// The admin cluster is the source of truth.
	allLocalProjects := clustersv1alpha1.VerrazzanoProjectList{}
	err = s.LocalClient.List(s.Context, &allLocalProjects, listOptions)
	if err != nil {
		s.Log.Errorf("Failed to list VerrazzanoProject on local cluster: %v", err)
		return nil
	}
	for i, project := range allLocalProjects.Items {
		// Delete each VerrazzanoProject object that is not on the admin cluster
		if !projectListContains(allAdminProjects, project.Name, project.Namespace) {
			err := s.LocalClient.Delete(s.Context, &allLocalProjects.Items[i])
			if err != nil {
				s.Log.Errorf("Failed to delete VerrazzanoProject with name %q and namespace %q: %v", project.Name, project.Namespace, err)
			}
		}
	}

	// Update the list of namespaces being watched for multi-cluster objects
	s.ProjectNamespaces, err = s.getManagedNamespaces()
	if err != nil {
		s.Log.Errorf("Failed to get the list of Verrazzano managed namespaces: %v", err)
		return err
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
		mutateVerrazzanoProject(vp, &vpNew)
		return nil
	})
}

func (s *Syncer) updateVerrazzanoProjectStatus(name types.NamespacedName, newCond clustersv1alpha1.Condition, newClusterStatus clustersv1alpha1.ClusterLevelStatus) error {
	fetched := clustersv1alpha1.VerrazzanoProject{}
	err := s.AdminClient.Get(s.Context, name, &fetched)
	if err != nil {
		return err
	}
	fetched.Status.Conditions = append(fetched.Status.Conditions, newCond)
	clusters.SetClusterLevelStatus(&fetched.Status, newClusterStatus)
	return s.AdminClient.Status().Update(s.Context, &fetched)
}

// mutateVerrazzanoProject mutates the VerrazzanoProject to reflect the contents of the parent VerrazzanoProject
func mutateVerrazzanoProject(vp clustersv1alpha1.VerrazzanoProject, vpNew *clustersv1alpha1.VerrazzanoProject) {
	vpNew.Spec.Template = vp.Spec.Template
	vpNew.Spec.Placement = vp.Spec.Placement
	// Mark the VerrazzanoProject we synced from Admin cluster with verrazzano-managed=true, to
	// distinguish from any (though unlikely) that the user might have created on managed cluster
	if vpNew.Labels == nil {
		vpNew.Labels = map[string]string{}
	}
	if vp.Labels != nil {
		vpNew.Labels = vp.Labels
	}
	vpNew.Labels[vzconst.VerrazzanoManagedLabelKey] = constants.LabelVerrazzanoManagedDefault
	vpNew.Annotations = vp.Annotations
}

// projectListContains returns boolean indicating if the list contains the object with the specified name and namespace
func projectListContains(projectList *clustersv1alpha1.VerrazzanoProjectList, name string, namespace string) bool {
	for _, item := range projectList.Items {
		if item.Name == name && item.Namespace == namespace {
			return true
		}
	}
	return false
}
