// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"fmt"

	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/files"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/report"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Minimal definition of object that only contains the fields that will be analyzed
type managedChartsList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []managedChart `json:"items"`
}
type managedChart struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Status            cattleStatus `json:"status,omitempty"`
}

// AnalyzeManagedCharts - analyze the status of ManagedCharts objects
func AnalyzeManagedCharts(clusterRoot string, issueReporter *report.IssueReporter) error {
	list := &managedChartsList{}
	err := files.UnmarshallFileInClusterRoot(clusterRoot, "managedchart.management.cattle.io.json", list)
	if err != nil {
		return err
	}

	for _, managedChart := range list.Items {
		err = analyzeManagedChart(clusterRoot, managedChart, issueReporter)
		if err != nil {
			return err
		}
	}

	return nil
}

// analyzeManagedChart - analyze a single ManagedChart object and report any issues
func analyzeManagedChart(clusterRoot string, managedChart managedChart, issueReporter *report.IssueReporter) error {

	var messages []string
	var subMessage string
	for _, condition := range managedChart.Status.Conditions {
		if condition.Status != corev1.ConditionTrue {
			switch condition.Type {
			case "Ready":
				subMessage = "is not ready"
			case "Processed":
				subMessage = "is not processed"
			case "Defined":
				subMessage = "is not defined"
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
			message := fmt.Sprintf("Rancher managedChart resource %q in namespace %q %s %s%s", managedChart.Name, managedChart.Namespace, subMessage, reason, msg)
			messages = append([]string{message}, messages...)
		}
	}

	if len(messages) > 0 {
		issueReporter.AddKnownIssueMessagesFiles(report.RancherIssues, clusterRoot, messages, []string{})
	}

	return nil
}
