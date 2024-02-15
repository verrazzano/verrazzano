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

const ociClustersResource = "ociclusters.infrastructure.cluster.x-k8s.io"

// Minimal definition of ociclusters.infrastructure.cluster.x-k8s.io object that only contains the fields that
// will be analyzed.
type ociClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ociCluster `json:"items"`
}
type ociCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Status            ociClusterStatus `json:"status,omitempty"`
}
type ociClusterStatus struct {
	Conditions []capiCondition `json:"conditions,omitempty"`
	Ready      bool            `json:"ready,omitempty"`
}

// AnalyzeOCIClusters handles the checking of the status of cluster-api ocicluster resources.
func AnalyzeOCIClusters(clusterRoot string, namespace string, issueReporter *report.IssueReporter) error {
	resourceRoot := clusterRoot
	if len(namespace) != 0 {
		resourceRoot = filepath.Join(clusterRoot, namespace)
	}
	list := &ociClusterList{}
	err := files.UnmarshallFileInClusterRoot(resourceRoot, "ocicluster.infrastructure.cluster.x-k8s.io.json", list)
	if err != nil {
		return err
	}

	for _, ociCluster := range list.Items {
		analyzeOCICluster(clusterRoot, ociCluster, issueReporter)
	}

	return nil
}

// analyzeOCICluster - analyze a single cluster API ocicluster and report any issues
func analyzeOCICluster(clusterRoot string, ociCluster ociCluster, issueReporter *report.IssueReporter) {

	var messages []string
	var subMessage string
	if !ociCluster.Status.Ready {
		if len(ociCluster.Status.Conditions) == 0 {
			message := fmt.Sprintf("%q resource %q in namespace %q, %s", ociClustersResource, ociCluster.Name, ociCluster.Namespace, "is not ready")
			messages = append([]string{message}, messages...)
		} else {
			for _, condition := range ociCluster.Status.Conditions {
				if condition.Status != corev1.ConditionTrue {
					switch condition.Type {
					case "ClusterReady":
						subMessage = "cluster is not ready"
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
		}
	}

	if len(messages) > 0 {
		messages = append([]string{fmt.Sprintf("%q resource %q in namespace %q", ociClustersResource, ociCluster.Name, ociCluster.Namespace)}, messages...)
		issueReporter.AddKnownIssueMessagesFiles(report.ClusterAPIClusterIssues, clusterRoot, messages, []string{})
	}
}
