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
	vzlogInit "github.com/verrazzano/verrazzano/pkg/log"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/util/wait"
)

type metricName string

const (
	AppconfigReconcileCounter              metricName = "appconfig reconcile counter"
	AppconfigReconcileError                metricName = "appconfig reconcile error"
	AppconfigReconcileDuration             metricName = "appconfig reconcile duration"
	CohworkloadReconcileCounter            metricName = "coh reconcile counter"
	CohworkloadReconcileError              metricName = "coh reconcile error"
	CohworkloadReconcileDuration           metricName = "coh reconcile duration"
	HelidonReconcileCounter                metricName = "helidon reconcile counter"
	HelidonReconcileError                  metricName = "helidon reconcile error"
	HelidonReconcileDuration               metricName = "helidon reconcile duration"
	IngresstraitReconcileCounter           metricName = "ingress reconcile counter"
	IngresstraitReconcileError             metricName = "ingress reconcile error"
	IngresstraitReconcileDuration          metricName = "ingress reconcile duration"
	AppconfigHandleCounter                 metricName = "appconfig handle counter"
	AppconfigHandleError                   metricName = "appconfig handle error"
	AppconfigHandleDuration                metricName = "appconfig hanlde duration"
	IstioHandleCounter                     metricName = "istio handle counter"
	IstioHandleError                       metricName = "istio handle error"
	IstioHandleDuration                    metricName = "istio hanlde duration"
	LabelerPodHandleCounter                metricName = "LabelerPod handle counter"
	LabelerPodHandleError                  metricName = "LabelerPod handle error"
	LabelerPodHandleDuration               metricName = "LabelerPod hanlde duration"
	BindingUpdaterHandleCounter            metricName = "BindingUpdater handle counter"
	BindingUpdaterHandleError              metricName = "BindingUpdater handle error"
	BindingUpdaterHandleDuration           metricName = "BindingUpdater handle duration"
	MultiClusterAppconfigPodHandleCounter  metricName = "MultiClusterAppconfig handle counter"
	MultiClusterAppconfigPodHandleError    metricName = "MultiClusterAppconfig handle error"
	MultiClusterAppconfigPodHandleDuration metricName = "MultiClusterAppconfig hanlde duration"
	MultiClusterCompHandleCounter          metricName = "MultiClusterComp handle counter"
	MultiClusterCompHandleError            metricName = "MultiClusterComp handle error"
	MultiClusterCompHandleDuration         metricName = "MultiClusterComp hanlde duration"
	MultiClusterConfigmapHandleCounter     metricName = "MultiClusterConfigmap  handle counter"
	MultiClusterConfigmapHandleError       metricName = "MultiClusterConfigmap  handle error"
	MultiClusterConfigmapHandleDuration    metricName = "MultiClusterConfigmap  handle duration"
	MultiClusterSecretHandleCounter        metricName = "MultiClusterSecret handle counter"
	MultiClusterSecretHandleError          metricName = "MultiClusterSecret handle error"
	MultiClusterSecretHandleDuration       metricName = "MultiClusterSecret hanlde duration"
	VzProjHandleCounter                    metricName = "VzProj handle counter"
	VzProjHandleError                      metricName = "VzProj handle error"
	VzProjHandleDuration                   metricName = "VzProj hanlde duration"
)

