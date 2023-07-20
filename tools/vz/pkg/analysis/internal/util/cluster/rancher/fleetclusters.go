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

const fleetClusterResource = "cluster.fleet.cattle.io"

// Minimal definition that only contains the fields that will be analyzed
type fleetClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []fleetCluster `json:"items"`
}
type fleetCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Status            fleetClusterStatus `json:"status,omitempty"`
}
type fleetClusterStatus struct {
	AgentMigrated           bool              `json:"agentMigrated,omitempty"`
	AgentNamespaceMigrated  bool              `json:"agentNamespaceMigrated,omitempty"`
	CattleNamespaceMigrated bool              `json:"cattleNamespaceMigrated,omitempty"`
	Conditions              []cattleCondition `json:"conditions,omitempty"`
}

// AnalyzeFleetClusters - analyze the status of Rancher fleet clusters resources
func AnalyzeFleetClusters(clusterRoot string, issueReporter *report.IssueReporter) error {
	list := &fleetClusterList{}
	err := files.UnmarshallFileInClusterRoot(clusterRoot, fmt.Sprintf("%s.json", fleetClusterResource), list)
	if err != nil {
		return err
	}

	for _, cluster := range list.Items {
		err = analyzeFleetCluster(clusterRoot, cluster, issueReporter)
		if err != nil {
			return err
		}
	}

	return nil
}

// analyzeFleetCluster - analyze a single Rancher fleet cluster and report any issues
func analyzeFleetCluster(clusterRoot string, cluster fleetCluster, issueReporter *report.IssueReporter) error {

	var messages []string
	var subMessage string
	for _, condition := range cluster.Status.Conditions {
		if condition.Status != corev1.ConditionTrue {
			switch condition.Type {
			case "Ready":
				subMessage = "is not ready"
			case "Processed":
				subMessage = "is not processed"
			case "Imported":
				subMessage = "is not imported"
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
			message := fmt.Sprintf("Rancher %s resource %q in namespace %s %s%s%s", fleetClusterResource, cluster.Name, cluster.Namespace, subMessage, reason, msg)
			messages = append([]string{message}, messages...)
		}
	}

	if !cluster.Status.AgentMigrated {
		message := fmt.Sprintf("Rancher %s resource %q in namespace %s agent not migrated", fleetClusterResource, cluster.Name, cluster.Namespace)
		messages = append([]string{message}, messages...)
	}
	if !cluster.Status.AgentNamespaceMigrated {
		message := fmt.Sprintf("Rancher %s resource %q in namespace %s agent namespace not migrated", fleetClusterResource, cluster.Name, cluster.Namespace)
		messages = append([]string{message}, messages...)
	}
	if !cluster.Status.CattleNamespaceMigrated {
		message := fmt.Sprintf("Rancher %s resource %q in namespace %s cattle namespace not migrated", fleetClusterResource, cluster.Name, cluster.Namespace)
		messages = append([]string{message}, messages...)
	}

	if len(messages) > 0 {
		issueReporter.AddKnownIssueMessagesFiles(report.RancherIssues, clusterRoot, messages, []string{})
	}

	return nil
}
