// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusters

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// MCLocalRegistrationSecretFullName is the full NamespacedName of the cluster local registration secret
var MCLocalRegistrationSecretFullName = types.NamespacedName{
	Namespace: constants.VerrazzanoSystemNamespace,
	Name:      constants.MCLocalRegistrationSecret}

// MCRegistrationSecretFullName is the full NamespacedName of the cluster registration secret
var MCRegistrationSecretFullName = types.NamespacedName{
	Namespace: constants.VerrazzanoSystemNamespace,
	Name:      constants.MCRegistrationSecret}

// ElasticsearchDetails represents all the details needed
// to determine how to connect to an Elasticsearch instance
type ElasticsearchDetails struct {
	URL        string
	SecretName string
}

// MultiClusterResource interface abstracts methods common to all MultiClusterXXX resource types
// It is defined outside the api resources package since deep-copy code generation cannot handle
// interface types
type MultiClusterResource interface {
	runtime.Object
	GetName() string
	GetNamespace() string
	GetStatus() clustersv1alpha1.MultiClusterResourceStatus
}

// StatusUpdateMessage represents a message sent to the Multi Cluster agent by the controllers
// when a MultiCluster Resource's status is updated, with the updates
type StatusUpdateMessage struct {
	NewCondition     clustersv1alpha1.Condition
	NewClusterStatus clustersv1alpha1.ClusterLevelStatus
	Resource         MultiClusterResource
}

// StatusNeedsUpdate determines based on the current state and conditions of a MultiCluster
// resource, as well as the new state and condition to be set, whether the status update
// needs to be done
func StatusNeedsUpdate(curStatus clustersv1alpha1.MultiClusterResourceStatus,
	newCondition clustersv1alpha1.Condition,
	newClusterStatus clustersv1alpha1.ClusterLevelStatus) bool {

	foundClusterLevelStatus := false
	for _, existingClusterStatus := range curStatus.Clusters {
		if existingClusterStatus.Name == newClusterStatus.Name &&
			existingClusterStatus.State == newClusterStatus.State {
			foundClusterLevelStatus = true
		}
	}

	if !foundClusterLevelStatus {
		return true
	}

	foundCondition := false
	for _, existingCond := range curStatus.Conditions {
		if existingCond.Status == newCondition.Status &&
			existingCond.Message == newCondition.Message &&
			existingCond.Type == newCondition.Type {
			foundCondition = true
			break
		}
	}

	return !foundCondition
}

// GetConditionFromResult - Based on the result of a create/update operation on the
// embedded resource, returns the Condition and State that must be set on a MultiCluster
// resource's Status
func GetConditionFromResult(err error, opResult controllerutil.OperationResult, msgPrefix string) clustersv1alpha1.Condition {
	var condition clustersv1alpha1.Condition
	if err != nil {
		condition = clustersv1alpha1.Condition{
			Type:               clustersv1alpha1.DeployFailed,
			Status:             corev1.ConditionTrue,
			Message:            err.Error(),
			LastTransitionTime: time.Now().Format(time.RFC3339),
		}
	} else {
		msg := fmt.Sprintf("%v %v", msgPrefix, opResult)
		condition = clustersv1alpha1.Condition{
			Type:               clustersv1alpha1.DeployComplete,
			Status:             corev1.ConditionTrue,
			Message:            msg,
			LastTransitionTime: time.Now().Format(time.RFC3339),
		}
	}
	return condition
}

// CreateClusterLevelStatus creates and returns a ClusterLevelStatus object based on the condition
// of an operation on a cluster
func CreateClusterLevelStatus(condition clustersv1alpha1.Condition, clusterName string) clustersv1alpha1.ClusterLevelStatus {
	var state clustersv1alpha1.StateType
	if condition.Type == clustersv1alpha1.DeployComplete {
		state = clustersv1alpha1.Succeeded
	} else if condition.Type == clustersv1alpha1.DeployFailed {
		state = clustersv1alpha1.Failed
	} else {
		state = clustersv1alpha1.Pending
	}
	return clustersv1alpha1.ClusterLevelStatus{
		Name: clusterName, State: state, LastUpdateTime: condition.LastTransitionTime}
}

