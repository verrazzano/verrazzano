// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricsexporter

import (
	"fmt"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/grafanadashboards"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/networkpolicies"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/vmo"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/util/wait"
)

var MetricsExp MetricsExporter

type metricName string

const (
	component                      = "component"
	ReconcileCounter    metricName = "reconcile counter"
	ReconcileError      metricName = "reconcile error"
	ReconcileDuration   metricName = "reconcile duration"
	AvailableComponents metricName = "available components"
	EnabledComponents   metricName = "enabled components"
)

func init() {
	RequiredInitialization()
	RegisterMetrics(zap.S())
}

// This function initializes the metrics object, but does not register the metrics
func RequiredInitialization() {
	MetricsExp = MetricsExporter{
		internalConfig: initConfiguration(),
		internalData: data{
			simpleCounterMetricMap:   initSimpleCounterMetricMap(),
			simpleGaugeMetricMap:     initSimpleGaugeMetricMap(),
			durationMetricMap:        initDurationMetricMap(),
			componentHealth:          initComponentHealthMetrics(),
			componentInstallDuration: initComponentInstallDurationMetrics(),
			componentUpgradeDuration: initComponentUpgradeDurationMetrics(),
		},
	}
	// initialize component availability metric to false
	for _, component := range registry.GetComponents() {
		if IsNonMetricComponent(component.Name()) {
			continue
		}
		MetricsExp.internalData.componentHealth.SetComponentHealth(component.GetJSONName(), false, false)
		SetComponentInstallDurationMetric(component.GetJSONName(), 0)
		SetComponentUpgradeDurationMetric(component.GetJSONName(), 0)

	}

}

// This function begins the process of registering metrics
func RegisterMetrics(log *zap.SugaredLogger) {
	InitializeAllMetricsArray()
	go registerMetricsHandlers(log)
}

// This function initializes the simpleCounterMetricMap for the metricsExporter object
func initSimpleCounterMetricMap() map[metricName]*SimpleCounterMetric {
	return map[metricName]*SimpleCounterMetric{
		ReconcileCounter: {
			prometheus.NewCounter(prometheus.CounterOpts{
				Name: "vz_platform_operator_reconcile_total",
				Help: "The number of times the reconcile function has been called in the verrazzano-platform-operator",
			}),
		},
		ReconcileError: {
			prometheus.NewCounter(prometheus.CounterOpts{
				Name: "vz_platform_operator_error_reconcile_total",
				Help: "The number of times the reconcile function has returned an error in the verrazzano-platform-operator",
			}),
		},
	}
}

func initComponentHealthMetrics() *ComponentHealth {
	return &ComponentHealth{
		available: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vz_platform_operator_component_health",
			Help: "Is component enabled and available",
		}, []string{component}),
	}
}

func initComponentInstallDurationMetrics() *ComponentInstallDuration {
	return &ComponentInstallDuration{
		installDuration: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vz_platform_operator_component_install_duration_seconds",
			Help: "The duration of the latest installation of each component in seconds",
		}, []string{component}),
	}
}

func initComponentUpgradeDurationMetrics() *ComponentUpgradeDuration {
	return &ComponentUpgradeDuration{
		upgradeDuration: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vz_platform_operator_component_upgrade_duration_seconds",
			Help: "The duration of the latest upgrade of each component in seconds",
		}, []string{component}),
	}
}

// This function initializes the simpleGaugeMetricMap for the metricsExporter object
func initSimpleGaugeMetricMap() map[metricName]*SimpleGaugeMetric {
	return map[metricName]*SimpleGaugeMetric{
		AvailableComponents: {
			metric: prometheus.NewGauge(prometheus.GaugeOpts{
				Name: "vz_platform_operator_component_health_total",
				Help: "The number of currently available Verrazzano components",
			}),
		},
		EnabledComponents: {
			metric: prometheus.NewGauge(prometheus.GaugeOpts{
				Name: "vz_platform_operator_component_enabled_total",
				Help: "The number of currently enabled Verrazzano components",
			}),
		},
	}
}

// This function initializes the durationMetricMap for the metricsExporter object
func initDurationMetricMap() map[metricName]*DurationMetric {
	return map[metricName]*DurationMetric{
		ReconcileDuration: {
			metric: prometheus.NewSummary(prometheus.SummaryOpts{
				Name: "vz_platform_operator_reconcile_duration",
				Help: "The duration in seconds of vpo reconcile process",
			}),
		},
	}
}

