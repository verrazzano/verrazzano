// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Synchronize MultiClusterConfigMap objects to the local cluster
func (s *Syncer) syncMCConfigMapObjects() error {
	// Get all the MultiClusterConfigMap objects from the admin cluster
	allMCConfigMaps := clustersv1alpha1.MultiClusterConfigMapList{}
	err := s.AdminClient.List(s.Context, &allMCConfigMaps)
	if err != nil {
		return client.IgnoreNotFound(err)
	}
	return nil
}
