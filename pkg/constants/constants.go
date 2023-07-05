// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package constants

import (
	"time"
)

// VerrazzanoClusterIssuerName Name of the Verrazzano Cert-Manager cluster issuer
const VerrazzanoClusterIssuerName = "verrazzano-cluster-issuer"

// RestartVersionAnnotation - the annotation used by user to tell Verrazzano application to restart its components
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

// VerrazzanoCAPINamespace is the system namespace for Cluster API resources
const VerrazzanoCAPINamespace = "verrazzano-capi"

// VerrazzanoMultiClusterNamespace is the multi-cluster namespace for Verrazzano
const VerrazzanoMultiClusterNamespace = "verrazzano-mc"

// VerrazzanoMonitoringNamespace is the namespace for monitoring components
const VerrazzanoMonitoringNamespace = "verrazzano-monitoring"

// CertManagerNamespace - the CertManager namespace
const CertManagerNamespace = "cert-manager"

// ExternalDNSNamespace - the ExternalDNS namespace
const ExternalDNSNamespace = VerrazzanoSystemNamespace

// KeycloakNamespace - the keycloak namespace
const KeycloakNamespace = "keycloak"

// MySQLOperatorNamespace indicates the namespace to be used for the MySQLOperator installation
const MySQLOperatorNamespace = "mysql-operator"

// RancherSystemNamespace - the Rancher cattle-system namespace
const RancherSystemNamespace = "cattle-system"

// IstioSystemNamespace - the Istio system namespace
const IstioSystemNamespace = "istio-system"

// PrometheusOperatorNamespace - the namespace where Verrazzano installs Prometheus Operator
// and its related components.
const PrometheusOperatorNamespace = "verrazzano-monitoring"

// ArgoCDNamespace - the Argocd namespace
const ArgoCDNamespace = "argocd"

// LabelIstioInjection - constant for a Kubernetes label that is applied by Verrazzano
const LabelIstioInjection = "istio-injection"

// LabelVerrazzanoNamespace - constant for a Kubernetes label that is used by network policies
const LabelVerrazzanoNamespace = "verrazzano.io/namespace"

// LegacyOpensearchSecretName legacy secret name for Opensearch credentials
const LegacyOpensearchSecretName = "verrazzano"

// VerrazzanoESInternal is the name of the Verrazzano internal Opensearch secret in the Verrazzano system namespace
const VerrazzanoESInternal = "verrazzano-es-internal"

// VerrazzanoPromInternal is the name of the Verrazzano internal Prometheus secret in the Verrazzano system namespace
const VerrazzanoPromInternal = "verrazzano-prom-internal"

// RancherTLSCA is a tls secret that contains CA if private CA is being used
const RancherTLSCA = "tls-ca"

// RancherTLSCAKey is the key containing the CA in the secret specified by the RancherTLSCA constant
const RancherTLSCAKey = "cacerts.pem"

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

// FluentBitDaemonSetName - The name of the FluentBit DaemonSet
const FluentBitDaemonSetName = "fluent-bit"

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
const DefaultOpensearchURL = "http://verrazzano-authproxy-opensearch:8775"

// Default Jaeger OpenSearch URL
const DefaultJaegerOSURL = "http://verrazzano-authproxy-opensearch.verrazzano-system:8775"

// DefaultOperatorOSURL is the default OpenSearch URL for opensearch-operator based OpenSearch
const DefaultOperatorOSURL = "http://verrazzano-authproxy-opensearch-logging:8775"

// DefaultOperatorOSURLWithNS is the default OpenSearch URL for opensearch-operator based OpenSearch with namespace suffix
const DefaultOperatorOSURLWithNS = "http://verrazzano-authproxy-opensearch-logging.verrazzano-system:8775"

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

// MysqlBackupMutatingWebhookName specifies the name of mysql webhook.
const MysqlBackupMutatingWebhookName = "verrazzano-mysql-backup"

// MysqlBackupMutatingWebhookPath specifies the path of mysql webhook.
const MysqlBackupMutatingWebhookPath = "/mysql-backup-job-annotate"

// VerrazzanoClusterRancherName is the name for the Rancher cluster role and secret used to grant permissions to the Verrazzano cluster user
const VerrazzanoClusterRancherName = "verrazzano-cluster-registrar"

// VerrazzanoClusterRancherUsername is the username in Rancher used to identify the Verrazzano cluster user
const VerrazzanoClusterRancherUsername = "vz-cluster-reg"

// ArgoCDClusterRancherSecretName is the name of secret for the Verrazzano Argo CD cluster user
// #nosec
const ArgoCDClusterRancherSecretName = "verrazzano-argocd-secret"

// ArgoCDClusterRancherUsername is the username in Rancher used to identify the Verrazzano Argo CD cluster user
const ArgoCDClusterRancherUsername = "vz-argoCD-reg"

// Components Names
const (
	Istio                 = "istio"
	ExternalDNS           = "external-dns"
	IngressController     = "ingress-controller"
	IngressDefaultBackend = "ingress-controller-ingress-nginx-defaultbackend"
	MySQL                 = "mysql"
	CertManager           = "cert-manager"
	Rancher               = "rancher"
	Keycloak              = "keycloak"
	Grafana               = "grafana"
	JaegerOperator        = "jaeger-operator"
	Opensearch            = "opensearch"
	Velero                = "velero"
	Verrazzano            = "verrazzano"
	Fluentd               = "fluentd"
	MySQLOperator         = "mysql-operator"
)

// ThanosQueryIngress is the name of the ingress for the Thanos Query
const ThanosQueryIngress = "thanos-query-frontend"

// ThanosQueryStoreIngress is the name of the ingress for the Thanos Query Store API
const ThanosQueryStoreIngress = "thanos-query-store"
