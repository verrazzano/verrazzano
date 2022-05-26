// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package reconciler

import (
	"context"
	"fmt"
	vzlogInit "github.com/verrazzano/verrazzano/pkg/log"
	vzstring "github.com/verrazzano/verrazzano/pkg/string"
	modulesv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/modules/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/module/modules"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"go.uber.org/zap"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const FinalizerName = "modules.finalizer.verrazzano.io"

type Reconciler struct {
	client.StatusWriter
	ChartDir string
	spi.ModuleComponent
}

func (r *Reconciler) SetStatusWriter(writer client.StatusWriter) {
	r.StatusWriter = writer
}

func (r *Reconciler) ReconcileModule(ctx spi.ComponentContext) error {
	// Delete module if it is being deleted
	if ctx.Module().IsBeingDeleted() {
		if err := r.UpdateStatus(ctx, modulesv1alpha1.CondUninstall); err != nil {
			return err
		}
		return r.Uninstall(ctx)
	}
	return r.doReconcile(ctx)
}

func (r *Reconciler) doReconcile(ctx spi.ComponentContext) error {
	module := ctx.Module()
	log := ctx.Log()
	// Initialize the module if this is the first time we are reconciling it
	if err := initializeModule(ctx); err != nil {
		return err
	}
	condition := module.Status.Conditions[len(module.Status.Conditions)-1].Type
	switch condition {
	case modulesv1alpha1.CondPreInstall:
		log.Progressf("Module %s pre-install is running", module.Name)
		if err := r.PreInstall(ctx); err != nil {
			return err
		}
		if err := r.Install(ctx); err != nil {
			return err
		}
		return r.UpdateStatus(ctx, modulesv1alpha1.CondInstallStarted)
	case modulesv1alpha1.CondInstallStarted:
		if r.IsReady(ctx) {
			log.Progressf("Module %s post-install is running", module.Name)
			if err := r.PostInstall(ctx); err != nil {
				return err
			}
			module.Status.ObservedGeneration = module.Generation
			ctx.Log().Infof("Module %s is ready", ctx.Module().Name)
			return r.UpdateStatus(ctx, modulesv1alpha1.CondInstallComplete)
		}
		return modules.NotReadyErrorf("Install: Module %s is not ready", module.Name)
	case modulesv1alpha1.CondPreUpgrade:
		log.Progressf("Module %s pre-upgrade is running", module.Name)
		if err := r.PreUpgrade(ctx); err != nil {
			return err
		}
		if err := r.Upgrade(ctx); err != nil {
			return err
		}
		return r.UpdateStatus(ctx, modulesv1alpha1.CondUpgradeStarted)
	case modulesv1alpha1.CondInstallComplete, modulesv1alpha1.CondUpgradeComplete:
		return r.ReadyPhase(ctx)
	case modulesv1alpha1.CondUpgradeStarted:
		if r.IsReady(ctx) {
			log.Progressf("Module %s post-upgrade is running", module.Name)
			if err := r.PostUpgrade(ctx); err != nil {
				return err
			}
			module.Status.ObservedGeneration = module.Generation
			return r.UpdateStatus(ctx, modulesv1alpha1.CondUpgradeComplete)
		}
		return modules.NotReadyErrorf("Upgrade: Module %s is not ready", module.Name)
	}
	return nil
}

//ReadyPhase reconciles put the Module back to pending state if the generation has changed
func (r *Reconciler) ReadyPhase(ctx spi.ComponentContext) error {
	if NeedsReconcile(ctx) {
		return r.UpdateStatus(ctx, modulesv1alpha1.CondPreUpgrade)
	}
	return nil
}

//Uninstall cleans up the Helm Chart and removes the Module finalizer so Kubernetes can clean the resource
func (r *Reconciler) Uninstall(ctx spi.ComponentContext) error {
	if err := r.ModuleComponent.Uninstall(ctx); err != nil {
		return err
	}
	if err := removeFinalizer(ctx); err != nil {
		return err
	}
	ctx.Log().Infof("Uninstalled Module %s", ctx.Module().Name)
	return nil
}

func initializeModule(ctx spi.ComponentContext) error {
	if err := addFinalizer(ctx); err != nil {
		return err
	}
	initializeModuleStatus(ctx)
	return nil
}

func initializeModuleStatus(ctx spi.ComponentContext) {
	module := ctx.Module()
	if module.Status.Phase == nil {
		module.SetPhase(modulesv1alpha1.PhasePreinstall)
		module.Status.Conditions = []modulesv1alpha1.Condition{
			NewCondition(string(modulesv1alpha1.PhasePreinstall), modulesv1alpha1.CondPreInstall),
		}
	}
}

func addFinalizer(ctx spi.ComponentContext) error {
	module := ctx.Module()
	if needsFinalizer(module) {
		module.Finalizers = append(module.Finalizers, FinalizerName)
		err := ctx.Client().Update(context.TODO(), module)
		_, err = vzlogInit.IgnoreConflictWithLog(fmt.Sprintf("Failed to add finalizer to ingress trait %s", module.Name), err, zap.S())
		return err
	}
	return nil
}

func needsFinalizer(module *modulesv1alpha1.Module) bool {
	return module.GetDeletionTimestamp().IsZero() && !vzstring.SliceContainsString(module.Finalizers, FinalizerName)
}

func removeFinalizer(ctx spi.ComponentContext) error {
	module := ctx.Module()
	if needsFinalizerRemoval(module) {
		module.Finalizers = vzstring.RemoveStringFromSlice(module.Finalizers, FinalizerName)
		err := ctx.Client().Update(context.TODO(), module)
		return vzlogInit.ConflictWithLog(fmt.Sprintf("Failed to remove finalizer from module %s/%s", module.Namespace, module.Name), err, zap.S())
	}
	return nil
}

func needsFinalizerRemoval(module *modulesv1alpha1.Module) bool {
	return !needsFinalizer(module)
}
