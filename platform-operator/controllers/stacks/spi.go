// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package stacks

import (
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

type StackContext interface {
	spi.ComponentContext
	// GetStackConfigMap returns the ConfigMap containing the stack configuration
	GetStackConfigMap() v1.ConfigMap
}

// getConfigMapInstallOverridesSig is an optional function called to generate additional overrides
// based on a ConfigMap containing the component configuration
type getConfigMapInstallOverridesSig func(cm v1.ConfigMap, cr runtime.Object) interface{}

type StackComponent interface {
	spi.Component
	ReconcileStack(ctx StackContext) error
}

type HelmStackComponent struct {
	helm.HelmComponent
	GetConfigMapInstallOverridesFunc getConfigMapInstallOverridesSig
}

type stackContext struct {
	spi.ComponentContext
	stackConfig v1.ConfigMap
}

// GetConfigMapOverrides returns the list of install overrides for a ConfigMap based component configuration
func (h HelmStackComponent) GetConfigMapInstallOverrides(cm v1.ConfigMap, cr runtime.Object) interface{} {
	if h.GetConfigMapInstallOverridesFunc != nil {
		return h.GetConfigMapInstallOverridesFunc(cm, cr)
	}
	if _, ok := cr.(*v1beta1.Verrazzano); ok {
		return []v1beta1.Overrides{}
	}
	return []v1alpha1.Overrides{}

}

// GetStackConfigMap returns the config map that contains the configuration of the stack
func (c stackContext) GetStackConfigMap() v1.ConfigMap {
	return c.stackConfig
}

// NewStackContext creates a new StackContext which wraps the given config map
func NewStackContext(log vzlog.VerrazzanoLogger,
	c clipkg.Client,
	actualCR *v1alpha1.Verrazzano,
	actualV1beta1CR *v1beta1.Verrazzano,
	stackConfig v1.ConfigMap,
	dryRun bool) (StackContext, error) {

	compCtx, err := spi.NewContext(log, c, actualCR, actualV1beta1CR, dryRun)
	if err != nil {
		return nil, err
	}
	return stackContext{
		ComponentContext: compCtx,
		stackConfig:      stackConfig,
	}, nil
}
