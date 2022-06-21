// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metrics

import (
	"testing"

	promoperapi "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	asserts "github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	corev1 "k8s.io/api/core/v1"
)

func TestPopulatePodMonitor(t *testing.T) {
	trueVal := true
	falseVal := false
	testNamespace := "test-namespace"

	tests := []struct {
		name        string
		info        ScrapeInfo
		expectError bool
	}{
		{
			name:        "empty info",
			info:        ScrapeInfo{},
			expectError: false,
		},
		{
			name: "true value test",
			info: ScrapeInfo{
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
			info: ScrapeInfo{
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
			podMonitor := &promoperapi.PodMonitor{}
			err := PopulatePodMonitor(tt.info, podMonitor, testNamespace, vzlog.DefaultLogger())
			if tt.expectError {
				asserts.Error(t, err)
			} else {
				asserts.NoError(t, err)
				asserts.Equal(t, tt.info.Ports, len(podMonitor.Spec.PodMetricsEndpoints))
				if len(podMonitor.Spec.PodMetricsEndpoints) == 0 {
					return
				}
				asserts.Equal(t, podMonitor.Spec.PodMetricsEndpoints[0].RelabelConfigs[0].Replacement, tt.info.ClusterName)
				asserts.Equal(t, 9, len(podMonitor.Spec.PodMetricsEndpoints[0].RelabelConfigs))
				if tt.info.BasicAuthSecret != nil {
					asserts.NotNil(t, podMonitor.Spec.PodMetricsEndpoints[0].BasicAuth)
				}
				if tt.info.IstioEnabled == nil || tt.info.IstioEnabled == &falseVal {
					asserts.Equal(t, "http", podMonitor.Spec.PodMetricsEndpoints[0].Scheme)
				} else {
					asserts.Equal(t, "https", podMonitor.Spec.PodMetricsEndpoints[0].Scheme)
					asserts.NotNil(t, podMonitor.Spec.PodMetricsEndpoints[0].TLSConfig)
				}
				if tt.info.VZPrometheusLabels == nil || tt.info.VZPrometheusLabels == &falseVal {
					asserts.Contains(t, podMonitor.Spec.PodMetricsEndpoints[0].RelabelConfigs[1].SourceLabels,
						promoperapi.LabelName("__meta_kubernetes_pod_annotation_prometheus_io_scrape"))
					asserts.Contains(t, podMonitor.Spec.PodMetricsEndpoints[0].RelabelConfigs[1].SourceLabels,
						promoperapi.LabelName("test"))
				} else {
					asserts.Contains(t, podMonitor.Spec.PodMetricsEndpoints[0].RelabelConfigs[1].SourceLabels,
						promoperapi.LabelName("__meta_kubernetes_pod_annotation_verrazzano_io_metricsEnabled"))
					if len(podMonitor.Spec.PodMetricsEndpoints) >= 1 {
						asserts.Contains(t, podMonitor.Spec.PodMetricsEndpoints[1].RelabelConfigs[1].SourceLabels,
							promoperapi.LabelName("__meta_kubernetes_pod_annotation_verrazzano_io_metricsEnabled1"))
					}
					asserts.Contains(t, podMonitor.Spec.PodMetricsEndpoints[0].RelabelConfigs[1].SourceLabels,
						promoperapi.LabelName("test"))
				}
			}
		})
	}
}
