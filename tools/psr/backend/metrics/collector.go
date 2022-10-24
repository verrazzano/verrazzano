// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/verrazzano/verrazzano/tools/psr/backend/spi"
)

type workerCollector struct {
	providers []spi.WorkerMetricsProvider
}

// MetricItem contains the information for a single metric
type MetricItem struct {
	Val  int64
	Desc *prometheus.Desc
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
