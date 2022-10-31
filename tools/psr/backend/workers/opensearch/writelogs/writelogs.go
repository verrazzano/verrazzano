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

const (
	loggedLinesCountTotal     = "logged_lines_count"
	loggedLinesCountTotalHelp = "The total number of lines logged"

	loggedCharsCountTotal     = "logged_chars_total"
	loggedCharsCountTotalHelp = "The total number of characters logged"
)

type logWriter struct {
	spi.Worker
	metricDescList []prometheus.Desc
	*workerMetrics
}

var _ spi.Worker = logWriter{}

// workerMetrics holds the metrics produced by the worker. Metrics must be thread safe.
type workerMetrics struct {
	loggedLinesCountTotal metrics.MetricItem
	loggedCharsCountTotal metrics.MetricItem
}

func NewWriteLogsWorker() (spi.Worker, error) {
	constLabels := prometheus.Labels{}

	w := logWriter{workerMetrics: &workerMetrics{}}

	d := prometheus.NewDesc(
		prometheus.BuildFQName(metrics.PsrNamespace, w.GetWorkerDesc().MetricsName, loggedLinesCountTotal),
		loggedLinesCountTotalHelp,
		nil,
		constLabels,
	)
	w.metricDescList = append(w.metricDescList, *d)
	w.workerMetrics.loggedLinesCountTotal.Desc = d

	d = prometheus.NewDesc(
		prometheus.BuildFQName(metrics.PsrNamespace, w.GetWorkerDesc().MetricsName, loggedCharsCountTotal),
		loggedCharsCountTotalHelp,
		nil,
		constLabels,
	)
	w.metricDescList = append(w.metricDescList, *d)
	w.workerMetrics.loggedCharsCountTotal.Desc = d

	return w, nil
}

// GetWorkerDesc returns the WorkerDesc for the worker
func (w logWriter) GetWorkerDesc() spi.WorkerDesc {
	return spi.WorkerDesc{
		EnvName:     config.WorkerTypeWriteLogs,
		Description: "The writelogs worker writes logs to STDOUT, putting a load on OpenSearch",
		MetricsName: "writelogs",
	}
}

func (w logWriter) GetEnvDescList() []osenv.EnvVarDesc {
	return []osenv.EnvVarDesc{}
}

func (w logWriter) WantIterationInfoLogged() bool {
	return false
}

func (w logWriter) DoWork(conf config.CommonConfig, log vzlog.VerrazzanoLogger) error {
	lc := atomic.AddInt64(&w.workerMetrics.loggedLinesCountTotal.Val, 1)
	logMsg := fmt.Sprintf("Writelogs worker logging line %v", lc)
	log.Infof(logMsg)
	atomic.AddInt64(&w.workerMetrics.loggedCharsCountTotal.Val, int64(len(logMsg)))
	return nil
}

func (w logWriter) GetMetricDescList() []prometheus.Desc {
	return w.metricDescList
}

func (w logWriter) GetMetricList() []prometheus.Metric {
	metrics := []prometheus.Metric{}

	m := prometheus.MustNewConstMetric(
		w.workerMetrics.loggedLinesCountTotal.Desc,
		prometheus.CounterValue,
		float64(atomic.LoadInt64(&w.workerMetrics.loggedLinesCountTotal.Val)))
	metrics = append(metrics, m)

	m = prometheus.MustNewConstMetric(
		w.workerMetrics.loggedCharsCountTotal.Desc,
		prometheus.CounterValue,
		float64(atomic.LoadInt64(&w.workerMetrics.loggedCharsCountTotal.Val)))
	metrics = append(metrics, m)

	return metrics
}
