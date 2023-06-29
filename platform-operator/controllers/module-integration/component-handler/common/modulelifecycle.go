// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano-modules/common/handlerspi"
	moduleapi "github.com/verrazzano/verrazzano-modules/module-operator/apis/platform/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func (h BaseHandler) GetModuleLifecycle(ctx handlerspi.HandlerContext) (*moduleapi.ModuleLifecycle, error) {
	mlc := moduleapi.ModuleLifecycle{}
	nsn := types.NamespacedName{
		Name:      h.MlcName,
		Namespace: h.ModuleCR.Namespace,
	}

	if err := ctx.Client.Get(context.TODO(), nsn, &mlc); err != nil {
		ctx.Log.Progressf("Retrying get for ModuleLifecycle %v: %v", nsn, err)
		return nil, err
	}
	return &mlc, nil
}

// DeleteModuleLifecycle deletes a moduleLifecycle
func (h BaseHandler) DeleteModuleLifecycle(ctx handlerspi.HandlerContext) error {
	mlc := moduleapi.ModuleLifecycle{
		ObjectMeta: metav1.ObjectMeta{
			Name:      h.MlcName,
			Namespace: h.ModuleCR.Namespace,
		},
	}

	if err := ctx.Client.Delete(context.TODO(), &mlc); err != nil {
		ctx.Log.ErrorfThrottled("Failed trying to delete ModuleLifecycles/%s: %v", mlc.Namespace, mlc.Name, err)
		return err
	}
	return nil
}

func DeriveModuleLifeCycleName(moduleCRName string, lifecycleClassName moduleapi.LifecycleClassType, action moduleapi.ModuleLifecycleActionType) string {
	return fmt.Sprintf("%s-%s-%s", moduleCRName, lifecycleClassName, action)
}
