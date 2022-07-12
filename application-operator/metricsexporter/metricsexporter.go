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
	//reconcileRequeue    prometheus.Counter
	reconcileDuration DurationMetrics
}

type DurationMetrics struct {
	durationStartTime *prometheus.Timer
	//	processDuration   prometheus.Summary
}

// type WebhookMetrics struct {
// 	webhookSuccessful prometheus.Counter
// 	webhookFailed     prometheus.Counter
// 	webhookDuration   DurationMetrics
// }

var (
	reconcileMap = map[string]ReconcileMetrics{
		"appconfig": {
			reconcileSuccessful: prometheus.NewCounter(prometheus.CounterOpts{
				Name: "appconfig_reconcile_successful_events_total",
				Help: "The total number of processed Reconcile events for appconfig",
			}),
			reconcileFailed: prometheus.NewCounter(prometheus.CounterOpts{
				Name: "appconfig_reconcile_failed_events_total",
				Help: "The total number of failed Reconcile events for appconfig",
			}),
		},
		"coherenceworkload": {
			reconcileSuccessful: prometheus.NewCounter(prometheus.CounterOpts{
				Name: "coherenceworkload_reconcile_puller_events_total",
				Help: "The total number of processed Reconcile events for coherenceworkload",
			}),
		},
	}

	//Successfull reconcile process
	// appconfigControllerMetrics = ReconcileMetrics{
	// 	reconcileSuccessful: prometheus.NewCounter(prometheus.CounterOpts{
	// 		Name: "appconfig_reconcile_puller_events_total",
	// 		Help: "The total number of processed Reconcile events for appconfig",
	// 	}),
	// }
	// appconfigReconcileProcessed = prometheus.NewCounter(prometheus.CounterOpts{
	// 	Name: "appconfig_reconcile_puller_events_total",
	// 	Help: "The total number of processed Reconcile events for appconfig",
	// })

	// cohworkloadReconcileProcessed = prometheus.NewCounter(prometheus.CounterOpts{
	// 	Name: "cohworkload_reconcile_puller_events_total",
	// 	Help: "The total number of processed Reconcile events for cohworkload",
	// })

	// helidonworkloadReconcileProcessed = prometheus.NewCounter(prometheus.CounterOpts{
	// 	Name: "helidonworkload_reconcile_puller_events_total",
	// 	Help: "The total number of processed Reconcile events for helidonworkload",
	// })

	// ingresstraitloadReconcileProcessed = prometheus.NewCounter(prometheus.CounterOpts{
	// 	Name: "ingresstrait_reconcile_puller_events_total",
	// 	Help: "The total number of processed Reconcile events for ingresstrait",
	// })

	// // Successfull Webhook process

	// appconfigWebhookProcessed = prometheus.NewCounter(prometheus.CounterOpts{
	// 	Name: "appconfig_webhook_puller_events_total",
	// 	Help: "The total number of processed webhook events for appconfig",
	// })

	// istioWebhookProcessed = prometheus.NewCounter(prometheus.CounterOpts{
	// 	Name: "istio_webhook_puller_events_total",
	// 	Help: "The total number of processed webhook events for cohworkload",
	// })
	// //multiclustercomponent
	// //multiclusterconfigmap
	// //multiclustersecret
	// //verrazzanoproject
	// helidonworkloadWebhookProcessed = prometheus.NewCounter(prometheus.CounterOpts{
	// 	Name: "helidonworkload_webhook_puller_events_total",
	// 	Help: "The total number of processed webhook events for helidonworkload",
	// })

	// ingresstraitloadWebhookProcessed = prometheus.NewCounter(prometheus.CounterOpts{
	// 	Name: "ingresstrait_webhook_puller_events_total",
	// 	Help: "The total number of processed webhook events for ingresstrait",
	// })

	// // Failed reconcile process
	// cohworkloadReconcileFailed = prometheus.NewCounter(prometheus.CounterOpts{
	// 	Name: "cohworkload_reconcile_failed_events_total",
	// 	Help: "The total number of failed Reconcile events for appconfig",
	// })
	// helidonworkloadReconcileFailed = prometheus.NewCounter(prometheus.CounterOpts{
	// 	Name: "helidonworkload_reconcile_failed_events_total",
	// 	Help: "The total number of failed Reconcile events for helidonworload",
	// })
	// ingresstraitReconcileFailed = prometheus.NewCounter(prometheus.CounterOpts{
	// 	Name: "ingresstrait_reconcile_failed_events_total",
	// 	Help: "The total number of failed Reconcile events for ingresstrait",
	// })
	// appconfigReconcileFailed = prometheus.NewCounter(prometheus.CounterOpts{
	// 	Name: "appconfig_reconcile_failed_events_total",
	// 	Help: "The total number of failed Reconcile events for appconfig",
	// })

	// // Duration Metrics
	// reconcileTimer *prometheus.Timer

	// appconfigReconcileDuration = prometheus.NewSummary(prometheus.SummaryOpts{
	// 	Name: "vao_appconfig_reconcile_duration",
	// 	Help: "Duration of Reconcile process for appconfig",
	// })

	// // Reque process
	// cohworkloadRequeProcessed = prometheus.NewCounter(prometheus.CounterOpts{
	// 	Name: "cohworkload_reconcile_reque_events_total",
	// 	Help: "The total number of failed Reconcile events for appconfig",
	// })
	// helidonworkloadRequeProcessed = prometheus.NewCounter(prometheus.CounterOpts{
	// 	Name: "helidonworkload_reconcile_reque_events_total",
	// 	Help: "The total number of failed Reconcile events for helidonworload",
	// })
	// ingresstraitRequeProcessed = prometheus.NewCounter(prometheus.CounterOpts{
	// 	Name: "ingresstrait_reconcile_reque_events_total",
	// 	Help: "The total number of failed Reconcile events for ingresstrait",
	// })
	// appconfigRequeProcessed = prometheus.NewCounter(prometheus.CounterOpts{
	// 	Name: "appconfig_reconcile_reque_events_total",
	// 	Help: "The total number of failed Reconcile events for appconfig",
	// })

	allMetrics    = []prometheus.Collector{reconcileMap["appconfig"].reconcileSuccessful, reconcileMap["coherenceworkload"].reconcileSuccessful}
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

// Duration metrics function

func (r ReconcileMetrics) GetDurationMetrics() DurationMetrics {
	return r.reconcileDuration
}

// func (d DurationMetrics) DurationTimerStart() {
// 	d.durationStartTime = prometheus.NewTimer(d.processDuration)
// }
func (d DurationMetrics) DurationTimerStop() {
	d.durationStartTime.ObserveDuration()
}

// Old code for metric functions --------------------------------

// func AppconfigIncrementEventsProcessed() {

// 	appconfigReconcileProcessed.Inc()
// }

// func CohworkloadIncrementEventsProcessed() {

// 	cohworkloadReconcileProcessed.Inc()
// }

// func HelidonworkloadIncrementEventsProcessed() {

// 	helidonworkloadReconcileProcessed.Inc()
// }
// func IngresstraitloadIncrementEventsProcessed() {

// 	ingresstraitloadReconcileProcessed.Inc()
// }

// //Successfull Webhook process incrementation

// func AppconfigIncrementWebhookProcessed() {

// 	appconfigWebhookProcessed.Inc()
// }

// func IstioIncrementWebhookProcessed() {

// 	istioWebhookProcessed.Inc()
// }

// func HelidonworkloadIncrementWebhookProcessed() {

// 	helidonworkloadWebhookProcessed.Inc()
// }
// func IngresstraitloadIncrementWebhookProcessed() {

// 	ingresstraitloadWebhookProcessed.Inc()
// }

// // Failed processing incrementation

// func AppconfigIncrementFailedProcess() {
// 	appconfigReconcileFailed.Inc()
// }
// func HelidonworkloadIncrementFailedProcess() {
// 	helidonworkloadReconcileFailed.Inc()
// }
// func CohworkloadIncrementFailedProcess() {
// 	cohworkloadReconcileFailed.Inc()
// }
// func IngresstraitIncrementFailedProcess() {
// 	ingresstraitReconcileFailed.Inc()
// }

// // Reque process incrementation

// func AppconfigIncrementRequeProcess() {
// 	appconfigRequeProcessed.Inc()
// }
// func HelidonworkloadIncrementRequeProcess() {
// 	helidonworkloadRequeProcessed.Inc()
// }
// func CohworkloadIncrementRequeProcess() {
// 	cohworkloadRequeProcessed.Inc()
// }
// func IngresstraitIncrementRequeProcess() {
// 	ingresstraitRequeProcessed.Inc()
// }

// // Reconcile Duration

// func AppconfigReconcileTimerStart() {
// 	reconcileTimer = prometheus.NewTimer(appconfigReconcileDuration)

// }
// func AppconfigReconcileTimerEnd() {
// 	reconcileTimer.ObserveDuration()
// }
