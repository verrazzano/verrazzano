// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	"fmt"

	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
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
			_, err := s.createOrUpdateMCAppConfig(mcAppConfig)
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
	for _, mcAppConfig := range allLocalMCAppConfigs.Items {
		// Delete each MultiClusterApplicationConfiguration object that is not on the admin cluster
		if !appConfigListContains(&allAdminMCAppConfigs, mcAppConfig.Name, mcAppConfig.Namespace) {
			err := s.LocalClient.Delete(s.Context, &mcAppConfig)
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

// appConfigListContains returns boolean indicating if the list contains the object with the specified name and namespace
func appConfigListContains(mcAdminList *clustersv1alpha1.MultiClusterApplicationConfigurationList, name string, namespace string) bool {
	for _, item := range mcAdminList.Items {
		if item.Name == name && item.Namespace == namespace {
			return true
		}
	}
	return false
}
