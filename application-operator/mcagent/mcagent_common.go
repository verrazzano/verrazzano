// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Syncer contains context for synchronize operations
type Syncer struct {
	AdminClient           client.Client
	LocalClient           client.Client
	Log                   logr.Logger
	ManagedClusterName    string
	Context               context.Context
	AgentSecretFound      bool
	SecretResourceVersion string

	// List of namespaces to watch for multi-cluster objects.
	ProjectNamespaces   []string
	StatusUpdateChannel chan clusters.StatusUpdateMessage
}

// Check if the placement is for this cluster
func (s *Syncer) isThisCluster(placement clustersv1alpha1.Placement) bool {
	// Loop through the cluster list looking for the cluster name
	for _, cluster := range placement.Clusters {
		if cluster.Name == s.ManagedClusterName {
			return true
		}
	}
	return false
}

// processStatusUpdates monitors the StatusUpdateChannel for any 
// received messages and processes a batch of 10
func (s *Syncer) processStatusUpdates() {
	s.Log.Info("processStatusUpdates: starting")
	for i:=0;i<10;i++ {
		// Use a select with default so as to not block on the channel if there are no updates
		select {
		case msg := <-s.StatusUpdateChannel:
			s.Log.Info(fmt.Sprintf("processStatusUpdates: Received status update %s with condition type %s for %s/%s from cluster %s",
				msg.NewClusterStatus.State, msg.NewCondition.Type, msg.ResourceNamespace, msg.ResourceName, msg.NewClusterStatus.Name))
		default:
			s.Log.Info("No status updates available, exiting processStatusUpdates")
			break
		}
	}
}
