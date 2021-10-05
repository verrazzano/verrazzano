// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	"fmt"
	"strings"

	oamv1alpha2 "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
	vzstring "github.com/verrazzano/verrazzano/pkg/string"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Synchronize MultiClusterApplicationConfiguration objects to the local cluster
func (s *Syncer) syncMCApplicationConfigurationObjects(namespace string) error {
	// Get all the MultiClusterApplicationConfiguration objects from the admin cluster
	allAdminMCAppConfigs := clustersv1alpha1.MultiClusterApplicationConfigurationList{}
	listOptions := &client.ListOptions{Namespace: namespace}
	err := s.AdminClient.List(s.Context, &allAdminMCAppConfigs, listOptions)
	if err != nil {
		return client.IgnoreNotFound(err)
	}
	for _, mcAppConfig := range allAdminMCAppConfigs.Items {
		if s.isThisCluster(mcAppConfig.Spec.Placement) {
			// Synchronize the components referenced by the application
			err := s.syncComponentList(mcAppConfig)
			if err != nil {
				s.Log.Error(err, "Error syncing components referenced by object",
					"MultiClusterApplicationConfiguration",
					types.NamespacedName{Namespace: mcAppConfig.Namespace, Name: mcAppConfig.Name})
			}
			// Synchronize the MultiClusterApplicationConfiguration even if there were errors
			// handling the application components.  For compatibility with v1.0.0 it is valid
			// for none of the OAM Components to be found because they may all be wrapped in
			// an MultiClusterComponent resource.
			_, err = s.createOrUpdateMCAppConfig(mcAppConfig)
			if err != nil {
				s.Log.Error(err, "Error syncing object",
					"MultiClusterApplicationConfiguration",
					types.NamespacedName{Namespace: mcAppConfig.Namespace, Name: mcAppConfig.Name})
			}
		}
	}

	// Delete orphaned MultiClusterApplicationConfiguration resources.
	// Get the list of MultiClusterApplicationConfiguration resources on the
	// local cluster and compare to the list received from the admin cluster.
	// The admin cluster is the source of truth.
	allLocalMCAppConfigs := clustersv1alpha1.MultiClusterApplicationConfigurationList{}
	err = s.LocalClient.List(s.Context, &allLocalMCAppConfigs, listOptions)
	if err != nil {
		s.Log.Error(err, "failed to list MultiClusterApplicationConfiguration on local cluster")
		return nil
	}
	for i, mcAppConfig := range allLocalMCAppConfigs.Items {
		// Delete each MultiClusterApplicationConfiguration object that is not on the admin cluster or no longer placed on this cluster
		if !s.appConfigPlacedOnCluster(&allAdminMCAppConfigs, mcAppConfig.Name, mcAppConfig.Namespace) {
			err := s.LocalClient.Delete(s.Context, &allLocalMCAppConfigs.Items[i])
			if err != nil {
				s.Log.Error(err, fmt.Sprintf("failed to delete MultiClusterApplicationConfiguration with name %q in namespace %q", mcAppConfig.Name, mcAppConfig.Namespace))
			}
			// Delete the OAM components listed in application
			err = s.deleteComponentList(mcAppConfig)
			if err != nil {
				s.Log.Error(err, fmt.Sprintf("error deleting the OAM Components listed in the MultiClusterApplicationConfiguration with name %q in namespace %q", mcAppConfig.Name, mcAppConfig.Namespace))
			}
		}
	}

	return nil
}

func (s *Syncer) createOrUpdateMCAppConfig(mcAppConfig clustersv1alpha1.MultiClusterApplicationConfiguration) (controllerutil.OperationResult, error) {
	var mcAppConfigNew clustersv1alpha1.MultiClusterApplicationConfiguration
	mcAppConfigNew.Namespace = mcAppConfig.Namespace
	mcAppConfigNew.Name = mcAppConfig.Name

	// Create or update on the local cluster
	return controllerutil.CreateOrUpdate(s.Context, s.LocalClient, &mcAppConfigNew, func() error {
		mutateMCAppConfig(mcAppConfig, &mcAppConfigNew)
		return nil
	})
}

func mutateMCAppConfig(mcAppConfig clustersv1alpha1.MultiClusterApplicationConfiguration, mcAppConfigNew *clustersv1alpha1.MultiClusterApplicationConfiguration) {
	mcAppConfigNew.Spec.Placement = mcAppConfig.Spec.Placement
	mcAppConfigNew.Spec.Template = mcAppConfig.Spec.Template
	mcAppConfigNew.Labels = mcAppConfig.Labels
}

// appConfigPlacedOnCluster returns boolean indicating if the list contains the object with the specified name and namespace and the placement
// includes the local cluster
func (s *Syncer) appConfigPlacedOnCluster(mcAdminList *clustersv1alpha1.MultiClusterApplicationConfigurationList, name string, namespace string) bool {
	for _, item := range mcAdminList.Items {
		if item.Name == name && item.Namespace == namespace {
			return s.isThisCluster(item.Spec.Placement)
		}
	}
	return false
}

