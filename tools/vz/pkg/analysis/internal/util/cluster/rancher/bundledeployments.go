// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"fmt"

	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/files"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/report"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
func AnalyzeBundleDeployments(log *zap.SugaredLogger, clusterRoot string, issueReporter *report.IssueReporter) error {
	list := &bundleDeploymentsList{}
	err := files.UnmarshallFileInClusterRoot(clusterRoot, "bundledeployment.fleet.cattle.io.json", list)
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
			message := fmt.Sprintf("Rancher BundledDeployment resource %q %s %s%s", bundleDeployment.Name, subMessage, reason, msg)
			messages = append([]string{message}, messages...)
		}
	}

	if !bundleDeployment.Status.Ready {
		message := fmt.Sprintf("Rancher BundledDeployment resource %q in namespace %s is not ready", bundleDeployment.Name, bundleDeployment.Namespace)
		messages = append([]string{message}, messages...)
	}

	if len(messages) > 0 {
		issueReporter.AddKnownIssueMessagesFiles(report.RancherIssues, clusterRoot, messages, []string{})
	}

	return nil
}
