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

const capiClustersResource = "clusters.cluster.x-k8s.io"

// Minimal definition of clusters.cluster.x-k8s.io object that only contains the fields that will be analyzed
type clusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []cluster `json:"items"`
}
type cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Status            capiStatus `json:"status,omitempty"`
}

// AnalyzeClusters handles the checking of the status of cluster-api cluster resources.
func AnalyzeClusters(clusterRoot string, namespace string, issueReporter *report.IssueReporter) error {
	resourceRoot := clusterRoot
	if len(namespace) != 0 {
		resourceRoot = filepath.Join(clusterRoot, namespace)
	}
	list := &clusterList{}
	err := files.UnmarshallFileInClusterRoot(resourceRoot, "cluster.cluster.x-k8s.io.json", list)
	if err != nil {
		return err
	}

	for _, cluster := range list.Items {
		analyzeCluster(clusterRoot, cluster, issueReporter)
	}

	return nil
}

// analyzeCluster - analyze a single cluster API cluster and report any issues
func analyzeCluster(clusterRoot string, cluster cluster, issueReporter *report.IssueReporter) {

	var messages []string
	var subMessage string
	for _, condition := range cluster.Status.Conditions {
		if condition.Status != corev1.ConditionTrue {
			switch condition.Type {
			case "ControlPlaneInitialized":
				subMessage = "control plane not initialized"
			case "ControlPlaneReady":
				subMessage = "control plane is not ready"
			case "InfrastructureReady":
				subMessage = "infrastructure is not ready"
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
		messages = append([]string{fmt.Sprintf("%q resource %q in namespace %q", capiClustersResource, cluster.Name, cluster.Namespace)}, messages...)
		issueReporter.AddKnownIssueMessagesFiles(report.ClusterAPIClusterIssues, clusterRoot, messages, []string{})
	}
}
