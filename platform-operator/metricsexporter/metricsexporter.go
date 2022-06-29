// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package metricsexporter

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/util/wait"
)

// PreInstall called multiple times but only successfully completed once
//
//InitalizeMetricsEndpoint creates and serves a /metrics endpoint at 9100 for Prometheus to scrape metrics from
func InitalizeMetricsEndpoint() {
	go wait.Until(func() {
		http.Handle("/metrics", promhttp.Handler())
		err := http.ListenAndServe(":9100", nil)
		if err != nil {
			zap.S().Errorf("Failed to start metrics server for verrazzano-platform-operator: %v", err)
		}
	}, time.Second*3, wait.NeverStop)
}
