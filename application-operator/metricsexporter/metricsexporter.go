// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricsexporter

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/util/wait"
)

// InitalizeMetricsEndpoint creates and serves a /metrics endpoint at 9100 for Prometheus to scrape metrics from
func InitalizeMetricsEndpoint() {
	go registerMetricsHandlers()

	go wait.Until(func() {
		http.Handle("/metrics", promhttp.Handler())
		err := http.ListenAndServe(":9100", nil)
		if err != nil {
			zap.S().Errorf("Failed to start metrics server for VAO: %v", err)
		}
	}, time.Second*3, wait.NeverStop)

}
func initializeFailedMetricsArray() {
	for i, metric := range allMetrics {
		failedMetrics[metric] = i
	}
}

func registerMetricsHandlers() {
	initializeFailedMetricsArray()
	for err := registerMetricsHandlersHelper(); err != nil; err = registerMetricsHandlersHelper() {
		zap.S().Errorf("Failed to register some metrics for VMI: %v", err)
		time.Sleep(time.Second)
	}
}

func registerMetricsHandlersHelper() error {
	var errorObserved error = nil
	for metric, i := range failedMetrics {
		err := registry.Register(metric)
		if err != nil {
			zap.S().Errorf("Failed to register metric index %v for VAO", i)
			errorObserved = err
		} else {
			delete(failedMetrics, metric)
		}
	}
	return errorObserved
}

type ReconcileMetrics struct {
	reconcileSuccessful prometheus.Counter
	reconcileFailed     prometheus.Counter
	//reconcileRequeue  prometheus.Counter
	reconcileDuration DurationMetrics
}

type DurationMetrics struct {
	durationStartTime *prometheus.Timer
	processDuration   prometheus.Summary
}

// type WebhookMetrics struct {
// 	webhookSuccessful prometheus.Counter
// 	webhookFailed     prometheus.Counter
// 	webhookDuration   DurationMetrics
// }

var (
	// Reconcile Metrics
	reconcileMap = map[string]ReconcileMetrics{
		"appconfig": {
			reconcileSuccessful: prometheus.NewCounter(prometheus.CounterOpts{
				Name: "vao_appconfig_reconcile_successful_events_total",
				Help: "The total number of processed Reconcile events for appconfig",
			}),
			reconcileFailed: prometheus.NewCounter(prometheus.CounterOpts{
				Name: "vao_appconfig_reconcile_failed_events_total",
				Help: "The total number of failed Reconcile events for appconfig",
			}),
			reconcileDuration: DurationMetrics{
				processDuration: prometheus.NewSummary(prometheus.SummaryOpts{
					Name: "vao_appconfig_reconcile_duration",
					Help: "The duration in seconds of appconfig reconcile process",
				}),
			},
		},
		"coherenceworkload": {
			reconcileSuccessful: prometheus.NewCounter(prometheus.CounterOpts{
				Name: "coherenceworkload_reconcile_puller_events_total",
				Help: "The total number of processed Reconcile events for coherenceworkload",
			}),
		},
	}

	allMetrics    = []prometheus.Collector{reconcileMap["appconfig"].reconcileSuccessful, reconcileMap["appconfig"].reconcileFailed, reconcileMap["coherenceworkload"].reconcileSuccessful}
	failedMetrics = map[prometheus.Collector]int{}
	registry      = prometheus.DefaultRegisterer
)

// Metric Functions --------------------------------

func GetReconcileMetricsObject(s string) ReconcileMetrics {
	return reconcileMap[s]
}

// Successfull/Failed reconcile process incrementation

func (r ReconcileMetrics) VerifyReconcileResult(err error) {
	if err == nil {
		r.reconcileSuccessful.Inc()
	}
	if err != nil {
		r.reconcileFailed.Inc()
	}
}
func (r ReconcileMetrics) GetReconcileFailed() prometheus.Counter {
	return r.reconcileFailed
}
func (r ReconcileMetrics) GetReconcileSuccessful() prometheus.Counter {
	return r.reconcileSuccessful
}
func (r ReconcileMetrics) IncreaseFailedReconcileMetric() {
	r.reconcileFailed.Inc()
}

// Duration metrics function

func (r ReconcileMetrics) GetDurationMetrics() DurationMetrics {
	return r.reconcileDuration
}

func (d DurationMetrics) DurationTimerStart() {
	d.durationStartTime = prometheus.NewTimer(d.processDuration)
}
func (d DurationMetrics) DurationTimerStop() {
	d.durationStartTime.ObserveDuration()
}
