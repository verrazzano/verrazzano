// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package writelogs

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tools/psr/backend/config"
	"github.com/verrazzano/verrazzano/tools/psr/backend/metrics"
	"github.com/verrazzano/verrazzano/tools/psr/backend/osenv"
	"github.com/verrazzano/verrazzano/tools/psr/backend/spi"
	"sync/atomic"
)

type worker struct {
	metricDescList []prometheus.Desc
	*workerMetrics
}

var _ spi.Worker = worker{}

// workerMetrics holds the metrics produced by the worker. Metrics must be thread safe.
type workerMetrics struct {
	loggedLinesCountTotal metrics.MetricItem
	loggedCharsCountTotal metrics.MetricItem
}

func NewWriteLogsWorker() (spi.Worker, error) {
	w := worker{workerMetrics: &workerMetrics{
		loggedLinesCountTotal: metrics.MetricItem{
			Name: "logged_lines_count_total",
			Help: "The total number of lines logged",
			Type: prometheus.CounterValue,
		},
		loggedCharsCountTotal: metrics.MetricItem{
			Name: "logged_chars_total",
			Help: "The total number of characters logged",
			Type: prometheus.CounterValue,
		},
	}}

	w.metricDescList = []prometheus.Desc{
		*w.loggedLinesCountTotal.BuildMetricDesc(w.GetWorkerDesc().MetricsName),
		*w.loggedCharsCountTotal.BuildMetricDesc(w.GetWorkerDesc().MetricsName),
	}
	return w, nil
}

// GetWorkerDesc returns the WorkerDesc for the worker
func (w worker) GetWorkerDesc() spi.WorkerDesc {
	return spi.WorkerDesc{
		WorkerType:  config.WorkerTypeWriteLogs,
		Description: "The writelogs worker writes logs to STDOUT, putting a load on OpenSearch",
		MetricsName: "writelogs",
	}
}

func (w worker) GetEnvDescList() []osenv.EnvVarDesc {
	return []osenv.EnvVarDesc{}
}

func (w worker) WantLoopInfoLogged() bool {
	return false
}

func (w worker) DoWork(conf config.CommonConfig, log vzlog.VerrazzanoLogger) error {
	lc := atomic.AddInt64(&w.workerMetrics.loggedLinesCountTotal.Val, 1)
	logMsg := fmt.Sprintf("Writelogs worker logging line %v", lc)
	log.Infof(logMsg)
	atomic.AddInt64(&w.workerMetrics.loggedCharsCountTotal.Val, int64(len(logMsg)))
	return nil
}

func (w worker) GetMetricDescList() []prometheus.Desc {
	return w.metricDescList
}

func (w worker) GetMetricList() []prometheus.Metric {
	return []prometheus.Metric{
		w.loggedLinesCountTotal.BuildMetric(),
		w.loggedCharsCountTotal.BuildMetric(),
	}
}