func (s *Syncer) updateMultiClusterAppConfigStatus(name types.NamespacedName, newCond clustersv1alpha1.Condition, newClusterStatus clustersv1alpha1.ClusterLevelStatus) error {
	var fetched clustersv1alpha1.MultiClusterApplicationConfiguration
	err := s.AdminClient.Get(s.Context, name, &fetched)
	if err != nil {
		return err
	}
	fetched.Status.Conditions = append(fetched.Status.Conditions, newCond)
	clusters.SetClusterLevelStatus(&fetched.Status, newClusterStatus)
	return s.AdminClient.Status().Update(s.Context, &fetched)
}

// syncComponentList - Synchronize the list of OAM Components contained in the MultiClusterApplicationConfiguration
func (s *Syncer) syncComponentList(mcAppConfig clustersv1alpha1.MultiClusterApplicationConfiguration) error {
	var errorStrings []string

	// Loop through the component list and get them one at a time.
	for _, component := range mcAppConfig.Spec.Template.Spec.Components {
		objectKey := types.NamespacedName{Name: component.ComponentName, Namespace: mcAppConfig.Namespace}
		oamComp := &oamv1alpha2.Component{}
		err := s.AdminClient.Get(s.Context, objectKey, oamComp)
		if err != nil {
			return err
		}
		_, err = s.createOrUpdateComponent(*oamComp, mcAppConfig.Name)
		if err != nil {
			errorStrings = append(errorStrings, err.Error())
		}
	}

	// Check if any errors were collected while processing the list
	if len(errorStrings) > 0 {
		return fmt.Errorf(strings.Join(errorStrings, "\n"))
	}
	return nil
}

// deleteComponentList - Delete the list of OAM Components contained in the MultiClusterApplicationConfiguration
//                       that is being deleted.
func (s *Syncer) deleteComponentList(mcAppConfig clustersv1alpha1.MultiClusterApplicationConfiguration) error {
	var errorStrings []string

	// Get list of OAM Applications from the local client
	listOptions := &client.ListOptions{Namespace: mcAppConfig.Namespace}
	appList := clustersv1alpha1.MultiClusterApplicationConfigurationList{}
	err := s.LocalClient.List(s.Context, &appList, listOptions)
	if err != nil {
		return err
	}

	// Loop through the component list and get them one at a time.
	for _, component := range mcAppConfig.Spec.Template.Spec.Components {
		// Get the OAM component
		oamComp := oamv1alpha2.Component{}
		err := s.LocalClient.Get(s.Context, types.NamespacedName{Name: component.ComponentName, Namespace: mcAppConfig.Namespace}, &oamComp)
		if err != nil {
			errorStrings = append(errorStrings, err.Error())
			continue
		}

		// Remove this MultiClusterApplicationConfiguration from the label list
		newLabels := vzstring.RemoveFromCommaSeparatedString(oamComp.Labels[mcAppConfigsLabel], mcAppConfig.Name)
		if newLabels == "" {
			// Ok to delete this component because it is not shared by another MultiClusterApplicationConfiguration
			err := s.LocalClient.Delete(s.Context, &oamComp)
			if err != nil {
				errorStrings = append(errorStrings, err.Error())
			}
		} else {
			// Update the OAM Component labels to remove the name of the MultiClusterApplicationConfiguration that is deleted
			oamComp.Labels[mcAppConfigsLabel] = newLabels
			err := s.LocalClient.Update(s.Context, &oamComp)
			if err != nil {
				errorStrings = append(errorStrings, err.Error())
			}
		}
	}

	// Check if any errors were collected while processing the list
	if len(errorStrings) > 0 {
		return fmt.Errorf(strings.Join(errorStrings, "\n"))
	}
	return nil
}

// createOrUpdateComponent - create or update an OAM Component
func (s *Syncer) createOrUpdateComponent(srcComp oamv1alpha2.Component, mcAppConfigName string) (controllerutil.OperationResult, error) {
	var oamComp oamv1alpha2.Component
	oamComp.Namespace = srcComp.Namespace
	oamComp.Name = srcComp.Name

	return controllerutil.CreateOrUpdate(s.Context, s.LocalClient, &oamComp, func() error {
		s.mutateComponent(s.ManagedClusterName, mcAppConfigName, srcComp, &oamComp)
		return nil
	})
}

// mutateComponent mutates the OAM component to reflect the contents of the parent MultiClusterComponent
func (s *Syncer) mutateComponent(managedClusterName string, mcAppConfigName string, component oamv1alpha2.Component, componentNew *oamv1alpha2.Component) {
	// Initialize the labels field
	if componentNew.Labels == nil {
		componentNew.Labels = make(map[string]string)
		if component.Labels != nil {
			componentNew.Labels = component.Labels
		}
	}

	// Add the name of this MultiClusterApplicationConfiguration to the label list
	componentNew.Labels[mcAppConfigsLabel] = vzstring.AppendToCommaSeparatedString(componentNew.Labels[mcAppConfigsLabel], mcAppConfigName)

	componentNew.Labels[managedClusterLabel] = managedClusterName
	componentNew.Spec = component.Spec
	componentNew.Annotations = component.Annotations
}
