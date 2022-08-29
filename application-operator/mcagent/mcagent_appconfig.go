// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	"fmt"
	"reflect"
	"strings"

	oamv1alpha2 "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	vzstring "github.com/verrazzano/verrazzano/pkg/string"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	// When placements are changed a forbidden error can be returned.  In this case,
	// we want to fall through and delete orphaned resources.
	if err != nil && !apierrors.IsNotFound(err) && !apierrors.IsForbidden(err) {
		return err
	}

	for _, mcAppConfig := range allAdminMCAppConfigs.Items {
		if s.isThisCluster(mcAppConfig.Spec.Placement) {
			// Synchronize the components referenced by the application
			err := s.syncComponentList(mcAppConfig)
			if err != nil {
				s.Log.Errorw(fmt.Sprintf("Failed syncing components referenced by object: %v", err),
					"MultiClusterApplicationConfiguration",
					types.NamespacedName{Namespace: mcAppConfig.Namespace, Name: mcAppConfig.Name})
			}
			// Synchronize the MultiClusterApplicationConfiguration even if there were errors
			// handling the application components.  For compatibility with v1.0.0 it is valid
			// for none of the OAM Components to be found because they may all be wrapped in
			// an MultiClusterComponent resource.
			_, err = s.createOrUpdateMCAppConfig(mcAppConfig)
			if err != nil {
				s.Log.Errorw(fmt.Sprintf("Failed syncing object: %c", err),
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
		s.Log.Errorf("Failed to list MultiClusterApplicationConfiguration on local cluster: %v", err)
		return nil
	}
	for i, mcAppConfig := range allLocalMCAppConfigs.Items {
		// Delete each MultiClusterApplicationConfiguration object that is not on the admin cluster or no longer placed on this cluster
		if !s.appConfigPlacedOnCluster(&allAdminMCAppConfigs, mcAppConfig.Name, mcAppConfig.Namespace) {
			err := s.LocalClient.Delete(s.Context, &allLocalMCAppConfigs.Items[i])
			if err != nil {
				s.Log.Errorf("Failed to delete MultiClusterApplicationConfiguration with name %q in namespace %q: %v", mcAppConfig.Name, mcAppConfig.Namespace, err)
			}
		}
	}

	// Delete OAM components no longer associated with any MultiClusterApplicationConfiguration
	err = s.deleteOrphanedComponents(namespace)
	if err != nil {
		s.Log.Errorf("Failed deleting orphaned OAM Components in namespace %q: %v", namespace, err)
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
	// Mark the MC app config we synced from Admin cluster with verrazzano-managed=true, to
	// distinguish from any (though unlikely) that the user might have created directly
	if mcAppConfigNew.Labels == nil {
		mcAppConfigNew.Labels = map[string]string{}
	}
	mcAppConfigNew.Labels[vzconst.VerrazzanoManagedLabelKey] = constants.LabelVerrazzanoManagedDefault
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
			// If the OAM component object is not found then we check if the MultiClusterComponent object exists.
			if apierrors.IsNotFound(err) {
				mcComp := &clustersv1alpha1.MultiClusterComponent{}
				errmc := s.LocalClient.Get(s.Context, objectKey, mcComp)
				// Return the OAM component not found error if we fail to get the MultiClusterComponent
				// with the same name.
				if errmc != nil {
					return err
				}
				// MulticlusterComponent object found so nothing to do
				continue
			} else {
				return err
			}
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
	// Mark the component synced from Admin cluster with verrazzano-managed=true, to distinguish
	// those directly created by user on managed cluster
	componentNew.Labels[vzconst.VerrazzanoManagedLabelKey] = constants.LabelVerrazzanoManagedDefault
	componentNew.Spec = component.Spec
	componentNew.Annotations = component.Annotations
}

// deleteOrphanedComponents - delete OAM components that are no longer associated with any MultiClusterApplicationConfigurations.
// Also update the contents of the mcAppConfigsLabel for a component if the list of applications it is shared by has changed.
func (s *Syncer) deleteOrphanedComponents(namespace string) error {
	// Only process OAM components that were synced to the local system
	labels := &metav1.LabelSelector{
		MatchLabels: map[string]string{
			managedClusterLabel: s.ManagedClusterName,
		},
	}
	selector, err := metav1.LabelSelectorAsSelector(labels)
	if err != nil {
		return err
	}
	listOptions := &client.ListOptions{Namespace: namespace, LabelSelector: selector}
	oamCompList := &oamv1alpha2.ComponentList{}
	err = s.LocalClient.List(s.Context, oamCompList, listOptions)
	if err != nil {
		return err
	}

	// Nothing to do if no OAM components found
	if len(oamCompList.Items) == 0 {
		return nil
	}

	// Get the list of MultiClusterApplicationConfiguration objects
	listOptions2 := &client.ListOptions{Namespace: namespace}
	mcAppConfigList := clustersv1alpha1.MultiClusterApplicationConfigurationList{}
	err = s.LocalClient.List(s.Context, &mcAppConfigList, listOptions2)
	if err != nil {
		return err
	}

	// Process the list of OAM Components checking to see if they are part of any MultiClusterApplicationConfiguration
	for i, oamComp := range oamCompList.Items {
		// Don't delete OAM component objects that have a MultiClusterComponent objects. These will be deleted
		// when syncing MultiClusterComponent objects.
		mcComp := &clustersv1alpha1.MultiClusterComponent{}
		objectKey := types.NamespacedName{Name: oamComp.Name, Namespace: namespace}
		err = s.LocalClient.Get(s.Context, objectKey, mcComp)
		if err == nil {
			continue
		}
		var actualAppConfigs []string
		// Loop through the MultiClusterApplicationConfiguration objects checking for a reference
		for _, mcAppConfig := range mcAppConfigList.Items {
			for _, component := range mcAppConfig.Spec.Template.Spec.Components {
				// If we get a match, maintain a list of applications this OAM component is shared by
				if component.ComponentName == oamComp.Name {
					actualAppConfigs = append(actualAppConfigs, mcAppConfig.Name)
				}
			}
		}
		if len(actualAppConfigs) == 0 {
			// Delete the orphaned OAM Component
			s.Log.Debugf("Deleting orphaned OAM Component %s in namespace %s", oamComp.Name, oamComp.Namespace)
			err = s.LocalClient.Delete(s.Context, &oamCompList.Items[i])
			if err != nil {
				return err
			}
		} else {
			// Has the list of applications this component is associated with changed?  If so update the label.
			if !reflect.DeepEqual(strings.Split(oamComp.Labels[mcAppConfigsLabel], ","), actualAppConfigs) {
				oamComp.Labels[mcAppConfigsLabel] = strings.Join(actualAppConfigs, ",")
				err = s.LocalClient.Update(s.Context, &oamCompList.Items[i])
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}
