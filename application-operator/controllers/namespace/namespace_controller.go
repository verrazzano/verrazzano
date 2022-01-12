// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package namespace

import (
	"context"
	"github.com/go-logr/logr"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/application-operator/controllers"
	vzstring "github.com/verrazzano/verrazzano/pkg/string"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const namespaceControllerFinalizer = "namespaces.verrazzano.io"

const namespaceField = "namespace"

// Reconciler reconciles a Verrazzano object
type NamespaceController struct {
	client.Client
	scheme     *runtime.Scheme
	controller controller.Controller
	log        logr.Logger
}

// NewNamespaceController - Creates and configures the namespace controller
func NewNamespaceController(mgr ctrl.Manager, logger logr.Logger) (*NamespaceController, error) {
	nc := &NamespaceController{
		Client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
		log:    logger,
	}
	return nc, nc.setupWithManager(mgr)
}

// SetupWithManager creates a new controller and adds it to the manager
func (nc *NamespaceController) setupWithManager(mgr ctrl.Manager) error {
	var err error
	nc.controller, err = ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{
			RateLimiter: controllers.NewDefaultRateLimiter(),
		}).
		For(&corev1.Namespace{}).
		Build(nc)
	return err
}

// Reconcile - Watches for and manages namespace activity as it relates to Verrazzano platform services
func (nc *NamespaceController) Reconcile(req reconcile.Request) (reconcile.Result, error) {
	ctx := context.Background()
	nc.log.Info("Reconciling namespace", namespaceField, req.Name)

	// fetch the namespace
	ns := corev1.Namespace{}
	if err := nc.Client.Get(ctx, req.NamespacedName, &ns); err != nil {
		if k8serrors.IsNotFound(err) {
			nc.log.V(1).Info("Namespace does not exist", namespaceField, req.Name)
		} else {
			nc.log.Error(err, "Failed to fetch namespace", namespaceField, req.Name)
		}
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}

	if !ns.ObjectMeta.DeletionTimestamp.IsZero() {
		// Finalizer is present, perform any required cleanup and remove the finalizer
		if vzstring.SliceContainsString(ns.Finalizers, namespaceControllerFinalizer) {
			if err := nc.reconcileNamespaceDelete(ctx, &ns); err != nil {
				return ctrl.Result{}, err
			}
			return nc.removeFinalizer(ctx, &ns)
		}
	}

	return ctrl.Result{}, nc.reconcileNamespace(ctx, &ns)
}

// removeFinalizer - Remove the finalizer and update the namespace resource if the post-delete processing is successful
func (nc *NamespaceController) removeFinalizer(ctx context.Context, ns *corev1.Namespace) (reconcile.Result, error) {
	nc.log.V(1).Info("Removing finalizer")
	ns.Finalizers = vzstring.RemoveStringFromSlice(ns.Finalizers, namespaceControllerFinalizer)
	err := nc.Update(ctx, ns)
	if err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}

// reconcileNamespace - Reconcile any namespace changes
func (nc *NamespaceController) reconcileNamespace(ctx context.Context, ns *corev1.Namespace) error {
	if err := nc.reconcileOCILogging(ctx, ns); err != nil {
		nc.log.Error(err, "Error occurred during OCI Logging reconciliation")
		return err
	}
	nc.log.V(1).Info("Reconciled namespace %s successfully", namespaceField, ns.Name)
	return nil
}

// reconcileNamespaceDelete - Reconcile any post-delete changes required
func (nc *NamespaceController) reconcileNamespaceDelete(ctx context.Context, ns *corev1.Namespace) error {
	// Update the OCI Logging configuration to remove the namespace configuration
	// If the annotation is not present, remove any existing logging configuration
	return nc.removeOCILogging(ctx, ns)
}

// reconcileOCILogging - Configure OCI logging based on the annotation if present
func (nc *NamespaceController) reconcileOCILogging(ctx context.Context, ns *corev1.Namespace) error {
	// If the annotation is present, add the finalizer if necessary and update the logging configuration
	if loggingOCID, ok := ns.Annotations[constants.OCILoggingIDAnnotation]; ok {
		var added bool
		if ns.Finalizers, added = vzstring.SliceAddString(ns.Finalizers, namespaceControllerFinalizer); added {
			if err := nc.Update(ctx, ns); err != nil {
				return err
			}
		}
		nc.log.V(1).Info("Updating logging configuration for namespace", namespaceField, ns.Name, "log-id", loggingOCID)
		updated, err := addNamespaceLoggingFunc(ctx, nc.Client, ns.Name, loggingOCID)
		if err != nil {
			return err
		}
		if updated {
			nc.log.Info("Updated logging configuration for namespace", namespaceField, ns.Name)
		}
		return nil
	}
	// If the annotation is not present, remove any existing logging configuration
	return nc.removeOCILogging(ctx, ns)
}

// removeOCILogging - Remove OCI logging if the namespace is deleted
func (nc *NamespaceController) removeOCILogging(ctx context.Context, ns *corev1.Namespace) error {
	removed, err := removeNamespaceLoggingFunc(ctx, nc.Client, ns.Name)
	if err != nil {
		return err
	}
	if removed {
		nc.log.Info("Removed logging configuration for namespace", namespaceField, ns.Name)
	}
	return nil
}

// addNamespaceLoggingFuncSig - Type for add namespace logging  function, for unit testing
type addNamespaceLoggingFuncSig func(_ context.Context, _ client.Client, _ string, _ string) (bool, error)

// addNamespaceLoggingFunc - Variable to allow replacing add namespace logging func for unit tests
var addNamespaceLoggingFunc addNamespaceLoggingFuncSig = AddNamespaceLogging

// AddNamespaceLogging - placeholder for logging update
func AddNamespaceLogging(_ context.Context, _ client.Client, _ string, _ string) (bool, error) {
	return true, nil
}

// removeNamespaceLoggingFuncSig - Type for remove namespace logging function, for unit testing
type removeNamespaceLoggingFuncSig func(_ context.Context, _ client.Client, _ string) (bool, error)

// removeNamespaceLoggingFunc - Variable to allow replacing remove namespace logging func for unit tests
var removeNamespaceLoggingFunc removeNamespaceLoggingFuncSig = RemoveNamespaceLogging

// RemoveNamespaceLogging - placeholder for logging update
func RemoveNamespaceLogging(_ context.Context, _ client.Client, _ string) (bool, error) {
	return true, nil
}
