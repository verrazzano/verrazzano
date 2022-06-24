// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricsexporter

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// InitalizeMetricsEndpoint creates a goroutine so this process will not halt execcution of the code
func InitalizeMetricsEndpoint() {
	go InitalizeMetricsEndpointHelper()
}

//InitalizeMetricsEndpointHelper creates and serves a /metrics endpoint at 9100 for Prometheus to scrape metrics from
func InitalizeMetricsEndpointHelper() {
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":9100", nil)

}