// ComputeEffectiveState computes the overall state of the multi cluster resource from the statuses
// at the level of the individual clusters it is placed in
func ComputeEffectiveState(status clustersv1alpha1.MultiClusterResourceStatus, placement clustersv1alpha1.Placement) clustersv1alpha1.StateType {
	clustersSucceeded := 0
	clustersFound := 0
	clustersPending := 0
	clustersFailed := 0

	// In some cases (such as VerrazzanoProject, which has no placement section), there may not be
	// any specified cluster placements. In this case assume the known placements as the ones for
	// which we have received updates from the cluster
	knownClusterPlacements := placement.Clusters
	if knownClusterPlacements == nil {
		for _, clusterStatus := range status.Clusters {
			knownClusterPlacements = append(knownClusterPlacements, clustersv1alpha1.Cluster{Name: clusterStatus.Name})
		}
	}

	for _, cluster := range knownClusterPlacements {
		for _, clusterStatus := range status.Clusters {
			if clusterStatus.Name == cluster.Name {
				clustersFound++
				if clusterStatus.State == clustersv1alpha1.Pending {
					clustersPending++
				} else if clusterStatus.State == clustersv1alpha1.Succeeded {
					clustersSucceeded++
				} else if clusterStatus.State == clustersv1alpha1.Failed {
					clustersFailed++
				}
			}
		}
	}
	// If any cluster has a failed status, mark the overall state as failed
	if clustersFailed > 0 {
		return clustersv1alpha1.Failed
	}

	// if all clusters succeeded, mark the overall state as succeeded
	if clustersSucceeded == len(knownClusterPlacements) {
		return clustersv1alpha1.Succeeded
	}

	// otherwise, overall state is pending
	return clustersv1alpha1.Pending
}

// SetClusterLevelStatus - given a multi cluster resource status object, and a new cluster status
// to be updated, either add or update the cluster status as appropriate
func SetClusterLevelStatus(status *clustersv1alpha1.MultiClusterResourceStatus, newClusterStatus clustersv1alpha1.ClusterLevelStatus) {
	foundClusterIdx := -1
	for i, clusterStatus := range status.Clusters {
		if clusterStatus.Name == newClusterStatus.Name {
			foundClusterIdx = i
		}
	}
	if foundClusterIdx == -1 {
		status.Clusters = append(status.Clusters, newClusterStatus)
	} else {
		status.Clusters[foundClusterIdx] = newClusterStatus
		status.Clusters[foundClusterIdx].LastUpdateTime = time.Now().Format(time.RFC3339)
	}
}

// NewScheme creates a new scheme that includes this package's object to use for testing
func NewScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	clustersv1alpha1.AddToScheme(scheme)
	return scheme
}

// IsPlacedInThisCluster determines whether the given Placement represents placement in the current
// cluster. Current cluster's identity is determined from the verrazzano-cluster secret
func IsPlacedInThisCluster(ctx context.Context, rdr client.Reader, placement clustersv1alpha1.Placement) bool {
	var clusterSecret corev1.Secret

	err := fetchClusterSecret(ctx, rdr, &clusterSecret)
	if err != nil {
		return false
	}
	thisCluster := string(clusterSecret.Data[constants.ClusterNameData])
	for _, placementCluster := range placement.Clusters {
		if thisCluster == placementCluster.Name {
			return true
		}
	}

	return false
}

// IgnoreNotFoundWithLog returns nil if err is a "Not Found" error, and if not, logs an error
// message that the resource could not be fetched and returns the original error
func IgnoreNotFoundWithLog(resourceType string, err error, logger logr.Logger) error {
	if apierrors.IsNotFound(err) {
		return nil
	}
	logger.Info("Failed to fetch resource", "type", resourceType, "err", err)
	return err
}

