// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package reconciler

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/controller"
	vzlogInit "github.com/verrazzano/verrazzano/pkg/log"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzstring "github.com/verrazzano/verrazzano/pkg/string"
	modulesv1beta2 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta2"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/experimental/controllers/platformctrl/common"
	"github.com/verrazzano/verrazzano/platform-operator/experimental/controllers/platformctrl/modlifecycle/delegates"
	"go.uber.org/zap"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

const FinalizerName = "modulelifecycle.finalizer.verrazzano.io"

type helmDelegateReconciler struct {
	client.StatusWriter
	comp spi.Component
}

var _ delegates.DelegateLifecycleReconciler = &helmDelegateReconciler{}

func (r *helmDelegateReconciler) Reconcile(log vzlog.VerrazzanoLogger, client client.Client, mlc *modulesv1beta2.ModuleLifecycle) (ctrl.Result, error) {
	ctx, err := spi.NewMinimalContext(client, log)
	if err != nil {
		return newRequeueWithDelay(), err
	}
	// Delete underlying resources if it is being deleted
	if mlc.IsBeingDeleted() {
		if err := UpdateStatus(ctx.Client(), mlc, string(modulesv1beta2.CondUninstall), modulesv1beta2.CondUninstall); err != nil {
			return ctrl.Result{}, err
		}
		if err := r.Uninstall(ctx); err != nil {
			return newRequeueWithDelay(), err
		}
		if err := removeFinalizer(ctx, mlc); err != nil {
			return newRequeueWithDelay(), err
		}
		ctx.Log().Infof("Uninstall of %s complete", common.GetNamespacedName(mlc.ObjectMeta))
		return ctrl.Result{}, nil
	}
	if mlc.Generation == mlc.Status.ObservedGeneration {
		log.Debugf("Skipping reconcile for %s, observed generation has not change", common.GetNamespacedName(mlc.ObjectMeta))
		return newRequeueWithDelay(), err
	}
	if err := r.doReconcile(ctx, mlc); err != nil {
		return newRequeueWithDelay(), err
	}
	return ctrl.Result{}, nil
}

func newRequeueWithDelay() ctrl.Result {
	return controller.NewRequeueWithDelay(3, 10, time.Second)
}

func (r *helmDelegateReconciler) doReconcile(ctx spi.ComponentContext, mlc *modulesv1beta2.ModuleLifecycle) error {
	log := ctx.Log()
	// Initialize the module if this is the first time we are reconciling it
	if err := initializeModule(ctx, mlc); err != nil {
		return err
	}
	condition := mlc.Status.Conditions[len(mlc.Status.Conditions)-1].Type
	switch condition {
	case modulesv1beta2.CondPreInstall:
		return r.handlePreInstall(ctx, mlc, log)
	case modulesv1beta2.CondInstallStarted:
		return r.handleInstallStarted(ctx, mlc, log)
	case modulesv1beta2.CondPreUpgrade:
		return r.handlePreUpgrade(ctx, mlc, log)
	case modulesv1beta2.CondInstallComplete, modulesv1beta2.CondUpgradeComplete:
		return r.ReadyState(ctx, mlc)
	case modulesv1beta2.CondUpgradeStarted:
		return r.handleUpgradeStarted(ctx, mlc, log)
	}
	return nil
}

func (r *helmDelegateReconciler) handleUpgradeStarted(ctx spi.ComponentContext, mlc *modulesv1beta2.ModuleLifecycle, log vzlog.VerrazzanoLogger) error {
	if r.comp.IsReady(ctx) {
		log.Progressf("Post-upgrade for %s is running", common.GetNamespacedName(mlc.ObjectMeta))
		if err := r.comp.PostUpgrade(ctx); err != nil {
			return err
		}
		mlc.Status.ObservedGeneration = mlc.Generation
		return UpdateStatus(ctx.Client(), mlc, string(modulesv1beta2.CondUpgradeComplete), modulesv1beta2.CondUpgradeComplete)
	}
	return delegates.NotReadyErrorf("Upgrade for %s is not ready", common.GetNamespacedName(mlc.ObjectMeta))
}

func (r *helmDelegateReconciler) handlePreInstall(ctx spi.ComponentContext, mlc *modulesv1beta2.ModuleLifecycle, log vzlog.VerrazzanoLogger) error {
	log.Progressf("Pre-install for %s is running", common.GetNamespacedName(mlc.ObjectMeta))
	if err := r.comp.PreInstall(ctx); err != nil {
		return err
	}
	if err := r.comp.Install(ctx); err != nil {
		return err
	}
	return UpdateStatus(ctx.Client(), mlc, string(modulesv1beta2.CondInstallStarted), modulesv1beta2.CondInstallStarted)
}

