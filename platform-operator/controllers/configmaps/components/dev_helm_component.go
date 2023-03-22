// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package components

import (
	"fmt"

	"github.com/verrazzano/verrazzano/pkg/helm"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	helmcomp "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"path/filepath"
)

const (
	devComponentConfigMapKindLabel         = "experimental.verrazzano.io/configmap-kind"
	devComponentConfigMapKindHelmComponent = "HelmComponent"
	devComponentConfigMapAPIVersionLabel   = "experimental.verrazzano.io/configmap-apiversion"
	devComponentConfigMapAPIVersionv1beta2 = "v1beta2"
	componentNameKey                       = "name"
	componentNamespaceKey                  = "namespace"
	chartPathKey                           = "chartPath"
	overridesKey                           = "overrides"
)

type devComponent struct {
	helmcomp.HelmComponent
}

var _ spi.Component = devComponent{}

func newDevHelmComponent(cm v1.ConfigMap) (devComponent, error) {
	componentName, ok := cm.Data[componentNameKey]
	if !ok {
		return devComponent{}, fmt.Errorf("ConfigMap %s does not contain the name field, cannot reconcile component", cm.Name)
	}

	componentNamespace, ok := cm.Data[componentNamespaceKey]
	if !ok {
		return devComponent{}, fmt.Errorf("ConfigMap %s does not contain the namespace field, cannot reconcile component %s", cm.Name, componentName)
	}

	chartPath, ok := cm.Data[chartPathKey]
	if !ok {
		return devComponent{}, fmt.Errorf("ConfigMap %s does not contain the chartPath field, cannot reconcile component %s", cm.Name, componentName)
	}

	return devComponent{
		helmcomp.HelmComponent{
			ReleaseName:             componentName,
			ChartDir:                filepath.Join(config.GetThirdPartyDir(), chartPath),
			ChartNamespace:          componentNamespace,
			IgnoreNamespaceOverride: true,
			GetInstallOverridesFunc: func(_ runtime.Object) interface{} {
				return []v1alpha1.Overrides{{
					ConfigMapRef: &v1.ConfigMapKeySelector{
						Key: overridesKey,
						LocalObjectReference: v1.LocalObjectReference{
							Name: cm.Name,
						},
					},
				}}
			},
			ImagePullSecretKeyname: constants.GlobalImagePullSecName,
		},
	}, nil
}

// IsReady Indicates whether a component is available and ready
func (h devComponent) IsReady(context spi.ComponentContext) bool {
	if context.IsDryRun() {
		context.Log().Debugf("IsReady() dry run for %s", h.ReleaseName)
		return true
	}

	if deployed, _ := helm.IsReleaseDeployed(h.ReleaseName, h.ChartNamespace); deployed {
		return true
	}
	return false
}
