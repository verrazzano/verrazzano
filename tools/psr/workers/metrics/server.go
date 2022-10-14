// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metrics

import (
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"net/http"
	"os"
)

func StartMetricsServerOrDie() {
	http.Handle("/metrics", promhttp.Handler())
	if err := http.ListenAndServe(":9090", nil); err != nil {
		zap.S().Errorf("Failed to start metrics server: %v", err)
		os.Exit(1)
	}
}
