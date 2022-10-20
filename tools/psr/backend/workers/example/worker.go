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
	"time"
)

import (
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
)

const (
	msgSize = "PSR_MSG_SIZE"
)

type ExampleWorker struct{}

var _ spi.Worker = ExampleWorker{}

// metrics holds the metrics produced by the worker. Metrics must be thread safe.
type metrics struct {
	loopCount   int64
	elapsedSecs int64
}

func (w ExampleWorker) GetEnvDescList() []config.EnvVarDesc {
	return []config.EnvVarDesc{
		{Key: msgSize, DefaultVal: "20", Required: false}}
}

func (w ExampleWorker) Work(conf config.CommonConfig, log vzlog.VerrazzanoLogger) {
	startTime := time.Now().Unix()
	m := metrics{}
	for {
		atomic.AddInt64(&m.loopCount, 1)
		secs := time.Now().Unix() - startTime
		atomic.SwapInt64(&m.elapsedSecs, secs)
		log.Infof("Example Worker Doing Work")
		log.Infof("Loop Count: %v, Elapsed Secs: %v", m.loopCount, m.elapsedSecs)
		time.Sleep(conf.IterationSleepNanos)
	}
}

func (w ExampleWorker) GetMetricDescList() []prometheus.Desc {
	return nil
}

func (w ExampleWorker) GetMetricList() []prometheus.Metric {
	return nil
}
