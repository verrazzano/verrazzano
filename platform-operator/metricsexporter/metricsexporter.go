// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricsexporter

import (
	"fmt"
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
		Name: "vpo_reconcile_counter",
		Help: "The number of times the reconcile function has been called in the Verrazzano-platform-operator",
	})
	reconcileLastDurationMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "vpo_reconcile_duration_seconds",
		Help: "The duration of each reconcile call in the Verrazzano-platform-operator in seconds"},
		[]string{"reconcile_index"},
	)
	verrazzanoAuthproxyInstallTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_authproxy_install_duration_seconds",
		Help: "The duration of the latest installation of the authproxy component in seconds",
	})
	oamInstallTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_oam_install_duration_seconds",
		Help: "The duration of the latest installation of the oam component in seconds",
	})
	appoperInstallTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_appoper_install_duration_seconds",
		Help: "The duration of the latest installation of the appoper component in seconds",
	})
	istioInstallTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_istio_install_duration_seconds",
		Help: "The duration of the latest installation of the istio component in seconds",
	})
	weblogicInstallTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_weblogic_install_duration_seconds",
		Help: "The duration of the latest installation of the weblogic component in seconds",
	})
	nginxInstallTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_nginx_install_duration_seconds",
		Help: "The duration of the latest installation of the nginx component in seconds",
	})
	certManagerInstallTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_certManager_install_duration_seconds",
		Help: "The duration of the latest installation of the certManager component in seconds",
	})
	externalDNSInstallTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_externalDNS_install_duration_seconds",
		Help: "The duration of the latest installation of the externalDNS component in seconds",
	})
	rancherInstallTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_rancher_install_duration_seconds",
		Help: "The duration of the latest installation of the rancher component in seconds",
	})
	verrazzanoInstallTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "verrazzano_install_duration_seconds",
		Help: "The duration of the latest installation of the verrazzano component in seconds",
	})
	verrazzanoMonitoringOperatorInstallTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vmo_install_duration_seconds",
		Help: "The duration of the latest installation of the verrazzano-monitoring-operator component in seconds",
	})
	openSearchInstallTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_open_search_install_duration_seconds",
		Help: "The duration of the latest installation of the opensearch component in seconds",
	})
	openSearchDashboardsInstallTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_open_search_dashboards_install_duration_seconds",
		Help: "The duration of the latest installation of the opensearch-dashboards component in seconds",
	})
	grafanaInstallTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_grafana_install_duration_seconds",
		Help: "The duration of the latest installation of the grafana component in seconds",
	})
	coherenceInstallTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_coherence_install_duration_seconds",
		Help: "The duration of the latest installation of the coherence component in seconds",
	})
	mySQLInstallTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_my_sql_install_duration_seconds",
		Help: "The duration of the latest installatio of nthe mysql component in seconds",
	})
	keycloakInstallTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_keycloak_install_duration_seconds",
		Help: "The duration of the latest installation of the keycloak component in seconds",
	})
	kialiInstallTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_kiali_install_duration_seconds",
		Help: "The duration of the latest installation of the kiali component in seconds",
	})
	prometheusOperatorInstallTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_prometheus_operator_install_duration_seconds",
		Help: "The duration of the latest installation of the prometheus-operator component in seconds",
	})
	prometheusAdapterInstallTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_prometheus_adapter_latest_duration_seconds",
		Help: "The duration of the latest installation of the prometheus-adapter component in seconds",
	})
	kubeStateMetricsInstallTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_kube_state_metrics_install_duration_seconds",
		Help: "The duration of the latest installation of the kube-state-metrics component in seconds",
	})
	prometheusPushGatewayInstallTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_prometheus_push_gateway_install_duration_seconds",
		Help: "The duration of the latest installation of the prometheus-push-gateway component in seconds",
	})
	prometheusNodeExporterInstallTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_prometheus_node_exporter_install_duration_seconds",
		Help: "The duration of the latest installation of the prometheus-node-exporter component in seconds",
	})
	jaegerOperatorInstallTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_jaeger_operator_install_duration_seconds",
		Help: "The duration of the latest installation of the jaeger-operator component in seconds",
	})
	verrazzanoConsoleInstallTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "verrazzano_console_install_duration_seconds",
		Help: "The duration of the latest installation of the verrazzano-console component in seconds",
	})
	fluentdInstallTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_fluentd_install_duration_secinds",
		Help: "The duration of the latest installation of the fluentd component in seconds",
	})
	verrazzanoAuthproxyUpgradeTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_authproxy_upgrade_duration_seconds",
		Help: "The duration of the latest upgrade of the authproxy component in seconds",
	})
	oamUpgradeTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_oam_upgrade_duration_seconds",
		Help: "The duration of the latest upgrade of the oam component in seconds",
	})
	appoperUpgradeTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_appoper_upgrade_duration_seconds",
		Help: "The duration of the latest upgrade of the appoper component in seconds",
	})
	istioUpgradeTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_istio_upgrade_duration_seconds",
		Help: "The duration of the latest upgrade of the istio component in seconds",
	})
	weblogicUpgradeTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_weblogic_upgrade_duration_seconds",
		Help: "The duration of the latest upgrade of the weblogic component in seconds",
	})
	nginxUpgradeTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_nginx_upgrade_duration_seconds",
		Help: "The duration of the latest upgrade of the nginx component in seconds",
	})
	certManagerUpgradeTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_certManager_upgrade_duration_seconds",
		Help: "The duration of the latest upgrade of the certManager component in seconds",
	})
	externalDNSUpgradeTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_externalDNS_upgrade_duration_seconds",
		Help: "The duration of the latest upgrade of the externalDNS component in seconds",
	})
	rancherUpgradeTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_rancher_upgrade_duration_seconds",
		Help: "The duration of the latest upgrade of the rancher component in seconds",
	})
	verrazzanoUpgradeTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "verrazzano_upgrade_duration_seconds",
		Help: "The duration of the latest upgrade of the verrazzano component in seconds",
	})
	verrazzanoMonitoringOperatorUpgradeTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vmo_upgrade_duration_seconds",
		Help: "The duration of the latest upgrade of the verrazzano-monitoring-operator component in seconds",
	})
	openSearchUpgradeTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_open_search_upgrade_duration_seconds",
		Help: "The duration of the latest upgrade of the opensearch component in seconds",
	})
	openSearchDashboardsUpgradeTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_open_search_dashboards_upgrade_duration_seconds",
		Help: "The duration of the latest upgrade of the opensearch-dashboards component in seconds",
	})
	grafanaUpgradeTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_grafana_upgrade_duration_seconds",
		Help: "The duration of the latest upgrade of the grafana component in seconds",
	})
	coherenceUpgradeTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_coherence_upgrade_duration_seconds",
		Help: "The duration of the latest upgrade of the coherence component in seconds",
	})
	mySQLUpgradeTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_my_sql_upgrade_duration_seconds",
		Help: "The duration of the latest upgrade of the mysql component in seconds",
	})
	keycloakUpgradeTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_keycloak_upgrade_duration_seconds",
		Help: "The duration of the latest upgrade of the keycloak component in seconds",
	})
	kialiUpgradeTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_kiali_upgrade_duration_seconds",
		Help: "The duration of the latest upgrade of the upgrade component in seconds",
	})
	prometheusOperatorUpgradeTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_prometheus_operator_upgrade_duration_seconds",
		Help: "The duration of the latest upgrade of the prometheus-operator component in seconds",
	})
	prometheusAdapterUpgradeTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_prometheus_adapter_upgrade_duration_seconds",
		Help: "The duration of the latest upgrade of the prometheus-adapter component in seconds",
	})
	kubeStateMetricsUpgradeTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_kube_state_metrics_upgrade_duration_seconds",
		Help: "The duration of the latest upgrade of the kube-state-metrics component in seconds",
	})
	prometheusPushGatewayUpgradeTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_prometheus_push_gateway_upgrade_duration_seconds",
		Help: "The duration of the latest upgrade of the prometheus-push-gateway component in seconds",
	})
	prometheusNodeExporterUpgradeTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_prometheus_node_exporter_upgrade_duration_seconds",
		Help: "The duration of the latest upgrade of the prometheus-node-exporter component in seconds",
	})
	jaegerOperatorUpgradeTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_jaeger_operator_upgrade_duration_seconds",
		Help: "The duration of the latest upgrade of the jaeger-operator component in seconds",
	})
	verrazzanoConsoleUpgradeTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "verrazzano_console_upgrade_duration_seconds",
		Help: "The duration of the latest upgrade of the verrazzano-console component in seconds",
	})
	fluentdUpgradeTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_fluentd_upgrade_duration_seconds",
		Help: "The duration of the latest upgrade of the fluentd component in seconds",
	})
	verrazzanoAuthproxyUpdateTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_authproxy_update_duration_seconds",
		Help: "The duration of the latest update of the authproxy component in seconds",
	})
	oamUpdateTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_oam_update_duration_seconds",
		Help: "The duration of the latest update of the oam component in seconds",
	})
	appoperUpdateTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_appoper_update_duration_seconds",
		Help: "The duration of the latest update of the appoper component in seconds",
	})
	istioUpdateTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_istio_update_duration_seconds",
		Help: "The duration of the latest update of the istio component in seconds",
	})
	weblogicUpdateTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_weblogic_update_duration_seconds",
		Help: "The duration of the latest update of the weblogic component in seconds",
	})
	nginxUpdateTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_nginx_update_duration_seconds",
		Help: "The duration of the latest update of the nginx component in seconds",
	})
	certManagerUpdateTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_certManager_update_duration_seconds",
		Help: "The duration of the latest update of the certManager component in seconds",
	})
	externalDNSUpdateTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_externalDNS_update_duration_seconds",
		Help: "The duration of the latest update of the externalDNS component in seconds",
	})
	rancherUpdateTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_rancher_update_duration_seconds",
		Help: "The duration of the latest update of the rancher component in seconds",
	})
	verrazzanoUpdateTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "verrazzano_update_duration_seconds",
		Help: "The duration of the latest update of the verrazzano component in seconds",
	})
	verrazzanoMonitoringOperatorUpdateTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vmo_update_duration_seconds",
		Help: "The duration of the latest update of the verrazzano-monitoring-operator component in seconds",
	})
	openSearchUpdateTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_open_search_update_duration_seconds",
		Help: "The duration of the latest update of the opensearch component in seconds",
	})
	openSearchDashboardsUpdateTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_open_search_dashboards_update_duration_seconds",
		Help: "The duration of the latest update of the opensearch-dashboards component in seconds",
	})
	grafanaUpdateTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_grafana_update_duration_seconds",
		Help: "The duration of the latest update of the grafana component in seconds",
	})
	coherenceUpdateTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_coherence_update_duration_seconds",
		Help: "The duration of the latest update of the coherence component in seconds",
	})
	mySQLUpdateTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_my_sql_update_duration_seconds",
		Help: "The duration of the latest update of the mysql component in seconds",
	})
	keycloakUpdateTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_keycloak_update_duration_seconds",
		Help: "The duration of the latest update of the keycloak component in seconds",
	})
	kialiUpdateTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_kiali_update_duration_seconds",
		Help: "The duration of the latest update of the kiali component in seconds",
	})
	prometheusOperatorUpdateTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_prometheus_operator_update_duration_seconds",
		Help: "The duration of the latest update of the prometheus-operator component in seconds",
	})
	prometheusAdapterUpdateTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_prometheus_adapter_update_duration_seconds",
		Help: "The duration of the latest update of the prometheus-adapter component in seconds",
	})
	kubeStateMetricsUpdateTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_kube_state_metrics_update_duration_seconds",
		Help: "The duration of the latest update of the kube-state-metrics component in seconds",
	})
	prometheusPushGatewayUpdateTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_prometheus_push_gateway_update_duration_seconds",
		Help: "The duration of the latest update of the prometheus-push-gateway component in seconds",
	})
	prometheusNodeExporterUpdateTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_prometheus_node_exporter_update_duration_seconds",
		Help: "The duration of the latest update of the prometheus-node-exporter component in seconds",
	})
	jaegerOperatorUpdateTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_jaeger_operator_update_duration_seconds",
		Help: "The duration of the latest update of the jaeger-operator component in seconds",
	})
	verrazzanoConsoleUpdateTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "verrazzano_console_update_duration_seconds",
		Help: "The duration of the latest update of the verrazzano-console component in seconds",
	})
	fluentdUpdateTimeMetric = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "vz_fluentd_update_duration_seconds",
		Help: "The duration of the latest update of the fluentd component in seconds",
	})
	allMetrics = []prometheus.Collector{verrazzanoAuthproxyInstallTimeMetric,
		oamInstallTimeMetric,
		appoperInstallTimeMetric,
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
		appoperUpgradeTimeMetric,
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
		appoperUpdateTimeMetric,
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
		"verrazzano-application-operator": appoperInstallTimeMetric,
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
		"verrazzano-application-operator": appoperUpgradeTimeMetric,
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
		"verrazzano-application-operator": appoperUpdateTimeMetric,
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
	durationTime := (float64(time.Now().UnixMilli() - startTime)) / 1000.0
	fmt.Println(durationTime)
	reconcileLastDurationMetric.WithLabelValues(strconv.Itoa(reconcileIndex)).Set(float64(durationTime))
	reconcileIndex = reconcileIndex + 1
}

//Reminder to take out this comment block later
//Change this function to get the times for the upcoming update Condition Type
//For each component go through the list
//Can update current code to get rid of installnothappened and for update just change the condition to its repsective update condition
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
			zap.S().Errorf("Failed to register metric index %v for VPO", i)
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
