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
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/appoper"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/authproxy"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/coherence"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/console"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/externaldns"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/fluentd"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/grafana"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/istio"
	jaegeroperator "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/jaeger/operator"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/keycloak"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/kiali"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/mysql"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/nginx"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/oam"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/opensearch"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/opensearchdashboards"
	promadapter "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/prometheus/adapter"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/prometheus/kubestatemetrics"
	promnodeexporter "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/prometheus/nodeexporter"
	promoperator "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/prometheus/operator"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/prometheus/pushgateway"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/rancher"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/verrazzano"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/vmo"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/weblogic"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/util/wait"
)

var MetricsExp MetricsExporter

// Initalized but not populated

// Deciding register all of the metrics internraly when we initalize it controller wise

// Want variable name but for it to be internal
type metricsOperation string
type metricName string

const (
	operationInstall metricsOperation = "install"
	operationUpgrade metricsOperation = "upgrade"
	millisPerSecond  float64          = 1000.0
	reconcileCounter metricName       = "reconcile counter"
)

// List 1.) To provide same initalization interface 1a.) Decide we should initalize server and metrics at same time or different (right now different)
// 2.) Provide common interface for creating a new metrics (similar to metric component Struct)
// 3.) Provide a common interface for updating and interacting with metrics (Add error handling)
// 4.) One structure encopassess all metrics might have different member functions of metrics and updates structure
// 6.) Internally have map that maps to string that has metrics
// 7.) Maybe have external object that provides metrics constant so can access them whatever strings
// 8.) Metrics will be can't dynamically register or unregister a metrics
// 9. Functions generate UpdateCount("authproxy") ->
// 10.) Have different struct for each type of metric and for each function share the same type of metric
func InitRegisterStart(log *zap.SugaredLogger) {
	RequiredInitialization()
	RegisterMetrics(log)
	StartMetricsServer(log)
}
func TestInitalization() {
	RequiredInitialization()
}
func RequiredInitialization() {
	MetricsExp = MetricsExporter{
		internalConfig: initConfiguration(),
		internalData: data{
			simpleCounterMetricMap: initSimpleCounterMetricMap(),
			simpleGaugeMetricMap:   initSimpleGaugeMetricMap(),
			durationMetricMap:      initDurationMetricMap(),
			metricsComponentMap:    initMetricComponentMap(),
		},
	}

}

func RegisterMetrics(log *zap.SugaredLogger) {
	InitializeAllMetricsArray()
	go registerMetricsHandlers(log)
}

// This function returns a pointer to a new MetricComponent Object
func newMetricsComponent(name string) *MetricsComponent {
	return &MetricsComponent{
		LatestInstallDuration: &SimpleGaugeMetric{

			metric: prometheus.NewGauge(prometheus.GaugeOpts{
				Name: fmt.Sprintf("vz_%s_install_duration_seconds", name),
				Help: fmt.Sprintf("The duration of the latest installation of the %s component in seconds", name),
			}),
		},
		LatestUpgradeDuration: &SimpleGaugeMetric{
			prometheus.NewGauge(prometheus.GaugeOpts{
				Name: fmt.Sprintf("vz_%s_upgrade_duration_seconds", name),
				Help: fmt.Sprintf("The duration of the latest upgrade of the %s component in seconds", name),
			}),
		},
	}
}

func initSimpleCounterMetricMap() map[string]*SimpleCounterMetric {
	return map[string]*SimpleCounterMetric{
		"reconcile_counter": {
			prometheus.NewCounter(prometheus.CounterOpts{
				Name: "vpo_reconcile_counter",
				Help: "The number of times the reconcile function has been called in the Verrazzano-platform-operator",
			}),
		},
		"reconcile_error": {
			prometheus.NewCounter(prometheus.CounterOpts{
				Name: "vpo_error_reconcile_counter",
				Help: "The number of times the reconcile function has returned an error in the Verrazzano-platform-operator",
			}),
		},
	}
}
func initMetricComponentMap() map[string]*MetricsComponent {
	return map[string]*MetricsComponent{
		authproxy.ComponentName:            newMetricsComponent("authproxy"),
		oam.ComponentName:                  newMetricsComponent("oam"),
		appoper.ComponentName:              newMetricsComponent("appoper"),
		istio.ComponentName:                newMetricsComponent("istio"),
		weblogic.ComponentName:             newMetricsComponent("weblogic"),
		nginx.ComponentName:                newMetricsComponent("nginx"),
		certmanager.ComponentName:          newMetricsComponent("certManager"),
		externaldns.ComponentName:          newMetricsComponent("externalDNS"),
		rancher.ComponentName:              newMetricsComponent("rancher"),
		verrazzano.ComponentName:           newMetricsComponent("verrazzano"),
		vmo.ComponentName:                  newMetricsComponent("verrazzano_monitoring_operator"),
		opensearch.ComponentName:           newMetricsComponent("opensearch"),
		opensearchdashboards.ComponentName: newMetricsComponent("opensearch_dashboards"),
		grafana.ComponentName:              newMetricsComponent("grafana"),
		coherence.ComponentName:            newMetricsComponent("coherence"),
		mysql.ComponentName:                newMetricsComponent("mysql"),
		keycloak.ComponentName:             newMetricsComponent("keycloak"),
		kiali.ComponentName:                newMetricsComponent("kiali"),
		promoperator.ComponentName:         newMetricsComponent("prometheus_operator"),
		promadapter.ComponentName:          newMetricsComponent("prometheus_adapter"),
		kubestatemetrics.ComponentName:     newMetricsComponent("kube_state_metrics"),
		pushgateway.ComponentName:          newMetricsComponent("prometheus_push_gateway"),
		promnodeexporter.ComponentName:     newMetricsComponent("prometheus_node_exporter"),
		jaegeroperator.ComponentName:       newMetricsComponent("jaeger_operator"),
		console.ComponentName:              newMetricsComponent("verrazzano_console"),
		fluentd.ComponentName:              newMetricsComponent("fluentd"),
	}
}

