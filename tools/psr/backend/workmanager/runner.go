// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package workmanager

import (
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tools/psr/backend/config"
	"github.com/verrazzano/verrazzano/tools/psr/backend/metrics"
	"github.com/verrazzano/verrazzano/tools/psr/backend/spi"
)

// WorkerRunner interface specifies a workerRunner that loops calling a worker
type WorkerRunner interface {
	// StartWorkerRunners runs the worker use case in a loop
	RunWorker(config.CommonConfig, vzlog.VerrazzanoLogger) error

	// WorkerMetricsProvider is an interface to get prometheus metrics information for the worker to do work
	spi.WorkerMetricsProvider
}

// workerRunner is needed to run the worker
type workerRunner struct {
	spi.Worker
	metricDescList []prometheus.Desc
	*runnerMetrics
	prevWorkFailed bool
}

var _ WorkerRunner = workerRunner{}

// runnerMetrics holds the metrics produced by the workerRunner. Metrics must be thread safe.
type runnerMetrics struct {
	loopCount                  metrics.MetricItem
	workerThreadCount          metrics.MetricItem
	workerLoopNanoSeconds      metrics.MetricItem
	workerDurationTotalSeconds metrics.MetricItem
}

// NewRunner creates a new workerRunner
func NewRunner(worker spi.Worker, conf config.CommonConfig, log vzlog.VerrazzanoLogger) (WorkerRunner, error) {
	r := workerRunner{Worker: worker, runnerMetrics: &runnerMetrics{
		loopCount: metrics.MetricItem{
			Name: "loop_count_total",
			Help: "The total number of loops executed",
			Type: prometheus.CounterValue,
		},
		workerThreadCount: metrics.MetricItem{
			Name: "worker_thread_count_total",
			Help: "The total number of worker threads (goroutines) running",
			Type: prometheus.CounterValue,
		},
		workerLoopNanoSeconds: metrics.MetricItem{
			Name: "worker_last_loop_nanoseconds",
			Help: "The number of nanoseconds that the worker took to run the last loop of doing work",
			Type: prometheus.GaugeValue,
		},
		workerDurationTotalSeconds: metrics.MetricItem{
			Name: "worker_running_seconds_total",
			Help: "The total number of seconds that the worker has been running",
			Type: prometheus.CounterValue,
		},
	}}

	r.metricDescList = []prometheus.Desc{
		*r.loopCount.BuildMetricDesc(r.GetWorkerDesc().MetricsPrefix),
		*r.workerThreadCount.BuildMetricDesc(r.GetWorkerDesc().MetricsPrefix),
		*r.workerLoopNanoSeconds.BuildMetricDesc(r.GetWorkerDesc().MetricsPrefix),
		*r.workerDurationTotalSeconds.BuildMetricDesc(r.GetWorkerDesc().MetricsPrefix),
	}

	return r, nil
}

// GetMetricDescList returns the prometheus metrics descriptors for the worker metrics.  Must be thread safe
func (r workerRunner) GetMetricDescList() []prometheus.Desc {
	return r.metricDescList
}

// GetMetricList returns the realtime metrics for the worker.  Must be thread safe
func (r workerRunner) GetMetricList() []prometheus.Metric {
	return []prometheus.Metric{
		r.loopCount.BuildMetric(),
		r.workerThreadCount.BuildMetric(),
		r.workerLoopNanoSeconds.BuildMetric(),
		r.workerDurationTotalSeconds.BuildMetric(),
	}
}

// StartWorkerRunners runs the worker in a loop
func (r workerRunner) RunWorker(conf config.CommonConfig, log vzlog.VerrazzanoLogger) error {
	if conf.NumLoops == 0 {
		return nil
	}

	r.incThreadCount()
	startTimeSecs := time.Now().Unix()
	for {
		loopCount := atomic.AddInt64(&r.runnerMetrics.loopCount.Val, 1)

		// call the wrapped worker.  Log any error but keep working
		startLoop := time.Now().UnixNano()
		err := r.Worker.DoWork(conf, log)
		if err != nil {
			r.prevWorkFailed = true
			log.ErrorfThrottled("Failed calling %s to do work: %v", r.Worker.GetWorkerDesc().WorkerType, err)
		} else {
			if r.prevWorkFailed {
				// If we had a failure on the prev call then log success so you can tell
				// get is working just be looking at the pod log.
				log.Info("Next call to DoWork from workerRunner successful after previous DoWork failed")
			}
			if loopCount == 1 {
				log.Info("First call to DoWork succeeded")
			}
			r.prevWorkFailed = false
		}
		log.GetZapLogger().Sync()
		durationSecondsTotal := time.Now().Unix() - startTimeSecs
		atomic.StoreInt64(&r.runnerMetrics.workerLoopNanoSeconds.Val, time.Now().UnixNano()-startLoop)
		atomic.StoreInt64(&r.runnerMetrics.workerDurationTotalSeconds.Val, durationSecondsTotal)
		if r.Worker.WantLoopInfoLogged() {
			log.Infof("Loop Count: %v, Total seconds from start of the first worker loop until now: %v", loopCount, durationSecondsTotal)
		}
		if loopCount == conf.NumLoops && conf.NumLoops != config.UnlimitedWorkerLoops {
			return nil
		}
		time.Sleep(conf.LoopSleepNanos)
	}
}

func (r workerRunner) incThreadCount() {
	atomic.AddInt64(&r.runnerMetrics.workerThreadCount.Val, 1)
}
