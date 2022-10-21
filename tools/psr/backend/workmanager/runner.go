// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package workmanager

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tools/psr/backend/config"
	"github.com/verrazzano/verrazzano/tools/psr/backend/spi"
	"sync/atomic"
	"time"
)

type WorkerRunner interface {
	GetEnvDescList() []config.EnvVarDesc
	RunWorker(config.CommonConfig, vzlog.VerrazzanoLogger) error
	spi.WorkerMetricsProvider
}

// Runner is needed to run the worker
type Runner struct {
	spi.Worker
}

// metrics holds the metrics produced by the worker. Metrics must be thread safe.
type metrics struct {
	loopCount   int64
	elapsedSecs int64
}

func (r Runner) GetEnvDescList() []config.EnvVarDesc {
	return []config.EnvVarDesc{}
}

func (r Runner) RunWorker(conf config.CommonConfig, log vzlog.VerrazzanoLogger) error {
	startTime := time.Now().Unix()
	m := metrics{}
	for {
		atomic.AddInt64(&m.loopCount, 1)
		secs := time.Now().Unix() - startTime
		atomic.SwapInt64(&m.elapsedSecs, secs)

		// call the wrapped worker.  Log any error but keep working
		err := r.Worker.Work(conf, log)
		if err != nil {
			log.Error("Failed calling %s to do work: %v", err)
		}
		if r.Worker.WantIterationInfoLogged() {
			log.Infof("Loop Count: %v, Elapsed Secs: %v", m.loopCount, m.elapsedSecs)
		}
		time.Sleep(conf.IterationSleepNanos)
	}
}

func (r Runner) GetMetricDescList() []prometheus.Desc {
	return nil
}

func (r Runner) GetMetricList() []prometheus.Metric {
	return nil
}
