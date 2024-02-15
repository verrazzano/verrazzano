// Copyright (c) 2023, 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"fmt"
	"path/filepath"

	"github.com/verrazzano/verrazzano/tools/vz/pkg/internal/util/files"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/internal/util/report"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const bundleDeploymentResource = "bundledeployment.fleet.cattle.io"

// Minimal definition of object that only contains the fields that will be analyzed
type bundleDeploymentsList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []bundleDeployment `json:"items"`
}
type bundleDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Status            bundleDeploymentStatus `json:"status,omitempty"`
}
type bundleDeploymentStatus struct {
	Ready      bool              `json:"ready,omitempty"`
	Conditions []cattleCondition `json:"conditions,omitempty"`
}

// AnalyzeBundleDeployments - analyze the status of BundleDeployment objects
func AnalyzeBundleDeployments(clusterRoot string, namespace string, issueReporter *report.IssueReporter) error {
	resourceRoot := clusterRoot
	if len(namespace) != 0 {
		resourceRoot = filepath.Join(clusterRoot, namespace)
	}

	list := &bundleDeploymentsList{}
	err := files.UnmarshallFileInClusterRoot(resourceRoot, fmt.Sprintf("%s.json", bundleDeploymentResource), list)
	if err != nil {
		return err
	}

	for _, deployment := range list.Items {
		err = analyzeBundleDeployment(clusterRoot, deployment, issueReporter)
		if err != nil {
			return err
		}
	}

	return nil
}

// analyzeBundleDeployment - analyze a single BundleDeployment and report any issues
func analyzeBundleDeployment(clusterRoot string, bundleDeployment bundleDeployment, issueReporter *report.IssueReporter) error {

	var messages []string
	var subMessage string
	for _, condition := range bundleDeployment.Status.Conditions {
		if condition.Status != corev1.ConditionTrue {
			switch condition.Type {
			case "Installed":
				subMessage = "is not installed"
			case "Deployed":
				subMessage = "is not deployed"
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
			message := fmt.Sprintf("\t%s %s%s", subMessage, reason, msg)
			messages = append([]string{message}, messages...)
		}
	}

	if !bundleDeployment.Status.Ready {
		messages = append([]string{"\tis not ready"}, messages...)
	}

	if len(messages) > 0 {
		messages = append([]string{fmt.Sprintf("Rancher BundledDeployment resource %q", bundleDeployment.Name)}, messages...)
		issueReporter.AddKnownIssueMessagesFiles(report.RancherIssues, clusterRoot, messages, []string{})
	}

	return nil
}
