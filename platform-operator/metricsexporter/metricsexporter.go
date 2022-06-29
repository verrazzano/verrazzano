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
	upgradeStartTimeMap = map[string]int64{}
	updateStartTimeMap  = map[string]int64{}

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
	verrazzanoAuthproxyUpgradeTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "authproxy_component_upgrade_time",
		Help: "The upgrade time for the authproxy component",
	})
	oamUpgradeTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "oam_component_upgrade_time",
		Help: "The upgrade time for the oam component",
	})
	apopperUpgradeTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "apopper_component_upgrade_time",
		Help: "The upgrade time for the apopper component",
	})
	istioUpgradeTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "istio_component_upgrade_time",
		Help: "The upgrade time for the istio component",
	})
	weblogicUpgradeTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "weblogic_component_upgrade_time",
		Help: "The upgrade time for the weblogic component",
	})
	nginxUpgradeTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "nginx_component_upgrade_time",
		Help: "The upgrade time for the nginx component",
	})
	certManagerUpgradeTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "certManager_component_upgrade_time",
		Help: "The upgrade time for the certManager component",
	})
	externalDNSUpgradeTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "externalDNS_component_upgrade_time",
		Help: "The upgrade time for the externalDNS component",
	})
	rancherUpgradeTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "rancher_component_upgrade_time",
		Help: "The upgrade time for the rancher component",
	})
	verrazzanoUpgradeTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "verrazzano_component_upgrade_time",
		Help: "The upgrade time for the verrazzano component",
	})
	verrazzanoMonitoringOperatorUpgradeTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "verrazzano_monitoring_operator_component_upgrade_time",
		Help: "The upgrade time for the verrazzano-monitoring-operator component",
	})
	openSearchUpgradeTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "open_search_component_upgrade_time",
		Help: "The upgrade time for the opensearch component",
	})
	openSearchDashboardsUpgradeTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "open_search_dashboards_component_upgrade_time",
		Help: "The upgrade time for the opensearch-dashboards component",
	})
	grafanaUpgradeTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "grafana_component_upgrade_time",
		Help: "The upgrade time for the grafana component",
	})
	coherenceUpgradeTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "coherence_component_upgrade_time",
		Help: "The upgrade time for the coherence component",
	})
	mySQLUpgradeTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "my_sql_component_upgrade_time",
		Help: "The upgrade time for the mysql component",
	})
	keycloakUpgradeTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "keycloak_component_upgrade_time",
		Help: "The upgrade time for the keycloak component",
	})
	kialiUpgradeTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "kiali_component_upgrade_time",
		Help: "The upgrade time for the upgrade component",
	})
	prometheusOperatorUpgradeTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "prometheus_operator_upgrade_time",
		Help: "The upgrade time for the prometheus-operator component",
	})
	prometheusAdapterUpgradeTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "prometheus_adapter_upgrade_time",
		Help: "The upgrade time for the prometheus-adapter component",
	})
	kubeStateMetricsUpgradeTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "kube_state_metrics_upgrade_time",
		Help: "The upgrade time for the kube-state-metrics component",
	})
	prometheusPushGatewayUpgradeTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "prometheus_push_gateway_upgrade_time",
		Help: "The upgrade time for the prometheus-push-gateway component",
	})
	prometheusNodeExporterUpgradeTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "prometheus_node_exporter_upgrade_time",
		Help: "The upgrade time for the prometheus-node-exporter component",
	})
	jaegerOperatorUpgradeTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "jaeger_operator_upgrade_time",
		Help: "The upgrade time for the jaeger-operator component",
	})
	verrazzanoConsoleUpgradeTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "verrazzano_console_upgrade_time",
		Help: "The upgrade time for the verrazzano-console component",
	})
	fluentdUpgradeTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "fluentd_upgrade_time",
		Help: "The upgrade time for the fluentd component",
	})
	testingUpgradeTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "test_component_upgrade_time",
		Help: "The upgrade time for the fake component",
	})
	//Ask about duplicate metric multiple objects most likely
	enabledTestingUpgradeTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "enabled_test_component_upgrade_time",
		Help: "The upgrade time for the fake component",
	})
	disabledTestingUpgradeTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "disabled_test_component_upgrade_time",
		Help: "The upgrade time for the fake component",
	})
	verrazzanoAuthproxyUpdateTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "authproxy_component_update_time",
		Help: "The update time for the authproxy component",
	})
	oamUpdateTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "oam_component_update_time",
		Help: "The update time for the oam component",
	})
	apopperUpdateTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "apopper_component_update_time",
		Help: "The update time for the apopper component",
	})
	istioUpdateTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "istio_component_update_time",
		Help: "The update time for the istio component",
	})
	weblogicUpdateTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "weblogic_component_update_time",
		Help: "The update time for the weblogic component",
	})
	nginxUpdateTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "nginx_component_update_time",
		Help: "The update time for the nginx component",
	})
	certManagerUpdateTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "certManager_component_update_time",
		Help: "The update time for the certManager component",
	})
	externalDNSUpdateTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "externalDNS_component_update_time",
		Help: "The update time for the externalDNS component",
	})
	rancherUpdateTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "rancher_component_update_time",
		Help: "The update time for the rancher component",
	})
	verrazzanoUpdateTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "verrazzano_component_update_time",
		Help: "The update time for the verrazzano component",
	})
	verrazzanoMonitoringOperatorUpdateTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "verrazzano_monitoring_operator_component_update_time",
		Help: "The update time for the verrazzano-monitoring-operator component",
	})
	openSearchUpdateTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "open_search_component_update_time",
		Help: "The update time for the opensearch component",
	})
	openSearchDashboardsUpdateTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "open_search_dashboards_component_update_time",
		Help: "The update time for the opensearch-dashboards component",
	})
	grafanaUpdateTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "grafana_component_update_time",
		Help: "The update time for the grafana component",
	})
	coherenceUpdateTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "coherence_component_update_time",
		Help: "The update time for the coherence component",
	})
	mySQLUpdateTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "my_sql_component_update_time",
		Help: "The update time for the mysql component",
	})
	keycloakUpdateTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "keycloak_component_update_time",
		Help: "The update time for the keycloak component",
	})
	kialiUpdateTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "kiali_component_update_time",
		Help: "The update time for the kiali component",
	})
	prometheusOperatorUpdateTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "prometheus_operator_update_time",
		Help: "The update time for the prometheus-operator component",
	})
	prometheusAdapterUpdateTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "prometheus_adapter_update_time",
		Help: "The update time for the prometheus-adapter component",
	})
	kubeStateMetricsUpdateTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "kube_state_metrics_update_time",
		Help: "The update time for the kube-state-metrics component",
	})
	prometheusPushGatewayUpdateTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "prometheus_push_gateway_update_time",
		Help: "The update time for the prometheus-push-gateway component",
	})
	prometheusNodeExporterUpdateTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "prometheus_node_exporter_update_time",
		Help: "The update time for the prometheus-node-exporter component",
	})
	jaegerOperatorUpdateTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "jaeger_operator_update_time",
		Help: "The update time for the jaeger-operator component",
	})
	verrazzanoConsoleUpdateTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "verrazzano_console_update_time",
		Help: "The update time for the verrazzano-console component",
	})
	fluentdUpdateTimeMetric = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "fluentd_update_time",
		Help: "The update time for the fluentd component",
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
	installDesiredNumberMap = map[string]int{
		"verrazzano-authproxy":            0,
		"oam-kubernetes-runtime":          0,
		"verrazzano-application-operator": 0,
		"istio":                           0,
		"weblogic-operator":               0,
		"ingress-controller":              0,
		"cert-manager":                    0,
		"external-dns":                    0,
		"rancher":                         0,
		"verrazzano":                      0,
		"verrazzano-monitoring-operator":  0,
		"opensearch":                      0,
		"opensearch-dashboards":           0,
		"grafana":                         0,
		"coherence-operator":              0,
		"mysql":                           0,
		"keycloak":                        0,
		"kiali-server":                    0,
		"prometheus-operator":             0,
		"prometheus-adapter":              0,
		"kube-state-metrics":              0,
		"prometheus-pushgateway":          0,
		"prometheus-node-exporter":        0,
		"jaeger-operator":                 0,
		"verrazzano-console":              0,
		"fluentd":                         0,
	}
	installCompletedNumberMap = map[string]int{
		"verrazzano-authproxy":            0,
		"oam-kubernetes-runtime":          0,
		"verrazzano-application-operator": 0,
		"istio":                           0,
		"weblogic-operator":               0,
		"ingress-controller":              0,
		"cert-manager":                    0,
		"external-dns":                    0,
		"rancher":                         0,
		"verrazzano":                      0,
		"verrazzano-monitoring-operator":  0,
		"opensearch":                      0,
		"opensearch-dashboards":           0,
		"grafana":                         0,
		"coherence-operator":              0,
		"mysql":                           0,
		"keycloak":                        0,
		"kiali-server":                    0,
		"prometheus-operator":             0,
		"prometheus-adapter":              0,
		"kube-state-metrics":              0,
		"prometheus-pushgateway":          0,
		"prometheus-node-exporter":        0,
		"jaeger-operator":                 0,
		"verrazzano-console":              0,
		"fluentd":                         0,
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
		"":                                testingUpgradeTimeMetric,
		"EnabledComponent":                enabledTestingUpgradeTimeMetric,
		"DisabledComponent":               disabledTestingUpgradeTimeMetric,
	}
	upgradeDesiredNumberMap = map[string]int{
		"verrazzano-authproxy":            0,
		"oam-kubernetes-runtime":          0,
		"verrazzano-application-operator": 0,
		"istio":                           0,
		"weblogic-operator":               0,
		"ingress-controller":              0,
		"cert-manager":                    0,
		"external-dns":                    0,
		"rancher":                         0,
		"verrazzano":                      0,
		"verrazzano-monitoring-operator":  0,
		"opensearch":                      0,
		"opensearch-dashboards":           0,
		"grafana":                         0,
		"coherence-operator":              0,
		"mysql":                           0,
		"keycloak":                        0,
		"kiali-server":                    0,
		"prometheus-operator":             0,
		"prometheus-adapter":              0,
		"kube-state-metrics":              0,
		"prometheus-pushgateway":          0,
		"prometheus-node-exporter":        0,
		"jaeger-operator":                 0,
		"verrazzano-console":              0,
		"fluentd":                         0,
		"":                                0,
		"EnabledComponent":                0,
		"DisabledComponent":               0,
	}
	upgradeCompletedNumberMap = map[string]int{
		"verrazzano-authproxy":            0,
		"oam-kubernetes-runtime":          0,
		"verrazzano-application-operator": 0,
		"istio":                           0,
		"weblogic-operator":               0,
		"ingress-controller":              0,
		"cert-manager":                    0,
		"external-dns":                    0,
		"rancher":                         0,
		"verrazzano":                      0,
		"verrazzano-monitoring-operator":  0,
		"opensearch":                      0,
		"opensearch-dashboards":           0,
		"grafana":                         0,
		"coherence-operator":              0,
		"mysql":                           0,
		"keycloak":                        0,
		"kiali-server":                    0,
		"prometheus-operator":             0,
		"prometheus-adapter":              0,
		"kube-state-metrics":              0,
		"prometheus-pushgateway":          0,
		"prometheus-node-exporter":        0,
		"jaeger-operator":                 0,
		"verrazzano-console":              0,
		"fluentd":                         0,
		"":                                0,
		"EnabledComponent":                0,
		"DisabledComponent":               0,
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
	updateDesiredNumberMap = map[string]int{
		"verrazzano-authproxy":            0,
		"oam-kubernetes-runtime":          0,
		"verrazzano-application-operator": 0,
		"istio":                           0,
		"weblogic-operator":               0,
		"ingress-controller":              0,
		"cert-manager":                    0,
		"external-dns":                    0,
		"rancher":                         0,
		"verrazzano":                      0,
		"verrazzano-monitoring-operator":  0,
		"opensearch":                      0,
		"opensearch-dashboards":           0,
		"grafana":                         0,
		"coherence-operator":              0,
		"mysql":                           0,
		"keycloak":                        0,
		"kiali-server":                    0,
		"prometheus-operator":             0,
		"prometheus-adapter":              0,
		"kube-state-metrics":              0,
		"prometheus-pushgateway":          0,
		"prometheus-node-exporter":        0,
		"jaeger-operator":                 0,
		"verrazzano-console":              0,
		"fluentd":                         0,
	}
	updateCompletedNumberMap = map[string]int{
		"verrazzano-authproxy":            0,
		"oam-kubernetes-runtime":          0,
		"verrazzano-application-operator": 0,
		"istio":                           0,
		"weblogic-operator":               0,
		"ingress-controller":              0,
		"cert-manager":                    0,
		"external-dns":                    0,
		"rancher":                         0,
		"verrazzano":                      0,
		"verrazzano-monitoring-operator":  0,
		"opensearch":                      0,
		"opensearch-dashboards":           0,
		"grafana":                         0,
		"coherence-operator":              0,
		"mysql":                           0,
		"keycloak":                        0,
		"kiali-server":                    0,
		"prometheus-operator":             0,
		"prometheus-adapter":              0,
		"kube-state-metrics":              0,
		"prometheus-pushgateway":          0,
		"prometheus-node-exporter":        0,
		"jaeger-operator":                 0,
		"verrazzano-console":              0,
		"fluentd":                         0,
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
func AddStartTime(startTime int64, componentName string, metric string) {
	if metric == "install" {
		installStartTimeMap[componentName] = startTime
	}
	if metric == "upgrade" {
		upgradeStartTimeMap[componentName] = startTime
	}
	if metric == "update" {
		updateStartTimeMap[componentName] = startTime
	}
}
func CollectTimeMetric(componentName string, metric string) {
	endTime := time.Now().UnixNano()
	if metric == "install" {
		totalInstallTime := float64((endTime - installStartTimeMap[componentName])) / 1000000000.0
		installMetricsMap[componentName].Set(totalInstallTime)
		installCompletedNumberMap[componentName] = installCompletedNumberMap[componentName] + 1
	}
	if metric == "upgrade" {
		totalUpgradeTime := float64((endTime - upgradeStartTimeMap[componentName])) / 1000000000.0
		upgradeMetricsMap[componentName].Set(totalUpgradeTime)
		upgradeCompletedNumberMap[componentName] = upgradeCompletedNumberMap[componentName] + 1
	}
	if metric == "update" {
		totalUpdateTime := float64((endTime - updateStartTimeMap[componentName])) / 1000000000.0
		updateMetricsMap[componentName].Set(totalUpdateTime)
		updateCompletedNumberMap[componentName] = updateCompletedNumberMap[componentName] + 1
	}
}
func CheckIfNewOperationHasToBegin(componentName string, metric string) bool {
	if metric == "update" {
		if updateDesiredNumberMap[componentName] == updateCompletedNumberMap[componentName] {
			updateDesiredNumberMap[componentName] = updateDesiredNumberMap[componentName] + 1
			return true
		}
	}
	if metric == "install" {
		if installDesiredNumberMap[componentName] == updateCompletedNumberMap[componentName] {
			installDesiredNumberMap[componentName] = installDesiredNumberMap[componentName] + 1
			return true
		}
	}
	if metric == "upgrade" {
		if upgradeDesiredNumberMap[componentName] == upgradeCompletedNumberMap[componentName] {
			upgradeDesiredNumberMap[componentName] = upgradeDesiredNumberMap[componentName] + 1
			return true
		}
	}
	return false
}
