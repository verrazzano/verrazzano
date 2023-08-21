// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package dex

import (
	"context"
	"github.com/verrazzano/verrazzano/pkg/bom"
	v8oconst "github.com/verrazzano/verrazzano/pkg/constants"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

// GetOverrides gets the installation overrides for the Dex component
func GetOverrides(object runtime.Object) interface{} {
	if effectiveCR, ok := object.(*vzapi.Verrazzano); ok {
		if effectiveCR.Spec.Components.Dex != nil {
			return effectiveCR.Spec.Components.Dex.ValueOverrides
		}
		return []vzapi.Overrides{}
	} else if effectiveCR, ok := object.(*v1beta1.Verrazzano); ok {
		if effectiveCR.Spec.Components.Dex != nil {
			return effectiveCR.Spec.Components.Dex.ValueOverrides
		}
		return []v1beta1.Overrides{}
	}
	return []vzapi.Overrides{}
}

// AppendDexOverrides appends the default overrides for the Dex component
func AppendDexOverrides(ctx spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	bomFile, err := bom.NewBom(config.GetDefaultBOMFilePath())
	if err != nil {
		return kvs, err
	}

	image, err := bomFile.BuildImageOverrides(ComponentName)
	if err != nil {
		return kvs, ctx.Log().ErrorfNewErr("Failed to build Dex image overrides from the Verrazzano BOM: %v", err)
	}
	kvs = append(kvs, image...)

	return kvs, nil
}

// preInstallUpgrade handles pre-install and pre-upgrade processing for the Dex Component
func preInstallUpgrade(ctx spi.ComponentContext) error {
	// Do nothing if dry run
	if ctx.IsDryRun() {
		ctx.Log().Debug("Dex preInstallUpgrade dry run")
		return nil
	}

	// Create the dex namespace if not already created
	ctx.Log().Debugf("Creating namespace %s for Dex", constants.DexNamespace)
	return ensureDexNamespace(ctx)
}

// ensureDexNamespace ensures that the dex namespace is created with the right labels.
func ensureDexNamespace(ctx spi.ComponentContext) error {
	// Create the dex namespace
	namespace := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: constants.DexNamespace}}
	_, err := controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), &namespace, func() error {
		if namespace.Labels == nil {
			namespace.Labels = map[string]string{}
		}
		namespace.Labels[v8oconst.LabelVerrazzanoNamespace] = constants.DexNamespace
		return nil
	})
	return err
}

// Create static user
// Create static clients
