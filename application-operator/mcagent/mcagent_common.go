// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-logr/logr"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
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
	AgentSecretValid      bool
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
// received messages and processes a batch of them
func (s *Syncer) processStatusUpdates() {
	for i := 0; i < constants.StatusUpdateBatchSize; i++ {
		// Use a select with default so as to not block on the channel if there are no updates
		select {
		case msg := <-s.StatusUpdateChannel:
			s.Log.Info(fmt.Sprintf("processStatusUpdates: Received status update %s with condition type %s for %s/%s from cluster %s",
				msg.NewClusterStatus.State, msg.NewCondition.Type, msg.Resource.GetNamespace(), msg.Resource.GetName(), msg.NewClusterStatus.Name))
			err := s.performAdminStatusUpdate(msg)
			if err != nil {
				if errors.IsConflict(err) {
					// Retry on conflict - put the message back on the channel, but don't block
					// trying to do it - if we block, there could be a deadlock since the agent
					// thread is both reading and writing to the channel
					s.requeueMsgNonBlocking(msg, err)
				} else {
					s.Log.Error(err, fmt.Sprintf("processStatusUpdates: status update failed for %s/%s from cluster %s: %s",
						msg.Resource.GetNamespace(), msg.Resource.GetName(),
						msg.NewClusterStatus.Name, err.Error()))
				}
			}
		default:
			break
		}
	}
}

// AgentReadyToSync - the agent has all the information it needs to sync resources i.e.
// there is an agent secret and a kubernetes client to the Admin cluster was created
func (s *Syncer) AgentReadyToSync() bool {
	return s.AgentSecretFound && s.AgentSecretValid
}

func (s *Syncer) performAdminStatusUpdate(msg clusters.StatusUpdateMessage) error {
	fullResourceName := types.NamespacedName{Name: msg.Resource.GetName(), Namespace: msg.Resource.GetNamespace()}
	typeName := reflect.TypeOf(msg.Resource).String()
	if strings.Contains(typeName, reflect.TypeOf(clustersv1alpha1.MultiClusterApplicationConfiguration{}).String()) {
		return s.updateMultiClusterAppConfigStatus(fullResourceName, msg.NewCondition, msg.NewClusterStatus)
	} else if strings.Contains(typeName, reflect.TypeOf(clustersv1alpha1.MultiClusterComponent{}).String()) {
		return s.updateMultiClusterComponentStatus(fullResourceName, msg.NewCondition, msg.NewClusterStatus)
	} else if strings.Contains(typeName, reflect.TypeOf(clustersv1alpha1.MultiClusterConfigMap{}).String()) {
		return s.updateMultiClusterConfigMapStatus(fullResourceName, msg.NewCondition, msg.NewClusterStatus)
	} else if strings.Contains(typeName, reflect.TypeOf(clustersv1alpha1.MultiClusterLoggingScope{}).String()) {
		return s.updateMultiClusterLoggingScopeStatus(fullResourceName, msg.NewCondition, msg.NewClusterStatus)
	} else if strings.Contains(typeName, reflect.TypeOf(clustersv1alpha1.MultiClusterSecret{}).String()) {
		return s.updateMultiClusterSecretStatus(fullResourceName, msg.NewCondition, msg.NewClusterStatus)
	} else if strings.Contains(typeName, reflect.TypeOf(clustersv1alpha1.VerrazzanoProject{}).String()) {
		return s.updateVerrazzanoProjectStatus(fullResourceName, msg.NewCondition, msg.NewClusterStatus)
	}
	return fmt.Errorf("received status update message for unknown resource type %s", typeName)
}

func (s *Syncer) requeueMsgNonBlocking(msg clusters.StatusUpdateMessage, conflictErr error) {
	select {
	case s.StatusUpdateChannel <- msg:
		s.Log.Info(fmt.Sprintf("processStatusUpdates: requeued status update with conflict for %s/%s from cluster %s: %s",
			msg.Resource.GetNamespace(), msg.Resource.GetName(), msg.NewClusterStatus.Name, conflictErr.Error()))
	default:
		s.Log.Error(conflictErr, fmt.Sprintf("processStatusUpdates: failed to requeue status update with conflict for %s/%s from cluster %s",
			msg.Resource.GetNamespace(), msg.Resource.GetName(), msg.NewClusterStatus.Name))
	}
}
