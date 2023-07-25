// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusterapi

import (
	"fmt"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/files"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/report"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"path/filepath"
)

const ocneConfigsResource = "ocneconfigs.bootstrap.cluster.x-k8s.io"

// Minimal definition of ocneconfigs.bootstrap.cluster.x-k8s.io object that only contains the fields that will be analyzed.
type ocneConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ocneConfig `json:"items"`
}
type ocneConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Status            capiStatus `json:"status,omitempty"`
}

// AnalyzeOCNEConfig handles the checking of the status of cluster-qpi ocneConfig resources.
func AnalyzeOCNEConfig(clusterRoot string, namespace string, issueReporter *report.IssueReporter) error {
	resourceRoot := clusterRoot
	if len(namespace) != 0 {
		resourceRoot = filepath.Join(clusterRoot, namespace)
	}
	list := &ocneConfigList{}
	err := files.UnmarshallFileInClusterRoot(resourceRoot, "ocneconfig.bootstrap.cluster.x-k8s.io.json", list)
	if err != nil {
		return err
	}

	for _, ocneConfig := range list.Items {
		analyzeOCNEConfig(clusterRoot, ocneConfig, issueReporter)
	}

	return nil
}

// analyzeOCNEConfig - analyze a single cluster API ocneConfig and report any issues
func analyzeOCNEConfig(clusterRoot string, ocneConfig ocneConfig, issueReporter *report.IssueReporter) {

	var messages []string
	var subMessage string
	for _, condition := range ocneConfig.Status.Conditions {
		if condition.Status != corev1.ConditionTrue {
			switch condition.Type {
			case "CertificatesAvailable":
				subMessage = "certificates are not available"
			case "DataSecretAvailable":
				subMessage = "data secret is not available"
			case "Ready":
				subMessage = "is not ready"
			default:
				continue
			}
			// Add a message for the issue
			var message string
			if len(condition.Reason) == 0 {
				message = fmt.Sprintf("%q resource %q in namespace %q, %s", ocneConfigsResource, ocneConfig.Name, ocneConfig.Namespace, subMessage)
			} else {
				message = fmt.Sprintf("%q resource %q in namespace %q, %s - reason is %s", ocneConfigsResource, ocneConfig.Name, ocneConfig.Namespace, subMessage, condition.Reason)
			}
			messages = append([]string{message}, messages...)

		}
	}

	if len(messages) > 0 {
		issueReporter.AddKnownIssueMessagesFiles(report.ClusterAPIClusterNotReady, clusterRoot, messages, []string{})
	}
}
