// Copyright (c) 2023, 2024, Oracle and/or its affiliates.
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

const machineDeploymentsResource = "machinedeployments.cluster.x-k8s.io"

// Minimal definition of machinedeployments.cluster.x-k8s.io object that only contains the fields that will be analyzed.
type machineDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []machineDeployment `json:"items"`
}
type machineDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Status            capiStatus `json:"status,omitempty"`
}

// AnalyzeMachineDeployments handles the checking of the status of cluster-api machine deploymet resources.
func AnalyzeMachineDeployments(clusterRoot string, namespace string, issueReporter *report.IssueReporter) error {
	resourceRoot := clusterRoot
	if len(namespace) != 0 {
		resourceRoot = filepath.Join(clusterRoot, namespace)
	}
	list := &machineDeploymentList{}
	err := files.UnmarshallFileInClusterRoot(resourceRoot, "machinedeployment.cluster.x-k8s.io.json", list)
	if err != nil {
		return err
	}

	for _, machineDeployment := range list.Items {
		analyzeMachineDeployment(clusterRoot, machineDeployment, issueReporter)
	}

	return nil
}

// analyzeMachineDeployment - analyze a single cluster API machine deployment and report any issues
func analyzeMachineDeployment(clusterRoot string, machineDeployment machineDeployment, issueReporter *report.IssueReporter) {

	var messages []string
	var subMessage string
	for _, condition := range machineDeployment.Status.Conditions {
		if condition.Status != corev1.ConditionTrue {
			switch condition.Type {
			case "Available":
				subMessage = "is not available"
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
		messages = append([]string{fmt.Sprintf("%q resource %q in namespace %q", machineDeploymentsResource, machineDeployment.Name, machineDeployment.Namespace)}, messages...)
		issueReporter.AddKnownIssueMessagesFiles(report.ClusterAPIClusterIssues, clusterRoot, messages, []string{})
	}
}
