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

// Metric Functions --------------------------------

func GetReconcileMetricsObject(s string) ReconcileMetrics {
	return reconcileMap[s]
}

// Successfull/Failed reconcile process incrementation

func (r ReconcileMetrics) VerifyReconcileResult(err error, log *zap.SugaredLogger) {
	if err == nil {
		r.reconcileSuccessful.Inc()
		log.Debug("The Reconcile Process has been Successful")
	}
	if err != nil {
		r.reconcileFailed.Inc()
		log.Debug("The Reconcile Process has failed")
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

func (r *ReconcileMetrics) GetDurationMetrics() *DurationMetrics {
	return &r.reconcileDuration
}

func (d *DurationMetrics) DurationTimerStart(log *zap.SugaredLogger) {
	d.durationStartTime = prometheus.NewTimer(d.processDuration)
	log.Debug("Duration timer started")
}
func (d *DurationMetrics) DurationTimerStop(log *zap.SugaredLogger) {
	d.durationStartTime.ObserveDuration()
	log.Debug("Duration timer stopped")
}
