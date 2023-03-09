// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package thanos

import (
	"github.com/verrazzano/verrazzano/pkg/bom"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"k8s.io/apimachinery/pkg/runtime"
)

// GetOverrides gets the install overrides for the Thanos component
func GetOverrides(object runtime.Object) interface{} {
	if effectiveCR, ok := object.(*vzapi.Verrazzano); ok {
		if effectiveCR.Spec.Components.Thanos != nil {
			return effectiveCR.Spec.Components.Thanos.ValueOverrides
		}
		return []vzapi.Overrides{}
	} else if effectiveCR, ok := object.(*v1beta1.Verrazzano); ok {
		if effectiveCR.Spec.Components.Thanos != nil {
			return effectiveCR.Spec.Components.Thanos.ValueOverrides
		}
		return []v1beta1.Overrides{}
	}
	return []vzapi.Overrides{}
}

// AppendOverrides appends the default overrides for the Thanos component
func AppendOverrides(_ spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	bomFile, err := bom.NewBom(config.GetDefaultBOMFilePath())
	if err != nil {
		return kvs, err
	}
	image, err := bomFile.BuildImageOverrides(ComponentName)
	if err != nil {
		return kvs, err
	}
	return append(kvs, image...), nil
}
