// Copyright (c) 2023, Oracle and/or its affiliates.
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
func AnalyzeGitJobs(clusterRoot string, namespace string, issueReporter *report.IssueReporter) error {
	resourceRoot := clusterRoot
	if len(namespace) != 0 {
		resourceRoot = filepath.Join(clusterRoot, namespace)
	}

	list := &gitJobList{}
	err := files.UnmarshallFileInClusterRoot(resourceRoot, fmt.Sprintf("%s.json", gitJobResource), list)
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
		switch condition.Type {
		case "Stalled":
			if condition.Status == corev1.ConditionTrue {
				subMessage = "is stalled"
			}
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

	if len(messages) > 0 {
		if len(status.JobStatus) > 0 {
			messages = append([]string{fmt.Sprintf("\tthe Rancher GitJob status is %s", status.JobStatus)}, messages...)
		}
		messages = append([]string{fmt.Sprintf("Rancher %s resource %q in namespace %s", gitJobResource, job.Name, job.Namespace)}, messages...)
		issueReporter.AddKnownIssueMessagesFiles(report.RancherIssues, clusterRoot, messages, []string{})
	}

	return nil
}
