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

const gitJobResource = "gitjob.gitjob.cattle.io"

// Minimal definition of object that only contains the fields that will be analyzed
type gitJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []gitJob `json:"items"`
}
type gitJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Status            gitJobStatus `json:"status,omitempty"`
}
type gitJobStatus struct {
	Conditions []cattleCondition `json:"conditions,omitempty"`
	JobStatus  string            `json:"jobStatus,omitempty"`
}

// AnalyzeGitJobs - analyze the status of GitJob objects
func AnalyzeGitJobs(clusterRoot string, issueReporter *report.IssueReporter) error {
	list := &gitJobList{}
	err := files.UnmarshallFileInClusterRoot(clusterRoot, fmt.Sprintf("%s.json", gitJobResource), list)
	if err != nil {
		return err
	}

	for _, job := range list.Items {
		err = analyzeGitJob(clusterRoot, job, issueReporter)
		if err != nil {
			return err
		}
	}

	return nil
}

// analyzeGitJob - analyze a single GitJob and report any issues
func analyzeGitJob(clusterRoot string, job gitJob, issueReporter *report.IssueReporter) error {

	var messages []string
	var subMessage string
	status := job.Status
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
			message := fmt.Sprintf("Rancher %s resource %q in namespace %s %s %s%s", gitJobResource, job.Name, job.Namespace, subMessage, reason, msg)
			messages = append([]string{message}, messages...)
		}
	}

	if len(messages) > 0 {
		messages = append([]string{fmt.Sprintf("The Rancher GitJob status is %s", status.JobStatus)}, messages...)
		issueReporter.AddKnownIssueMessagesFiles(report.RancherIssues, clusterRoot, messages, []string{})
	}

	return nil
}
