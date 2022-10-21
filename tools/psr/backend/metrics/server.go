// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/verrazzano/verrazzano/tools/psr/backend/spi"
	"go.uber.org/zap"
	"net/http"
	"os"
	"time"
)

func StartMetricsServerOrDie(providers []spi.WorkerMetricsProvider) {
	reg := prometheus.NewPedanticRegistry()
	rc := runCollector{providers: providers}
	reg.MustRegister(
		rc,
	)

	// Register the custom collector and start the metrics server
	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	server := http.Server{
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		Addr:         "0.0.0.0:9090"}

	if err := server.ListenAndServe(); err != nil {
		zap.S().Errorf("Failed to start metrics server: %v", err)
		os.Exit(1)
	}
}
