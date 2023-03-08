// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package monitoring

import "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"

type Monitoring struct {
	// If true, then the monitoring stack will be installed
	Enabled bool `json:"enabled,omitempty"`

	// The Grafana component configuration.
	// +optional
	Grafana *v1beta1.GrafanaComponent `json:"grafana,omitempty"`

	// The kube-state-metrics  component configuration.
	// +optional
	KubeStateMetrics *v1beta1.KubeStateMetricsComponent `json:"kubeStateMetrics,omitempty"`

	// The Prometheus Adapter component configuration.
	// +optional
	PrometheusAdapter *v1beta1.PrometheusAdapterComponent `json:"prometheusAdapter,omitempty"`

	// The Prometheus Node Exporter component configuration.
	// +optional
	PrometheusNodeExporter *v1beta1.PrometheusNodeExporterComponent `json:"prometheusNodeExporter,omitempty"`

	// The Prometheus Operator component configuration.
	// +optional
	PrometheusOperator *v1beta1.PrometheusOperatorComponent `json:"prometheusOperator,omitempty"`

	// The Prometheus Pushgateway component configuration.
	// +optional
	PrometheusPushgateway *v1beta1.PrometheusPushgatewayComponent `json:"prometheusPushgateway,omitempty"`
}
