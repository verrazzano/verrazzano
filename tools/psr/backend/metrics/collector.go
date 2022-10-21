// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/verrazzano/verrazzano/tools/psr/backend/spi"
)

type runCollector struct {
	providers []spi.WorkerMetricsProvider
}

func (rc runCollector) Describe(ch chan<- *prometheus.Desc) {
	// Loop through the metrics providers. Usually it is just the runner and a worker
	for _, p := range rc.providers {
		// Get the metrics for the provider and send the descriptor to the channel
		dd := p.GetMetricDescList()
		for i := range dd {
			ch <- &dd[i]
		}
	}
}

func (rc runCollector) Collect(ch chan<- prometheus.Metric) {
	// Loop through the metrics providers. Usually it is just the runner and a worker
	for _, p := range rc.providers {
		// Get the metrics for the provider and send the metric to the channel
		dd := p.GetMetricList()
		for i := range dd {
			ch <- dd[i]
		}
	}
}
