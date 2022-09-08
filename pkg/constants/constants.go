// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package constants

import (
	"time"

	platformOperatorConstants "github.com/verrazzano/verrazzano/platform-operator/constants"
)

// RestartVersionAnnotation - the annotation used by user to tell Verrazzano applicaton to restart its components
const RestartVersionAnnotation = "verrazzano.io/restart-version"

// VerrazzanoRestartAnnotation is the annotation used to restart platform workloads
const VerrazzanoRestartAnnotation = "verrazzano.io/restartedAt"

// LifecycleActionAnnotation - the annotation perform lifecycle actions on a workload
const LifecycleActionAnnotation = "verrazzano.io/lifecycle-action"

// LifecycleActionStop - the annotation value used to stop a workload
const LifecycleActionStop = "stop"

// LifecycleActionStart - the annotation value used to start a workload
const LifecycleActionStart = "start"

// VerrazzanoWebLogicWorkloadKind - the VerrazzanoWebLogicWorkload resource kind
const VerrazzanoWebLogicWorkloadKind = "VerrazzanoWebLogicWorkload"

// VerrazzanoCoherenceWorkloadKind - the VerrazzanoCoherenceWorkload resource kind
const VerrazzanoCoherenceWorkloadKind = "VerrazzanoCoherenceWorkload"

// VerrazzanoHelidonWorkloadKind - the VerrazzanoHelidonWorkload resource kind
const VerrazzanoHelidonWorkloadKind = "VerrazzanoHelidonWorkload"

// ContainerizedWorkloadKind - the ContainerizedWorkload resource kind
const ContainerizedWorkloadKind = "ContainerizedWorkload"

// DeploymentWorkloadKind - the Deployment workload resource kind
const DeploymentWorkloadKind = "Deployment"

// StatefulSetWorkloadKind - the StatefulSet workload resource kind
const StatefulSetWorkloadKind = "StatefulSet"

// DaemonSetWorkloadKind - the DaemonSet workload resource kind
const DaemonSetWorkloadKind = "DaemonSet"

// VerrazzanoInstallNamespace is the namespace for installing the verrazzano-platform-operator
const VerrazzanoInstallNamespace = "verrazzano-install"

// VerrazzanoSystemNamespace is the system namespace for Verrazzano
const VerrazzanoSystemNamespace = "verrazzano-system"

// VerrazzanoMultiClusterNamespace is the multi-cluster namespace for Verrazzano
const VerrazzanoMultiClusterNamespace = "verrazzano-mc"

// CertManagerNamespace - the CertManager namespace
const CertManagerNamespace = "cert-manager"

// KeycloakNamespace - the keycloak namespace
const KeycloakNamespace = "keycloak"

// MySQLOperatorNamespace indicates the namespace to be used for the MySQLOperator installation
const MySQLOperatorNamespace = "mysql-operator"

// RancherSystemNamespace - the Rancher cattle-system namespace
const RancherSystemNamespace = "cattle-system"

// IstioSystemNamespace - the Istio system namespace
const IstioSystemNamespace = "istio-system"

// IngressNamespace - the NGINX ingress namespace
const IngressNamespace = "ingress-nginx"

// PrometheusOperatorNamespace - the namespace where Verrazzano installs Prometheus Operator
// and its related components.
const PrometheusOperatorNamespace = "verrazzano-monitoring"

// LabelIstioInjection - constant for a Kubernetes label that is applied by Verrazzano
const LabelIstioInjection = "istio-injection"

// LabelVerrazzanoNamespace - constant for a Kubernetes label that is used by network policies
const LabelVerrazzanoNamespace = "verrazzano.io/namespace"

// LegacyElasticsearchSecretName legacy secret name for Elasticsearch credentials
const LegacyElasticsearchSecretName = "verrazzano"

// VerrazzanoESInternal is the name of the Verrazzano internal Elasticsearch secret in the Verrazzano system namespace
const VerrazzanoESInternal = "verrazzano-es-internal"

// VerrazzanoPromInternal is the name of the Verrazzano internal Prometheus secret in the Verrazzano system namespace
const VerrazzanoPromInternal = "verrazzano-prom-internal"

// AdditionalTLS is an optional tls secret that contains additional CA
const AdditionalTLS = "tls-ca-additional"

// AdditionalTLSCAKey is the key containing the CA in the secret specified by the AdditionalTLS constant
const AdditionalTLSCAKey = "ca-additional.pem"

// VMCAgentPollingTimeInterval - The time interval at which mcagent polls Verrazzano Managed CLuster resource on the admin cluster.
const VMCAgentPollingTimeInterval = 60 * time.Second

// MaxTimesVMCAgentPollingTime - The constant used to set max polling time for vmc agent to determine VMC state
const MaxTimesVMCAgentPollingTime = 3

// FluentdDaemonSetName - The name of the Fluentd DaemonSet
const FluentdDaemonSetName = "fluentd"

// KubeSystem - The name of the kube-system namespace
const KubeSystem = "kube-system"

// DefaultVerrazzanoCASecretName Default self-signed CA secret name
// #nosec
const DefaultVerrazzanoCASecretName = "verrazzano-ca-certificate-secret"

// VmiPromConfigName - The name of the prometheus config map
const VmiPromConfigName string = "vmi-system-prometheus-config"

const PrometheusJobNameKey = "job_name"

// TestPrometheusJobScrapeInterval - The string 0s representing a test only prometheus config scrape interval
const TestPrometheusJobScrapeInterval = "0s"

// TestPrometheusJob - Name of a test prometheus scraper job
const TestPrometheusScrapeJob = "test_job"

// Default OpenSearch URL
const DefaultOpensearchURL = "http://verrazzano-authproxy-elasticsearch:8775"

