// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	"encoding/json"
	"fmt"
	oamv1 "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	metricsTraitType = "metricstraits.oam.verrazzano.io"
)

//MetricsTraitDefaulter supplies default MetricsTrait
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

//Default method adds default MetricsTrait to ApplicationConfiguration
func (m *MetricsTraitDefaulter) Default(appConfig *oamv1.ApplicationConfiguration, dryRun bool) error {
	appliesToWorkloads, err := m.getAppliesToWorkloads()
	if err != nil {
		return err
	}
	for i := range appConfig.Spec.Components {
		found := m.findMetricsTrait(&appConfig.Spec.Components[i])
		if !found {
			applies, err := m.appliesToWorkload(appConfig, &appConfig.Spec.Components[i], appliesToWorkloads)
			if err != nil {
				return err
			}
			if applies {
				m.addDefaultTrait(&appConfig.Spec.Components[i])
			}
		}
	}
	return nil
}

// Cleanup is not used by the metrics trait defaulter
func (m *MetricsTraitDefaulter) Cleanup(appConfig *oamv1.ApplicationConfiguration, dryRun bool) error {
	return nil
}

// getAppliesToWorkloads gets the set of AppliesToWorkloads from the metrics trait definition
func (m *MetricsTraitDefaulter) getAppliesToWorkloads() (map[string]struct{}, error) {
	// get the metrics trait definition
	appliesToWorkloads := make(map[string]struct{})
	td := &oamv1.TraitDefinition{}
	err := m.Client.Get(context.TODO(), types.NamespacedName{Name: metricsTraitType}, td)
	if err != nil {
		return nil, err
	}
	// build the set of AppliesToWorkloads
	for _, wl := range td.Spec.AppliesToWorkloads {
		appliesToWorkloads[wl] = struct{}{}
	}
	return appliesToWorkloads, nil
}

// appliesToWorkload determines if the workload specified in the given component is in the list of workload kinds
// that the metrics trait applies to.
func (m *MetricsTraitDefaulter) appliesToWorkload(appConfig *oamv1.ApplicationConfiguration, component *oamv1.ApplicationConfigurationComponent, appliesToWorkloads map[string]struct{}) (bool, error) {
	// get the workload from the given component
	comp := &oamv1.Component{}
	err := m.Client.Get(context.TODO(), types.NamespacedName{Namespace: appConfig.Namespace, Name: component.ComponentName}, comp)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	var workload map[string]interface{}
	err = json.Unmarshal(comp.Spec.Workload.Raw, &workload)
	if err != nil {
		return false, err
	}
	_, applies := appliesToWorkloads[fmt.Sprintf("%s.%s", workload["apiVersion"], workload["kind"])]
	return applies, nil
}
