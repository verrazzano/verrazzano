// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package metricsexporter

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/util/wait"
)

var (
	reconcileIndex         int = 0
	reconcileCounterMetric     = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "counter_for_reconcile_function",
		Help: "The number of times the reconcile function has been called in the VPO",
	})
	reconcileLastDurationMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "reconcileLastTime",
		Help: "The duration of each reconcile call"},
		[]string{"reconcile_index"},
	)
	verrazzanoAuthproxyInstallTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "authproxy_component_install_time",
		Help: "The install time for the authproxy component",
	})
	oamInstallTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "oam_component_install_time",
		Help: "The install time for the oam component",
	})
	apopperInstallTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "apopper_component_install_time",
		Help: "The install time for the apopper component",
	})
	istioInstallTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "istio_component_install_time",
		Help: "The install time for the istio component",
	})
	weblogicInstallTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "weblogic_component_install_time",
		Help: "The install time for the weblogic component",
	})
	nginxInstallTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "nginx_component_install_time",
		Help: "The install time for the nginx component",
	})
	certManagerInstallTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "certManager_component_install_time",
		Help: "The install time for the certManager component",
	})
	externalDNSInstallTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "externalDNS_component_install_time",
		Help: "The install time for the externalDNS component",
	})
	rancherInstallTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "rancher_component_install_time",
		Help: "The install time for the rancher component",
	})
	verrazzanoInstallTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "verrazzano_component_install_time",
		Help: "The install time for the verrazzano component",
	})
	verrazzanoMonitoringOperatorInstallTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "verrazzano_monitoring_operator_component_install_time",
		Help: "The install time for the verrazzano-monitoring-operator component",
	})
	openSearchInstallTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "open_search_component_install_time",
		Help: "The install time for the opensearch component",
	})
	openSearchDashboardsInstallTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "open_search_dashboards_component_install_time",
		Help: "The install time for the opensearch-dashboards component",
	})
	grafanaInstallTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "grafana_component_install_time",
		Help: "The install time for the grafana component",
	})
	coherenceInstallTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "coherence_component_install_time",
		Help: "The install time for the coherence component",
	})
	mySQLInstallTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "my_sql_component_install_time",
		Help: "The install time for the mysql component",
	})
	keycloakInstallTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "keycloak_component_install_time",
		Help: "The install time for the keycloak component",
	})
	kialiInstallTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "kiali_component_install_time",
		Help: "The install time for the kiali component",
	})
	prometheusOperatorInstallTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "prometheus_operator_install_time",
		Help: "The install time for the prometheus-operator component",
	})
	prometheusAdapterInstallTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "prometheus_adapter_install_time",
		Help: "The install time for the prometheus-adapter component",
	})
	kubeStateMetricsInstallTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "kube_state_metrics_install_time",
		Help: "The install time for the kube-state-metrics component",
	})
	prometheusPushGatewayInstallTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "prometheus_push_gateway_install_time",
		Help: "The install time for the prometheus-push-gateway component",
	})
	prometheusNodeExporterInstallTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "prometheus_node_exporter_install_time",
		Help: "The install time for the prometheus-node-exporter component",
	})
	jaegerOperatorInstallTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "jaeger_operator_install_time",
		Help: "The install time for the jaeger-operator component",
	})
	verrazzanoConsoleInstallTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "verrazzano_console_install_time",
		Help: "The install time for the verrazzano-console component",
	})
	fluentdInstallTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "fluentd_install_time",
		Help: "The install time for the fluentd component",
	})
	verrazzanoAuthproxyUpgradeTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "authproxy_component_upgrade_time",
		Help: "The upgrade time for the authproxy component",
	})
	oamUpgradeTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "oam_component_upgrade_time",
		Help: "The upgrade time for the oam component",
	})
	apopperUpgradeTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "apopper_component_upgrade_time",
		Help: "The upgrade time for the apopper component",
	})
	istioUpgradeTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "istio_component_upgrade_time",
		Help: "The upgrade time for the istio component",
	})
	weblogicUpgradeTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "weblogic_component_upgrade_time",
		Help: "The upgrade time for the weblogic component",
	})
	nginxUpgradeTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "nginx_component_upgrade_time",
		Help: "The upgrade time for the nginx component",
	})
	certManagerUpgradeTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "certManager_component_upgrade_time",
		Help: "The upgrade time for the certManager component",
	})
	externalDNSUpgradeTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "externalDNS_component_upgrade_time",
		Help: "The upgrade time for the externalDNS component",
	})
	rancherUpgradeTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "rancher_component_upgrade_time",
		Help: "The upgrade time for the rancher component",
	})
	verrazzanoUpgradeTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "verrazzano_component_upgrade_time",
		Help: "The upgrade time for the verrazzano component",
	})
	verrazzanoMonitoringOperatorUpgradeTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "verrazzano_monitoring_operator_component_upgrade_time",
		Help: "The upgrade time for the verrazzano-monitoring-operator component",
	})
	openSearchUpgradeTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "open_search_component_upgrade_time",
		Help: "The upgrade time for the opensearch component",
	})
	openSearchDashboardsUpgradeTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "open_search_dashboards_component_upgrade_time",
		Help: "The upgrade time for the opensearch-dashboards component",
	})
	grafanaUpgradeTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "grafana_component_upgrade_time",
		Help: "The upgrade time for the grafana component",
	})
	coherenceUpgradeTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "coherence_component_upgrade_time",
		Help: "The upgrade time for the coherence component",
	})
	mySQLUpgradeTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "my_sql_component_upgrade_time",
		Help: "The upgrade time for the mysql component",
	})
	keycloakUpgradeTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "keycloak_component_upgrade_time",
		Help: "The upgrade time for the keycloak component",
	})
	kialiUpgradeTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "kiali_component_upgrade_time",
		Help: "The upgrade time for the upgrade component",
	})
	prometheusOperatorUpgradeTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "prometheus_operator_upgrade_time",
		Help: "The upgrade time for the prometheus-operator component",
	})
	prometheusAdapterUpgradeTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "prometheus_adapter_upgrade_time",
		Help: "The upgrade time for the prometheus-adapter component",
	})
	kubeStateMetricsUpgradeTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "kube_state_metrics_upgrade_time",
		Help: "The upgrade time for the kube-state-metrics component",
	})
	prometheusPushGatewayUpgradeTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "prometheus_push_gateway_upgrade_time",
		Help: "The upgrade time for the prometheus-push-gateway component",
	})
	prometheusNodeExporterUpgradeTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "prometheus_node_exporter_upgrade_time",
		Help: "The upgrade time for the prometheus-node-exporter component",
	})
	jaegerOperatorUpgradeTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "jaeger_operator_upgrade_time",
		Help: "The upgrade time for the jaeger-operator component",
	})
	verrazzanoConsoleUpgradeTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "verrazzano_console_upgrade_time",
		Help: "The upgrade time for the verrazzano-console component",
	})
	fluentdUpgradeTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "fluentd_upgrade_time",
		Help: "The upgrade time for the fluentd component",
	})
	verrazzanoAuthproxyUpdateTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "authproxy_component_update_time",
		Help: "The update time for the authproxy component",
	})
	oamUpdateTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "oam_component_update_time",
		Help: "The update time for the oam component",
	})
	apopperUpdateTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "apopper_component_update_time",
		Help: "The update time for the apopper component",
	})
	istioUpdateTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "istio_component_update_time",
		Help: "The update time for the istio component",
	})
	weblogicUpdateTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "weblogic_component_update_time",
		Help: "The update time for the weblogic component",
	})
	nginxUpdateTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "nginx_component_update_time",
		Help: "The update time for the nginx component",
	})
	certManagerUpdateTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "certManager_component_update_time",
		Help: "The update time for the certManager component",
	})
	externalDNSUpdateTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "externalDNS_component_update_time",
		Help: "The update time for the externalDNS component",
	})
	rancherUpdateTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "rancher_component_update_time",
		Help: "The update time for the rancher component",
	})
	verrazzanoUpdateTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "verrazzano_component_update_time",
		Help: "The update time for the verrazzano component",
	})
	verrazzanoMonitoringOperatorUpdateTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "verrazzano_monitoring_operator_component_update_time",
		Help: "The update time for the verrazzano-monitoring-operator component",
	})
	openSearchUpdateTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "open_search_component_update_time",
		Help: "The update time for the opensearch component",
	})
	openSearchDashboardsUpdateTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "open_search_dashboards_component_update_time",
		Help: "The update time for the opensearch-dashboards component",
	})
	grafanaUpdateTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "grafana_component_update_time",
		Help: "The update time for the grafana component",
	})
	coherenceUpdateTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "coherence_component_update_time",
		Help: "The update time for the coherence component",
	})
	mySQLUpdateTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "my_sql_component_update_time",
		Help: "The update time for the mysql component",
	})
	keycloakUpdateTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "keycloak_component_update_time",
		Help: "The update time for the keycloak component",
	})
	kialiUpdateTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "kiali_component_update_time",
		Help: "The update time for the kiali component",
	})
	prometheusOperatorUpdateTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "prometheus_operator_update_time",
		Help: "The update time for the prometheus-operator component",
	})
	prometheusAdapterUpdateTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "prometheus_adapter_update_time",
		Help: "The update time for the prometheus-adapter component",
	})
	kubeStateMetricsUpdateTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "kube_state_metrics_update_time",
		Help: "The update time for the kube-state-metrics component",
	})
	prometheusPushGatewayUpdateTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "prometheus_push_gateway_update_time",
		Help: "The update time for the prometheus-push-gateway component",
	})
	prometheusNodeExporterUpdateTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "prometheus_node_exporter_update_time",
		Help: "The update time for the prometheus-node-exporter component",
	})
	jaegerOperatorUpdateTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "jaeger_operator_update_time",
		Help: "The update time for the jaeger-operator component",
	})
	verrazzanoConsoleUpdateTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "verrazzano_console_update_time",
		Help: "The update time for the verrazzano-console component",
	})
	fluentdUpdateTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "fluentd_update_time",
		Help: "The update time for the fluentd component",
	})
	allMetrics = []prometheus.Collector{verrazzanoAuthproxyInstallTimeMetric,
		oamInstallTimeMetric,
		apopperInstallTimeMetric,
		istioInstallTimeMetric,
		weblogicInstallTimeMetric,
		nginxInstallTimeMetric,
		certManagerInstallTimeMetric,
		externalDNSInstallTimeMetric,
		rancherInstallTimeMetric,
		verrazzanoInstallTimeMetric,
		verrazzanoMonitoringOperatorInstallTimeMetric,
		openSearchInstallTimeMetric,
		openSearchDashboardsInstallTimeMetric,
		grafanaInstallTimeMetric,
		coherenceInstallTimeMetric,
		mySQLInstallTimeMetric,
		keycloakInstallTimeMetric,
		kialiInstallTimeMetric,
		prometheusOperatorInstallTimeMetric,
		prometheusAdapterInstallTimeMetric,
		kubeStateMetricsInstallTimeMetric,
		prometheusPushGatewayInstallTimeMetric,
		prometheusNodeExporterInstallTimeMetric,
		jaegerOperatorInstallTimeMetric,
		verrazzanoConsoleInstallTimeMetric,
		fluentdInstallTimeMetric,
		verrazzanoAuthproxyUpgradeTimeMetric,
		oamUpgradeTimeMetric,
		apopperUpgradeTimeMetric,
		istioUpgradeTimeMetric,
		weblogicUpgradeTimeMetric,
		nginxUpgradeTimeMetric,
		certManagerUpgradeTimeMetric,
		externalDNSUpgradeTimeMetric,
		rancherUpgradeTimeMetric,
		verrazzanoUpgradeTimeMetric,
		verrazzanoMonitoringOperatorUpgradeTimeMetric,
		openSearchUpgradeTimeMetric,
		openSearchDashboardsUpgradeTimeMetric,
		grafanaUpgradeTimeMetric,
		coherenceUpgradeTimeMetric,
		mySQLUpgradeTimeMetric,
		keycloakUpgradeTimeMetric,
		kialiUpgradeTimeMetric,
		prometheusOperatorUpgradeTimeMetric,
		prometheusAdapterUpgradeTimeMetric,
		kubeStateMetricsUpgradeTimeMetric,
		prometheusPushGatewayUpgradeTimeMetric,
		prometheusNodeExporterUpgradeTimeMetric,
		jaegerOperatorUpgradeTimeMetric,
		verrazzanoConsoleUpgradeTimeMetric,
		fluentdUpgradeTimeMetric,
		verrazzanoAuthproxyUpdateTimeMetric,
		oamUpdateTimeMetric,
		apopperUpdateTimeMetric,
		istioUpdateTimeMetric,
		weblogicUpdateTimeMetric,
		nginxUpdateTimeMetric,
		certManagerUpdateTimeMetric,
		externalDNSUpdateTimeMetric,
		rancherUpdateTimeMetric,
		verrazzanoUpdateTimeMetric,
		verrazzanoMonitoringOperatorUpdateTimeMetric,
		openSearchUpdateTimeMetric,
		openSearchDashboardsUpdateTimeMetric,
		grafanaUpdateTimeMetric,
		coherenceUpdateTimeMetric,
		mySQLUpdateTimeMetric,
		keycloakUpdateTimeMetric,
		kialiUpdateTimeMetric,
		prometheusOperatorUpdateTimeMetric,
		prometheusAdapterUpdateTimeMetric,
		kubeStateMetricsUpdateTimeMetric,
		prometheusPushGatewayUpdateTimeMetric,
		prometheusNodeExporterUpdateTimeMetric,
		jaegerOperatorUpdateTimeMetric,
		verrazzanoConsoleUpdateTimeMetric,
		fluentdUpdateTimeMetric,
		reconcileCounterMetric,
		reconcileLastDurationMetric,
	}
	failedMetrics = map[prometheus.Collector]int{}
	registry      = prometheus.DefaultRegisterer

	installMetricsMap = map[string]prometheus.Gauge{
		"verrazzano-authproxy":            verrazzanoAuthproxyInstallTimeMetric,
		"oam-kubernetes-runtime":          oamInstallTimeMetric,
		"verrazzano-application-operator": apopperInstallTimeMetric,
		"istio":                           istioInstallTimeMetric,
		"weblogic-operator":               weblogicInstallTimeMetric,
		"ingress-controller":              nginxInstallTimeMetric,
		"cert-manager":                    certManagerInstallTimeMetric,
		"external-dns":                    externalDNSInstallTimeMetric,
		"rancher":                         rancherInstallTimeMetric,
		"verrazzano":                      verrazzanoInstallTimeMetric,
		"verrazzano-monitoring-operator":  verrazzanoMonitoringOperatorInstallTimeMetric,
		"opensearch":                      openSearchInstallTimeMetric,
		"opensearch-dashboards":           openSearchDashboardsInstallTimeMetric,
		"grafana":                         grafanaInstallTimeMetric,
		"coherence-operator":              coherenceInstallTimeMetric,
		"mysql":                           mySQLInstallTimeMetric,
		"keycloak":                        keycloakInstallTimeMetric,
		"kiali-server":                    kialiInstallTimeMetric,
		"prometheus-operator":             prometheusOperatorInstallTimeMetric,
		"prometheus-adapter":              prometheusAdapterInstallTimeMetric,
		"kube-state-metrics":              kubeStateMetricsInstallTimeMetric,
		"prometheus-pushgateway":          prometheusPushGatewayInstallTimeMetric,
		"prometheus-node-exporter":        prometheusNodeExporterInstallTimeMetric,
		"jaeger-operator":                 jaegerOperatorInstallTimeMetric,
		"verrazzano-console":              verrazzanoConsoleInstallTimeMetric,
		"fluentd":                         fluentdInstallTimeMetric,
	}
	upgradeMetricsMap = map[string]prometheus.Gauge{
		"verrazzano-authproxy":            verrazzanoAuthproxyUpgradeTimeMetric,
		"oam-kubernetes-runtime":          oamUpgradeTimeMetric,
		"verrazzano-application-operator": apopperUpgradeTimeMetric,
		"istio":                           istioUpgradeTimeMetric,
		"weblogic-operator":               weblogicUpgradeTimeMetric,
		"ingress-controller":              nginxUpgradeTimeMetric,
		"cert-manager":                    certManagerUpgradeTimeMetric,
		"external-dns":                    externalDNSUpgradeTimeMetric,
		"rancher":                         rancherUpgradeTimeMetric,
		"verrazzano":                      verrazzanoUpgradeTimeMetric,
		"verrazzano-monitoring-operator":  verrazzanoMonitoringOperatorUpgradeTimeMetric,
		"opensearch":                      openSearchUpgradeTimeMetric,
		"opensearch-dashboards":           openSearchDashboardsUpgradeTimeMetric,
		"grafana":                         grafanaUpgradeTimeMetric,
		"coherence-operator":              coherenceUpgradeTimeMetric,
		"mysql":                           mySQLUpgradeTimeMetric,
		"keycloak":                        keycloakUpgradeTimeMetric,
		"kiali-server":                    kialiUpgradeTimeMetric,
		"prometheus-operator":             prometheusOperatorUpgradeTimeMetric,
		"prometheus-adapter":              prometheusAdapterUpgradeTimeMetric,
		"kube-state-metrics":              kubeStateMetricsUpgradeTimeMetric,
		"prometheus-pushgateway":          prometheusPushGatewayUpgradeTimeMetric,
		"prometheus-node-exporter":        prometheusNodeExporterUpgradeTimeMetric,
		"jaeger-operator":                 jaegerOperatorUpgradeTimeMetric,
		"verrazzano-console":              verrazzanoConsoleUpgradeTimeMetric,
		"fluentd":                         fluentdUpgradeTimeMetric,
	}
	updateMetricsMap = map[string]prometheus.Gauge{
		"verrazzano-authproxy":            verrazzanoAuthproxyUpdateTimeMetric,
		"oam-kubernetes-runtime":          oamUpdateTimeMetric,
		"verrazzano-application-operator": apopperUpdateTimeMetric,
		"istio":                           istioUpdateTimeMetric,
		"weblogic-operator":               weblogicUpdateTimeMetric,
		"ingress-controller":              nginxUpdateTimeMetric,
		"cert-manager":                    certManagerUpdateTimeMetric,
		"external-dns":                    externalDNSUpdateTimeMetric,
		"rancher":                         rancherUpdateTimeMetric,
		"verrazzano":                      verrazzanoUpdateTimeMetric,
		"verrazzano-monitoring-operator":  verrazzanoMonitoringOperatorUpdateTimeMetric,
		"opensearch":                      openSearchUpdateTimeMetric,
		"opensearch-dashboards":           openSearchDashboardsUpdateTimeMetric,
		"grafana":                         grafanaUpdateTimeMetric,
		"coherence-operator":              coherenceUpdateTimeMetric,
		"mysql":                           mySQLUpdateTimeMetric,
		"keycloak":                        keycloakUpdateTimeMetric,
		"kiali-server":                    kialiUpdateTimeMetric,
		"prometheus-operator":             prometheusOperatorUpdateTimeMetric,
		"prometheus-adapter":              prometheusAdapterUpdateTimeMetric,
		"kube-state-metrics":              kubeStateMetricsUpdateTimeMetric,
		"prometheus-pushgateway":          prometheusPushGatewayUpdateTimeMetric,
		"prometheus-node-exporter":        prometheusNodeExporterUpdateTimeMetric,
		"jaeger-operator":                 jaegerOperatorUpdateTimeMetric,
		"verrazzano-console":              verrazzanoConsoleUpdateTimeMetric,
		"fluentd":                         fluentdUpdateTimeMetric,
	}
)

