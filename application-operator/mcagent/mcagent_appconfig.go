// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	"fmt"

	oamv1alpha2 "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
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
				s.Log.Error(err, fmt.Sprintf("failed to delete MultiClusterApplicationConfiguration with name %q and namespace %q", mcAppConfig.Name, mcAppConfig.Namespace))
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
	// Loop through the component list and get them one at a time.
	for _, component := range mcAppConfig.Spec.Template.Spec.Components {
		objectKey := types.NamespacedName{Name: component.ComponentName, Namespace: mcAppConfig.Namespace}
		oamComp := &oamv1alpha2.Component{}
		err := s.AdminClient.Get(s.Context, objectKey, oamComp)
		if err != nil {
			return err
		}
		_, err = s.createOrUpdateComponent(*oamComp)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Syncer) createOrUpdateComponent(srcComp oamv1alpha2.Component) (controllerutil.OperationResult, error) {
	var oamComp oamv1alpha2.Component
	oamComp.Namespace = srcComp.Namespace
	oamComp.Name = srcComp.Name

	return controllerutil.CreateOrUpdate(s.Context, s.LocalClient, &oamComp, func() error {
		s.mutateComponent(srcComp, &oamComp)
		return nil
	})
}

// mutateComponent mutates the OAM component to reflect the contents of the parent MultiClusterComponent
func (s *Syncer) mutateComponent(srcComp oamv1alpha2.Component, localComp *oamv1alpha2.Component) {
	localComp.Spec = srcComp.Spec
	localComp.Labels = srcComp.Labels
	localComp.Annotations = srcComp.Annotations
}
