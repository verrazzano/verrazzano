// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package spi

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tools/psr/backend/config"
)

// Worker is an interface that must be implemented by all workers
type Worker interface {
	// GetEnvDescList get the Environment variable descriptors used for worker configuration
	GetEnvDescList() []config.EnvVarDesc

	// DoWork implements the worker use case
	DoWork(config.CommonConfig, vzlog.VerrazzanoLogger) error

	// WantIterationInfoLogged returns true if the runner should log information for each iteration
	WantIterationInfoLogged() bool

	// WorkerMetricsProvider is an interface to get prometheus metrics information for the worker
	WorkerMetricsProvider
}

// WorkerMetricsProvider is an interface that provides Prometheus metrics information
type WorkerMetricsProvider interface {
	// GetMetricDescList returns the prometheus metrics descriptors for the worker metrics.  Must be thread safe
	GetMetricDescList() []prometheus.Desc

	// GetMetricList returns the realtime metrics for the worker.  Must be thread safe
	GetMetricList() []prometheus.Metric
}
