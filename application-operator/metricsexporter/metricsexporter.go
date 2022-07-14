// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricsexporter

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

func initializeFailedMetricsArray() {
	for i, metric := range allMetrics {
		failedMetrics[metric] = i
	}
}

func registerMetricsHandlers() {
	initializeFailedMetricsArray()
	for err := registerMetricsHandlersHelper(); err != nil; err = registerMetricsHandlersHelper() {
		zap.S().Errorf("Failed to register some metrics for VAO: %v", err)
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

	allMetrics    = []prometheus.Collector{reconcileMap["appconfig"].reconcileDuration.processDuration, reconcileMap["appconfig"].reconcileSuccessful, reconcileMap["appconfig"].reconcileFailed, reconcileMap["coherenceworkload"].reconcileSuccessful}
	failedMetrics = map[prometheus.Collector]int{}
	registry      = prometheus.DefaultRegisterer
)
