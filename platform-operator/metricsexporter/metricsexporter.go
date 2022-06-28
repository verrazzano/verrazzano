// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package metricsexporter

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/util/wait"
)

var (
	//InstallStartTimeMap is a map that will have its keys as the component name and the time since the epoch in seconds as its value
	//It will be used to store the "true" time when a component install successfully begins
	installStartTimeMap = map[string]int64{}

	verrazzanoAuthproxyInstallTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "authproxy_component_install_time",
		Help: "The install time for the authproxy component",
	})
	oamInstallTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "oam_component_install_time",
		Help: "The install time for the oam component",
	})
	apopperInstallTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "apopper_component_install_time",
		Help: "The install time for the apopper component",
	})
	istioInstallTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "istio_component_install_time",
		Help: "The install time for the istio component",
	})
	weblogicInstallTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "weblogic_component_install_time",
		Help: "The install time for the weblogic component",
	})
	nginxInstallTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "nginx_component_install_time",
		Help: "The install time for the nginx component",
	})
	certManagerInstallTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "certManager_component_install_time",
		Help: "The install time for the certManager component",
	})
	externalDNSInstallTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "externalDNS_component_install_time",
		Help: "The install time for the externalDNS component",
	})
	rancherInstallTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "rancher_component_install_time",
		Help: "The install time for the rancher component",
	})
	verrazzanoInstallTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "verrazzano_component_install_time",
		Help: "The install time for the verrazzano component",
	})
	verrazzanoMonitoringOperatorInstallTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "verrazzano_monitoring_operator_component_install_time",
		Help: "The install time for the verrazzano-monitoring-operator component",
	})
	openSearchInstallTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "open_search_component_install_time",
		Help: "The install time for the opensearch component",
	})
	openSearchDashboardsInstallTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "open_search_dashboards_component_install_time",
		Help: "The install time for the opensearch-dashboards component",
	})
	grafanaInstallTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "grafana_component_install_time",
		Help: "The install time for the grafana component",
	})
	coherenceInstallTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "coherence_component_install_time",
		Help: "The install time for the coherence component",
	})
	mySQLInstallTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "my_sql_component_install_time",
		Help: "The install time for the mysql component",
	})
	keycloakInstallTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "keycloak_component_install_time",
		Help: "The install time for the keycloak component",
	})
	kialiInstallTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "kiali_component_install_time",
		Help: "The install time for the kiali component",
	})
	prometheusOperatorInstallTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "prometheus_operator_install_time",
		Help: "The install time for the prometheus-operator component",
	})
	prometheusAdapterInstallTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "prometheus_adapter_install_time",
		Help: "The install time for the prometheus-adapter component",
	})
	kubeStateMetricsInstallTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "kube_state_metrics_install_time",
		Help: "The install time for the kube-state-metrics component",
	})
	prometheusPushGatewayInstallTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "prometheus_push_gateway_install_time",
		Help: "The install time for the prometheus-push-gateway component",
	})
	prometheusNodeExporterInstallTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "prometheus_node_exporter_install_time",
		Help: "The install time for the prometheus-node-exporter component",
	})
	jaegerOperatorInstallTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "jaeger_operator_install_time",
		Help: "The install time for the jaeger-operator component",
	})
	verrazzanoConsoleInstallTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "verrazzano_console_install_time",
		Help: "The install time for the verrazzano-console component",
	})
	fluentdInstallTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "fluentd_install_time",
		Help: "The install time for the fluentd component",
	})

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
)

//InitalizeMetricsEndpoint creates and serves a /metrics endpoint at 9100 for Prometheus to scrape metrics from
func InitalizeMetricsEndpoint() {
	go wait.Until(func() {
		http.Handle("/metrics", promhttp.Handler())
		err := http.ListenAndServe(":9100", nil)
		if err != nil {
			zap.S().Errorf("Failed to start metrics server for verrazzano-platform-operator: %v", err)
		}
	}, time.Second*3, wait.NeverStop)
}
func AddInstallStartTime(startTime int64, componentName string) {
	installStartTimeMap[componentName] = startTime
}
func CollectInstallTimeMetric(componentName string) {
	endTime := time.Now().UnixMilli()
	totalInstallTime := float64((endTime - installStartTimeMap[componentName])) / 1000.0
	installMetricsMap[componentName].Set(totalInstallTime)

}
func CheckIfAlreadyBeingMonitored(componentName string) bool {
	_, monitored := installStartTimeMap[componentName]
	return monitored
}
