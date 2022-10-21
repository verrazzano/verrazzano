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
	// RunWorker runs the worker use case in a loop
	RunWorker(config.CommonConfig, vzlog.VerrazzanoLogger) error

	// WorkerMetricsProvider is an interface to get prometheus metrics information for the worker
	spi.WorkerMetricsProvider
}

// runner is needed to run the worker
type runner struct {
	spi.Worker
	metricDescList []prometheus.Desc
	*runnerMetrics
}

var _ WorkerRunner = runner{}

// metricInfo contains the information for a single metric
type metricInfo struct {
	val  int64
	desc *prometheus.Desc
}

// runnerMetrics holds the metrics produced by the runner. Metrics must be thread safe.
type runnerMetrics struct {
	loopCount   metricInfo
	elapsedSecs metricInfo
}

// NewRunner creates a new runner
func NewRunner(worker spi.Worker, conf config.CommonConfig, log vzlog.VerrazzanoLogger) (WorkerRunner, error) {
	constLabels := prometheus.Labels{}

	r := runner{Worker: worker, runnerMetrics: &runnerMetrics{}}

	d := prometheus.NewDesc(
		prometheus.BuildFQName("psr", "loggen", "loop_count"),
		"The number of loop iterations executed",
		nil,
		constLabels,
	)
	r.metricDescList = append(r.metricDescList, *d)
	r.runnerMetrics.loopCount.desc = d

	return r, nil
}

// GetMetricDescList returns the prometheus metrics descriptors for the worker metrics.  Must be thread safe
func (r runner) GetMetricDescList() []prometheus.Desc {
	return r.metricDescList
}

// GetMetricList returns the realtime metrics for the worker.  Must be thread safe
func (r runner) GetMetricList() []prometheus.Metric {
	metrics := []prometheus.Metric{}

	m := prometheus.MustNewConstMetric(
		r.runnerMetrics.loopCount.desc,
		prometheus.GaugeValue,
		float64(atomic.LoadInt64(&r.runnerMetrics.loopCount.val)))
	metrics = append(metrics, m)

	return metrics
}

// RunWorker runs the worker in a loop
func (r runner) RunWorker(conf config.CommonConfig, log vzlog.VerrazzanoLogger) error {
	startTime := time.Now().Unix()
	for {
		atomic.AddInt64(&r.runnerMetrics.loopCount.val, 1)
		secs := time.Now().Unix() - startTime
		atomic.StoreInt64(&r.runnerMetrics.elapsedSecs.val, secs)

		// call the wrapped worker.  Log any error but keep working
		err := r.Worker.DoWork(conf, log)
		if err != nil {
			log.Error("Failed calling %s to do work: %v", err)
		}
		if r.Worker.WantIterationInfoLogged() {
			log.Infof("Loop Count: %v, Elapsed Secs: %v", r.runnerMetrics.loopCount, r.runnerMetrics.elapsedSecs)
		}
		time.Sleep(conf.IterationSleepNanos)
	}
}
