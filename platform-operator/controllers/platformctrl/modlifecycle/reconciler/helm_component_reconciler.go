// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package reconciler

import (
	"context"
	"fmt"
	vzlogInit "github.com/verrazzano/verrazzano/pkg/log"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzstring "github.com/verrazzano/verrazzano/pkg/string"
	modulesv1beta2 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta2"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/platformctrl/modlifecycle/delegates"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"go.uber.org/zap"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const FinalizerName = "modules.finalizer.verrazzano.io"

type Reconciler struct {
	client.StatusWriter
	spi.Component
}

var _ delegates.DelegateReconciler = &Reconciler{}

func (r *Reconciler) SetStatusWriter(writer client.StatusWriter) {
	r.StatusWriter = writer
}

func (r *Reconciler) ReconcileModule(log vzlog.VerrazzanoLogger, client client.Client, mlc *modulesv1beta2.ModuleLifecycle) error {
	ctx, err := spi.NewMinimalContext(client, log)
	if err != nil {
		return err
	}
	// Delete module if it is being deleted
	if mlc.IsBeingDeleted() {
		if err := r.UpdateStatus(ctx, modulesv1beta2.CondUninstall); err != nil {
			return err
		}
		if err := r.Uninstall(ctx); err != nil {
			return err
		}
		if err := removeFinalizer(ctx, mlc); err != nil {
			return err
		}
		ctx.Log().Infof("Uninstalled Module %s", mlc.Name)
	}
	return r.doReconcile(ctx, mlc)
}

func (r *Reconciler) doReconcile(ctx spi.ComponentContext, mlc *modulesv1beta2.ModuleLifecycle) error {
	log := ctx.Log()
	// Initialize the module if this is the first time we are reconciling it
	if err := initializeModule(ctx, mlc); err != nil {
		return err
	}
	condition := mlc.Status.Conditions[len(mlc.Status.Conditions)-1].Type
	switch condition {
	case modulesv1beta2.CondPreInstall:
		log.Progressf("Module %s pre-install is running", mlc.Name)
		if err := r.PreInstall(ctx); err != nil {
			return err
		}
		if err := r.Install(ctx); err != nil {
			return err
		}
		return r.UpdateStatus(ctx, modulesv1beta2.CondInstallStarted)
	case modulesv1beta2.CondInstallStarted:
		if r.IsReady(ctx) {
			log.Progressf("Module %s post-install is running", mlc.Name)
			if err := r.PostInstall(ctx); err != nil {
				return err
			}
			mlc.Status.ObservedGeneration = mlc.Generation
			ctx.Log().Infof("Module %s is ready", mlc.Name)
			return r.UpdateStatus(ctx, modulesv1beta2.CondInstallComplete)
		}
		return delegates.NotReadyErrorf("Install: Module %s is not ready", mlc.Name)
	case modulesv1beta2.CondPreUpgrade:
		log.Progressf("Module %s pre-upgrade is running", mlc.Name)
		if err := r.PreUpgrade(ctx); err != nil {
			return err
		}
		if err := r.Upgrade(ctx); err != nil {
			return err
		}
		return r.UpdateStatus(ctx, modulesv1beta2.CondUpgradeStarted)
	case modulesv1beta2.CondInstallComplete, modulesv1beta2.CondUpgradeComplete:
		return r.ReadyState(ctx)
	case modulesv1beta2.CondUpgradeStarted:
		if r.IsReady(ctx) {
			log.Progressf("Module %s post-upgrade is running", mlc.Name)
			if err := r.PostUpgrade(ctx); err != nil {
				return err
			}
			mlc.Status.ObservedGeneration = mlc.Generation
			return r.UpdateStatus(ctx, modulesv1beta2.CondUpgradeComplete)
		}
		return delegates.NotReadyErrorf("Upgrade: Module %s is not ready", mlc.Name)
	}
	return nil
}

// ReadyState reconciles put the Module back to pending state if the generation has changed
func (r *Reconciler) ReadyState(ctx spi.ComponentContext) error {
	if NeedsReconcile(ctx) {
		return r.UpdateStatus(ctx, modulesv1beta2.CondPreUpgrade)
	}
	return nil
}

// Uninstall cleans up the Helm Chart and removes the Module finalizer so Kubernetes can clean the resource
func (r *Reconciler) Uninstall(ctx spi.ComponentContext) error {
	if err := r.Component.PreUninstall(ctx); err != nil {
		return err
	}
	if err := r.Component.Uninstall(ctx); err != nil {
		return err
	}
	if err := r.Component.PostUninstall(ctx); err != nil {
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
	if len(mlc.Status.State) > 0 {
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

// TODO: Needed for shim layer to v1 components
//func (r *Reconciler) createComponentContext(log vzlog.VerrazzanoLogger, verrazzanos *vzapi.VerrazzanoList, module *modulesv1beta2.ModuleLifecycle) (spi.ComponentContext, error) {
//	var moduleCtx spi.ComponentContext
//	var err error
//if len(verrazzanos.Items) > 0 {
//	moduleCtx, err = spi.NewModuleContext(log, r.Client, &verrazzanos.Items[0], module, false)
//} else {
//	moduleCtx, err = spi.NewMinimalModuleContext(r.Client, log, module, false)
//}
//if err != nil {
//	log.Errorf("Failed to create module context: %v", err)
//}
//	return moduleCtx, err
//}
