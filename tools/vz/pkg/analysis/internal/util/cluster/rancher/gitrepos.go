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
type gitRepoList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []gitRepo `json:"items"`
}
type gitRepo struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Status            gitRepoStatus `json:"status,omitempty"`
}
type gitRepoStatus struct {
	Conditions           []cattleCondition `json:"conditions,omitempty"`
	DesiredReadyClusters int               `json:"desiredReadyClusters,omitempty"`
	ReadyClusters        int               `json:"readyClusters,omitempty"`
}

// AnalyzeGitRepos - analyze the status of GitRepo objects
func AnalyzeGitRepos(clusterRoot string, issueReporter *report.IssueReporter) error {
	list := &gitRepoList{}
	err := files.UnmarshallFileInClusterRoot(clusterRoot, "gitrepo.fleet.cattle.io.json", list)
	if err != nil {
		return err
	}

	for _, gitRepo := range list.Items {
		err = analyzeGitRepo(clusterRoot, gitRepo, issueReporter)
		if err != nil {
			return err
		}
	}

	return nil
}

// analyzeGitRepo - analyze a single GitRepo and report any issues
func analyzeGitRepo(clusterRoot string, gitRepo gitRepo, issueReporter *report.IssueReporter) error {

	var messages []string
	var subMessage string
	status := gitRepo.Status
	for _, condition := range status.Conditions {
		if condition.Status != corev1.ConditionTrue {
			switch condition.Type {
			case "Ready":
				subMessage = "is not ready"
			case "Accepted":
				subMessage = "is not accepted"
			case "ImageSynced":
				subMessage = "image not synced"
			case "Synced":
				subMessage = "is not synced"
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
			message := fmt.Sprintf("GitRepo resource %q in namespace %s %s %s%s", gitRepo.Name, gitRepo.Namespace, subMessage, reason, msg)
			messages = append([]string{message}, messages...)
		}
	}

	if status.DesiredReadyClusters != status.ReadyClusters {
		message := fmt.Sprintf("GitRepo resource %q in namespace %s expected %d to be ready, actual ready is %d", gitRepo.Name, gitRepo.Namespace, status.DesiredReadyClusters, status.ReadyClusters)
		messages = append([]string{message}, messages...)
	}

	if len(messages) > 0 {
		issueReporter.AddKnownIssueMessagesFiles(report.RancherIssues, clusterRoot, messages, []string{})
	}

	return nil
}
