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

const capiClusterResourceSetsResource = "clusterresourcesets.addons.cluster.x-k8s.io"

// Minimal definition of clusterresourcesets.addons.cluster.x-k8s.io object that only contains the fields that will be analyzed
type clusterResourceSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []clusterResourceSet `json:"items"`
}
type clusterResourceSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Status            capiStatus `json:"status,omitempty"`
}

// AnalyzeClusterResourceSets handles the checking of the status of cluster-api clusterResoureSet resources.
func AnalyzeClusterResourceSets(clusterRoot string, namespace string, issueReporter *report.IssueReporter) error {
	resourceRoot := clusterRoot
	if len(namespace) != 0 {
		resourceRoot = filepath.Join(clusterRoot, namespace)
	}
	list := &clusterResourceSetList{}
	err := files.UnmarshallFileInClusterRoot(resourceRoot, "clusterresourceset.addons.cluster.x-k8s.io.json", list)
	if err != nil {
		return err
	}

	for _, clusterResourceSet := range list.Items {
		analyzeClusterResourceSet(clusterRoot, clusterResourceSet, issueReporter)
	}

	return nil
}

// analyzeClusterResourceSet - analyze a single cluster API clusterResourceSet and report any issues
func analyzeClusterResourceSet(clusterRoot string, clusterResourceSet clusterResourceSet, issueReporter *report.IssueReporter) {

	var messages []string
	var subMessage string
	for _, condition := range clusterResourceSet.Status.Conditions {
		if condition.Status != corev1.ConditionTrue {
			switch condition.Type {
			case "ResourcesApplied":
				subMessage = "is not applied"
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
		messages = append([]string{fmt.Sprintf("%q resource %q in namespace %q", capiClusterResourceSetsResource, clusterResourceSet.Name, clusterResourceSet.Namespace)}, messages...)
		issueReporter.AddKnownIssueMessagesFiles(report.ClusterAPIClusterIssues, clusterRoot, messages, []string{})
	}
}
