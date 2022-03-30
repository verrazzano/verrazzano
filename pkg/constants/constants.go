// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package constants

import "time"

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

// VerrazzanoSystemNamespace is the system namespace for Verrazzano
const VerrazzanoSystemNamespace = "verrazzano-system"

// VerrazzanoMultiClusterNamespace is the multi-cluster namespace for Verrazzano
const VerrazzanoMultiClusterNamespace = "verrazzano-mc"

// KeycloakNamespace - the keycloak namespace
const KeycloakNamespace = "keycloak"

// RancherSystemNamespace - the Rancher cattle-system namespace
const RancherSystemNamespace = "cattle-system"

// RancherOperatorSystemNamespace - the Rancher operator system namespace
const RancherOperatorSystemNamespace = "rancher-operator-system"

// VerrazzanoMonitoringNamespace - the keycloak namespace
const VerrazzanoMonitoringNamespace = "monitoring"

// IstioSystemNamespace - the Istio system namespace
const IstioSystemNamespace = "istio-system"

// IngressNamespace - the NGINX ingress namespace
const IngressNamespace = "ingress-nginx"

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

// VMCAgentPollingTimeInterval - The time interval at which mcagent polls Verrazzano Managed CLuster resource on the admin cluster.
const VMCAgentPollingTimeInterval = 60 * time.Second

// MaxTimesVMCAgentPollingTime - The constant used to set max polling time for vmc agent to determine VMC state
const MaxTimesVMCAgentPollingTime = 3

// FluentdDaemonSetName - The name of the Fluentd DaemonSet
const FluentdDaemonSetName = "fluentd"

// KubeSystem - The name of the kube-system namespace
const KubeSystem = "kube-system"
