// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/verrazzano/verrazzano/tools/psr/backend/spi"
	"sync/atomic"
)

type workerCollector struct {
	providers []spi.WorkerMetricsProvider
}

func (rc workerCollector) Describe(ch chan<- *prometheus.Desc) {
	// Loop through the metrics providers. Usually it is just the runner and a worker
	for _, p := range rc.providers {
		// Get the metrics for the provider and send the descriptor to the channel
		dd := p.GetMetricDescList()
		for i := range dd {
			ch <- &dd[i]
		}
	}
}

func (rc workerCollector) Collect(ch chan<- prometheus.Metric) {
	// Loop through the metrics providers. Usually it is just the runner and a worker
	for _, p := range rc.providers {
		// Get the metrics for the provider and send the metric to the channel
		mm := p.GetMetricList()
		for i := range mm {
			ch <- mm[i]
		}
	}
}

// MetricItem contains the information for a single metric
type MetricItem struct {
	Val         int64
	Desc        *prometheus.Desc
	Name        string
	Help        string
	Type        prometheus.ValueType
	ConstLabels prometheus.Labels
	VarLabels   []string
}

// BuildMetric builds the prometheus metrics from the MetricItem
func (m *MetricItem) BuildMetric() prometheus.Metric {
	return prometheus.MustNewConstMetric(
		m.Desc,
		m.Type,
		float64(atomic.LoadInt64(&m.Val)),
	)
}

// BuildMetricDesc builds the MetricItem description from info about the metric and worker
func (m *MetricItem) BuildMetricDesc(workerMetricsName string) *prometheus.Desc {
	d := prometheus.NewDesc(
		prometheus.BuildFQName(PsrNamespace, workerMetricsName, m.Name),
		m.Help,
		m.VarLabels,
		m.ConstLabels,
	)
	m.Desc = d
	return d
}