// Default Jaeger OpenSearch URL
const DefaultJaegerOSURL = "http://verrazzano-authproxy-elasticsearch.verrazzano-system:8775"

// DefaultJaegerSecretName is the Jaeger secret name used by the default Jaeger instance
// #nosec
const DefaultJaegerSecretName = "verrazzano-jaeger-secret"

// JaegerInstanceName is the name of the default Jaeger instance
const JaegerInstanceName = "jaeger-operator-jaeger"

// JaegerQueryComponentName is the name of the collector component
const JaegerCollectorComponentName = "collector"

// JaegerQueryComponentName is the name of the collector component
const JaegerQueryComponentName = "query"

// VerrazzanoManagedLabelKey is a constant for a Kubernetes label that is applied to Verrazzano application namespaces
const VerrazzanoManagedLabelKey = "verrazzano-managed"

// PromAdditionalScrapeConfigsSecretName is the name of the secret that contains the additional scrape configurations loaded by Prometheus
const PromAdditionalScrapeConfigsSecretName = "additional-scrape-configs"

// PromAdditionalScrapeConfigsSecretKey is the name of the key in the additional scrape configurations secret that contains the scrape config YAML
const PromAdditionalScrapeConfigsSecretKey = "jobs"

// MetricsTemplateKind is the Kind of the MetricsTemplate custom resource
const MetricsTemplateKind = "MetricsTemplate"

// MetricsTemplateAPIVersion is the APIVersion of the MetricsTemplate custom resource
const MetricsTemplateAPIVersion = "app.verrazzano.io/v1alpha1"

// SecretKind is the kind for a secret
const SecretKind = "Secret"

// Components Names
const (
	OamKubernetesRuntime          = "oam-kubernetes-runtime"
	KialiServer                   = "kiali-server"
	WeblogicOperator              = "weblogic-operator"
	VerrazzanoAuthproxy           = "verrazzano-authproxy"
	Istio                         = "istio"
	ExternalDNS                   = "external-dns"
	VerrazzanoApplicationOperator = "verrazzano-application-operator"
	CoherenceOperator             = "coherence-operator"
	IngressController             = "ingress-controller"
	IngressDefaultBackend         = "ingress-controller-ingress-nginx-defaultbackend"
	MySQL                         = "mysql"
	CertManager                   = "cert-manager"
	Rancher                       = "rancher"
	PrometheusPushgateway         = "prometheus-pushgateway"
	PrometheusAdapter             = "prometheus-adapter"
	KubeStateMetrics              = "kube-state-metrics"
	PrometheusNodeExporter        = "prometheus-node-exporter"
	PrometheusOperator            = "prometheus-operator"
	Keycloak                      = "keycloak"
	VerrazzanoMonitoringOperator  = "verrazzano-monitoring-operator"
	Grafana                       = "grafana"
	JaegerOperator                = "jaeger-operator"
	OpensearchDashboards          = "opensearch-dashboards"
	Opensearch                    = "opensearch"
	Velero                        = "velero"
	VerrazzanoConsole             = "verrazzano-console"
	Verrazzano                    = "verrazzano"
	Fluentd                       = "fluentd"
	RancherBackup                 = "rancher-backup"
	MySQLOperator                 = "mysql-operator"
)

const (
	RancherFleetSystemNamespace      = "cattle-fleet-system"
	RancherFleetLocalSystemNamespace = "cattle-fleet-local-system"
)

var ComponentNameToNamespacesMap = map[string][]string{
	OamKubernetesRuntime:          {VerrazzanoSystemNamespace},
	KialiServer:                   {VerrazzanoSystemNamespace},
	WeblogicOperator:              {VerrazzanoSystemNamespace},
	VerrazzanoAuthproxy:           {VerrazzanoSystemNamespace},
	Istio:                         {IstioSystemNamespace},
	ExternalDNS:                   {CertManagerNamespace},
	VerrazzanoApplicationOperator: {VerrazzanoSystemNamespace},
	CoherenceOperator:             {VerrazzanoSystemNamespace},
	IngressController:             {platformOperatorConstants.IngressNginxNamespace},
	MySQL:                         {KeycloakNamespace},
	CertManager:                   {CertManagerNamespace},
	Rancher:                       {RancherSystemNamespace, RancherFleetSystemNamespace, RancherFleetLocalSystemNamespace},
	PrometheusPushgateway:         {platformOperatorConstants.VerrazzanoMonitoringNamespace},
	PrometheusAdapter:             {platformOperatorConstants.VerrazzanoMonitoringNamespace},
	KubeStateMetrics:              {platformOperatorConstants.VerrazzanoMonitoringNamespace},
	PrometheusNodeExporter:        {platformOperatorConstants.VerrazzanoMonitoringNamespace},
	PrometheusOperator:            {platformOperatorConstants.VerrazzanoMonitoringNamespace},
	Keycloak:                      {KeycloakNamespace},
	VerrazzanoMonitoringOperator:  {VerrazzanoSystemNamespace},
	Grafana:                       {VerrazzanoSystemNamespace},
	JaegerOperator:                {platformOperatorConstants.VerrazzanoMonitoringNamespace},
	OpensearchDashboards:          {VerrazzanoSystemNamespace},
	Opensearch:                    {VerrazzanoSystemNamespace},
	Velero:                        {platformOperatorConstants.VeleroNameSpace},
	VerrazzanoConsole:             {VerrazzanoSystemNamespace},
	Verrazzano:                    {VerrazzanoSystemNamespace},
	Fluentd:                       {VerrazzanoSystemNamespace},
	RancherBackup:                 {platformOperatorConstants.RancherBackupNamesSpace},
	MySQLOperator:                 {MySQLOperatorNamespace},
}
