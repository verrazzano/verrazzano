// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricsexporter

import (
	"fmt"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/util/wait"
)

type metricName string

const (
	AppconfigReconcileCounter     metricName = "reconcile counter"
	AppconfigReconcileError       metricName = "reconcile error"
	AppconfigReconcileDuration    metricName = "reconcile duration"
	CohworkloadReconcileCounter   metricName = "coh reconcile counter"
	CohworkloadReconcileError     metricName = "coh reconcile error"
	CohworkloadReconcileDuration  metricName = "coh reconcile duration"
	HelidonReconcileCounter       metricName = "helidon reconcile counter"
	HelidonReconcileError         metricName = "helidon reconcile error"
	HelidonReconcileDuration      metricName = "helidon reconcile duration"
	IngresstraitReconcileCounter  metricName = "ingress reconcile counter"
	IngresstraitReconcileError    metricName = "ingress reconcile error"
	IngresstraitReconcileDuration metricName = "ingress reconcile duration"
)

// InitRegisterStart initalizes the metrics object, registers the metrics, and then starts the server
func InitRegisterStart(log *zap.SugaredLogger) {
	RequiredInitialization()
	RegisterMetrics(log)
	StartMetricsServer(log)
}

// RequiredInitialization initalizes the metrics object, but does not register the metrics
func RequiredInitialization() {
	MetricsExp = metricsExporter{
		internalConfig: initConfiguration(),
		internalData: data{
			simpleCounterMetricMap: initCounterMetricMap(),
			durationMetricMap:      initDurationMetricMap(),
		},
	}
}

// RegisterMetrics begins the process of registering metrics
func RegisterMetrics(log *zap.SugaredLogger) {
	InitializeAllMetricsArray()
	go registerMetricsHandlers(log)
}

// InitializeAllMetricsArray initalizes the allMetrics array
func InitializeAllMetricsArray() {
	// Loop through all metrics declarations in metric maps
	for _, value := range MetricsExp.internalData.simpleCounterMetricMap {
		MetricsExp.internalConfig.allMetrics = append(MetricsExp.internalConfig.allMetrics, value.metric)
	}
	for _, value := range MetricsExp.internalData.durationMetricMap {
		MetricsExp.internalConfig.allMetrics = append(MetricsExp.internalConfig.allMetrics, value.metric)
	}

}

// initCounterMetricMap initalizes the simpleCounterMetricMap for the metricsExporter object
func initCounterMetricMap() map[metricName]*SimpleCounterMetric {
	return map[metricName]*SimpleCounterMetric{
		AppconfigReconcileCounter: {
			metric: prometheus.NewCounter(prometheus.CounterOpts{
				Name: "vao_appconfig_successful_reconcile_total",
				Help: "Tracks how many times a the appconfig reconcile process is successful"}),
		},
		AppconfigReconcileError: {
			metric: prometheus.NewCounter(prometheus.CounterOpts{
				Name: "vao_appconfig_error_reconcile_total",
				Help: "Tracks how many times a the appconfig reconcile process has failed"}),
		},
		CohworkloadReconcileCounter: {
			metric: prometheus.NewCounter(prometheus.CounterOpts{
				Name: "vao_cohworkload_error_reconcile_total",
				Help: "Tracks how many times a the cohworkload reconcile process has failed"}),
		},
		CohworkloadReconcileError: {
			metric: prometheus.NewCounter(prometheus.CounterOpts{
				Name: "vao_cohworkload_error_reconcile_total",
				Help: "Tracks how many times a the cohworkload reconcile process has failed"}),
		},
		HelidonReconcileCounter: {
			metric: prometheus.NewCounter(prometheus.CounterOpts{
				Name: "vao_helidonworkload_error_reconcile_total",
				Help: "Tracks how many times a the helidonworkload reconcile process has failed"}),
		},
		HelidonReconcileError: {
			metric: prometheus.NewCounter(prometheus.CounterOpts{
				Name: "vao_helidonworkload_error_reconcile_total",
				Help: "Tracks how many times a the helidonworkload reconcile process has failed"}),
		},
		IngresstraitReconcileCounter: {
			metric: prometheus.NewCounter(prometheus.CounterOpts{
				Name: "vao_ingresstrait_error_reconcile_total",
				Help: "Tracks how many times a the ingresstrait reconcile process has failed"}),
		},
		IngresstraitReconcileError: {
			metric: prometheus.NewCounter(prometheus.CounterOpts{
				Name: "vao_ingresstrait_error_reconcile_total",
				Help: "Tracks how many times a the ingresstrait reconcile process has failed"}),

		},
	}
}

