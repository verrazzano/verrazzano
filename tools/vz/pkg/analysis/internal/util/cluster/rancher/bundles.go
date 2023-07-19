// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"fmt"
	"os"

	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/files"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/report"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Minimal definition of object that only contains the fields that will be analyzed
type bundleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []bundle `json:"items"`
}
type bundle struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Status            bundleStatus `json:"status,omitempty"`
}
type bundleStatus struct {
	Unavailable           int               `json:"unavailable,omitempty"`
	UnavailablePartitions int               `json:"unavailablePartitions,omitempty"`
	Conditions            []cattleCondition `json:"conditions,omitempty"`
}

// AnalyzeBundles - analyze the status of Bundle objects
func AnalyzeBundles(clusterRoot string, issueReporter *report.IssueReporter) error {
	snapshotFiles, err := os.ReadDir(clusterRoot)
	if err != nil {
		return err
	}
	for _, f := range snapshotFiles {
		if f.IsDir() {
			list := &bundleList{}
			err := files.UnmarshallFileInNamespace(clusterRoot, f.Name(), "bundle.fleet.cattle.io.json", list)
			if err != nil {
				return err
			}

			for _, bundle := range list.Items {
				err = analyzeBundle(clusterRoot, bundle, issueReporter)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// analyzeBundle - analyze a single Bundle and report any issues
func analyzeBundle(clusterRoot string, bundle bundle, issueReporter *report.IssueReporter) error {

	var messages []string
	var subMessage string
	for _, condition := range bundle.Status.Conditions {
		if condition.Status != corev1.ConditionTrue {
			switch condition.Type {
			case "Processed":
				subMessage = "is not processed"
			case "Ready":
				subMessage = "is not ready"
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
			message := fmt.Sprintf("Rancher Bundle resource %q %s %s%s", bundle.Name, subMessage, reason, msg)
			messages = append([]string{message}, messages...)
		}
	}

	if bundle.Status.Unavailable > 0 {
		message := fmt.Sprintf("Rancher Bundle resource %q in namespace %s has %d unavailable", bundle.Name, bundle.Namespace, bundle.Status.Unavailable)
		messages = append([]string{message}, messages...)
	}
	if bundle.Status.UnavailablePartitions > 0 {
		message := fmt.Sprintf("Rancher Bundle resource %q in namespace %s has %d unavailable partitions", bundle.Name, bundle.Namespace, bundle.Status.UnavailablePartitions)
		messages = append([]string{message}, messages...)
	}

	if len(messages) > 0 {
		issueReporter.AddKnownIssueMessagesFiles(report.RancherIssues, clusterRoot, messages, []string{})
	}

	return nil
}
