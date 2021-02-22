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
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// MCRegistrationSecretFullName is the full NamespacedName of the cluster registration secret
var MCRegistrationSecretFullName = types.NamespacedName{
	Namespace: constants.VerrazzanoSystemNamespace,
	Name:      constants.MCRegistrationSecret}

// StatusNeedsUpdate determines based on the current state and conditions of a MultiCluster
// resource, as well as the new state and condition to be set, whether the status update
// needs to be done
func StatusNeedsUpdate(curConditions []clustersv1alpha1.Condition, curState clustersv1alpha1.StateType,
	newCondition clustersv1alpha1.Condition, newState clustersv1alpha1.StateType) bool {
	if newState == clustersv1alpha1.Failed {
		return true
	}
	if newState != curState {
		return true
	}
	foundStatus := false
	for _, existingCond := range curConditions {
		if existingCond.Status == newCondition.Status &&
			existingCond.Message == newCondition.Message &&
			existingCond.Type == newCondition.Type {
			foundStatus = true
		}
	}
	return !foundStatus
}

// GetConditionAndStateFromResult - Based on the result of a create/update operation on the
// embedded resource, returns the Condition and State that must be set on a MultiCluster
// resource's Status
func GetConditionAndStateFromResult(err error, opResult controllerutil.OperationResult, msgPrefix string) (clustersv1alpha1.Condition, clustersv1alpha1.StateType) {
	var condition clustersv1alpha1.Condition
	var state clustersv1alpha1.StateType
	if err != nil {
		condition = clustersv1alpha1.Condition{
			Type:               clustersv1alpha1.DeployFailed,
			Status:             corev1.ConditionTrue,
			Message:            err.Error(),
			LastTransitionTime: time.Now().Format(time.RFC3339),
		}
		state = clustersv1alpha1.Failed
	} else {
		msg := fmt.Sprintf("%v %v", msgPrefix, opResult)
		condition = clustersv1alpha1.Condition{
			Type:               clustersv1alpha1.DeployComplete,
			Status:             corev1.ConditionTrue,
			Message:            msg,
			LastTransitionTime: time.Now().Format(time.RFC3339),
		}
		state = clustersv1alpha1.Ready
	}

	return condition, state
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

	err := rdr.Get(ctx, MCRegistrationSecretFullName, &clusterSecret)
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
