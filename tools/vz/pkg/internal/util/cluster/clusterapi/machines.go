// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusterapi

import (
	"fmt"
	"path/filepath"

	"github.com/verrazzano/verrazzano/tools/vz/pkg/internal/util/files"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/internal/util/report"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const machinesResource = "machines.cluster.x-k8s.io"

// Minimal definition of machines.cluster.x-k8s.io object that only contains the fields that will be analyzed.
type machineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []machine `json:"items"`
}
type machine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Status            capiStatus `json:"status,omitempty"`
}

// AnalyzeMachines handles the checking of the status of cluster-api machine resources.
func AnalyzeMachines(clusterRoot string, namespace string, issueReporter *report.IssueReporter) error {
	resourceRoot := clusterRoot
	if len(namespace) != 0 {
		resourceRoot = filepath.Join(clusterRoot, namespace)
	}
	list := &machineList{}
	err := files.UnmarshallFileInClusterRoot(resourceRoot, "machine.cluster.x-k8s.io.json", list)
	if err != nil {
		return err
	}

	for _, machine := range list.Items {
		analyzeMachine(clusterRoot, machine, issueReporter)
	}

	return nil
}

// analyzeMachine - analyze a single cluster API machine and report any issues
func analyzeMachine(clusterRoot string, machine machine, issueReporter *report.IssueReporter) {

	var messages []string
	var subMessage string
	for _, condition := range machine.Status.Conditions {
		if condition.Status != corev1.ConditionTrue {
			switch condition.Type {
			case "APIServerPodHealthy":
				subMessage = "kube-apiserver pod is not healthy"
			case "BootstrapReady":
				subMessage = "bootstrap provider is not ready"
			case "ControllerManagerPodHealthy":
				subMessage = "kube-controller-manager pod is not healthy"
			case "EtcdMemberHealthy":
				subMessage = "member's etcd is not healthy"
			case "EtcdPodHealthy":
				subMessage = "pod's etcd is not healthy"
			case "InfrastructureReady":
				subMessage = "infrastructure provider is not ready"
			case "NodeHealthy":
				subMessage = "Kubernetes node is not healthy"
			case "SchedulerPodHealthy":
				subMessage = "kube-scheduler pod is not healthy"
			case "Ready":
				subMessage = "is not ready"
			default:
				continue
			}
			// Add a message for the issue
			var message string
			if len(condition.Reason) == 0 {
				message = fmt.Sprintf("\t%s", subMessage)
			} else {
				message = fmt.Sprintf("\t%s - reason is %s", subMessage, condition.Reason)
			}
			messages = append([]string{message}, messages...)

		}
	}

	if len(messages) > 0 {
		messages = append([]string{fmt.Sprintf("%q resource %q in namespace %q", machinesResource, machine.Name, machine.Namespace)}, messages...)
		issueReporter.AddKnownIssueMessagesFiles(report.ClusterAPIClusterIssues, clusterRoot, messages, []string{})
	}
}
