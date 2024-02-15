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

const clusterRepoResource = "clusterrepo.catalog.cattle.io"

// Minimal definition of object that only contains the fields that will be analyzed
type clusterRepoList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []clusterRepo `json:"items"`
}
type clusterRepo struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              clusterRepoSpec `json:"spec,omitempty"`
	Status            cattleStatus    `json:"status,omitempty"`
}
type clusterRepoSpec struct {
	GitBranch string `json:"gitBranch,omitempty"`
	GitRepo   string `json:"gitRepo,omitempty"`
}

// AnalyzeClusterRepos - analyze the status of ClusterRepo objects
func AnalyzeClusterRepos(clusterRoot string, namespace string, issueReporter *report.IssueReporter) error {
	resourceRoot := clusterRoot
	if len(namespace) != 0 {
		resourceRoot = filepath.Join(clusterRoot, namespace)
	}

	list := &clusterRepoList{}
	err := files.UnmarshallFileInClusterRoot(resourceRoot, fmt.Sprintf("%s.json", clusterRepoResource), list)
	if err != nil {
		return err
	}

	for _, cluster := range list.Items {
		err = analyzeClusterRepo(clusterRoot, cluster, issueReporter)
		if err != nil {
			return err
		}
	}

	return nil
}

// analyzeClusterRepo - analyze a single ClusterRepo and report any issues
func analyzeClusterRepo(clusterRoot string, clusterRepo clusterRepo, issueReporter *report.IssueReporter) error {

	var messages []string
	var subMessage string
	for _, condition := range clusterRepo.Status.Conditions {
		if condition.Status != corev1.ConditionTrue {
			switch condition.Type {
			case "Downloaded":
				subMessage = fmt.Sprintf("in repo %s on branch %s not downloaded", clusterRepo.Spec.GitRepo, clusterRepo.Spec.GitBranch)
			case "FollowerDownloaded":
				subMessage = "follower not downloaded"
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

	if len(messages) > 0 {
		messages = append([]string{fmt.Sprintf("Rancher %s resource %q", clusterRepoResource, clusterRepo.Name)}, messages...)
		issueReporter.AddKnownIssueMessagesFiles(report.RancherIssues, clusterRoot, messages, []string{})
	}

	return nil
}
