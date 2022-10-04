// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	"encoding/json"
	"fmt"

	oamv1 "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/controllers/metricstrait"
	vznav "github.com/verrazzano/verrazzano/application-operator/controllers/navigation"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MetricsTraitDefaulter supplies default MetricsTrait
type MetricsTraitDefaulter struct {
	Client client.Client
}

var (
	apiVersion          = v1alpha1.SchemeGroupVersion.String()
	defaultMetricsTrait = []byte(fmt.Sprintf(traitTemp, apiVersion, v1alpha1.MetricsTraitKind))
)

const traitTemp = `{
	"apiVersion": "%s",
	"kind": "%s"
 }`

func (m *MetricsTraitDefaulter) findMetricsTrait(component *oamv1.ApplicationConfigurationComponent) bool {
	for _, trait := range component.Traits {
		var rawTrait map[string]interface{}
		json.Unmarshal(trait.Trait.Raw, &rawTrait)
		if rawTrait["apiVersion"] == apiVersion && rawTrait["kind"] == v1alpha1.MetricsTraitKind {
			return true
		}
	}
	return false
}

func (m *MetricsTraitDefaulter) addDefaultTrait(component *oamv1.ApplicationConfigurationComponent) {
	rawTrait := runtime.RawExtension{Raw: defaultMetricsTrait}
	componentTrait := oamv1.ComponentTrait{Trait: rawTrait}
	component.Traits = append(component.Traits, componentTrait)
}

// Default method adds default MetricsTrait to ApplicationConfiguration
func (m *MetricsTraitDefaulter) Default(appConfig *oamv1.ApplicationConfiguration, dryRun bool, log *zap.SugaredLogger) error {
	for i := range appConfig.Spec.Components {
		appConfigComponent := &appConfig.Spec.Components[i]
		if m.shouldDefaultTraitBeAdded(appConfig, appConfigComponent, log) {
			m.addDefaultTrait(appConfigComponent)
		}
	}
	return nil
}

// Cleanup is not used by the metrics trait defaulter
func (m *MetricsTraitDefaulter) Cleanup(appConfig *oamv1.ApplicationConfiguration, dryRun bool, log *zap.SugaredLogger) error {
	return nil
}

// shouldDefaultTraitBeAdded method verifies whether a trait should be applied to the component
func (m *MetricsTraitDefaulter) shouldDefaultTraitBeAdded(appConfig *oamv1.ApplicationConfiguration, appConfigComponent *oamv1.ApplicationConfigurationComponent, log *zap.SugaredLogger) bool {
	found := m.findMetricsTrait(appConfigComponent)
	if found {
		return false
	}

	var component oamv1.Component
	err := m.Client.Get(context.TODO(), types.NamespacedName{Namespace: appConfig.GetNamespace(), Name: appConfigComponent.ComponentName}, &component)
	if err != nil {
		log.Debugf("Unable to get component info for component: %s of application configuration: %s/%s, error: %v, adding default metric trait", appConfigComponent.ComponentName, appConfig.GetNamespace(), appConfig.GetName(), err)
		return true

	}

	componentUnstructured, err := vznav.ConvertRawExtensionToUnstructured(&component.Spec.Workload)
	if err != nil || componentUnstructured == nil {
		log.Debugf("Unable to convert workload spec for component: %s of application configuration: %s/%s, error: %v, adding default metric trait", appConfigComponent.ComponentName, appConfig.GetNamespace(), appConfig.GetName(), err)
		return true
	}

	if componentUnstructured.GetNamespace() == "" {
		componentUnstructured.SetNamespace(component.GetNamespace())
	}

	workload, err := vznav.FetchWorkloadResource(context.TODO(), m.Client, vzlog.DefaultLogger(), componentUnstructured)
	if err != nil || workload == nil {
		log.Debugf("Unable to get workload resource for component: %s of application configuration: %s/%s, error: %v, adding default metric trait", appConfigComponent.ComponentName, appConfig.GetNamespace(), appConfig.GetName(), err)
		return true
	}

	apiVerKind, err := vznav.GetAPIVersionKindOfUnstructured(workload)
	if err != nil || apiVerKind == "" {
		log.Debugf("Unable to determine api version and kind for workload of component: %s of application configuration: %s/%s, error: %v, adding default metric trait", appConfigComponent.ComponentName, appConfig.GetNamespace(), appConfig.GetName(), err)
		return true
	}

	workloadType := metricstrait.GetSupportedWorkloadType(apiVerKind)
	if workloadType != "" {
		log.Infof("Adding default metrics trait for supported component: %s of type %s of application configuration: %s/%s", appConfigComponent.ComponentName, workloadType, appConfig.GetNamespace(), appConfig.GetName())
		return true
	}

	return false
}
