// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package constants

// RestartVersionAnnotation - the annotation used by user to tell Verrazzano applicaton to restart its components
const RestartVersionAnnotation = "verrazzano.io/restart-version"

// LifecycleAnnotation - the annotation perform lifecycle actions on a workload
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
