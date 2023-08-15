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
	port := 7001
	types.InputArgs.IstioEnabled = false
	scrape := "verrazzano-system/vmi-system-prometheus-0"
	tests := []struct {
		name         string
		input        types.ConversionComponents
		istioEnabled bool
		workload     unstructured.Unstructured
	}{
		{
			name: "empty",
			input: types.ConversionComponents{
				AppName:       "test-appconf",
				ComponentName: "test-component",
				MetricsTrait: &vzapi.MetricsTrait{
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
				},
				Weblogicworkload: &unstructured.Unstructured{
					Object: map[string]interface{}{
						"apiVersion": "oam.verrazzano.io/v1alpha1",
						"kind":       "VerrazzanoWebLogicWorkload",
					},
				},
			},
			istioEnabled: false,
		},
		{
			name: "helidon input",
			input: types.ConversionComponents{
				AppName:       "test-appconf",
				ComponentName: "test-component",
				MetricsTrait: &vzapi.MetricsTrait{
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
				},
				Helidonworkload: &unstructured.Unstructured{
					Object: map[string]interface{}{
						"apiVersion": "oam.verrazzano.io/v1alpha1",
						"kind":       "VerrazzanoHelidonWorkload",
					},
				},
			},
			istioEnabled: false,
		},
		{
			name: "hello helidon",
			input: types.ConversionComponents{
				AppName:       "test-appconf",
				ComponentName: "test-component",
				MetricsTrait: &vzapi.MetricsTrait{
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
				},
				Coherenceworkload: &unstructured.Unstructured{
					Object: map[string]interface{}{
						"apiVersion": "oam.verrazzano.io/v1alpha1",
						"kind":       "VerrazzanoCoherenceWorkload",
					},
				},
			},
			istioEnabled: true,
		},
	}
	// Call the function with the sample inputs
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serviceMonitor, err := CreateServiceMonitor(&tt.input)
			assert.Nil(t, err, "Unexpected error returned from CreateServiceMonitor")
			assert.Equal(t, 1, len(serviceMonitor.Spec.Endpoints))
			if len(serviceMonitor.Spec.Endpoints) == 0 {
				return
			}
			assert.Equal(t, 10, len(serviceMonitor.Spec.Endpoints[0].RelabelConfigs))
			if types.InputArgs.IstioEnabled == false {
				assert.Equal(t, "http", serviceMonitor.Spec.Endpoints[0].Scheme)
			} else {
				assert.Equal(t, "https", serviceMonitor.Spec.Endpoints[0].Scheme)
			}

			//check type of workload
			var workload *unstructured.Unstructured
			if &tt.input.Helidonworkload != nil {
				workload = tt.input.Helidonworkload
			} else if &tt.input.Coherenceworkload != nil {
				workload = tt.input.Coherenceworkload
			} else if &tt.input.Weblogicworkload != nil {
				workload = tt.input.Weblogicworkload
			} else {
				workload = tt.input.Genericworkload
			}

			//check if workload is WLR or not
			wlsWorkload, err := operator.IsWLSWorkload(workload)
			if err != nil {
				t.Fatalf("error in reading yaml file: %v", err)
			}
			if wlsWorkload {
				assert.Contains(t, serviceMonitor.Spec.Endpoints[0].RelabelConfigs[1].SourceLabels,
					promoperapi.LabelName("__meta_kubernetes_pod_annotation_prometheus_io_scrape"))
				assert.Contains(t, serviceMonitor.Spec.Endpoints[0].RelabelConfigs[1].SourceLabels,
					promoperapi.LabelName("test"))
			} else {
				assert.Contains(t, serviceMonitor.Spec.Endpoints[0].RelabelConfigs[1].SourceLabels,
					promoperapi.LabelName("__meta_kubernetes_pod_annotation_verrazzano_io_metricsEnabled"))
			}
		})
	}
}