//InitalizeMetricsEndpoint creates and serves a /metrics endpoint at 9100 for Prometheus to scrape metrics from
func InitalizeMetricsEndpoint() {
	go registerMetricsHandlers()
	go wait.Until(func() {
		http.Handle("/metrics", promhttp.Handler())
		err := http.ListenAndServe(":9100", nil)
		if err != nil {
			zap.S().Errorf("Failed to start metrics server for verrazzano-platform-operator: %v", err)
		}
	}, time.Second*3, wait.NeverStop)
}

func CollectReconcileMetrics(startTime int64) {
	reconcileCounterMetric.Add(float64(1))
	durationTime := time.Now().UnixMilli() - startTime
	reconcileLastDurationMetric.WithLabelValues(strconv.Itoa(reconcileIndex)).Set(float64(durationTime))
	reconcileIndex = reconcileIndex + 1
}

func AnalyzeVZCR(CR vzapi.Verrazzano) {
	//Get the VZ CR Component Map (Store it in this function, so the state does not change)
	mapOfComponents := CR.Status.Components
	for componentName, componentStatusDetails := range mapOfComponents {
		latestInstallCompletionTime := ""
		latestInstallStartTime := ""
		latestUpgradeCompletionTime := ""
		latestUpgradeStartTime := ""
		latestUpdateCompletionTime := ""
		latestUpdateStartTime := ""
		possibleUpgradeStartTime := ""
		possibleUpdateStartTime := ""
		installNotHappened := true
		for _, status := range componentStatusDetails.Conditions {
			if status.Type == vzapi.CondInstallStarted && installNotHappened {
				latestInstallStartTime = status.LastTransitionTime
			}
			if status.Type == vzapi.CondInstallComplete && installNotHappened {
				latestInstallCompletionTime = status.LastTransitionTime
				installNotHappened = false
			}
			if status.Type == vzapi.CondUpgradeStarted {
				possibleUpgradeStartTime = status.LastTransitionTime
			}
			if status.Type == vzapi.CondUpgradeComplete {
				latestUpgradeCompletionTime = status.LastTransitionTime
				latestUpgradeStartTime = possibleUpgradeStartTime
			}
			if status.Type == vzapi.CondInstallStarted && !installNotHappened {
				possibleUpdateStartTime = status.LastTransitionTime
			}
			if status.Type == vzapi.CondInstallComplete && !installNotHappened {
				latestUpdateCompletionTime = status.LastTransitionTime
				latestUpdateStartTime = possibleUpdateStartTime
			}
		}
		if latestInstallCompletionTime != "" && latestInstallStartTime != "" {
			installStartInSeconds, _ := time.Parse(time.RFC3339, latestInstallStartTime)
			installStartInSecondsUnix := installStartInSeconds.Unix()
			installCompletionInSeconds, _ := time.Parse(time.RFC3339, latestInstallCompletionTime)
			installCompletionInSecondsUnix := installCompletionInSeconds.Unix()
			totalDurationOfInstall := (installCompletionInSecondsUnix - installStartInSecondsUnix)
			if _, ok := installMetricsMap[componentName]; ok {
				installMetricsMap[componentName].Set(float64(totalDurationOfInstall))
			}
		}
		if latestUpdateCompletionTime != "" && latestUpdateStartTime != "" && possibleUpdateStartTime != "" {
			updateStartInSeconds, _ := time.Parse(time.RFC3339, latestUpdateStartTime)
			updateStartInSecondsUnix := updateStartInSeconds.Unix()
			updateCompletionInSeconds, _ := time.Parse(time.RFC3339, latestUpdateCompletionTime)
			updateCompletionInSecondsUnix := updateCompletionInSeconds.Unix()
			totalDurationOfUpdate := (updateCompletionInSecondsUnix - updateStartInSecondsUnix)
			if _, ok := updateMetricsMap[componentName]; ok {
				updateMetricsMap[componentName].Set(float64(totalDurationOfUpdate))
			}
		}
		if latestUpgradeCompletionTime != "" && latestUpgradeStartTime != "" && possibleUpgradeStartTime != "" {
			upgradeStartInSeconds, _ := time.Parse(time.RFC3339, latestUpgradeStartTime)
			upgradeStartInSecondsUnix := upgradeStartInSeconds.Unix()
			upgradeCompletionInSeconds, _ := time.Parse(time.RFC3339, latestUpgradeCompletionTime)
			upgradeCompletionInSecondsUnix := upgradeCompletionInSeconds.Unix()
			totalDurationOfUpgrade := (upgradeCompletionInSecondsUnix - upgradeStartInSecondsUnix)
			if _, ok := upgradeMetricsMap[componentName]; ok {
				upgradeMetricsMap[componentName].Set(float64(totalDurationOfUpgrade))
			}
		}
	}
}
func registerMetricsHandlersHelper() error {
	var errorObserved error = nil
	for metric, i := range failedMetrics {
		err := registry.Register(metric)
		if err != nil {
			zap.S().Errorf("Failed to register metric index %v for VMI", i)
			errorObserved = err
		} else {
			delete(failedMetrics, metric)
		}
	}
	return errorObserved
}
func registerMetricsHandlers() {
	initializeFailedMetricsArray()
	for err := registerMetricsHandlersHelper(); err != nil; err = registerMetricsHandlersHelper() {
		zap.S().Errorf("Failed to register some metrics for VMI: %v", err)
		time.Sleep(time.Second)
	}
}
func initializeFailedMetricsArray() {
	for i, metric := range allMetrics {
		failedMetrics[metric] = i
	}
}
