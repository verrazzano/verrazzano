// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Package example has an example worker is used as the default worker when the helm chart is run without specifying a worker
// override file.
package example

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tools/psr/backend/config"
	"github.com/verrazzano/verrazzano/tools/psr/backend/osenv"
	"github.com/verrazzano/verrazzano/tools/psr/backend/spi"
)

type state struct {
	loggedLinesTotal int64
}
type exampleWorker struct {
	*state
}

var _ spi.Worker = exampleWorker{}

func NewExampleWorker() (spi.Worker, error) {
	return exampleWorker{&state{}}, nil
}

// GetWorkerDesc returns the WorkerDesc for the worker
func (w exampleWorker) GetWorkerDesc() spi.WorkerDesc {
	return spi.WorkerDesc{
		WorkerType:    config.WorkerTypeExample,
		Description:   "Example worker that demonstrates executing a fake use case",
		MetricsPrefix: config.WorkerTypeExample,
	}
}

func (w exampleWorker) GetEnvDescList() []osenv.EnvVarDesc {
	return []osenv.EnvVarDesc{}
}

func (w exampleWorker) GetMetricDescList() []prometheus.Desc {
	return nil
}

func (w exampleWorker) GetMetricList() []prometheus.Metric {
	return nil
}

func (w exampleWorker) WantLoopInfoLogged() bool {
	return true
}

func (w exampleWorker) PreconditionsMet() (bool, error) {
	return true, nil
}

func (w exampleWorker) DoWork(conf config.CommonConfig, log vzlog.VerrazzanoLogger) error {
	log.Infof("Example Worker doing work")
	w.state.loggedLinesTotal++
	return nil
}
