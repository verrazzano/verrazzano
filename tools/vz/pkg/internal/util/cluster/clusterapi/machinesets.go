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

const machineSetsResource = "machinesets.cluster.x-k8s.io"

// Minimal definition of machinesets.cluster.x-k8s.io object that only contains the fields that will be analyzed.
type machineSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []machineSet `json:"items"`
}
type machineSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Status            capiStatus `json:"status,omitempty"`
}

// AnalyzeMachineSetss handles the checking of the status of cluster-api machineSet resources.
func AnalyzeMachineSets(clusterRoot string, namespace string, issueReporter *report.IssueReporter) error {
	resourceRoot := clusterRoot
	if len(namespace) != 0 {
		resourceRoot = filepath.Join(clusterRoot, namespace)
	}
	list := &machineSetList{}
	err := files.UnmarshallFileInClusterRoot(resourceRoot, "machineset.cluster.x-k8s.io.json", list)
	if err != nil {
		return err
	}

	for _, machineSet := range list.Items {
		analyzeMachineSet(clusterRoot, machineSet, issueReporter)
	}

	return nil
}

// analyzeMachineSet - analyze a single cluster API machineSet and report any issues
func analyzeMachineSet(clusterRoot string, machineSet machineSet, issueReporter *report.IssueReporter) {

	var messages []string
	var subMessage string
	for _, condition := range machineSet.Status.Conditions {
		if condition.Status != corev1.ConditionTrue {
			switch condition.Type {
			case "MachinesCreated":
				subMessage = "machines are not created"
			case "MachinesReady":
				subMessage = "machines are not ready"
			case "Resized":
				subMessage = "is not resized"
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
		messages = append([]string{fmt.Sprintf("%q resource %q in namespace %q", machineSetsResource, machineSet.Name, machineSet.Namespace)}, messages...)
		issueReporter.AddKnownIssueMessagesFiles(report.ClusterAPIClusterIssues, clusterRoot, messages, []string{})
	}
}