// This function is used to determine whether a durationTime for a component metric should be set and what the duration time is
// If the start time is greater than the completion time, the metric will not be set
// After this check, the function calculates the duration time and tries to set the metric of the component
// If the component's name is not in the metric map, an error will be raised to prevent a seg fault
func metricParserHelperFunction(log vzlog.VerrazzanoLogger, componentName string, startTime string, completionTime string, typeofOperation string) {
	startInSeconds, err := time.Parse(time.RFC3339, startTime)
	if err != nil {
		log.Errorf("Error in parsing start time %s for operation %s for component %s", startTime, typeofOperation, componentName)
		return
	}
	startInSecondsUnix := startInSeconds.Unix()
	completionInSeconds, err := time.Parse(time.RFC3339, completionTime)
	if err != nil {
		log.Errorf("Error in parsing completion time %s for operation %s for component %s", completionTime, typeofOperation, componentName)
		return
	}
	completionInSecondsUnix := completionInSeconds.Unix()
	if startInSecondsUnix >= completionInSecondsUnix {
		log.Debug("Component %s is not updated, as there is an ongoing operation in progress")
		return
	}
	totalDuration := (completionInSecondsUnix - startInSecondsUnix)
	if typeofOperation == constants.InstallOperation {
		err := SetComponentInstallDurationMetric(componentName, totalDuration)
		if err != nil {
			log.Errorf(err.Error())
			return
		}
	}
	if typeofOperation == constants.UpgradeOperation {
		err := SetComponentUpgradeDurationMetric(componentName, totalDuration)
		if err != nil {
			log.Errorf(err.Error())
			return
		}
	}
}

func SetComponentInstallDurationMetric(JSONName string, totalDuration int64) error {
	metric, err := MetricsExp.internalData.componentInstallDuration.installDuration.GetMetricWithLabelValues(JSONName)
	if err != nil {
		return err
	}
	metric.Set(float64(totalDuration))
	return nil
}

func SetComponentUpgradeDurationMetric(JSONName string, totalDuration int64) error {
	metric, err := MetricsExp.internalData.componentUpgradeDuration.upgradeDuration.GetMetricWithLabelValues(JSONName)
	if err != nil {
		return err
	}
	metric.Set(float64(totalDuration))
	return nil
}

// This function is a helper function that assists in registering metrics
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
			// if a metric is registered, delete it from the failed metrics map so that it is not retried
			delete(MetricsExp.internalConfig.failedMetrics, metric)
		}
	}
	return errorObserved
}

// This function registers the metrics and provides error handling
func registerMetricsHandlers(log *zap.SugaredLogger) {
	initializeFailedMetricsArray() // Get list of metrics to register initially
	// loop until there is no error in registering
	for err := registerMetricsHandlersHelper(); err != nil; err = registerMetricsHandlersHelper() {
		log.Errorf("Failed to register metrics for VPO %v \n", err)
		time.Sleep(time.Second)
	}
	// register component health metrics vector
	MetricsExp.internalConfig.registry.MustRegister(MetricsExp.internalData.componentHealth.available)
	MetricsExp.internalConfig.registry.MustRegister(MetricsExp.internalData.componentInstallDuration.installDuration)
	MetricsExp.internalConfig.registry.MustRegister(MetricsExp.internalData.componentUpgradeDuration.upgradeDuration)
}

// This function initializes the failedMetrics array
func initializeFailedMetricsArray() {
	for i, metric := range MetricsExp.internalConfig.allMetrics {
		MetricsExp.internalConfig.failedMetrics[metric] = i
	}
}

// This function starts the metric server to begin emitting metrics to Prometheus
func StartMetricsServer(log *zap.SugaredLogger) {
	go wait.Until(func() {
		http.Handle("/metrics", promhttp.Handler())
		server := &http.Server{
			Addr:              ":9100",
			ReadHeaderTimeout: 3 * time.Second,
		}
		if err := server.ListenAndServe(); err != nil {
			log.Errorf("Failed to start metrics server for verrazzano-platform-operator: %v", err)
		}
	}, time.Second*3, wait.NeverStop)
}

