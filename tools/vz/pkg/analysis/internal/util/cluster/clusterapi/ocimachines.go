// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusterapi

import (
	"fmt"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/files"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/report"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"path/filepath"
)

const ociMachinesResource = "ocimachines.infrastructure.cluster.x-k8s.io"

// Minimal definition of ocimachines.infrastructure.cluster.x-k8s.io object that only contains the fields that will be analyzed.
type ociMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ociMachine `json:"items"`
}
type ociMachine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Status            ociMachineStatus `json:"status,omitempty"`
}
type ociMachineStatus struct {
	Conditions []ociMachineCondition `json:"conditions,omitempty"`
}
type ociMachineCondition struct {
	Reason string                 `json:"reason,omitempty"`
	Status corev1.ConditionStatus `json:"status"`
	Type   string                 `json:"type"`
}

// AnalyzeOCIMachine handles the checking of the status of cluster-qpi ociMachine resources.
func AnalyzeOCIMachine(clusterRoot string, namespace string, issueReporter *report.IssueReporter) error {
	resourceRoot := clusterRoot
	if len(namespace) != 0 {
		resourceRoot = filepath.Join(clusterRoot, namespace)
	}
	list := &ociMachineList{}
	err := files.UnmarshallFileInClusterRoot(resourceRoot, "ocimachine.infrastructure.cluster.x-k8s.io.json", list)
	if err != nil {
		return err
	}

	for _, ociMachine := range list.Items {
		analyzeOCIMachine(clusterRoot, ociMachine, issueReporter)
	}

	return nil
}

// analyzeOCIMachine - analyze a single cluster API ociMachine and report any issues
func analyzeOCIMachine(clusterRoot string, ociMachine ociMachine, issueReporter *report.IssueReporter) {

	var messages []string
	var subMessage string
	for _, condition := range ociMachine.Status.Conditions {
		if condition.Status != corev1.ConditionTrue {
			switch condition.Type {
			case "InstanceReady":
				subMessage = "OCI instance is not ready"
			case "Ready":
				subMessage = "is not ready"
			default:
				continue
			}
			// Add a message for the issue
			var message string
			if len(condition.Reason) == 0 {
				message = fmt.Sprintf("%q resource %q in namespace %q, %s", ociMachinesResource, ociMachine.Name, ociMachine.Namespace, subMessage)
			} else {
				message = fmt.Sprintf("%q resource %q in namespace %q, %s - reason is %s", ociMachinesResource, ociMachine.Name, ociMachine.Namespace, subMessage, condition.Reason)
			}
			messages = append([]string{message}, messages...)

		}
	}

	if len(messages) > 0 {
		issueReporter.AddKnownIssueMessagesFiles(report.ClusterAPIClusterNotReady, clusterRoot, messages, []string{})
	}
}
