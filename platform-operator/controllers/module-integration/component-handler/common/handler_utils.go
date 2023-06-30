// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"context"
	"fmt"
	moduleapi "github.com/verrazzano/verrazzano-modules/module-operator/apis/platform/v1alpha1"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/handlerspi"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"k8s.io/apimachinery/pkg/types"
)

func GetComponentAndContext(ctx handlerspi.HandlerContext) (spi.ComponentContext, spi.Component, error) {
	module := ctx.CR.(*moduleapi.Module)

	vz, err := GetVerrazzanoCR(ctx)
	if err != nil {
		return nil, nil, err
	}

	compCtx, err := spi.NewContext(vzlog.DefaultLogger(), ctx.Client, vz, nil, false)
	if err != nil {
		compCtx.Log().Errorf("Failed to create component context: %v", err)
		return nil, nil, err
	}

	found, comp := registry.FindComponent(module.Spec.ModuleName)
	if !found {
		compCtx.Log().Errorf("Failed to find component %s in registry: %s", module.Spec.ModuleName)
		return nil, nil, err
	}

	return compCtx, comp, nil
}

func GetVerrazzanoCR(ctx handlerspi.HandlerContext) (*vzapi.Verrazzano, error) {
	module := ctx.CR.(*moduleapi.Module)
	var name, ns string
	if module.Annotations != nil {
		ns = module.Annotations[constants.VerrazzanoCRNamespaceAnnotation]
		name = module.Annotations[constants.VerrazzanoCRNameAnnotation]
	}
	if name == "" || ns == "" {
		return nil, fmt.Errorf("Module %s is missing annotations for verrazzano CR name and namespace", module.Name)
	}

	vz := &vzapi.Verrazzano{}
	if err := ctx.Client.Get(context.TODO(), types.NamespacedName{Namespace: ns, Name: name}, vz); err != nil {
		return nil, err
	}
	return vz, nil
}
