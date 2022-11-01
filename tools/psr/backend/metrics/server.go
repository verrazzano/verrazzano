// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/verrazzano/verrazzano/tools/psr/backend/spi"
	"go.uber.org/zap"
	"net/http"
	"os"
	"time"
)

// StartMetricsServerOrDie starts the metrics server.  If there is an error the code exits
func StartMetricsServerOrDie(providers []spi.WorkerMetricsProvider) {
	reg := prometheus.NewPedanticRegistry()
	rc := workerCollector{providers: providers}
	reg.MustRegister(
		rc,
	)
	// Add the standard process and Go metrics to the custom registry.
	reg.MustRegister(
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		collectors.NewGoCollector(),
	)

	// Instrument the default metrics
	h1 := promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
	h2 := promhttp.InstrumentMetricHandler(reg, h1)
	http.Handle("/actuator/prometheus", h2)

	server := http.Server{
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		Addr:         "0.0.0.0:9090"}

	if err := server.ListenAndServe(); err != nil {
		zap.S().Errorf("Failed to start metrics server: %v", err)
		os.Exit(1)
	}
}
