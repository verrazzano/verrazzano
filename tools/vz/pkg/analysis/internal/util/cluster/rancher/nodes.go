// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"fmt"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/files"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/report"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
)

// Minimal definition of object that only contains the fields that will be analyzed
type nodeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []node `json:"items"`
}
type node struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Status            cattleStatus `json:"status,omitempty"`
}

// AnalyzeNodes - analyze the status of Node objects
func AnalyzeNodes(clusterRoot string, issueReporter *report.IssueReporter) error {
	snapshotFiles, err := os.ReadDir(clusterRoot)
	if err != nil {
		return err
	}
	for _, f := range snapshotFiles {
		if f.IsDir() {
			list := &nodeList{}
			err = files.UnmarshallFileInNamespace(clusterRoot, f.Name(), "node.management.cattle.io.json", list)
			if err != nil {
				return err
			}
			for _, node := range list.Items {
				err = analyzeNode(clusterRoot, node, issueReporter)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// analyzeNode - analyze a single Node object and report any issues
func analyzeNode(clusterRoot string, node node, issueReporter *report.IssueReporter) error {

	var messages []string
	var subMessage string
	for _, condition := range node.Status.Conditions {
		if condition.Status != corev1.ConditionTrue {
			switch condition.Type {
			case "Initialized":
				subMessage = "is not initialized"
			case "Provisioned":
				subMessage = "is not provisioned"
			case "Updated":
				subMessage = "is not updated"
			case "Registered":
				subMessage = "is not registered with Kubernetes"
			case "Removed":
				subMessage = "is not removed"
			case "Saved":
				subMessage = "is not saved"
			case "Ready":
				subMessage = "is not ready"
			case "Drained":
				subMessage = "is not drained"
			case "Upgraded":
				subMessage = "is not upgraded"
			default:
				continue
			}
			// Add a message for the issue
			reason := ""
			msg := ""
			if len(condition.Reason) > 0 {
				reason = fmt.Sprintf(", reason is %q", condition.Reason)
			}
			if len(condition.Message) > 0 {
				msg = fmt.Sprintf(", message is %q", condition.Message)
			}
			message := fmt.Sprintf("Rancher Node resource %q %s %s%s", node.Name, subMessage, reason, msg)
			messages = append([]string{message}, messages...)
		}
	}

	if len(messages) > 0 {
		issueReporter.AddKnownIssueMessagesFiles(report.RancherIssues, clusterRoot, messages, []string{})
	}

	return nil
}
