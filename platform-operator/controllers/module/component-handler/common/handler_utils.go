// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"context"
	"fmt"
	moduleapi "github.com/verrazzano/verrazzano-modules/module-operator/apis/platform/v1alpha1"
	modulestatus "github.com/verrazzano/verrazzano-modules/module-operator/controllers/module/status"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/result"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/spi/handlerspi"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	modulecatalog "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/catalog"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetComponentAndContext(ctx handlerspi.HandlerContext, operation string) (spi.ComponentContext, spi.Component, error) {
	module := ctx.CR.(*moduleapi.Module)
	vz, err := GetVerrazzanoCR(ctx)
	if err != nil {
		return nil, nil, err
	}

	return getComponentByNameAndContext(ctx, vz, module.Spec.ModuleName, operation)
}

func GetVerrazzanoCR(ctx handlerspi.HandlerContext) (*vzapi.Verrazzano, error) {
	nsn, err := GetVerrazzanoNSN(ctx)
	if err != nil {
		return nil, err
	}

	vz := &vzapi.Verrazzano{}
	if err := ctx.Client.Get(context.TODO(), *nsn, vz); err != nil {
		return nil, err
	}
	return vz, nil
}

func GetVerrazzanoNSN(ctx handlerspi.HandlerContext) (*types.NamespacedName, error) {
	vzlist := &vzapi.VerrazzanoList{}
	if err := ctx.Client.List(context.TODO(), vzlist); err != nil {
		return nil, err
	}
	if len(vzlist.Items) != 1 {
		return nil, fmt.Errorf("Failed, found %d Verrazzano CRs in the cluster.  There must be exactly 1 Verrazzano CR", len(vzlist.Items))
	}
	vz := vzlist.Items[0]
	return &types.NamespacedName{Namespace: vz.Namespace, Name: vz.Name}, nil
}

// AreDependenciesReady check if dependencies are ready using the Module condition
func AreDependenciesReady(ctx handlerspi.HandlerContext, depModulesNames []string) (res result.Result, deps []string) {
	var remainingDeps []string

	vz, err := GetVerrazzanoCR(ctx)
	if err != nil {
		return result.NewResultShortRequeueDelayWithError(err), nil
	}
	// Check if every dependency is ready, skip ones that are not enabled
	for _, moduleName := range depModulesNames {
		res := isDependencyReady(ctx, vz, moduleName)
		if res.ShouldRequeue() {
			remainingDeps = append(remainingDeps, moduleName)
		}
	}

	if len(remainingDeps) > 0 {
		return result.NewResultShortRequeueDelay(), remainingDeps
	}
	return result.NewResult(), nil
}

// isDependencyReady checks if a single dependency is ready.  Return requeue if not ready.
func isDependencyReady(ctx handlerspi.HandlerContext, vz *vzapi.Verrazzano, moduleName string) result.Result {
	compCtx, comp, err := getComponentByNameAndContext(ctx, vz, moduleName, "")
	if err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}
	if !comp.IsEnabled(compCtx.EffectiveCR()) {
		return result.NewResult()
	}
	module := moduleapi.Module{}
	nsn := types.NamespacedName{Namespace: vzconst.VerrazzanoInstallNamespace, Name: moduleName}
	if err := ctx.Client.Get(context.TODO(), nsn, &module, &client.GetOptions{}); err != nil {
		if !errors.IsNotFound(err) {
			ctx.Log.ErrorfThrottled("Failed to get Module %s, retrying: %v", moduleName, err)
			return result.NewResultShortRequeueDelayWithError(err)
		}
		return result.NewResultShortRequeueDelay()
	}

	cond := modulestatus.GetReadyCondition(&module)
	if cond == nil || cond.Status != corev1.ConditionTrue {
		return result.NewResultShortRequeueDelay()
	}
	if module.Status.LastSuccessfulGeneration != module.Generation {
		return result.NewResultShortRequeueDelay()
	}
	if module.Status.LastSuccessfulVersion != module.Spec.Version {
		return result.NewResultShortRequeueDelay()
	}

	// Make sure the module version matches the catalog version. This ensures that
	// dependent modules finish upgrade before module that depend on them start upgrade.
	catalog, err := modulecatalog.NewCatalog(config.GetCatalogPath())
	if err != nil {
		ctx.Log.ErrorfThrottled("Error loading module catalog: %v", err)
		return result.NewResultShortRequeueDelayWithError(err)
	}
	version := catalog.GetVersion(comp.Name())
	if version == "" {
		err = ctx.Log.ErrorfThrottledNewErr("Failed to find version for module %s in the module catalog", comp.Name())
		return result.NewResultShortRequeueDelayWithError(err)
	}
	if version != module.Status.LastSuccessfulVersion {
		return result.NewResultShortRequeueDelay()
	}

	return result.NewResult()
}

func getComponentByNameAndContext(ctx handlerspi.HandlerContext, vz *vzapi.Verrazzano, compName string, operation string) (spi.ComponentContext, spi.Component, error) {
	compCtx, err := spi.NewContext(ctx.Log, ctx.Client, vz, nil, false)
	if err != nil {
		compCtx.Log().Errorf("Failed to create component context: %v", err)
		return nil, nil, err
	}

	found, comp := registry.FindComponent(compName)
	if !found {
		compCtx.Log().Errorf("Failed to find component %s in registry: %s", compName)
		return nil, nil, err
	}

	return compCtx.Init(compName).Operation(operation), comp, nil
}

// CheckDependencies checks if the dependencies are ready
func CheckDependencies(ctx handlerspi.HandlerContext, action string, reason moduleapi.ModuleConditionReason) result.Result {
	module := ctx.CR.(*moduleapi.Module)

	_, comp, err := GetComponentAndContext(ctx, string(action))
	if err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}

	// Check if dependencies are ready
	if res, deps := AreDependenciesReady(ctx, comp.GetDependencies()); res.ShouldRequeue() {
		ctx.Log.Oncef("Component %s is waiting for dependent components to be installed", comp.Name())
		msg := fmt.Sprintf("Waiting for dependencies %v", deps)
		modulestatus.UpdateReadyConditionFailed(ctx, module, reason, msg)
		return res
	}
	return result.NewResult()
}
