// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cluster

import (
	"fmt"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/files"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/report"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"path/filepath"
)

const ociClusterResource = "ociclusters.infrastructure.cluster.x-k8s.io"

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
	Conditions []ociClusterCondition `json:"conditions,omitempty"`
	Ready      bool                  `json:"ready,omitempty"`
}
type ociClusterCondition struct {
	Message string                 `json:"message,omitempty"`
	Reason  string                 `json:"reason,omitempty"`
	Status  corev1.ConditionStatus `json:"status"`
	Type    string                 `json:"type"`
}

// AnalyzeOCIClusters handles the checking of the status of cluster-qpi ocicluster resources.
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
			message := fmt.Sprintf("%q resource %q in namespace %q, %s", ociClusterResource, ociCluster.Name, ociCluster.Namespace, "is not ready")
			messages = append([]string{message}, messages...)
		} else {
			for _, condition := range ociCluster.Status.Conditions {
				if condition.Status != corev1.ConditionTrue {
					switch condition.Type {
					case "Ready":
						subMessage = "is not ready"
					case "ClusterReady":
						subMessage = "cluster is not ready"
					default:
						continue
					}
					// Add a message for the issue
					var message string
					if len(condition.Reason) == 0 {
						message = fmt.Sprintf("%q resource %q in namespace %q, %s", ociClusterResource, ociCluster.Name, ociCluster.Namespace, subMessage)
					} else {
						message = fmt.Sprintf("%q resource %q in namespace %q, %s - reason is %s", ociClusterResource, ociCluster.Name, ociCluster.Namespace, subMessage, condition.Reason)
					}
					messages = append([]string{message}, messages...)

				}
			}
		}
	}

	if len(messages) > 0 {
		issueReporter.AddKnownIssueMessagesFiles(report.ClusterAPIClusterNotReady, clusterRoot, messages, []string{})
	}
}
