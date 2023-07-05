// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusters

import (
	"context"
	"fmt"
	"time"

	vzctrl "github.com/verrazzano/verrazzano/pkg/controller"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"

	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	controllerruntime "sigs.k8s.io/controller-runtime"
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
	GetPlacement() clustersv1alpha1.Placement
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
		Name: clusterName, State: state, Message: condition.Message, LastUpdateTime: condition.LastTransitionTime}
}

// ComputeEffectiveState computes the overall state of the multi cluster resource from the statuses
// at the level of the individual clusters it is placed in
func ComputeEffectiveState(status clustersv1alpha1.MultiClusterResourceStatus, placement clustersv1alpha1.Placement) clustersv1alpha1.StateType {
	clustersSucceeded := 0
	clustersFound := 0
	clustersPending := 0
	clustersFailed := 0

	for _, cluster := range placement.Clusters {
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

	// If any cluster has a pending status, mark the overall state as pending
	if clustersPending > 0 {
		return clustersv1alpha1.Pending
	}

	// if all clusters succeeded, mark the overall state as succeeded
	// The check for ">=" is because placement on the admin cluster is implied.
	if clustersSucceeded >= len(placement.Clusters) {
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
	_ = clustersv1alpha1.AddToScheme(scheme)
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
func IgnoreNotFoundWithLog(err error, log *zap.SugaredLogger) (reconcile.Result, error) {
	if apierrors.IsNotFound(err) {
		log.Debug("Resource has been deleted")
		return reconcile.Result{}, nil
	}
	if err != nil {
		log.Errorf("Failed to fetch resource: %v", err)
	}
	return NewRequeueWithDelay(), nil
}

// GetClusterName returns the cluster name for a this cluster, empty string if the cluster
// name cannot be determined due to an error.
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
		addOrUpdateCondition(mcStatus, newCondition)
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

// addOrUpdateCondition adds or updates the newCondition in the status' list of existing conditions
func addOrUpdateCondition(status *clustersv1alpha1.MultiClusterResourceStatus, condition clustersv1alpha1.Condition) {
	var matchingCondition *clustersv1alpha1.Condition
	for i, existingCondition := range status.Conditions {
		if condition.Type == existingCondition.Type &&
			condition.Status == existingCondition.Status &&
			condition.Message == existingCondition.Message {
			// the exact same condition already exists, don't update
			return
		}
		if condition.Type == existingCondition.Type {
			// use the index here since "existingCondition" is a copy and won't point to the object in the slice
			matchingCondition = &status.Conditions[i]
			break
		}
	}
	if matchingCondition == nil {
		status.Conditions = append(status.Conditions, condition)
	} else {
		matchingCondition.Message = condition.Message
		matchingCondition.Status = condition.Status
		matchingCondition.LastTransitionTime = condition.LastTransitionTime
	}
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

// DeleteAssociatedResource will retrieve and delete the resource specified by the name. It is used to delete
// the underlying resource corresponding to a MultiClusterxxx wrapper resource (e.g. the OAM app config corresponding
// to a MultiClusterApplicationConfiguration)
func DeleteAssociatedResource(ctx context.Context, c client.Client, mcResource client.Object, finalizerName string, resourceToDelete client.Object, name types.NamespacedName) error {
	// Get and delete the associated with the name specified by resourceToDelete
	err := c.Get(ctx, name, resourceToDelete)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
	} else {
		err = c.Delete(ctx, resourceToDelete)
		if err != nil {
			return err
		}
	}

	// Deletion succeeded, now we can remove the finalizer

	// assert the MC object is a controller util Object that can be processed by controllerutil.RemoveFinalizer
	controllerutil.RemoveFinalizer(mcResource, finalizerName)
	err = c.Update(ctx, mcResource)
	if err != nil {
		return err
	}

	return nil
}

// AddFinalizer adds a finalizer and updates the resource if that finalizer is not already attached to the resource
func AddFinalizer(ctx context.Context, r client.Client, obj client.Object, finalizerName string) (controllerruntime.Result, error) {
	if !controllerutil.ContainsFinalizer(obj, finalizerName) {
		controllerutil.AddFinalizer(obj, finalizerName)
		if err := r.Update(ctx, obj); err != nil {
			return controllerruntime.Result{}, err
		}
	}

	return controllerruntime.Result{}, nil
}

// GetRandomRequeueDelay returns a random delay between 2 and 8 secondsto be used for RequeueAfter
func GetRandomRequeueDelay() time.Duration {
	return GetRandomRequeueDelayInRange(2, 8)
}

// GetRandomRequeueDelayInRange returns a random delay in the given range in seconds, to be used for RequeueAfter
func GetRandomRequeueDelayInRange(lowSeconds, highSeconds int) time.Duration {
	// get a jittered delay to use for requeueing reconcile
	var seconds = rand.IntnRange(lowSeconds, highSeconds)
	return time.Duration(seconds) * time.Second
}

// NewRequeueWithDelay retruns a result set to requeue in 2 to 3 seconds
func NewRequeueWithDelay() reconcile.Result {
	return vzctrl.NewRequeueWithDelay(2, 3, time.Second)
}

// NewRequeueWithRandomDelay retruns a result set to requeue after a random delay
func NewRequeueWithRandomDelay(lowSeconds, highSeconds int) reconcile.Result {
	return controllerruntime.Result{Requeue: true, RequeueAfter: GetRandomRequeueDelayInRange(lowSeconds, highSeconds)}
}

// ShouldRequeue returns true if requeue is needed
func ShouldRequeue(r reconcile.Result) bool {
	return r.Requeue || r.RequeueAfter > 0
}

// GetResourceLogger will return the controller logger associated with the resource
func GetResourceLogger(controller string, namespacedName types.NamespacedName, obj client.Object) (vzlog.VerrazzanoLogger, error) {
	// Get the resource logger needed to log message using 'progress' and 'once' methods
	log, err := vzlog.EnsureResourceLogger(&vzlog.ResourceConfig{
		Name:           namespacedName.Name,
		Namespace:      namespacedName.Namespace,
		ID:             string(obj.GetUID()),
		Generation:     obj.GetGeneration(),
		ControllerName: controller,
	})
	if err != nil {
		zap.S().Errorf("Failed to create controller logger for %v: %v", namespacedName, err)
	}

	return log, err
}
