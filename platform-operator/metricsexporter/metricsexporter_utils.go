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
	"github.com/verrazzano/verrazzano/platform-operator/constants"
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
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/mysqloperator"
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
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/rancherbackup"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/velero"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/verrazzano"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/vmo"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/weblogic"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/util/wait"
)

var MetricsExp MetricsExporter

type metricName string

const (
	ReconcileCounter               metricName = "reconcile counter"
	ReconcileError                 metricName = "reconcile error"
	ReconcileDuration              metricName = "reconcile duration"
	authproxyMetricName            metricName = authproxy.ComponentName
	oamMetricName                  metricName = oam.ComponentName
	appoperMetricName              metricName = appoper.ComponentName
	istioMetricName                metricName = istio.ComponentName
	weblogicMetricName             metricName = weblogic.ComponentName
	nginxMetricName                metricName = nginx.ComponentName
	certmanagerMetricName          metricName = certmanager.ComponentName
	externaldnsMetricName          metricName = externaldns.ComponentName
	rancherMetricName              metricName = rancher.ComponentName
	verrazzanoMetricName           metricName = verrazzano.ComponentName
	vmoMetricName                  metricName = vmo.ComponentName
	opensearchMetricName           metricName = opensearch.ComponentName
	opensearchdashboardsMetricName metricName = opensearchdashboards.ComponentName
	grafanaMetricName              metricName = grafana.ComponentName
	coherenceMetricName            metricName = coherence.ComponentName
	mysqlMetricName                metricName = mysql.ComponentName
	mysqlOperatorMetricName        metricName = mysqloperator.ComponentName
	keycloakMetricname             metricName = keycloak.ComponentName
	kialiMetricName                metricName = kiali.ComponentName
	promoperatorMetricname         metricName = promoperator.ComponentName
	promadapterMetricname          metricName = promadapter.ComponentName
	kubestatemmetricsMetricName    metricName = kubestatemetrics.ComponentName
	pushgatewayMetricName          metricName = pushgateway.ComponentName
	promnodeexporterMetricname     metricName = promnodeexporter.ComponentName
	jaegeroperatorMetricName       metricName = jaegeroperator.ComponentName
	consoleMetricName              metricName = console.ComponentName
	fluentdMetricName              metricName = fluentd.ComponentName
	veleroMetricName               metricName = velero.ComponentName
	rancherBackupMetricName        metricName = rancherbackup.ComponentName
)

func init() {
	RequiredInitialization()
	RegisterMetrics(zap.S())
}

// This function initalizes the metrics object, but does not register the metrics
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

//This function begins the process of registering metrics
func RegisterMetrics(log *zap.SugaredLogger) {
	InitializeAllMetricsArray()
	go registerMetricsHandlers(log)
}

// This function returns a pointer to a new MetricComponent Object
func newMetricsComponent(name string) *MetricsComponent {
	return &MetricsComponent{
		latestInstallDuration: &SimpleGaugeMetric{

			metric: prometheus.NewGauge(prometheus.GaugeOpts{
				Name: fmt.Sprintf("vz_%s_install_duration_seconds", name),
				Help: fmt.Sprintf("The duration of the latest installation of the %s component in seconds", name),
			}),
		},
		latestUpgradeDuration: &SimpleGaugeMetric{
			prometheus.NewGauge(prometheus.GaugeOpts{
				Name: fmt.Sprintf("vz_%s_upgrade_duration_seconds", name),
				Help: fmt.Sprintf("The duration of the latest upgrade of the %s component in seconds", name),
			}),
		},
	}
}

//This function initalizes the simpleCounterMetricMap for the metricsExporter object
func initSimpleCounterMetricMap() map[metricName]*SimpleCounterMetric {
	return map[metricName]*SimpleCounterMetric{
		ReconcileCounter: {
			prometheus.NewCounter(prometheus.CounterOpts{
				Name: "vpo_reconcile_counter",
				Help: "The number of times the reconcile function has been called in the verrazzano-platform-operator",
			}),
		},
		ReconcileError: {
			prometheus.NewCounter(prometheus.CounterOpts{
				Name: "vpo_error_reconcile_counter",
				Help: "The number of times the reconcile function has returned an error in the verrazzano-platform-operator",
			}),
		},
	}
}

