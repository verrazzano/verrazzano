// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricsexporter

import (
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"go.uber.org/zap"
)

type metricComponent struct {
	LatestInstallDuration prometheus.Gauge
	LatestUpgradeDuration prometheus.Gauge
}

// This function populates an existing MetricComponent Struct with the chosen name of a component
func (m *metricComponent) init(name string) {
	m.LatestInstallDuration = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: fmt.Sprintf("vz_%s_install_duration_seconds", name),
		Help: fmt.Sprintf("The duration of the latest installation of the %s component in seconds", name),
	})
	m.LatestUpgradeDuration = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: fmt.Sprintf("vz_%s_upgrade_duration_seconds", name),
		Help: fmt.Sprintf("The duration of the latest upgrade of the %s component in seconds", name),
	})
}

// This function creates and populates a MetricComponent struct
func initalizeMetricsComponent(name string) metricComponent {
	metricComponent := metricComponent{}
	metricComponent.init(name)
	return metricComponent
}

// This function currently adds all defined metrics in the package to its list
// If one would like to add a new metric to this list, they must either update the metricmap with that metric or append it directly
func populateAllMetricsList(metricsMap map[string]metricComponent) []prometheus.Collector {
	metricsList := []prometheus.Collector{}
	for k := range metricsMap {
		metricsList = append(metricsList, metricsMap[k].LatestInstallDuration)
		metricsList = append(metricsList, metricsMap[k].LatestUpgradeDuration)
	}
	metricsList = append(metricsList, reconcileCounterMetric, reconcileErrorCounterMetric, reconcileLastDurationMetric)
	return metricsList
}

var (
	authproxyMetrics              = initalizeMetricsComponent("authproxy")
	oamMetrics                    = initalizeMetricsComponent("oam")
	appoperMetrics                = initalizeMetricsComponent("appoper")
	istioMetrics                  = initalizeMetricsComponent("istio")
	weblogicMetrics               = initalizeMetricsComponent("weblogic")
	nginxMetrics                  = initalizeMetricsComponent("nginx")
	certManagerMetrics            = initalizeMetricsComponent("certManager")
	externalDNSMetrics            = initalizeMetricsComponent("externalDNS")
	rancherMetrics                = initalizeMetricsComponent("rancher")
	verrazzanoMetrics             = initalizeMetricsComponent("verrazzano")
	vmoMetrics                    = initalizeMetricsComponent("verrazzano_monitoring_operator")
	opensearchMetrics             = initalizeMetricsComponent("opensearch")
	opensearchDashBoardsMetrics   = initalizeMetricsComponent("opensearch_dashboards")
	grafanaMetrics                = initalizeMetricsComponent("grafana")
	coherenceMetrics              = initalizeMetricsComponent("coherence")
	mySQLMetrics                  = initalizeMetricsComponent("mysql")
	keycloakMetrics               = initalizeMetricsComponent("keycloak")
	kialiMetrics                  = initalizeMetricsComponent("kiali")
	prometheusOperatorMetrics     = initalizeMetricsComponent("prometheus_operator")
	prometheusAdapterMetrics      = initalizeMetricsComponent("prometheus_adapter")
	kubeStateMetricsMetrics       = initalizeMetricsComponent("kube_state_metrics")
	prometheusPushGatewayMetrics  = initalizeMetricsComponent("prometheus_push_gateway")
	prometheusNodeExporterMetrics = initalizeMetricsComponent("prometheus_node_exporter")
	jaegerOperatorMetrics         = initalizeMetricsComponent("jaeger_operator")
	verrazzanoConsoleMetrics      = initalizeMetricsComponent("verrazzano_console")
	fluentdMetrics                = initalizeMetricsComponent("fluentd")

	reconcileIndex         int = 0
	reconcileCounterMetric     = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "vpo_reconcile_counter",
		Help: "The number of times the reconcile function has been called in the Verrazzano-platform-operator",
	})
	reconcileLastDurationMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "vpo_reconcile_duration_seconds",
		Help: "The duration of each reconcile call in the Verrazzano-platform-operator in seconds"},
		[]string{"reconcile_index"},
	)
	reconcileErrorCounterMetric = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "vpo_error_reconcile_counter",
		Help: "The number of times the reconcile function has returned an error in the Verrazzano-platform-operator",
	})
	// The metricsMap keys are the names of the components as they appear in the VZ CR
	metricsMap = map[string]metricComponent{
		"verrazzano-authproxy":            authproxyMetrics,
		"oam-kubernetes-runtime":          oamMetrics,
		"verrazzano-application-operator": appoperMetrics,
		"istio":                           istioMetrics,
		"weblogic-operator":               weblogicMetrics,
		"ingress-controller":              nginxMetrics,
		"cert-manager":                    certManagerMetrics,
		"external-dns":                    externalDNSMetrics,
		"rancher":                         rancherMetrics,
		"verrazzano":                      verrazzanoMetrics,
		"verrazzano-monitoring-operator":  vmoMetrics,
		"opensearch":                      opensearchMetrics,
		"opensearch-dashboards":           opensearchDashBoardsMetrics,
		"grafana":                         grafanaMetrics,
		"coherence-operator":              coherenceMetrics,
		"mysql":                           mySQLMetrics,
		"keycloak":                        keycloakMetrics,
		"kiali-server":                    kialiMetrics,
		"prometheus-operator":             prometheusOperatorMetrics,
		"prometheus-adapter":              prometheusAdapterMetrics,
		"kube-state-metrics":              kubeStateMetricsMetrics,
		"prometheus-pushgateway":          prometheusPushGatewayMetrics,
		"prometheus-node-exporter":        prometheusNodeExporterMetrics,
		"jaeger-operator":                 jaegerOperatorMetrics,
		"verrazzano-console":              verrazzanoConsoleMetrics,
		"fluentd":                         fluentdMetrics,
	}

	// This array will be populated with all the metrics from each map
	// Metrics not included in a map can be added to this array for registration
	allMetrics = populateAllMetricsList(metricsMap)
	// This map will be automatically populated with all metrics which were not registered correctly
	// Metrics in this map will be retried periodically
	failedMetrics = map[prometheus.Collector]int{}

	registry = prometheus.DefaultRegisterer
)