func init() {
	RequiredInitialization()
	RegisterMetrics()
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
func RegisterMetrics() {
	InitializeAllMetricsArray()
	go registerMetricsHandlers(zap.S())
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
				Name: "vao_helidonworkload_success_reconcile_total",
				Help: "Tracks how many times a the helidonworkload reconcile process has failed"}),
		},
		HelidonReconcileError: {
			metric: prometheus.NewCounter(prometheus.CounterOpts{
				Name: "vao_helidonworkload_error_reconcile_total",
				Help: "Tracks how many times a the helidonworkload reconcile process has failed"}),
		},
		IngresstraitReconcileCounter: {
			metric: prometheus.NewCounter(prometheus.CounterOpts{
				Name: "vao_ingresstrait_success_reconcile_total",
				Help: "Tracks how many times a the ingresstrait reconcile process has failed"}),
		},
		IngresstraitReconcileError: {
			metric: prometheus.NewCounter(prometheus.CounterOpts{
				Name: "vao_ingresstrait_error_reconcile_total",
				Help: "Tracks how many times a the ingresstrait reconcile process has failed"}),
		},
		AppconfigHandleCounter: {
			metric: prometheus.NewCounter(prometheus.CounterOpts{
				Name: "vao_appconfig_handle_total",
				Help: "Tracks how many times a the helidonworkload reconcile process has failed"}),
		},
		AppconfigHandleError: {
			metric: prometheus.NewCounter(prometheus.CounterOpts{
				Name: "vao_appconfig_error_handle_total",
				Help: "Tracks how many times a the helidonworkload reconcile process has failed"}),
		},
		IstioHandleCounter: {
			metric: prometheus.NewCounter(prometheus.CounterOpts{
				Name: "vao_istio_handle_total",
				Help: "Tracks how many times a the ingresstrait reconcile process has been successful"}),
		},
		IstioHandleError: {
			metric: prometheus.NewCounter(prometheus.CounterOpts{
				Name: "vao_istio_error_handle_total",
				Help: "Tracks how many times a the ingresstrait reconcile process has failed"}),
		},
		LabelerPodHandleCounter: {
			metric: prometheus.NewCounter(prometheus.CounterOpts{
				Name: "vao_labelerPod_handle__total",
				Help: "Tracks how many times a the helidonworkload reconcile process has failed"}),
		},
		LabelerPodHandleError: {
			metric: prometheus.NewCounter(prometheus.CounterOpts{
				Name: "vao_labelerpod_error_handle_total",
				Help: "Tracks how many times a the helidonworkload reconcile process has failed"}),
		},
		BindingUpdaterHandleCounter: {

			metric: prometheus.NewCounter(prometheus.CounterOpts{
				Name: "vao_bindingupdater_handle_total",
				Help: "Tracks how many times a the ingresstrait reconcile process has failed"}),
		},
		BindingUpdaterHandleError: {
			metric: prometheus.NewCounter(prometheus.CounterOpts{
				Name: "vao_bindingupdater_error_handle_total",
				Help: "Tracks how many times a the ingresstrait reconcile process has failed"}),
		},
		MultiClusterAppconfigPodHandleCounter: {
			metric: prometheus.NewCounter(prometheus.CounterOpts{
				Name: "vao_multiclusterappconfig_handle_total",
				Help: "Tracks how many times a the helidonworkload reconcile process has failed"}),
		},
		MultiClusterAppconfigPodHandleError: {
			metric: prometheus.NewCounter(prometheus.CounterOpts{
				Name: "vao_multiclusterappconfig_error_handle_total",
				Help: "Tracks how many times a the helidonworkload reconcile process has failed"}),
		},
		MultiClusterCompHandleCounter: {

			metric: prometheus.NewCounter(prometheus.CounterOpts{
				Name: "vao_multiclustercomp_handle_total",
				Help: "Tracks how many times a the ingresstrait reconcile process has failed"}),
		},
		MultiClusterCompHandleError: {
			metric: prometheus.NewCounter(prometheus.CounterOpts{
				Name: "vao_multiclustercomp_error_handle_total",
				Help: "Tracks how many times a the ingresstrait reconcile process has failed"}),
		},
		MultiClusterConfigmapHandleCounter: {
			metric: prometheus.NewCounter(prometheus.CounterOpts{
				Name: "vao_multiclustercomp_handle_total",
				Help: "Tracks how many times a the ingresstrait reconcile process has failed"}),
		},
		MultiClusterConfigmapHandleError: {
			metric: prometheus.NewCounter(prometheus.CounterOpts{
				Name: "vao_multiclustercomp_error_handle_total",
				Help: "Tracks how many times a the ingresstrait reconcile process has failed"}),
		},
		MultiClusterSecretHandleCounter: {
			metric: prometheus.NewCounter(prometheus.CounterOpts{
				Name: "vao_multiclustersecret_handle_total",
				Help: "Tracks how many times a the ingresstrait reconcile process has failed"}),
		},
		MultiClusterSecretHandleError: {
			metric: prometheus.NewCounter(prometheus.CounterOpts{
				Name: "vao_multiclustersecret_error_handle_total",
				Help: "Tracks how many times a the ingresstrait reconcile process has failed"}),
		},
		VzProjHandleCounter: {
			metric: prometheus.NewCounter(prometheus.CounterOpts{
				Name: "vao_vzproj_handle__total",
				Help: "Tracks how many times a the helidonworkload reconcile process has failed"}),
		},
		VzProjHandleError: {
			metric: prometheus.NewCounter(prometheus.CounterOpts{
				Name: "vao_vzproj_error_handle_total",
				Help: "Tracks how many times a the helidonworkload reconcile process has failed"}),
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
		AppconfigHandleDuration: {
			metric: prometheus.NewSummary(prometheus.SummaryOpts{
				Name: "vao_appconfig_handle_duration",
				Help: "The duration in seconds of vao Ingresstrait reconcile process",
			}),
		},
		IstioHandleDuration: {
			metric: prometheus.NewSummary(prometheus.SummaryOpts{
				Name: "vao_istio_handle_duration",
				Help: "The duration in seconds of vao Ingresstrait handle process",
			}),
		},
		LabelerPodHandleDuration: {
			metric: prometheus.NewSummary(prometheus.SummaryOpts{
				Name: "vao_labelerpod_handle_duration",
				Help: "The duration in seconds of vao Ingresstrait hanlde process",
			}),
		},
		MultiClusterConfigmapHandleDuration: {
			metric: prometheus.NewSummary(prometheus.SummaryOpts{
				Name: "vao_multiclusterconfigmap_handle_duration",
				Help: "The duration in seconds of vao Ingresstrait handle process",
			}),
		},
		MultiClusterAppconfigPodHandleDuration: {
			metric: prometheus.NewSummary(prometheus.SummaryOpts{
				Name: "vao_multiclusterappconfig_handle_duration",
				Help: "The duration in seconds of vao Ingresstrait reconcile process",
			}),
		},
		MultiClusterCompHandleDuration: {
			metric: prometheus.NewSummary(prometheus.SummaryOpts{
				Name: "vao_multiclustercomp_handle_duration",
				Help: "The duration in seconds of vao Ingresstrait reconcile process",
			}),
		},
		MultiClusterSecretHandleDuration: {
			metric: prometheus.NewSummary(prometheus.SummaryOpts{
				Name: "vao_multiclustersecret_handle_duration",
				Help: "The duration in seconds of vao Ingresstrait reconcile process",
			}),
		},
		VzProjHandleDuration: {
			metric: prometheus.NewSummary(prometheus.SummaryOpts{
				Name: "vao_vzproj_handle_duration",
				Help: "The duration in seconds of vao Ingresstrait reconcile process",
			}),
		},
		BindingUpdaterHandleDuration: {
			metric: prometheus.NewSummary(prometheus.SummaryOpts{
				Name: "vao_bindingupdater_handle_duration",
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
		log.Infof("Failed to register metrics for VMI %v", err)
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
func StartMetricsServer() error {
	vlog, err := vzlog.EnsureResourceLogger(&vzlog.ResourceConfig{
		Name:           "",
		Namespace:      "",
		ID:             "",
		Generation:     0,
		ControllerName: "metricsexporter",
	})
	if err != nil {
		return err
	}
	go wait.Until(func() {
		http.Handle("/metrics", promhttp.Handler())
		err := http.ListenAndServe(":9100", nil)
		if err != nil {
			vlog.Oncef("Failed to start metrics server for VMI: %v", err)
		}
	}, time.Second*3, wait.NeverStop)
	return nil
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
func ExposeControllerMetrics(controllerName string, successname metricName, errorname metricName, durationname metricName) (*SimpleCounterMetric, *SimpleCounterMetric, *DurationMetrics, *zap.SugaredLogger, error) {
	zapLogForMetrics := zap.S().With(vzlogInit.FieldController, controllerName)
	counterMetricObject, err := GetSimpleCounterMetric(successname)
	if err != nil {
		zapLogForMetrics.Error(err)
		return nil, nil, nil, nil, err
	}
	errorCounterMetricObject, err := GetSimpleCounterMetric(errorname)
	if err != nil {
		zapLogForMetrics.Error(err)
		return nil, nil, nil, nil, err
	}

	durationMetricObject, err := GetDurationMetric(durationname)
	if err != nil {
		zapLogForMetrics.Error(err)
		return nil, nil, nil, nil, err
	}
	return counterMetricObject, errorCounterMetricObject, durationMetricObject, zapLogForMetrics, nil
}
