// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package loggen

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
	loggedLinesCount     = "logged_lines_count"
	loggedLinesCountHelp = "The total number of lines logged"

	loggedTotalChars     = "logged_total_chars"
	loggedTotalCharsHelp = "The total number of characters logged"
)

type logGenerator struct {
	spi.Worker
	metricDescList []prometheus.Desc
	*loggenMetrics
}

var _ spi.Worker = logGenerator{}

// loggenMetrics holds the metrics produced by the worker. Metrics must be thread safe.
type loggenMetrics struct {
	loggedLinesCount metrics.MetricItem
	loggedTotalChars metrics.MetricItem
}

func NewLogGenerator() (spi.Worker, error) {
	constLabels := prometheus.Labels{}

	w := logGenerator{loggenMetrics: &loggenMetrics{}}

	d := prometheus.NewDesc(
		prometheus.BuildFQName(metrics.PsrNamespace, w.GetWorkerDesc().MetricsName, loggedLinesCount),
		loggedLinesCountHelp,
		nil,
		constLabels,
	)
	w.metricDescList = append(w.metricDescList, *d)
	w.loggenMetrics.loggedLinesCount.Desc = d

	d = prometheus.NewDesc(
		prometheus.BuildFQName(metrics.PsrNamespace, w.GetWorkerDesc().MetricsName, loggedTotalChars),
		loggedTotalCharsHelp,
		nil,
		constLabels,
	)
	w.metricDescList = append(w.metricDescList, *d)
	w.loggenMetrics.loggedTotalChars.Desc = d

	return w, nil
}

// GetWorkerDesc returns the WorkerDesc for the worker
func (w logGenerator) GetWorkerDesc() spi.WorkerDesc {
	return spi.WorkerDesc{
		EnvName:     config.WorkerTypeLogGen,
		Description: "The log generator worker generates logs to put a load on OpenSearch",
		MetricsName: "loggen",
	}
}

func (w logGenerator) GetEnvDescList() []osenv.EnvVarDesc {
	return []osenv.EnvVarDesc{}
}

func (w logGenerator) WantIterationInfoLogged() bool {
	return false
}

func (w logGenerator) DoWork(conf config.CommonConfig, log vzlog.VerrazzanoLogger) error {
	lc := atomic.AddInt64(&w.loggenMetrics.loggedLinesCount.Val, 1)
	logMsg := fmt.Sprintf("Loggen worker logging line %v", lc)
	log.Infof(logMsg)
	atomic.AddInt64(&w.loggenMetrics.loggedTotalChars.Val, int64(len(logMsg)))
	return nil
}

func (w logGenerator) GetMetricDescList() []prometheus.Desc {
	return w.metricDescList
}

func (w logGenerator) GetMetricList() []prometheus.Metric {
	metrics := []prometheus.Metric{}

	m := prometheus.MustNewConstMetric(
		w.loggenMetrics.loggedLinesCount.Desc,
		prometheus.GaugeValue,
		float64(atomic.LoadInt64(&w.loggenMetrics.loggedLinesCount.Val)))
	metrics = append(metrics, m)

	m = prometheus.MustNewConstMetric(
		w.loggenMetrics.loggedTotalChars.Desc,
		prometheus.GaugeValue,
		float64(atomic.LoadInt64(&w.loggenMetrics.loggedTotalChars.Val)))
	metrics = append(metrics, m)

	return metrics
}
