// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricsexporter

import (
	"fmt"
	"net/http"
	"time"

	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/appoper"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/clusteroperator"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/coherence"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/console"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/grafanadashboards"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/mysqloperator"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/networkpolicies"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/oam"
	promnodeexporter "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/prometheus/nodeexporter"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/prometheus/pushgateway"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/rancher"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/verrazzano"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/vmo"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/weblogic"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/authproxy"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/externaldns"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/fluentd"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/grafana"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/istio"
	jaegeroperator "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/jaeger/operator"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/keycloak"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/kiali"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/mysql"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/nginx"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/opensearch"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/opensearchdashboards"
	promadapter "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/prometheus/adapter"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/prometheus/kubestatemetrics"
	promoperator "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/prometheus/operator"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/rancherbackup"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/velero"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/util/wait"
)

var MetricsExp MetricsExporter

type metricName string

const (
	ReconcileCounter               metricName = "reconcile counter"
	ReconcileError                 metricName = "reconcile error"
	ReconcileDuration              metricName = "reconcile duration"
	AvailableComponents            metricName = "available components"
	EnabledComponents              metricName = "enabled components"
	authproxyMetricName            metricName = authproxy.ComponentJSONName
	oamMetricName                  metricName = oam.ComponentJSONName
	appoperMetricName              metricName = appoper.ComponentJSONName
	istioMetricName                metricName = istio.ComponentJSONName
	weblogicMetricName             metricName = weblogic.ComponentJSONName
	nginxMetricName                metricName = nginx.ComponentJSONName
	certmanagerMetricName          metricName = certmanager.ComponentJSONName
	clusterOperatorMetricName      metricName = clusteroperator.ComponentJSONName
	externaldnsMetricName          metricName = externaldns.ComponentJSONName
	rancherMetricName              metricName = rancher.ComponentJSONName
	verrazzanoMetricName           metricName = verrazzano.ComponentJSONName
	opensearchMetricName           metricName = opensearch.ComponentJSONName
	opensearchdashboardsMetricName metricName = opensearchdashboards.ComponentJSONName
	grafanaMetricName              metricName = grafana.ComponentJSONName
	coherenceMetricName            metricName = coherence.ComponentJSONName
	mysqlMetricName                metricName = mysql.ComponentJSONName
	mysqlOperatorMetricName        metricName = mysqloperator.ComponentJSONName
	keycloakMetricname             metricName = keycloak.ComponentJSONName
	kialiMetricName                metricName = kiali.ComponentJSONName
	promoperatorMetricname         metricName = promoperator.ComponentJSONName
	promadapterMetricname          metricName = promadapter.ComponentJSONName
	kubestatemmetricsMetricName    metricName = kubestatemetrics.ComponentJSONName
	pushgatewayMetricName          metricName = pushgateway.ComponentJSONName
	promnodeexporterMetricname     metricName = promnodeexporter.ComponentJSONName
	jaegeroperatorMetricName       metricName = jaegeroperator.ComponentJSONName
	consoleMetricName              metricName = console.ComponentJSONName
	fluentdMetricName              metricName = fluentd.ComponentJSONName
	veleroMetricName               metricName = velero.ComponentJSONName
	rancherBackupMetricName        metricName = rancherbackup.ComponentJSONName
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
			metricsComponentMap:      initMetricComponentMap(),
			componentHealth:          initComponentHealthMetrics(),
			componentInstallDuration: initComponentInstallDurationMetrics(),
		},
	}
	// initialize component availability metric to false
	for _, metricComponent := range MetricsExp.internalData.metricsComponentMap {
		MetricsExp.internalData.componentHealth.SetComponentHealth(metricComponent.metricName, false, false)
	}

}

// This function begins the process of registering metrics
func RegisterMetrics(log *zap.SugaredLogger) {
	InitializeAllMetricsArray()
	go registerMetricsHandlers(log)
}