func (r *helmDelegateReconciler) handlePreUpgrade(ctx spi.ComponentContext, mlc *modulesv1beta2.ModuleLifecycle, log vzlog.VerrazzanoLogger) error {
	log.Progressf("Pre-upgrade for %s is running", common.GetNamespacedName(mlc.ObjectMeta))
	if err := r.comp.PreUpgrade(ctx); err != nil {
		return err
	}
	if err := r.comp.Upgrade(ctx); err != nil {
		return err
	}
	return UpdateStatus(ctx.Client(), mlc, string(modulesv1beta2.CondUpgradeStarted), modulesv1beta2.CondUpgradeStarted)
}

func (r *helmDelegateReconciler) handleInstallStarted(ctx spi.ComponentContext, mlc *modulesv1beta2.ModuleLifecycle, log vzlog.VerrazzanoLogger) error {
	if r.comp.IsReady(ctx) {
		log.Progressf("Post-install for %s is running", common.GetNamespacedName(mlc.ObjectMeta))
		if err := r.comp.PostInstall(ctx); err != nil {
			return err
		}
		mlc.Status.ObservedGeneration = mlc.Generation
		ctx.Log().Infof("%s is ready", common.GetNamespacedName(mlc.ObjectMeta))
		return UpdateStatus(ctx.Client(), mlc, string(modulesv1beta2.CondInstallComplete), modulesv1beta2.CondInstallComplete)
	}
	return delegates.NotReadyErrorf("Install for %s is not ready", common.GetNamespacedName(mlc.ObjectMeta))
}

// ReadyState reconciles put the Module back to pending state if the generation has changed
func (r *helmDelegateReconciler) ReadyState(ctx spi.ComponentContext, mlc *modulesv1beta2.ModuleLifecycle) error {
	if needsReconcile(mlc) {
		return UpdateStatus(ctx.Client(), mlc, string(modulesv1beta2.CondPreUpgrade), modulesv1beta2.CondPreUpgrade)
	}
	return nil
}

// Uninstall cleans up the Helm Chart and removes the Module finalizer so Kubernetes can clean the resource
func (r *helmDelegateReconciler) Uninstall(ctx spi.ComponentContext) error {
	if err := r.comp.PreUninstall(ctx); err != nil {
		return err
	}
	if err := r.comp.Uninstall(ctx); err != nil {
		return err
	}
	if err := r.comp.PostUninstall(ctx); err != nil {
		return err
	}
	return nil
}

func initializeModule(ctx spi.ComponentContext, mlc *modulesv1beta2.ModuleLifecycle) error {
	if err := addFinalizer(ctx, mlc); err != nil {
		return err
	}
	initializeModuleStatus(ctx, mlc)
	return nil
}

func initializeModuleStatus(ctx spi.ComponentContext, mlc *modulesv1beta2.ModuleLifecycle) {
	if len(mlc.Status.State) == 0 {
		mlc.SetState(modulesv1beta2.StatePreinstall)
		mlc.Status.Conditions = []modulesv1beta2.ModuleLifecycleCondition{
			NewCondition(string(modulesv1beta2.StatePreinstall), modulesv1beta2.CondPreInstall),
		}
	}
}

func addFinalizer(ctx spi.ComponentContext, mlc *modulesv1beta2.ModuleLifecycle) error {
	if needsFinalizer(mlc) {
		mlc.Finalizers = append(mlc.Finalizers, FinalizerName)
		err := ctx.Client().Update(context.TODO(), mlc)
		_, err = vzlogInit.IgnoreConflictWithLog(fmt.Sprintf("Failed to add finalizer to ingress trait %s", mlc.Name), err, zap.S())
		return err
	}
	return nil
}

func needsFinalizer(mlc *modulesv1beta2.ModuleLifecycle) bool {
	return mlc.GetDeletionTimestamp().IsZero() && !vzstring.SliceContainsString(mlc.Finalizers, FinalizerName)
}

func removeFinalizer(ctx spi.ComponentContext, mlc *modulesv1beta2.ModuleLifecycle) error {
	if needsFinalizerRemoval(mlc) {
		mlc.Finalizers = vzstring.RemoveStringFromSlice(mlc.Finalizers, FinalizerName)
		err := ctx.Client().Update(context.TODO(), mlc)
		return vzlogInit.ConflictWithLog(fmt.Sprintf("Failed to remove finalizer from module %s/%s", mlc.Namespace, mlc.Name), err, zap.S())
	}
	return nil
}

func needsFinalizerRemoval(mlc *modulesv1beta2.ModuleLifecycle) bool {
	return !needsFinalizer(mlc)
}
