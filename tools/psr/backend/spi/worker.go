// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package spi

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tools/psr/backend/config"
	"github.com/verrazzano/verrazzano/tools/psr/backend/osenv"
)

// WorkerDesc contains basic information about a worker
type WorkerDesc struct {
	// WorkerType returns the worker type specified by the Env var
	WorkerType string

	// Description returns a description of the worker
	Description string

	// MetricsName returns the worker name used for metrics
	MetricsName string
}

// Worker is an interface that must be implemented by all workers
type Worker interface {

	// PreconditionsMet Checks for any worker preconditions to ensure they are met before DoWork() can be called;
	// returns true if any preconditions are met, or an error if there is an unrecoverable issue
	PreconditionsMet() (bool, error)

	// GetWorkerDesc returns the WorkerDesc for the worker
	GetWorkerDesc() WorkerDesc

	// GetEnvDescList get the Environment variable descriptors used for worker configuration
	GetEnvDescList() []osenv.EnvVarDesc

	// DoWork implements the worker use case
	DoWork(config.CommonConfig, vzlog.VerrazzanoLogger) error

	// WantLoopInfoLogged returns true if the runner should log information for each loop
	WantLoopInfoLogged() bool

	// SetMetricsDesc sets the worker metrics descriptions
	SetMetricsDesc() error

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
