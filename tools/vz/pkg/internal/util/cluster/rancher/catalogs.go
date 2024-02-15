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

const catalogResource = "catalog.management.cattle.io"

// Minimal definition of object that only contains the fields that will be analyzed
type catalogsList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []catalog `json:"items"`
}
type catalog struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              catalogSpec  `json:"spec,omitempty"`
	Status            cattleStatus `json:"status,omitempty"`
}
type catalogSpec struct {
	Branch string `json:"branch,omitempty"`
	URL    string `json:"url,omitempty"`
}

// AnalyzeCatalogs - analyze the status of Catalog objects
func AnalyzeCatalogs(clusterRoot string, namespace string, issueReporter *report.IssueReporter) error {
	resourceRoot := clusterRoot
	if len(namespace) != 0 {
		resourceRoot = filepath.Join(clusterRoot, namespace)
	}

	list := &catalogsList{}
	err := files.UnmarshallFileInClusterRoot(resourceRoot, fmt.Sprintf("%s.json", catalogResource), list)
	if err != nil {
		return err
	}

	for _, catalog := range list.Items {
		err = analyzeCatalog(clusterRoot, catalog, issueReporter)
		if err != nil {
			return err
		}
	}

	return nil
}

// analyzeCatalog - analyze a single Catalog and report any issues
func analyzeCatalog(clusterRoot string, catalog catalog, issueReporter *report.IssueReporter) error {

	var messages []string
	var subMessage string
	for _, condition := range catalog.Status.Conditions {
		if condition.Status != corev1.ConditionTrue {
			switch condition.Type {
			case "SecretsMigrated":
				subMessage = "secrets not migrated"
			case "Refreshed":
				subMessage = "not refreshed"
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
		messages = append([]string{fmt.Sprintf("Rancher %s resource %q on branch %s with URL %s", catalogResource, catalog.Name, catalog.Spec.Branch, catalog.Spec.URL)}, messages...)
		issueReporter.AddKnownIssueMessagesFiles(report.RancherIssues, clusterRoot, messages, []string{})
	}

	return nil
}
