// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package namespace

import (
	"context"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
	vzlog "github.com/verrazzano/verrazzano/pkg/log"
	"time"

	"github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/application-operator/controllers"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	vzstring "github.com/verrazzano/verrazzano/pkg/string"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const namespaceControllerFinalizer = "verrazzano.io/namespace"

const namespaceField = "namespace"

// Reconciler reconciles a Verrazzano object
type NamespaceController struct {
	client.Client
	scheme     *runtime.Scheme
	controller controller.Controller
	log        *zap.SugaredLogger
}

// NewNamespaceController - Creates and configures the namespace controller
func NewNamespaceController(mgr ctrl.Manager, log *zap.SugaredLogger) (*NamespaceController, error) {
	nc := &NamespaceController{
		Client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
		log:    log,
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
func (nc *NamespaceController) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	res, err := nc.doReconcile(req)
	if clusters.ShouldRequeue(res) {
		return res, nil
	}
	// Never return an error since it has already been logged and we don't want the
	// controller runtime to log again (with stack trace).  Just re-queue if there is an error.
	if err != nil {
		return clusters.NewRequeueWithDelay(), nil
	}

	return ctrl.Result{}, nil
}

// doReconcile performs the reconciliation operations for the namespace
func (nc *NamespaceController) doReconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := nc.log.With(vzlog.FieldResourceNamespace, req.Namespace, vzlog.FieldResourceNamespace, req.Name, vzlog.FieldController, "namespace")

	// fetch the namespace
	ns := corev1.Namespace{}
	if err := nc.Client.Get(ctx, req.NamespacedName, &ns); err != nil {
		if k8serrors.IsNotFound(err) {
			log.Infow("Failed to find namespace", namespaceField, req.Name)
		} else {
			log.Errorf("Failed to fetch namespace %s: %v", req.Name, err)
		}
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}

	if !ns.ObjectMeta.DeletionTimestamp.IsZero() {
		// Finalizer is present, perform any required cleanup and remove the finalizer
		if vzstring.SliceContainsString(ns.Finalizers, namespaceControllerFinalizer) {
			if err := nc.reconcileNamespaceDelete(ctx, &ns, log); err != nil {
				return ctrl.Result{}, err
			}
			return nc.removeFinalizer(ctx, &ns, log)
		}
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nc.reconcileNamespace(ctx, &ns, log)
}

// removeFinalizer - Remove the finalizer and update the namespace resource if the post-delete processing is successful
func (nc *NamespaceController) removeFinalizer(ctx context.Context, ns *corev1.Namespace, log *zap.SugaredLogger) (reconcile.Result, error) {
	log.Debug("Removing finalizer")
	ns.Finalizers = vzstring.RemoveStringFromSlice(ns.Finalizers, namespaceControllerFinalizer)
	err := nc.Update(ctx, ns)
	if err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}

// reconcileNamespace - Reconcile any namespace changes
func (nc *NamespaceController) reconcileNamespace(ctx context.Context, ns *corev1.Namespace, log *zap.SugaredLogger) error {
	if err := nc.reconcileOCILogging(ctx, ns, log); err != nil {
		log.Errorf("Failed to reconcile OCI Logging: %v", err)
		return err
	}
	log.Debugf("Reconciled namespace %s successfully", ns.Name)
	return nil
}

// reconcileNamespaceDelete - Reconcile any post-delete changes required
func (nc *NamespaceController) reconcileNamespaceDelete(ctx context.Context, ns *corev1.Namespace, log *zap.SugaredLogger) error {
	// Update the OCI Logging configuration to remove the namespace configuration
	// If the annotation is not present, remove any existing logging configuration
	return nc.removeOCILogging(ctx, ns, log)
}

// reconcileOCILogging - Configure OCI logging based on the annotation if present
func (nc *NamespaceController) reconcileOCILogging(ctx context.Context, ns *corev1.Namespace, log *zap.SugaredLogger) error {
	// If the annotation is present, add the finalizer if necessary and update the logging configuration
	if loggingOCID, ok := ns.Annotations[constants.OCILoggingIDAnnotation]; ok {
		var added bool
		if ns.Finalizers, added = vzstring.SliceAddString(ns.Finalizers, namespaceControllerFinalizer); added {
			if err := nc.Update(ctx, ns); err != nil {
				return err
			}
		}
		log.Debugw("Updating logging configuration for namespace", namespaceField, ns.Name, "log-id", loggingOCID)
		updated, err := addNamespaceLoggingFunc(ctx, nc.Client, ns.Name, loggingOCID)
		if err != nil {
			return err
		}
		if updated {
			log.Debugw("Updated logging configuration for namespace", namespaceField, ns.Name)
			err = nc.restartFluentd(ctx, log)
		}
		return err
	}
	// If the annotation is not present, remove any existing logging configuration
	return nc.removeOCILogging(ctx, ns, log)
}

// removeOCILogging - Remove OCI logging if the namespace is deleted
func (nc *NamespaceController) removeOCILogging(ctx context.Context, ns *corev1.Namespace, log *zap.SugaredLogger) error {
	removed, err := removeNamespaceLoggingFunc(ctx, nc.Client, ns.Name)
	if err != nil {
		return err
	}
	if removed {
		log.Debugw("Removed logging configuration for namespace", namespaceField, ns.Name)
		err = nc.restartFluentd(ctx, log)
	}
	return err
}

// restartFluentd - restarts the Fluentd pods by adding an annotation to the Fluentd daemonset.
func (nc *NamespaceController) restartFluentd(ctx context.Context, log *zap.SugaredLogger) error {
	log.Debug("Restarting Fluentd")
	daemonSet := &appsv1.DaemonSet{}
	dsName := types.NamespacedName{Name: vzconst.FluentdDaemonSetName, Namespace: constants.VerrazzanoSystemNamespace}

	if err := nc.Client.Get(ctx, dsName, daemonSet); err != nil {
		return err
	}

	if daemonSet.Spec.Template.ObjectMeta.Annotations == nil {
		daemonSet.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
	}
	daemonSet.Spec.Template.ObjectMeta.Annotations[vzconst.VerrazzanoRestartAnnotation] = time.Now().Format(time.RFC3339)

	if err := nc.Client.Update(ctx, daemonSet); err != nil {
		return err
	}

	return nil
}

// addNamespaceLoggingFuncSig - Type for add namespace logging  function, for unit testing
type addNamespaceLoggingFuncSig func(_ context.Context, _ client.Client, _ string, _ string) (bool, error)

// addNamespaceLoggingFunc - Variable to allow replacing add namespace logging func for unit tests
var addNamespaceLoggingFunc addNamespaceLoggingFuncSig = addNamespaceLogging

// removeNamespaceLoggingFuncSig - Type for remove namespace logging function, for unit testing
type removeNamespaceLoggingFuncSig func(_ context.Context, _ client.Client, _ string) (bool, error)

// removeNamespaceLoggingFunc - Variable to allow replacing remove namespace logging func for unit tests
var removeNamespaceLoggingFunc removeNamespaceLoggingFuncSig = removeNamespaceLogging
