// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package servicemonitor

import (
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/tools/oam-converter/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"

	promoperapi "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	asserts "github.com/stretchr/testify/assert"
	metrics "github.com/verrazzano/verrazzano/pkg/metrics"
	corev1 "k8s.io/api/core/v1"
)

func TestCreateServiceMonitor(t *testing.T) {
	trueVal := true
	falseVal := false

	tests := []struct {
		name        string
		info        metrics.ScrapeInfo
		expectError bool
	}{
		{
			name:        "empty info",
			info:        metrics.ScrapeInfo{},
			expectError: false,
		},
		{
			name: "true value test",
			info: metrics.ScrapeInfo{
				Ports:              5,
				BasicAuthSecret:    &corev1.Secret{},
				IstioEnabled:       &trueVal,
				VZPrometheusLabels: &trueVal,
				KeepLabels:         map[string]string{"test": "label"},
				ClusterName:        "local1",
			},
			expectError: false,
		},
		{
			name: "false value test",
			info: metrics.ScrapeInfo{
				Ports:              3,
				BasicAuthSecret:    &corev1.Secret{},
				IstioEnabled:       &falseVal,
				VZPrometheusLabels: &falseVal,
				KeepLabels:         map[string]string{"test": "label"},
				ClusterName:        "local2",
			},
			expectError: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metricsTrait := &vzapi.MetricsTrait{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-namespace",
				},
			}

			//create conversion component to house metricstrait
			var input *types.ConversionComponents
			input.MetricsTrait = metricsTrait
			serviceMonitor,err := CreateServiceMonitor(input)
			if tt.expectError {
				asserts.Error(t, err)
			} else {
				asserts.NoError(t, err)
				asserts.Equal(t, tt.info.Ports, len(serviceMonitor.Spec.Endpoints))
				if len(serviceMonitor.Spec.Endpoints) == 0 {
					return
				}
				asserts.Equal(t, serviceMonitor.Spec.Endpoints[0].RelabelConfigs[0].Replacement, tt.info.ClusterName)
				asserts.Equal(t, 10, len(serviceMonitor.Spec.Endpoints[0].RelabelConfigs))
				if tt.info.BasicAuthSecret != nil {
					asserts.NotNil(t, serviceMonitor.Spec.Endpoints[0].BasicAuth)
				}
				if tt.info.IstioEnabled == nil || tt.info.IstioEnabled == &falseVal {
					asserts.Equal(t, "http", serviceMonitor.Spec.Endpoints[0].Scheme)
				} else {
					asserts.Equal(t, "https", serviceMonitor.Spec.Endpoints[0].Scheme)
				}
				if tt.info.VZPrometheusLabels == nil || tt.info.VZPrometheusLabels == &falseVal {
					asserts.Contains(t, serviceMonitor.Spec.Endpoints[0].RelabelConfigs[1].SourceLabels,
						promoperapi.LabelName("__meta_kubernetes_pod_annotation_prometheus_io_scrape"))
					asserts.Contains(t, serviceMonitor.Spec.Endpoints[0].RelabelConfigs[1].SourceLabels,
						promoperapi.LabelName("test"))
				} else {
					asserts.Contains(t, serviceMonitor.Spec.Endpoints[0].RelabelConfigs[1].SourceLabels,
						promoperapi.LabelName("__meta_kubernetes_pod_annotation_verrazzano_io_metricsEnabled"))
					if len(serviceMonitor.Spec.Endpoints) >= 1 {
						asserts.Contains(t, serviceMonitor.Spec.Endpoints[1].RelabelConfigs[1].SourceLabels,
							promoperapi.LabelName("__meta_kubernetes_pod_annotation_verrazzano_io_metricsEnabled1"))
					}
					asserts.Contains(t, serviceMonitor.Spec.Endpoints[0].RelabelConfigs[1].SourceLabels,
						promoperapi.LabelName("test"))
				}
			}
		})
	}
}
