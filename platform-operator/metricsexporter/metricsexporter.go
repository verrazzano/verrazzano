// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricsexporter

import "github.com/prometheus/client_golang/prometheus"

type metricsExporter struct {
	MetricsMap                  map[string]metricsComponent
	MetricsList                 []prometheus.Collector
	FailedMetrics               map[prometheus.Collector]int
	Registry                    prometheus.Registerer
	ReconcileIndex              int
	ReconcileCounterMetric      prometheus.Counter
	ReconcileLastDurationMetric *prometheus.GaugeVec
	ReconcileErrorCounterMetric prometheus.Counter
}

func (m *metricsExporter) populateAllMetricsList() {
	metricsList := []prometheus.Collector{}
	for k := range m.MetricsMap {
		metricsList = append(metricsList, m.MetricsMap[k].LatestInstallDuration)
		metricsList = append(metricsList, m.MetricsMap[k].LatestUpgradeDuration)
	}
	metricsList = append(metricsList, m.ReconcileCounterMetric, m.ReconcileErrorCounterMetric, m.ReconcileLastDurationMetric)
	m.MetricsList = metricsList
}
