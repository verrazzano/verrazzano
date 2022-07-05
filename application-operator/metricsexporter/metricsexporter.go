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

var (
	appconfigReconcileProcessed = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "appconfig_reconcile_puller_events_total",
		Help: "The total number of processed Reconcile events for appconfig",
	})

	cohworkloadReconcileProcessed = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "cohworkload_reconcile_puller_events_total",
		Help: "The total number of processed Reconcile events for cohworkload",
	})

	helidonworkloadReconcileProcessed = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "helidonworkload_reconcile_puller_events_total",
		Help: "The total number of processed Reconcile events for helidonworkload",
	})

	ingresstraitloadReconcileProcessed = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "ingresstrait_reconcile_puller_events_total",
		Help: "The total number of processed Reconcile events for ingresstrait",
	})

	allMetrics    = []prometheus.Collector{appconfigReconcileProcessed, cohworkloadReconcileProcessed, helidonworkloadReconcileProcessed}
	failedMetrics = map[prometheus.Collector]int{}
	registry      = prometheus.DefaultRegisterer
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

func AppconfigIncrementEventsProcessed() {

	appconfigReconcileProcessed.Inc()

}

func CohworkloadIncrementEventsProcessed() {

	cohworkloadReconcileProcessed.Inc()

}

func HelidonworkloadIncrementEventsProcessed() {

	helidonworkloadReconcileProcessed.Inc()

}
func IngresstraitloadIncrementEventsProcessed() {

	ingresstraitloadReconcileProcessed.Inc()

}
