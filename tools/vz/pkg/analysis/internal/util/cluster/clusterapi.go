// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cluster

import (
	"encoding/json"
	"fmt"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/files"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/report"
	"go.uber.org/zap"
	"io"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
)

// Minimal definition of cluster.x-k8s.io object that only contains the fields that will be analyzed
type clusterAPIClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []clusterAPICluster `json:"items"`
}
type clusterAPICluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Status            clusterAPIClusterStatus `json:"status,omitempty"`
}
type clusterAPIClusterStatus struct {
	Conditions []clusterAPIClusterCondition `json:"conditions,omitempty"`
}
type clusterAPIClusterCondition struct {
	Message  string                 `json:"message,omitempty"`
	Reason   string                 `json:"reason,omitempty"`
	Severity string                 `json:"severity"`
	Status   corev1.ConditionStatus `json:"status"`
	Type     string                 `json:"type"`
}

// AnalyzeClusterAPIIssues handles the checking of cluster-api cluster resource.
func AnalyzeClusterAPIIssues(log *zap.SugaredLogger, clusterRoot string) error {
	log.Debugf("AnalyzeClusterAPIIssues called for %s", clusterRoot)

	var issueReporter = report.IssueReporter{
		PendingIssues: make(map[string]report.Issue),
	}

	return analyzeClusterAPICLusters(log, clusterRoot, &issueReporter)
}

// analyzeClusterAPICLusters handles the checking of the status of cluster-qpi cluster resources.
func analyzeClusterAPICLusters(log *zap.SugaredLogger, clusterRoot string, issueReporter *report.IssueReporter) error {
	namespaces, err := files.FindNamespaces(log, clusterRoot)
	if err != nil {
		return err
	}

	for _, namespace := range namespaces {
		clusterFile := files.FindFileInNamespace(clusterRoot, namespace, "cluster.cluster.x-k8s.io.json")
		// Parse the json into local struct
		file, err := os.Open(clusterFile)
		// File may not exist so just continue if that is the case
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return fmt.Errorf("failed to open file %s from cluster snapshot: %s", clusterFile, err.Error())
		}
		defer file.Close()

		fileBytes, err := io.ReadAll(file)
		if err != nil {
			return fmt.Errorf("failed reading JSON file %s from cluster snapshot: %s", clusterFile, err)
		}
		clusterList := &clusterAPIClusterList{}
		err = json.Unmarshal(fileBytes, &clusterList)
		if err != nil {
			return fmt.Errorf("failed to unmarshal Cluster API Cluster list from cluster snapshot: %s", err)
		}

		// Analyze each cluster API cluster resource.
		for _, cluster := range clusterList.Items {
			analyzeClusterAPICluster(clusterRoot, cluster, issueReporter)
		}

		// check if this is the correct CAPI json file and handle multiple cluster resources in the same namespace
	}

	issueReporter.Contribute(log, clusterRoot)

	return nil
}

// analyzeClusterAPICluster - analyze a single cluster API cluster and report any issues
func analyzeClusterAPICluster(clusterRoot string, cluster clusterAPICluster, issueReporter *report.IssueReporter) {

	var messages []string
	var subMessage string
	for _, condition := range cluster.Status.Conditions {
		if condition.Status != corev1.ConditionTrue {
			switch condition.Type {
			case "Ready":
				subMessage = "is not ready"
			case "ControlPlaneInitialized":
				subMessage = "control plane not initialized"
			case "ControlPlaneReady":
				subMessage = "control plane is not ready"
			case "InfrastructureReady":
				subMessage = "infrastructure is not ready"
			}
			// Add a message for the issue
			var message string
			if len(condition.Reason) == 0 {
				message = fmt.Sprintf("Cluster API cluster resource %q in namespace %q, %s", cluster.Name, cluster.Namespace, subMessage)
			} else {
				message = fmt.Sprintf("Cluster API cluster resource %q in namespace %q, %s - reason is %s", cluster.Name, cluster.Namespace, subMessage, condition.Reason)
			}
			messages = append([]string{message}, messages...)

		}
	}

	if len(messages) > 0 {
		issueReporter.AddKnownIssueMessagesFiles(report.ClusterAPIClusterNotReady, clusterRoot, messages, []string{})
	}
}
