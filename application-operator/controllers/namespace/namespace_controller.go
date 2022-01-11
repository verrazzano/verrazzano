// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package namespace

import (
	"context"
	"fmt"
	vzctrl "github.com/verrazzano/verrazzano/pkg/controller"

	"time"

	"github.com/go-logr/logr"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/application-operator/controllers"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	vzstring "github.com/verrazzano/verrazzano/pkg/string"
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
	log        logr.Logger
}

// NewNamespaceController - Creates and configures the namespace controller
func NewNamespaceController(mgr ctrl.Manager, logger logr.Logger) (*NamespaceController, error) {
	nc := &NamespaceController{
		Client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
		log:    logger.WithValues("function", "controller"),
	}

	// Launch a periodic task to scan all namespaces; this is required for cases where the configmap
	// may have been reset (e.g., post-upgrade)
	go scannerFunc(nc, logger.WithValues("function", "scanner"), 30, time.Second)

	var err error
	nc.controller, err = ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{
			RateLimiter: controllers.NewDefaultRateLimiter(),
		}).
		For(&corev1.Namespace{}).
		Build(nc)

	return nc, err
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
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nc.reconcileNamespace(ctx, nc.log, &ns)
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
func (nc *NamespaceController) reconcileNamespace(ctx context.Context, log logr.Logger, ns *corev1.Namespace) error {
	if err := nc.reconcileOCILogging(ctx, log, ns); err != nil {
		log.Error(err, "Error occurred during OCI Logging reconciliation")
		return err
	}
	log.V(1).Info("Reconciled namespace successfully", namespaceField, ns.Name)
	return nil
}

// reconcileNamespaceDelete - Reconcile any post-delete changes required
func (nc *NamespaceController) reconcileNamespaceDelete(ctx context.Context, ns *corev1.Namespace) error {
	// Update the OCI Logging configuration to remove the namespace configuration
	// If the annotation is not present, remove any existing logging configuration
	return nc.removeOCILogging(ctx, nc.log, ns)
}

// reconcileOCILogging - Configure OCI logging based on the annotation if present
func (nc *NamespaceController) reconcileOCILogging(ctx context.Context, log logr.Logger, ns *corev1.Namespace) error {
	// If the annotation is present, add the finalizer if necessary and update the logging configuration
	if loggingOCID, ok := ns.Annotations[constants.OCILoggingIDAnnotation]; ok {
		var added bool
		if ns.Finalizers, added = vzstring.SliceAddString(ns.Finalizers, namespaceControllerFinalizer); added {
			if err := nc.Update(ctx, ns); err != nil {
				return err
			}
		}
		log.V(1).Info("Updating logging configuration for namespace", namespaceField, ns.Name, "log-id", loggingOCID)
		updated, err := addNamespaceLoggingFunc(ctx, nc.Client, ns.Name, loggingOCID)
		if err != nil {
			return err
		}
		if updated {
			log.Info("Updated logging configuration for namespace", namespaceField, ns.Name)
			err = nc.restartFluentd(ctx)
		}
		return err
	}
	// If the annotation is not present, remove any existing logging configuration
	return nc.removeOCILogging(ctx, log, ns)
}

// removeOCILogging - Remove OCI logging if the namespace is deleted
func (nc *NamespaceController) removeOCILogging(ctx context.Context, log logr.Logger, ns *corev1.Namespace) error {
	removed, err := removeNamespaceLoggingFunc(ctx, nc.Client, ns.Name)
	if err != nil {
		return err
	}
	if removed {
		log.Info("Removed logging configuration for namespace", namespaceField, ns.Name)
		err = nc.restartFluentd(ctx)
	}
	return err
}

// restartFluentd - restarts the Fluentd pods by adding an annotation to the Fluentd daemonset.
func (nc *NamespaceController) restartFluentd(ctx context.Context) error {
	nc.log.Info("Restarting Fluentd")
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

func (nc *NamespaceController) scanNamespaces(ctx context.Context, log logr.Logger) (bool, error) {
	log.V(1).Info("Examining all namespaces")
	namespaceList := corev1.NamespaceList{}
	if err := nc.List(ctx, &namespaceList); err != nil {
		return false, err
	}
	for i := range namespaceList.Items {
		if err := nc.reconcileNamespace(ctx, log, &namespaceList.Items[i]); err != nil {
			return false, err
		}
	}
	return true, nil
}

// scannerFuncSig - Func type for namespace scanner, for unit testing
type scannerFuncSig func(nc *NamespaceController, log logr.Logger, period int, units time.Duration)

// scannerFunc - Var to allow overriding the scanner function, for unit testing
var scannerFunc scannerFuncSig = namespaceScanner

// scanOnce - indicates to the namespaceScanner routine that we should only execute once, for unit testing
var scanOnce = false

// namespaceScanner - Goroutine that reconciles all namespaces periodically
// - this will enable us to re-sync any changes (e.g., OCI Logging) that may need to be rebuilt post-upgrade, etc
func namespaceScanner(nc *NamespaceController, log logr.Logger, period int, units time.Duration) {
	periodDelay := time.Duration(period) * units
	delay := periodDelay
	for {
		log.V(1).Info(fmt.Sprintf("Delay %v seconds", delay.Seconds()))
		time.Sleep(delay)
		if completed, err := nc.scanNamespaces(context.Background(), log); !completed {
			// the scan didn't complete either due to an error or a failure to acquire the lock
			if err != nil {
				log.Error(err, "Error on periodic namespace scan")
			}
			delay = vzctrl.CalculateDelay(1, period, units)
			continue
		}
		delay = periodDelay
		if scanOnce {
			break
		}
	}
}

// addNamespaceLoggingFuncSig - Type for add namespace logging  function, for unit testing
type addNamespaceLoggingFuncSig func(_ context.Context, _ client.Client, _ string, _ string) (bool, error)

// addNamespaceLoggingFunc - Variable to allow replacing add namespace logging func for unit tests
var addNamespaceLoggingFunc addNamespaceLoggingFuncSig = addNamespaceLogging

// removeNamespaceLoggingFuncSig - Type for remove namespace logging function, for unit testing
type removeNamespaceLoggingFuncSig func(_ context.Context, _ client.Client, _ string) (bool, error)

// removeNamespaceLoggingFunc - Variable to allow replacing remove namespace logging func for unit tests
var removeNamespaceLoggingFunc removeNamespaceLoggingFuncSig = removeNamespaceLogging
