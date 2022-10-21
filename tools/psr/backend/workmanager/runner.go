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
	// Init initializes the runner. This is called once at startup, before RunWork is called
	Init(config.CommonConfig, vzlog.VerrazzanoLogger) error

	// GetEnvDescList get the Environment variable descriptors used for worker configuration
	GetEnvDescList() []config.EnvVarDesc

	// RunWorker runs the worker use case in a loop
	RunWorker(config.CommonConfig, vzlog.VerrazzanoLogger) error

	// WantIterationInfoLogged returns true if the runner should log information for each iteration
	WantIterationInfoLogged() bool

	// WorkerMetricsProvider is an interface to get prometheus metrics information for the worker
	spi.WorkerMetricsProvider
}

// Runner is needed to run the worker
type Runner struct {
	spi.Worker
	runnerMetrics
}

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

var metricDescList []prometheus.Desc

// GetEnvDescList get the Environment variable descriptors used for worker configuration
func (r Runner) GetEnvDescList() []config.EnvVarDesc {
	return []config.EnvVarDesc{}
}

// Init creates the metrics descriptors
func (r Runner) Init(conf config.CommonConfig, log vzlog.VerrazzanoLogger) error {
	constLabels := prometheus.Labels{}

	d := prometheus.NewDesc(
		prometheus.BuildFQName("PSR", "Worker", "loop_count"),
		"The number of loop iterations executed",
		nil,
		constLabels,
	)
	metricDescList = append(metricDescList, *d)

	if err := r.Worker.Init(conf, log); err != nil {
		return err
	}
	return nil
}

// GetMetricDescList returns the prometheus metrics descriptors for the worker metrics.  Must be thread safe
func (r Runner) GetMetricDescList() []prometheus.Desc {
	return metricDescList
}

// GetMetricList returns the realtime metrics for the worker.  Must be thread safe
func (r Runner) GetMetricList() []prometheus.Metric {
	metrics := []prometheus.Metric{}

	m := prometheus.MustNewConstMetric(
		r.runnerMetrics.loopCount.desc,
		prometheus.GaugeValue,
		float64(atomic.LoadInt64(&r.runnerMetrics.loopCount.val)))
	metrics = append(metrics, m)

	return metrics
}

// RunWorker runs the worker in a loop
func (r Runner) RunWorker(conf config.CommonConfig, log vzlog.VerrazzanoLogger) error {
	startTime := time.Now().Unix()
	for {
		atomic.AddInt64(&r.runnerMetrics.loopCount.val, 1)
		secs := time.Now().Unix() - startTime
		atomic.StoreInt64(&r.runnerMetrics.elapsedSecs.val, secs)

		// call the wrapped worker.  Log any error but keep working
		err := r.Worker.Work(conf, log)
		if err != nil {
			log.Error("Failed calling %s to do work: %v", err)
		}
		if r.Worker.WantIterationInfoLogged() {
			log.Infof("Loop Count: %v, Elapsed Secs: %v", r.runnerMetrics.loopCount, r.runnerMetrics.elapsedSecs)
		}
		time.Sleep(conf.IterationSleepNanos)
	}
}