// This function returns a pointer to a new MetricComponent Object
func newMetricsComponent(name string) *MetricsComponent {
	return &MetricsComponent{
		metricName: name,
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

// This function initializes the metricComponentMap for the metricsExporter object
func initMetricComponentMap() map[metricName]*MetricsComponent {
	return map[metricName]*MetricsComponent{
		authproxyMetricName:            newMetricsComponent(authproxy.ComponentJSONName),
		oamMetricName:                  newMetricsComponent(oam.ComponentJSONName),
		appoperMetricName:              newMetricsComponent(appoper.ComponentJSONName),
		istioMetricName:                newMetricsComponent(istio.ComponentJSONName),
		weblogicMetricName:             newMetricsComponent(weblogic.ComponentJSONName),
		nginxMetricName:                newMetricsComponent(nginx.ComponentJSONName),
		certmanagerMetricName:          newMetricsComponent(certmanager.ComponentJSONName),
		clusterOperatorMetricName:      newMetricsComponent(clusteroperator.ComponentJSONName),
		externaldnsMetricName:          newMetricsComponent(externaldns.ComponentJSONName),
		rancherMetricName:              newMetricsComponent(rancher.ComponentJSONName),
		verrazzanoMetricName:           newMetricsComponent(verrazzano.ComponentJSONName),
		opensearchMetricName:           newMetricsComponent(opensearch.ComponentJSONName),
		opensearchdashboardsMetricName: newMetricsComponent(opensearchdashboards.ComponentJSONName),
		grafanaMetricName:              newMetricsComponent(grafana.ComponentJSONName),
		coherenceMetricName:            newMetricsComponent(coherence.ComponentJSONName),
		mysqlMetricName:                newMetricsComponent(mysql.ComponentJSONName),
		mysqlOperatorMetricName:        newMetricsComponent(mysqloperator.ComponentJSONName),
		keycloakMetricname:             newMetricsComponent(keycloak.ComponentJSONName),
		kialiMetricName:                newMetricsComponent(kiali.ComponentJSONName),
		promoperatorMetricname:         newMetricsComponent(promoperator.ComponentJSONName),
		promadapterMetricname:          newMetricsComponent(promadapter.ComponentJSONName),
		kubestatemmetricsMetricName:    newMetricsComponent(kubestatemetrics.ComponentJSONName),
		pushgatewayMetricName:          newMetricsComponent(pushgateway.ComponentJSONName),
		promnodeexporterMetricname:     newMetricsComponent(promnodeexporter.ComponentJSONName),
		jaegeroperatorMetricName:       newMetricsComponent(jaegeroperator.ComponentJSONName),
		consoleMetricName:              newMetricsComponent(console.ComponentJSONName),
		fluentdMetricName:              newMetricsComponent(fluentd.ComponentJSONName),
		veleroMetricName:               newMetricsComponent(velero.ComponentJSONName),
		rancherBackupMetricName:        newMetricsComponent(rancherbackup.ComponentJSONName),
	}
}

func initComponentHealthMetrics() *ComponentHealth {
	return &ComponentHealth{
		available: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vz_platform_operator_component_health",
			Help: "Is component enabled and available",
		}, []string{"component"}),
	}
}

func initComponentInstallDurationMetrics() *ComponentInstallDuration {
	return &ComponentInstallDuration{
		installDuration: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "vz_platform_operator_component_install_duration_seconds",
			Help: "Is component enabled and available",
		}, []string{"component"}),
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
	SetComponentInstallDurationMetric(componentName, totalDuration)

}

// SetComponentAvailabilityMetric updates the components availability status metric
func SetComponentInstallDurationMetric(componentName metricName, totalDuration int64) error {
	compMetric, err := GetMetricComponent(componentName)
	if err != nil {
		return err
	}
	MetricsExp.internalData.componentHealth.SetInstallDuration(compMetric.metricName, totalDuration)
	return nil
}

// This member function returns the simpleGaugeMetric that holds the upgrade time for a component
func (c *ComponentHealth) SetInstallDuration(name string, totalDuration int64) (prometheus.Gauge, error) {
	metric, err := c.available.GetMetricWithLabelValues(name)
	if err != nil {
		return nil, err
	}
	metric.Set(float64(totalDuration))
	return metric, nil
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
			metricParserHelperFunction(log, metricName(componentJSONName), installStartTime, installCompletionTime, constants.InstallOperation)
		}
		if upgradeStartTime != "" && upgradeCompletionTime != "" {
			metricParserHelperFunction(log, metricName(componentJSONName), upgradeStartTime, upgradeCompletionTime, constants.UpgradeOperation)
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

// SetComponentAvailabilityMetric updates the components availability status metric
func SetComponentAvailabilityMetric(name string, availability vzapi.ComponentAvailability, isEnabled bool) error {
	compMetric, err := GetMetricComponent(metricName(name))
	if err != nil {
		return err
	}
	MetricsExp.internalData.componentHealth.SetComponentHealth(compMetric.metricName, availability == vzapi.ComponentAvailable, isEnabled)
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