// This function initalizes the metricComponentMap for the metricsExporter object
func initMetricComponentMap() map[metricName]*MetricsComponent {
	return map[metricName]*MetricsComponent{
		authproxyMetricName:            newMetricsComponent("authproxy"),
		oamMetricName:                  newMetricsComponent("oam"),
		appoperMetricName:              newMetricsComponent("appoper"),
		istioMetricName:                newMetricsComponent("istio"),
		weblogicMetricName:             newMetricsComponent("weblogic"),
		nginxMetricName:                newMetricsComponent("nginx"),
		certmanagerMetricName:          newMetricsComponent("certManager"),
		externaldnsMetricName:          newMetricsComponent("externalDNS"),
		rancherMetricName:              newMetricsComponent("rancher"),
		verrazzanoMetricName:           newMetricsComponent("verrazzano"),
		vmoMetricName:                  newMetricsComponent("verrazzano_monitoring_operator"),
		opensearchMetricName:           newMetricsComponent("opensearch"),
		opensearchdashboardsMetricName: newMetricsComponent("opensearch_dashboards"),
		grafanaMetricName:              newMetricsComponent("grafana"),
		coherenceMetricName:            newMetricsComponent("coherence"),
		mysqlMetricName:                newMetricsComponent("mysql"),
		mysqlOperatorMetricName:        newMetricsComponent("mysql_operator"),
		keycloakMetricname:             newMetricsComponent("keycloak"),
		kialiMetricName:                newMetricsComponent("kiali"),
		promoperatorMetricname:         newMetricsComponent("prometheus_operator"),
		promadapterMetricname:          newMetricsComponent("prometheus_adapter"),
		kubestatemmetricsMetricName:    newMetricsComponent("kube_state_metrics"),
		pushgatewayMetricName:          newMetricsComponent("prometheus_push_gateway"),
		promnodeexporterMetricname:     newMetricsComponent("prometheus_node_exporter"),
		jaegeroperatorMetricName:       newMetricsComponent("jaeger_operator"),
		consoleMetricName:              newMetricsComponent("verrazzano_console"),
		fluentdMetricName:              newMetricsComponent("fluentd"),
		veleroMetricName:               newMetricsComponent("velero"),
		rancherBackupMetricName:        newMetricsComponent("rancher-backup"),
	}
}

// This function initalizes the simpleGaugeMetricMap for the metricsExporter object
func initSimpleGaugeMetricMap() map[metricName]*SimpleGaugeMetric {
	return map[metricName]*SimpleGaugeMetric{}
}

// This function initalizes the durationMetricMap for the metricsExporter object
func initDurationMetricMap() map[metricName]*DurationMetric {
	return map[metricName]*DurationMetric{
		ReconcileDuration: {
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
func metricParserHelperFunction(log vzlog.VerrazzanoLogger, componentName metricName, startTime string, completionTime string, typeofOperation string) {
	_, ok := MetricsExp.internalData.metricsComponentMap[componentName]
	if !ok {
		log.Errorf("Component %s does not have metrics in the metrics map", componentName)
		return
	}
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
		installDurationMetricForComponent := MetricsExp.internalData.metricsComponentMap[componentName].getInstallDuration()
		installDurationMetricForComponent.Set(float64(totalDuration))
	}
	if typeofOperation == constants.UpgradeOperation {
		upgradeDurationMetricForComponent := MetricsExp.internalData.metricsComponentMap[componentName].getUpgradeDuration()
		upgradeDurationMetricForComponent.Set(float64(totalDuration))
	}

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
			//if a metric is registered, delete it from the failed metrics map so that it is not retried
			delete(MetricsExp.internalConfig.failedMetrics, metric)
		}
	}
	return errorObserved
}

// This function registers the metrics and provides error handling
func registerMetricsHandlers(log *zap.SugaredLogger) {
	initializeFailedMetricsArray() //Get list of metrics to register initially
	//loop until there is no error in registering
	for err := registerMetricsHandlersHelper(); err != nil; err = registerMetricsHandlersHelper() {
		log.Errorf("Failed to register metrics for VPO %v \n", err)
		time.Sleep(time.Second)
	}
}

// This function initalizes the failedMetrics array
func initializeFailedMetricsArray() {
	for i, metric := range MetricsExp.internalConfig.allMetrics {
		MetricsExp.internalConfig.failedMetrics[metric] = i
	}
}

// This function starts the metric server to begin emitting metrics to Prometheus
func StartMetricsServer(log *zap.SugaredLogger) {
	go wait.Until(func() {
		http.Handle("/metrics", promhttp.Handler())
		err := http.ListenAndServe(":9100", nil)
		if err != nil {
			log.Errorf("Failed to start metrics server for verrazzano-platform-operator: %v", err)
		}
	}, time.Second*3, wait.NeverStop)
}

// This functionn parses the VZ CR and extracts the install and update data for each component
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
		if installStartTime != "" && installCompletionTime != "" {
			metricParserHelperFunction(log, metricName(componentName), installStartTime, installCompletionTime, constants.InstallOperation)
		}
		if upgradeStartTime != "" && upgradeCompletionTime != "" {
			metricParserHelperFunction(log, metricName(componentName), upgradeStartTime, upgradeCompletionTime, constants.UpgradeOperation)
		}
	}
}

// This function initalizes the allMetrics array
func InitializeAllMetricsArray() {
	//loop through all metrics declarations in metric maps
	for _, value := range MetricsExp.internalData.simpleCounterMetricMap {
		MetricsExp.internalConfig.allMetrics = append(MetricsExp.internalConfig.allMetrics, value.metric)
	}
	for _, value := range MetricsExp.internalData.durationMetricMap {
		MetricsExp.internalConfig.allMetrics = append(MetricsExp.internalConfig.allMetrics, value.metric)
	}
	for _, value := range MetricsExp.internalData.metricsComponentMap {
		MetricsExp.internalConfig.allMetrics = append(MetricsExp.internalConfig.allMetrics, value.latestInstallDuration.metric, value.latestUpgradeDuration.metric)
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

// This function returns a metricComponent from the metricComponentMap given a metricName
func GetMetricComponent(name metricName) (*MetricsComponent, error) {
	metricComponent, ok := MetricsExp.internalData.metricsComponentMap[name]
	if !ok {
		return nil, fmt.Errorf("%v not found in metricsComponentMap due to metricName being defined, but not being a key in the map", name)
	}
	return metricComponent, nil
}