// This function is used to determine whether a durationTime for a component metric should be set and what the duration time is
// If the start time is greater than the completion time, the metric will not be set
// After this check, the function calculates the duration time and tries to set the metric of the component
// If the component's name is not in the metric map, an error will be raised to prevent a seg fault
func metricParserHelperFunction(log vzlog.VerrazzanoLogger, componentName string, startTime string, completionTime string, typeofOperation string) {
	startInSeconds, _ := time.Parse(time.RFC3339, startTime)
	startInSecondsUnix := startInSeconds.Unix()
	completionInSeconds, _ := time.Parse(time.RFC3339, completionTime)
	completionInSecondsUnix := completionInSeconds.Unix()
	if startInSecondsUnix >= completionInSecondsUnix {
		return
	}
	totalDuration := (completionInSecondsUnix - startInSecondsUnix)
	_, ok := metricsMap[componentName]
	if !ok {
		log.Errorf("Component %s does not have metrics in the metrics map", componentName)
		return
	}
	if typeofOperation == "upgrade" {
		metricsMap[componentName].LatestUpgradeDuration.Set(float64(totalDuration))
	}
	if typeofOperation == "install" {
		metricsMap[componentName].LatestInstallDuration.Set(float64(totalDuration))
	}

}
func registerMetricsHandlersHelper(log *zap.SugaredLogger) error {
	var errorObserved error
	for metric, i := range failedMetrics {
		err := registry.Register(metric)
		if err != nil {
			log.Errorf("Failed to register metric index %v for VPO", i)
			errorObserved = err
		} else {
			//if a metric is registered, delete it from the failed metrics map so that it is not retried
			delete(failedMetrics, metric)
		}
	}
	return errorObserved
}
func registerMetricsHandlers(log *zap.SugaredLogger) {
	initializeFailedMetricsArray() // Get list of metrics to register initially
	// loop until there is no error in registering
	for err := registerMetricsHandlersHelper(log); err != nil; err = registerMetricsHandlersHelper(log) {
		log.Errorf("Failed to register some metrics for VPO: %v", err)
		time.Sleep(time.Second)
	}
}
func initializeFailedMetricsArray() {
	for i, metric := range allMetrics {
		failedMetrics[metric] = i
	}
}
