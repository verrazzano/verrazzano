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
	Status            capiStatus `json:"status,omitempty"`
}

// AnalyzeOCIMachines handles the checking of the status of cluster-api ociMachine resources.
func AnalyzeOCIMachines(clusterRoot string, namespace string, issueReporter *report.IssueReporter) error {
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
				message = fmt.Sprintf("\t%s", subMessage)
			} else {
				message = fmt.Sprintf("\t%s - reason is %s", subMessage, condition.Reason)
			}
			messages = append([]string{message}, messages...)

		}
	}

	if len(messages) > 0 {
		messages = append([]string{fmt.Sprintf("%q resource %q in namespace %q", ociMachinesResource, ociMachine.Name, ociMachine.Namespace)}, messages...)
		issueReporter.AddKnownIssueMessagesFiles(report.ClusterAPIClusterIssues, clusterRoot, messages, []string{})
	}
}
