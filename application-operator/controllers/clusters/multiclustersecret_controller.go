// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package controllers

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
)

// MultiClusterSecretReconciler reconciles a MultiClusterSecret object
type MultiClusterSecretReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=clusters.verrazzano.io,resources=multiclustersecrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=clusters.verrazzano.io,resources=multiclustersecrets/status,verbs=get;update;patch

func (r *MultiClusterSecretReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	logger := r.Log.WithValues("multiclustersecret", req.NamespacedName)
	var mcSecret clustersv1alpha1.MultiClusterSecret
	result := reconcile.Result{}
	ctx := context.Background()
	err := r.fetchMultiClusterSecret(ctx, req.NamespacedName, &mcSecret)
	if err != nil {
		logger.Info("Failed to fetch MultiClusterSecret", "err", err)
		return result, client.IgnoreNotFound(err)
	}

	logger.Info("MultiClusterSecret create or update with underlying secret",
		"secret", mcSecret.Spec.Template.Metadata.Name,
		"placement", mcSecret.Spec.Placement.Clusters[0].Name)
	opResult, err := r.createOrUpdateSecret(ctx, mcSecret, &result)

	return r.updateStatus(ctx, &mcSecret, opResult, err)
}

func (r *MultiClusterSecretReconciler) updateStatus(ctx context.Context, mcSecret *clustersv1alpha1.MultiClusterSecret, opResult controllerutil.OperationResult, err error) (ctrl.Result, error) {
	var condition clustersv1alpha1.Condition
	var state clustersv1alpha1.StateType
	if err != nil {
		condition = clustersv1alpha1.Condition{Type: clustersv1alpha1.DeployFailed,
			Status: corev1.ConditionTrue,
			Message: err.Error(),
			LastTransitionTime: time.Now().Format(time.RFC3339),
		}
		state = clustersv1alpha1.Failed
	} else {
		msg := fmt.Sprintf("Secret %v", opResult)
		condition = clustersv1alpha1.Condition{Type: clustersv1alpha1.DeployComplete,
			Status: corev1.ConditionTrue,
			Message: msg,
			LastTransitionTime: time.Now().Format(time.RFC3339),
		}
		state = clustersv1alpha1.Ready
	}
	if statusNeedsUpdate(mcSecret.Status.Conditions, mcSecret.Status.State, condition, state) {
		mcSecret.Status.Conditions = append(mcSecret.Status.Conditions, condition)
		mcSecret.Status.State = state
		return reconcile.Result{Requeue: true}, r.Status().Update(ctx, mcSecret)
	} else {
		return reconcile.Result{}, nil
	}
}

func statusNeedsUpdate(curConditions []clustersv1alpha1.Condition, curState clustersv1alpha1.StateType,
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

func (r *MultiClusterSecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&clustersv1alpha1.MultiClusterSecret{}).
		Complete(r)
}

func (r *MultiClusterSecretReconciler) fetchMultiClusterSecret(ctx context.Context, name types.NamespacedName, mcSecretRef *clustersv1alpha1.MultiClusterSecret) error {
	return r.Get(ctx, name, mcSecretRef)
}

func (r *MultiClusterSecretReconciler) createOrUpdateSecret(ctx context.Context, mcSecret clustersv1alpha1.MultiClusterSecret, recRes *reconcile.Result) (controllerutil.OperationResult, error) {
	var secret corev1.Secret
	secret.Namespace = mcSecret.Namespace
	secret.Name = mcSecret.Name

	return controllerutil.CreateOrUpdate(ctx, r.Client, &secret, func() error {
		r.MutateSecret(mcSecret, &secret)
		// This SetControllerReference call will trigger garbage collection i.e. the secret
		// will automatically get deleted when the mcSecret is deleted
		return controllerutil.SetControllerReference(&mcSecret, &secret, r.Scheme)
	})

}

func (r *MultiClusterSecretReconciler) MutateSecret(mcSecret clustersv1alpha1.MultiClusterSecret, secret *corev1.Secret) {
	secret.Type = mcSecret.Spec.Template.Type
	secret.Data = mcSecret.Spec.Template.Data
	secret.StringData = mcSecret.Spec.Template.StringData
}
