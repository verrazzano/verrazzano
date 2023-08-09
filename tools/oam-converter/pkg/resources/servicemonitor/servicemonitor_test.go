// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package servicemonitor

import (
	"github.com/crossplane/oam-kubernetes-runtime/pkg/oam"
	promoperapi "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	operator "github.com/verrazzano/verrazzano/application-operator/controllers/metricstrait"
	"github.com/verrazzano/verrazzano/tools/oam-converter/pkg/types"
	k8smeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"testing"
)

func TestCreateServiceMonitor(t *testing.T) {
	//Create Conversion Component to house MetricsTrait
	var input types.ConversionComponents
	input.AppName = "test-appconf"
	input.ComponentName = "test-component"
	scrape := "verrazzano-system/vmi-system-prometheus-0"
	port := 7001
	types.InputArgs.IstioEnabled = false

	//Populate MetricsTrait
	trait := &vzapi.MetricsTrait{
		TypeMeta: k8smeta.TypeMeta{
			APIVersion: "oam.verrazzano.io/v1alpha1",
			Kind:       vzapi.MetricsTraitKind,
		},
		ObjectMeta: k8smeta.ObjectMeta{
			Namespace: "test-namespace",
			Name:      "test-trait-name",
			Labels:    map[string]string{oam.LabelAppName: "test-app", oam.LabelAppComponent: "test-comp"},
		},
		Spec: vzapi.MetricsTraitSpec{
			Scraper: &scrape,
			Port:    &port,
		},
	}

	workload := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "oam.verrazzano.io/v1alpha1",
			"kind":       "VerrazzanoHelidonWorkload",
		},
	}

	//Populate Conversion Component
	input.MetricsTrait = trait
	input.Helidonworkload = workload
	// Call the function with the sample inputs
	serviceMonitor, err := CreateServiceMonitor(&input)
	assert.Nil(t, err, "Unexpected error returned from CreateServiceMonitor")

	assert.Equal(t, 1, len(serviceMonitor.Spec.Endpoints))
	if len(serviceMonitor.Spec.Endpoints) == 0 {
		return
	}
	//assert.Equal(t, serviceMonitor.Spec.Endpoints[0].RelabelConfigs[0].Replacement, tt.info.ClusterName)
	assert.Equal(t, 10, len(serviceMonitor.Spec.Endpoints[0].RelabelConfigs))
	//if tt.info.BasicAuthSecret != nil {
	//	assert.NotNil(t, serviceMonitor.Spec.Endpoints[0].BasicAuth)
	//}
	if types.InputArgs.IstioEnabled == false {
		assert.Equal(t, "http", serviceMonitor.Spec.Endpoints[0].Scheme)
	} else {
		assert.Equal(t, "https", serviceMonitor.Spec.Endpoints[0].Scheme)
	}
	wlsWorkload, err := operator.IsWLSWorkload(workload)
	if wlsWorkload {
		assert.Contains(t, serviceMonitor.Spec.Endpoints[0].RelabelConfigs[1].SourceLabels,
			promoperapi.LabelName("__meta_kubernetes_pod_annotation_prometheus_io_scrape"))
		assert.Contains(t, serviceMonitor.Spec.Endpoints[0].RelabelConfigs[1].SourceLabels,
			promoperapi.LabelName("test"))
	} else {
		assert.Contains(t, serviceMonitor.Spec.Endpoints[0].RelabelConfigs[1].SourceLabels,
			promoperapi.LabelName("__meta_kubernetes_pod_annotation_verrazzano_io_metricsEnabled"))

	}
}
