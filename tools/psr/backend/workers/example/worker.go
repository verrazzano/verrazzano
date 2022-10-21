// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Package example had an example worker is used as the default worker when the helm chart is run without specifying a worker
// override file.
package example

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/verrazzano/verrazzano/tools/psr/backend/config"
	"github.com/verrazzano/verrazzano/tools/psr/backend/spi"
	"sync/atomic"
)

import (
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
)

type exampleWorker struct {
	loggedLinesTotal int64
}

var _ spi.Worker = exampleWorker{}

func NewExampleWorker() spi.Worker {
	return exampleWorker{}
}

func (w exampleWorker) GetEnvDescList() []config.EnvVarDesc {
	return []config.EnvVarDesc{}
}

func (w exampleWorker) GetMetricDescList() []prometheus.Desc {
	return nil
}

func (w exampleWorker) GetMetricList() []prometheus.Metric {
	return nil
}

func (w exampleWorker) WantIterationInfoLogged() bool {
	return true
}

func (w exampleWorker) Work(conf config.CommonConfig, log vzlog.VerrazzanoLogger) error {
	log.Infof("Example Worker doing work")
	atomic.AddInt64(&w.loggedLinesTotal, 1)
	return nil
}