func initSimpleGaugeMetricMap() map[metricName]*SimpleGaugeMetric {
	return map[metricName]*SimpleGaugeMetric{}
}

func initDurationMetricMap() map[metricName]*DurationMetric {
	return map[metricName]*DurationMetric{
		reconcileCounter: {
			metric: prometheus.NewSummary(prometheus.SummaryOpts{
				Name: "vpo_reconcile_duration",
				Help: "The duration in seconds of vpo reconcile process",
			}),
		},
	}
}

// This function is used to determine whether a durationTime for a component metric should be set and what the duration time is
// If the start time is greater than the completion time, the metric will not be set
// After this check, the function calculates the duration time and tries to set the metric of the component
// If the component's name is not in the metric map, an error will be raised to prevent a seg fault
func metricParserHelperFunction(log vzlog.VerrazzanoLogger, componentName string, startTime string, completionTime string, typeofOperation metricsOperation) error {
	startInSeconds, err := time.Parse(time.RFC3339, startTime)
	if err != nil {
		return fmt.Errorf("error in parsing start time %s for operation %s for component %s", startTime, typeofOperation, componentName)
	}
	startInSecondsUnix := startInSeconds.Unix()
	completionInSeconds, err := time.Parse(time.RFC3339, completionTime)
	if err != nil {
		return fmt.Errorf("error in parsing completion time %s for operation %s for component %s", completionTime, typeofOperation, componentName)
	}
	completionInSecondsUnix := completionInSeconds.Unix()
	if startInSecondsUnix >= completionInSecondsUnix {
		log.Debug("Component %s is not updated, as there is an ongoing operation in progress")
		return nil
	}
	totalDuration := (completionInSecondsUnix - startInSecondsUnix)
	_, ok := MetricsExp.internalData.metricsComponentMap[componentName]
	if !ok {
		log.Errorf("Component %s does not have metrics in the metrics map", componentName)
		return nil
	}
	if typeofOperation == operationUpgrade {
		MetricsExp.internalData.metricsComponentMap[componentName].LatestUpgradeDuration.Set(float64(totalDuration))
	}
	if typeofOperation == operationInstall {
		MetricsExp.internalData.metricsComponentMap[componentName].LatestInstallDuration.Set(float64(totalDuration))
	}
	return nil

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
		log.Errorf("Failed to register metrics for VPO %v \n", err)
		time.Sleep(time.Second)
	}
}
func initializeFailedMetricsArray() {
	for i, metric := range MetricsExp.internalConfig.allMetrics {
		MetricsExp.internalConfig.failedMetrics[metric] = i
	}
}
func StartMetricsServer(log *zap.SugaredLogger) {
	go registerMetricsHandlers(log)
	go wait.Until(func() {
		http.Handle("/metrics", promhttp.Handler())
		err := http.ListenAndServe(":9100", nil)
		if err != nil {
			log.Errorf("Failed to start metrics server for verrazzano-platform-operator: %v", err)
		}
	}, time.Second*3, wait.NeverStop)
}
func AnalyzeVerrazzanoResourceMetrics(log vzlog.VerrazzanoLogger, cr vzapi.Verrazzano) {
	mapOfComponents := cr.Status.Components
	for componentName, componentStatusDetails := range mapOfComponents {
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
		if (installStartTime == "" || installCompletionTime == "") && (upgradeStartTime == "" || upgradeCompletionTime == "") {
			continue
		}
		if installStartTime != "" && installCompletionTime != "" {
			err := metricParserHelperFunction(log, componentName, installStartTime, installCompletionTime, "install")
			if err != nil {
				log.Error(err)
			}
		}
		if upgradeStartTime != "" && upgradeCompletionTime != "" {
			err := metricParserHelperFunction(log, componentName, upgradeStartTime, upgradeCompletionTime, "upgrade")
			if err != nil {
				log.Error(err)
			}
		}
	}
}

func InitializeAllMetricsArray() {
	//loop through all metrics declarations in metric maps
	for _, value := range MetricsExp.internalData.simpleCounterMetricMap {
		MetricsExp.internalConfig.allMetrics = append(MetricsExp.internalConfig.allMetrics, value.metric)
	}
	for _, value := range MetricsExp.internalData.durationMetricMap {
		MetricsExp.internalConfig.allMetrics = append(MetricsExp.internalConfig.allMetrics, value.metric)
	}
	for _, value := range MetricsExp.internalData.metricsComponentMap {
		MetricsExp.internalConfig.allMetrics = append(MetricsExp.internalConfig.allMetrics, value.LatestInstallDuration.metric, value.LatestUpgradeDuration.metric)
	}
}
func initConfiguration() configuration {
	return configuration{
		allMetrics:    []prometheus.Collector{},
		failedMetrics: map[prometheus.Collector]int{},
		registry:      prometheus.DefaultRegisterer,
	}
}
func getCounterMetric(string Name)

// Implement Get Types, make const block with all of the metric strings, make the metricNames in the struct, make types private when possible