// FetchManagedClusterElasticSearchDetails fetches Elasticsearch details to use for system and
// application logs on this managed cluster. If this cluster is NOT a managed cluster (i.e. does not
// have the managed cluster secret), an empty ElasticsearchDetails will be returned
func FetchManagedClusterElasticSearchDetails(ctx context.Context, rdr client.Reader) ElasticsearchDetails {
	esDetails := ElasticsearchDetails{}
	regSecret := corev1.Secret{}
	err := rdr.Get(ctx, MCRegistrationSecretFullName, &regSecret)
	if err != nil {
		return esDetails
	}
	esDetails.URL = string(regSecret.Data[constants.ElasticsearchURLData])
	esDetails.SecretName = constants.MCRegistrationSecret
	return esDetails
}

// GetClusterName returns the cluster name for a managed cluster, empty string otherwise
func GetClusterName(ctx context.Context, rdr client.Reader) string {
	clusterSecret := corev1.Secret{}
	err := fetchClusterSecret(ctx, rdr, &clusterSecret)
	if err != nil {
		return ""
	}
	return string(clusterSecret.Data[constants.ClusterNameData])
}

// Try to get the registration secret that was created via the registration YAML apply.  If it doesn't
// exist then use the local registration secret that was created at install time.
func fetchClusterSecret(ctx context.Context, rdr client.Reader, clusterSecret *corev1.Secret) error {
	err := rdr.Get(ctx, MCRegistrationSecretFullName, clusterSecret)
	if err == nil {
		return nil
	}
	if !apierrors.IsNotFound(err) {
		return err
	}
	return rdr.Get(ctx, MCLocalRegistrationSecretFullName, clusterSecret)
}

// UpdateStatus determines whether a status update is needed for the specified mcStatus, given the new
// Condition to be added, and if so, computes the state and calls the callback function to perform
// the status update
func UpdateStatus(resource MultiClusterResource, mcStatus *clustersv1alpha1.MultiClusterResourceStatus, placement clustersv1alpha1.Placement, newCondition clustersv1alpha1.Condition, clusterName string, agentChannel chan StatusUpdateMessage, updateFunc func() error) (controllerruntime.Result, error) {

	clusterLevelStatus := CreateClusterLevelStatus(newCondition, clusterName)

	if StatusNeedsUpdate(*mcStatus, newCondition, clusterLevelStatus) {
		mcStatus.Conditions = append(mcStatus.Conditions, newCondition)
		SetClusterLevelStatus(mcStatus, clusterLevelStatus)
		mcStatus.State = ComputeEffectiveState(*mcStatus, placement)
		err := updateFunc()
		if err != nil {
			return reconcile.Result{}, err
		}
		if agentChannel != nil {
			// put the status update itself on the agent channel.
			// note that the send will block if the channel buffer is full, which means the
			// reconcile operation will not complete till unblocked
			msg := StatusUpdateMessage{
				NewCondition:     newCondition,
				NewClusterStatus: clusterLevelStatus,
				Resource:         resource,
			}
			agentChannel <- msg
		}
	}
	return reconcile.Result{}, nil
}

// SetEffectiveStateIfChanged - if the effective state of the resource has changed, set it on the
// in-memory multicluster resource's status. Returns the previous state, whether changed or not
func SetEffectiveStateIfChanged(placement clustersv1alpha1.Placement,
	statusPtr *clustersv1alpha1.MultiClusterResourceStatus) clustersv1alpha1.StateType {

	effectiveState := ComputeEffectiveState(*statusPtr, placement)
	if effectiveState != statusPtr.State {
		oldState := statusPtr.State
		statusPtr.State = effectiveState
		return oldState
	}
	return statusPtr.State
}
