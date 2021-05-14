// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/go-logr/logr"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
	vzstring "github.com/verrazzano/verrazzano/pkg/string"
	corev1 "k8s.io/api/core/v1"
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

// garbageCollect
// 	Remove multicluster resources that reside in namespaces that are no longer associated with
// 	a VerrazzanoProject and either do not exist on the admin cluster or are no longer placed on this cluster.
func (s *Syncer) garbageCollect() {
	mcLabel, err := labels.Parse(fmt.Sprintf("%s=%s", constants.LabelVerrazzanoManaged, constants.LabelVerrazzanoManagedDefault))
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

	// Create table that drives garbage collection for each multicluster resource type
	type gcObject struct {
		ObjectList clusters.MultiClusterResourceList
		Object     clusters.MultiClusterResource
	}
	gcObjectArray := []gcObject{
		{
			ObjectList: &clustersv1alpha1.MultiClusterApplicationConfigurationList{},
			Object:     &clustersv1alpha1.MultiClusterApplicationConfiguration{},
		},
		{
			ObjectList: &clustersv1alpha1.MultiClusterComponentList{},
			Object:     &clustersv1alpha1.MultiClusterComponent{},
		},
		{
			ObjectList: &clustersv1alpha1.MultiClusterConfigMapList{},
			Object:     &clustersv1alpha1.MultiClusterConfigMap{},
		},
		{
			ObjectList: &clustersv1alpha1.MultiClusterSecretList{},
			Object:     &clustersv1alpha1.MultiClusterSecret{},
		},
	}

	// Perform garbage collection on namespaces that are no longer associated with a VerrazzanoProject
	for _, namespace := range vpNamespaceList.Items {
		for _, mcObject := range gcObjectArray {
			if !vzstring.SliceContainsString(s.ProjectNamespaces, namespace.Name) {
				listOptions := &client.ListOptions{Namespace: namespace.Name}
				err = s.LocalClient.List(s.Context, mcObject.ObjectList, listOptions)
				if err != nil {
					s.Log.Error(err, "failed to list MultiClusterApplicationConfiguration on local cluster")
				}
				// Delete resources that are on the local cluster but no longer on the admin cluster or placed on this cluster
				for _, item := range mcObject.ObjectList.GetItems() {
					mcItem := item.(clusters.MultiClusterResource)
					err := s.AdminClient.Get(s.Context, types.NamespacedName{Name: mcItem.GetName(), Namespace: mcItem.GetNamespace()}, mcObject.Object)
					if errors.IsNotFound(err) || (err != nil && !s.isThisCluster(mcObject.Object.GetPlacement())) {
						s.Log.Info(fmt.Sprintf("perfoming garbage collection on %s with name %s in namespace %s", item.GetObjectKind().GroupVersionKind().Kind, mcItem.GetName(), mcItem.GetNamespace()))
						err := s.LocalClient.Delete(s.Context, mcItem)
						if err != nil {
							s.Log.Error(err, fmt.Sprintf("failed to delete %s with name %s in namespace %s", mcItem.GetObjectKind().GroupVersionKind().Kind, mcItem.GetName(), mcItem.GetNamespace()))
						}
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
