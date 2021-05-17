// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	vzstring "github.com/verrazzano/verrazzano/pkg/string"
	corev1 "k8s.io/api/core/v1"

	"github.com/go-logr/logr"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
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

type adminStatusUpdateFuncType = func(name types.NamespacedName, newCond clustersv1alpha1.Condition, newClusterStatus clustersv1alpha1.ClusterLevelStatus) error

const retryCount = 3

var (
	retryDelay = 3 * time.Second
)

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
			err := s.performAdminStatusUpdate(msg)
			if err != nil {
				s.Log.Error(err, fmt.Sprintf("processStatusUpdates: failed to update status on admin cluster for %s/%s from cluster %s after %d retries: %s",
					msg.Resource.GetNamespace(), msg.Resource.GetName(),
					msg.NewClusterStatus.Name, retryCount, err.Error()))
			}
		default:
			break
		}
	}
}

// garbageCollect delete resources that have been orphaned
func (s *Syncer) garbageCollect() {
	mcLabel, err := labels.Parse(fmt.Sprintf("%s=%s", constants.LabelVerrazzanoProject, constants.LabelVerrazzanoProjectDefault))
	if err != nil {
		s.Log.Error(err, "failed to create list selector on local cluster")
	}
	listOptionsGC := &client.ListOptions{LabelSelector: mcLabel}

	// Get the list of namespaces that were created or managed by VerrazzanoProjects
	vpNamespaceList := corev1.NamespaceList{}
	err = s.LocalClient.List(s.Context, &vpNamespaceList, listOptionsGC)
	if err != nil {
		s.Log.Error(err, "failed to get list of namespaces")
	}

	// Perform garbage collection on namespaces that are no longer associated with a VerrazzanoProject
	for _, namespace := range vpNamespaceList.Items {
		if !vzstring.SliceContainsString(s.ProjectNamespaces, namespace.Name) {
			allLocalMCAppConfigs := clustersv1alpha1.MultiClusterApplicationConfigurationList{}
			listOptions := &client.ListOptions{Namespace: namespace.Name}
			err = s.LocalClient.List(s.Context, &allLocalMCAppConfigs, listOptions)
			if err != nil {
				s.Log.Error(err, "failed to list MultiClusterApplicationConfiguration on local cluster")
			}
			for _, mcAppConfig := range allLocalMCAppConfigs.Items {
				// Delete each MultiClusterApplicationConfiguration object that is no longer exists on the admin cluster
				var appConfig clustersv1alpha1.MultiClusterApplicationConfiguration
				err := s.AdminClient.Get(s.Context, types.NamespacedName{Name: mcAppConfig.Name, Namespace: mcAppConfig.Namespace}, &appConfig)
				if errors.IsNotFound(err) {
					err := s.LocalClient.Delete(s.Context, &mcAppConfig)
					if err != nil {
						s.Log.Error(err, fmt.Sprintf("failed to delete MultiClusterApplicationConfiguration with name %q and namespace %q", mcAppConfig.Name, mcAppConfig.Namespace))
					}
				}
			}
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
	var statusUpdateFunc adminStatusUpdateFuncType
	if strings.Contains(typeName, reflect.TypeOf(clustersv1alpha1.MultiClusterApplicationConfiguration{}).String()) {
		statusUpdateFunc = s.updateMultiClusterAppConfigStatus
	} else if strings.Contains(typeName, reflect.TypeOf(clustersv1alpha1.MultiClusterComponent{}).String()) {
		statusUpdateFunc = s.updateMultiClusterComponentStatus
	} else if strings.Contains(typeName, reflect.TypeOf(clustersv1alpha1.MultiClusterConfigMap{}).String()) {
		statusUpdateFunc = s.updateMultiClusterConfigMapStatus
	} else if strings.Contains(typeName, reflect.TypeOf(clustersv1alpha1.MultiClusterSecret{}).String()) {
		statusUpdateFunc = s.updateMultiClusterSecretStatus
	} else if strings.Contains(typeName, reflect.TypeOf(clustersv1alpha1.VerrazzanoProject{}).String()) {
		statusUpdateFunc = s.updateVerrazzanoProjectStatus
	} else {
		return fmt.Errorf("received status update message for unknown resource type %s", typeName)
	}
	return s.adminStatusUpdateWithRetry(statusUpdateFunc, fullResourceName, msg.NewCondition, msg.NewClusterStatus)
}

func (s *Syncer) adminStatusUpdateWithRetry(statusUpdateFunc adminStatusUpdateFuncType,
	name types.NamespacedName,
	condition clustersv1alpha1.Condition,
	clusterStatus clustersv1alpha1.ClusterLevelStatus) error {
	var err error
	for tries := 0; tries < retryCount; tries++ {
		err = statusUpdateFunc(name, condition, clusterStatus)
		if err == nil {
			break
		}
		if !errors.IsConflict(err) {
			break
		}

		time.Sleep(retryDelay)
	}
	return err
}
