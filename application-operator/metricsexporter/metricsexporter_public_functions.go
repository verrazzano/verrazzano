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

// var MetricsExp MetricsExporter
type metricsOperation string
type metricName string

const (
	//millisPerSecond           float64    = 1000.0
	AppconfigReconcileCounter  metricName = "reconcile counter"
	AppconfigReconcileError    metricName = "reconcile error"
	AppconfigReconcileDuration metricName = "reconcile duration"
	appconfigMetricName        metricName = "appconfig"
	cohworkloadMetricName      metricName = "cohworkload"
	helidonworkloadMetricName  metricName = "helidonworkload"
	ingresstraitMetricName     metricName = "ingresstrait"
)

func InitRegisterStart(log *zap.SugaredLogger) {
	RequiredInitialization()
	RegisterMetrics(log)
	StartMetricsServer(log)
}

//This is intialized because adding the statement in the var block would create a cycle
func RequiredInitialization() {
	MetricsExp = metricsExporter{
		internalConfig: initConfiguration(),
		internalData: data{
			simpleCounterMetricMap: initCounterMetricMap(),
			durationMetricMap:      initDurationMetricMap(),
		},
	}
}

func RegisterMetrics(log *zap.SugaredLogger) {
	InitializeAllMetricsArray()
	go registerMetricsHandlers(log)
}

func InitializeAllMetricsArray() {
	//loop through all metrics declarations in metric maps
	for _, value := range MetricsExp.internalData.simpleCounterMetricMap {
		MetricsExp.internalConfig.allMetrics = append(MetricsExp.internalConfig.allMetrics, value.metric)
	}
	for _, value := range MetricsExp.internalData.durationMetricMap {
		MetricsExp.internalConfig.allMetrics = append(MetricsExp.internalConfig.allMetrics, value.metric)
	}

}

func initCounterMetricMap() map[metricName]*SimpleCounterMetric {
	return map[metricName]*SimpleCounterMetric{
		AppconfigReconcileCounter: {
			metric: prometheus.NewCounter(prometheus.CounterOpts{
				Name: "vao_appconfig_successful_reconcile_total",
				Help: "Tracks how many times a the reconcile process is successful"}),
		},
		AppconfigReconcileError: {
			metric: prometheus.NewCounter(prometheus.CounterOpts{
				Name: "vao_appconfig_error_reconcile_total",
				Help: "Tracks how many times a the reconcile process has failed"}),
		},
	}
}
func initDurationMetricMap() map[metricName]*DurationMetrics {
	return map[metricName]*DurationMetrics{
		AppconfigReconcileDuration: {
			metric: prometheus.NewSummary(prometheus.SummaryOpts{
				Name: "vao_appconfig_reconcile_duration",
				Help: "The duration in seconds of vao reconcile process",
			}),
		},
	}
}
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
			//if a metric is registered, delete it from the failed metrics map so that it is not retried
			delete(MetricsExp.internalConfig.failedMetrics, metric)
		}
	}
	return errorObserved
}

func registerMetricsHandlers(log *zap.SugaredLogger) {
	initializeFailedMetricsArray() //Get list of metrics to register initially
	//loop until there is no error in registering
	for err := registerMetricsHandlersHelper(); err != nil; err = registerMetricsHandlersHelper() {
		zap.S().Errorf("Failed to register metrics for VMI %v \n", err)
		time.Sleep(time.Second)
	}
}
func initializeFailedMetricsArray() {
	for i, metric := range MetricsExp.internalConfig.allMetrics {
		MetricsExp.internalConfig.failedMetrics[metric] = i
	}
}
func StartMetricsServer(log *zap.SugaredLogger) {
	go wait.Until(func() {
		http.Handle("/metrics", promhttp.Handler())
		err := http.ListenAndServe(":9100", nil)
		if err != nil {
			zap.S().Errorf("Failed to start metrics server for VMI: %v", err)
		}
	}, time.Second*3, wait.NeverStop)
}
func initConfiguration() configuration {
	return configuration{
		allMetrics:    []prometheus.Collector{},
		failedMetrics: map[prometheus.Collector]int{},
		registry:      prometheus.DefaultRegisterer,
	}
}

func GetSimpleCounterMetric(name metricName) (*SimpleCounterMetric, error) {
	counterMetric, ok := MetricsExp.internalData.simpleCounterMetricMap[name]
	if !ok {
		return nil, fmt.Errorf("%v not found in SimpleCounterMetricMap", name)
	}
	return counterMetric, nil
}
func GetDurationMetric(name metricName) (*DurationMetrics, error) {
	durationMetric, ok := MetricsExp.internalData.durationMetricMap[name]
	if !ok {
		return nil, fmt.Errorf("%v not found in durationMetricMap", name)
	}
	return durationMetric, nil
}
