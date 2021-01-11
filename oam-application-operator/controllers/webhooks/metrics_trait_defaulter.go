// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"encoding/json"
	"fmt"
	oamv1 "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/verrazzano/verrazzano/oam-application-operator/apis/oam/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
)

//MetricsTraitDefaulter supplies default MetricsTrait
type MetricsTraitDefaulter struct {
}

var (
	apiVersion          = v1alpha1.GroupVersion.String()
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

//Default method adds default MetricsTrait to ApplicationConfiguration
func (m *MetricsTraitDefaulter) Default(appConfig *oamv1.ApplicationConfiguration, dryRun bool) error {
	for i := range appConfig.Spec.Components {
		found := m.findMetricsTrait(&appConfig.Spec.Components[i])
		if !found {
			m.addDefaultTrait(&appConfig.Spec.Components[i])
		}
	}
	return nil
}

// Cleanup is not used by the metrics trait defaulter
func (m *MetricsTraitDefaulter) Cleanup(appConfig *oamv1.ApplicationConfiguration, dryRun bool) error {
	return nil
}
