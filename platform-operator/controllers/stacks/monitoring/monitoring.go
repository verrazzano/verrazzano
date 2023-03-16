// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package monitoring

import (
	"path/filepath"

	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/stacks/stackspi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/grafana"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	promadapter "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/prometheus/adapter"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/prometheus/kubestatemetrics"
	promnodeexporter "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/prometheus/nodeexporter"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	v1 "k8s.io/api/core/v1"
	// apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

// StackName is the name of the stack
const StackName = "verrazzano-monitoring-stack"

// StackNamespace is the namespace of the stack
const StackNamespace = constants.VerrazzanoMonitoringStackNamespace

// StackJSONName is the JSON name of the component in the configuration
const StackJSONName = "monitoring"

const chartDir = "prometheus-community/kube-prometheus-stack"

var dependencyComponentNames = []string{
	grafana.ComponentName,
	promadapter.ComponentName,
	promnodeexporter.ComponentName,
	kubestatemetrics.ComponentName,
}

type monitoringStack struct {
	stackspi.HelmStackComponent
}

func NewStackComponent() stackspi.StackComponent {
	return monitoringStack{
		stackspi.HelmStackComponent{
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
				Dependencies:              []string{},
				ValuesFile:                filepath.Join(config.GetHelmOverridesDir(), "monitoring-stack-values.yaml"),
				AppendOverridesFunc:       AppendVZOverrides,
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

// AppendVZOverrides appends the overrides that VZ wants to put in (namespaces mainly) i.e. not user overrides
func AppendVZOverrides(context spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	kvs, err := appendNamespaceOverrides(kvs)
	if err != nil {
		return kvs, err
	}
	return kvs, nil
	// return appendComponentValuesFiles(kvs, context.Log())
}

func appendNamespaceOverrides(kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	kvs = append(kvs,
		// the install namespaces of individual components - should we override these?
		/*bom.KeyValue{Key: "namespaceOverride", Value: StackNamespace},
		bom.KeyValue{Key: "grafana.namespaceOverride", Value: StackNamespace},
		bom.KeyValue{Key: "kube-state-metrics.namespaceOverride", Value: StackNamespace},
		bom.KeyValue{Key: "prometheus-adapter.namespaceOverride", Value: StackNamespace},
		bom.KeyValue{Key: "prometheus-node-exporter.namespaceOverride", Value: StackNamespace},
		bom.KeyValue{Key: "alertmanager.namespace", Value: StackNamespace},*/

		// the namespaces that prometheus operator monitors for prometheuses, alertmanagers etc
		bom.KeyValue{Key: "prometheusOperator.namespaces.releaseNamespace", Value: "true"},
	)
	return kvs, nil
}

func (m monitoringStack) ReconcileStack(ctx stackspi.StackContext) error {
	compContext := ctx.Init(StackName).Operation(constants.InstallOperation)
	compLog := compContext.Log()
	ctx.Log().Infof("Reconciling Monitoring stack with configmap %s", ctx.GetStackConfigMap().Name)
	if err := m.PreInstall(ctx); err != nil {
		compLog.Errorf("Failed preinstall: %v", err)
		return err
	}
	if err := m.Install(ctx); err != nil {
		compLog.Errorf("Failed install: %v", err)
		return err
	}
	m.IsReady(ctx)
	if err := m.PostInstall(ctx); err != nil {
		compLog.Errorf("Failed post install: %v", err)
		return err
	}

	return nil
}

// GetOverrides gets the install overrides
func GetOverrides(monitoringConfig v1.ConfigMap, object runtime.Object) interface{} {
	// TODO parse configmap data into monitoring config and extract overrides
	// var jsonValues apiextensionsv1.JSON

	if _, ok := object.(*vzapi.Verrazzano); ok {
		return []vzapi.Overrides{}
	} else if _, ok := object.(*installv1beta1.Verrazzano); ok {
		return []installv1beta1.Overrides{}
	}

	return []vzapi.Overrides{}
}