// initDurationMetricMap initalizes the DurationMetricMap for the metricsExporter object
func initDurationMetricMap() map[metricName]*DurationMetrics {
	return map[metricName]*DurationMetrics{
		AppconfigReconcileDuration: {
			metric: prometheus.NewSummary(prometheus.SummaryOpts{
				Name: "vao_appconfig_reconcile_duration",
				Help: "The duration in seconds of vao reconcile process",
			}),
		},
		CohworkloadReconcileDuration: {
			metric: prometheus.NewSummary(prometheus.SummaryOpts{
				Name: "vao_cohworkload_reconcile_duration",
				Help: "The duration in seconds of vao Cohworkload reconcile process",
			}),
		},
		HelidonReconcileDuration: {
			metric: prometheus.NewSummary(prometheus.SummaryOpts{
				Name: "vao_helidon_reconcile_duration",
				Help: "The duration in seconds of vao Helidon reconcile process",
			}),
		},
		IngresstraitReconcileDuration: {
			metric: prometheus.NewSummary(prometheus.SummaryOpts{
				Name: "vao_ingresstrait_reconcile_duration",
				Help: "The duration in seconds of vao Ingresstrait reconcile process",
			}),
		},
	}
}

// registerMetricsHandlersHelper is a helper function that assists in registering metrics
func registerMetricsHandlersHelper() error {
	var errorObserved error
	for metric := range MetricsExp.internalConfig.failedMetrics {
		err := MetricsExp.internalConfig.registry.Register(metric)
		if err != nil {
			if errorObserved != nil {
				errorObserved = errors.Wrap(errorObserved, err.Error())
			} else {
				errorObserved = err
			}
		} else {
			// If a metric is registered, delete it from the failed metrics map so that it is not retried
			delete(MetricsExp.internalConfig.failedMetrics, metric)
		}
	}
	return errorObserved
}

// registerMetricsHandlers registers the metrics and provides error handling
func registerMetricsHandlers(log *zap.SugaredLogger) {
	// Get list of metrics to register initially
	initializeFailedMetricsArray()
	// Loop until there is no error in registering
	for err := registerMetricsHandlersHelper(); err != nil; err = registerMetricsHandlersHelper() {
		zap.S().Errorf("Failed to register metrics for VMI %v \n", err)
		time.Sleep(time.Second)
	}
}

// initializeFailedMetricsArray initalizes the failedMetrics array
func initializeFailedMetricsArray() {
	for i, metric := range MetricsExp.internalConfig.allMetrics {
		MetricsExp.internalConfig.failedMetrics[metric] = i
	}
}

// StartMetricsServer starts the metric server to begin emitting metrics to Prometheus
func StartMetricsServer(log *zap.SugaredLogger) {
	go wait.Until(func() {
		http.Handle("/metrics", promhttp.Handler())
		err := http.ListenAndServe(":9100", nil)
		if err != nil {
			zap.S().Errorf("Failed to start metrics server for VMI: %v", err)
		}
	}, time.Second*3, wait.NeverStop)
}

// initConfiguration returns an empty struct of type configuration
func initConfiguration() configuration {
	return configuration{
		allMetrics:    []prometheus.Collector{},
		failedMetrics: map[prometheus.Collector]int{},
		registry:      prometheus.DefaultRegisterer,
	}
}

// GetSimpleCounterMetric returns a simpleCounterMetric from the simpleCounterMetricMap given a metricName
func GetSimpleCounterMetric(name metricName) (*SimpleCounterMetric, error) {
	counterMetric, ok := MetricsExp.internalData.simpleCounterMetricMap[name]
	if !ok {
		return nil, fmt.Errorf("%v not found in SimpleCounterMetricMap due to metricName being defined, but not being a key in the map", name)
	}
	return counterMetric, nil
}

// GetDurationMetric returns a durationMetric from the durationMetricMap given a metricName
func GetDurationMetric(name metricName) (*DurationMetrics, error) {
	durationMetric, ok := MetricsExp.internalData.durationMetricMap[name]
	if !ok {
		return nil, fmt.Errorf("%v not found in durationMetricMap due to metricName being defined, but not being a key in the map", name)
	}
	return durationMetric, nil
}