// This functionn parses the VZ CR and extracts the install and update data for each component
func AnalyzeVerrazzanoResourceMetrics(log vzlog.VerrazzanoLogger, cr vzapi.Verrazzano) {
	mapOfComponents := cr.Status.Components
	for componentName, componentStatusDetails := range mapOfComponents {
		// If component is not in the metricsMap, move on to the next component
		if IsNonMetricComponent(componentName) {
			continue
		}
		var installCompletionTime string
		var upgradeCompletionTime string
		var upgradeStartTime string
		var installStartTime string
		for _, status := range componentStatusDetails.Conditions {
			if status.Type == vzapi.CondInstallStarted {
				installStartTime = status.LastTransitionTime
			}
			if status.Type == vzapi.CondInstallComplete {
				installCompletionTime = status.LastTransitionTime
			}
			if status.Type == vzapi.CondUpgradeStarted {
				upgradeStartTime = status.LastTransitionTime
			}
			if status.Type == vzapi.CondUpgradeComplete {
				upgradeCompletionTime = status.LastTransitionTime
			}
		}
		found, component := registry.FindComponent(componentName)
		if !found {
			log.Errorf("No component %s found", componentName)
			return
		}
		componentJSONName := component.GetJSONName()
		if installStartTime != "" && installCompletionTime != "" {
			metricParserHelperFunction(log, componentJSONName, installStartTime, installCompletionTime, constants.InstallOperation)
		}
		if upgradeStartTime != "" && upgradeCompletionTime != "" {
			metricParserHelperFunction(log, componentJSONName, upgradeStartTime, upgradeCompletionTime, constants.UpgradeOperation)
		}
	}
}

// This function initializes the allMetrics array
func InitializeAllMetricsArray() {
	// loop through all metrics declarations in metric maps
	for _, value := range MetricsExp.internalData.simpleCounterMetricMap {
		MetricsExp.internalConfig.allMetrics = append(MetricsExp.internalConfig.allMetrics, value.metric)
	}
	for _, value := range MetricsExp.internalData.durationMetricMap {
		MetricsExp.internalConfig.allMetrics = append(MetricsExp.internalConfig.allMetrics, value.metric)
	}
	for _, value := range MetricsExp.internalData.simpleGaugeMetricMap {
		MetricsExp.internalConfig.allMetrics = append(MetricsExp.internalConfig.allMetrics, value.metric)
	}
}

// This function returns an empty struct of type configuration
func initConfiguration() configuration {
	return configuration{
		allMetrics:    []prometheus.Collector{},
		failedMetrics: map[prometheus.Collector]int{},
		registry:      prometheus.DefaultRegisterer,
	}
}

// This function returns a simpleCounterMetric from the simpleCounterMetricMap given a metricName
func GetSimpleCounterMetric(name metricName) (*SimpleCounterMetric, error) {
	counterMetric, ok := MetricsExp.internalData.simpleCounterMetricMap[name]
	if !ok {
		return nil, fmt.Errorf("%v not found in SimpleCounterMetricMap due to metricName being defined, but not being a key in the map", name)
	}
	return counterMetric, nil
}

// This function returns a durationMetric from the durationMetricMap given a metricName
func GetDurationMetric(name metricName) (*DurationMetric, error) {
	durationMetric, ok := MetricsExp.internalData.durationMetricMap[name]
	if !ok {
		return nil, fmt.Errorf("%v not found in durationMetricMap due to metricName being defined, but not being a key in the map", name)
	}
	return durationMetric, nil
}

// This function returns a simpleGaugeMetric from the simpleGaugeMetricMap given a metricName
func GetSimpleGaugeMetric(name metricName) (*SimpleGaugeMetric, error) {
	gaugeMetric, ok := MetricsExp.internalData.simpleGaugeMetricMap[name]
	if !ok {
		return nil, fmt.Errorf("%v not found in SimpleGaugeMetricMap due to metricName being defined, but not being a key in the map", name)
	}
	return gaugeMetric, nil
}

// SetComponentAvailabilityMetric updates the components availability status metric
func SetComponentAvailabilityMetric(JSONname string, availability vzapi.ComponentAvailability, isEnabled bool) error {
	_, err := MetricsExp.internalData.componentHealth.SetComponentHealth(JSONname, availability == vzapi.ComponentAvailable, isEnabled)
	if err != nil {
		return err
	}
	return nil
}

func IsNonMetricComponent(componentName string) bool {
	var nonMetricComponents = map[string]bool{
		vmo.ComponentName:               true,
		networkpolicies.ComponentName:   true,
		grafanadashboards.ComponentName: true,
	}
	return nonMetricComponents[componentName]
}
