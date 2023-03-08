// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package monitoring

import (
	"path/filepath"

	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/stacks"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

// StackName is the name of the stack
const StackName = "monitoring"

// StackNamespace is the namespace of the stack
const StackNamespace = constants.VerrazzanoMonitoringStackNamespace

// StackJSONName is the JSON name of the component in the configuration
const StackJSONName = "monitoring"

const chartDir = "prometheus-community/kube-prometheus-stack"

type monitoringStack struct {
	stacks.HelmStackComponent
}

func NewStackComponent() stacks.StackComponent {
	return monitoringStack{
		stacks.HelmStackComponent{
			GetConfigMapInstallOverridesFunc: GetOverrides,
			HelmComponent: helm.HelmComponent{
				ReleaseName:               StackName,
				JSONName:                  StackJSONName,
				ChartDir:                  filepath.Join(config.GetThirdPartyDir(), chartDir),
				ChartNamespace:            StackNamespace,
				IgnoreNamespaceOverride:   true,
				SupportsOperatorInstall:   true,
				SupportsOperatorUninstall: true,
				MinVerrazzanoVersion:      constants.VerrazzanoVersion1_6_0,
				ImagePullSecretKeyname:    "image.pullSecrets[0]",
				ValuesFile:                filepath.Join(config.GetHelmOverridesDir(), "monitoring-stack-values.yaml"),
				Dependencies:              []string{},
				AvailabilityObjects: &ready.AvailabilityObjects{
					DeploymentNames: []types.NamespacedName{
						{
							Name:      StackName,
							Namespace: StackNamespace,
						},
						// TODO get all availability objects and list them here or write an IsReady
						//  override similar to the one in promoperator_component.go
					},
				},
			},
		},
	}
}

func (m monitoringStack) ReconcileStack(ctx stacks.StackContext) error {
	ctx.Log().Infof("Reconciling Monitoring stack with configmap %s", ctx.GetStackConfigMap())
	return nil
}

// GetOverrides gets the install overrides
func GetOverrides(monitoringConfig v1.ConfigMap, object runtime.Object) interface{} {
	// TODO parse configmap data into monitoring config and extract overrides
	if _, ok := object.(*vzapi.Verrazzano); ok {
		return []vzapi.Overrides{}
	} else if _, ok := object.(*installv1beta1.Verrazzano); ok {
		return []installv1beta1.Overrides{}
	}

	return []vzapi.Overrides{}
}
