// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metrics

import (
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"net/http"
	"os"
	"time"
)

func StartMetricsServerOrDie() {
	http.Handle("/metrics", promhttp.Handler())
	server := http.Server{
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		Addr:         "0.0.0.0:9090"}

	if err := server.ListenAndServe(); err != nil {
		zap.S().Errorf("Failed to start metrics server: %v", err)
		os.Exit(1)
	}
}
