// Copyright (c) 2023, 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"fmt"
	"path/filepath"

	"github.com/verrazzano/verrazzano/tools/vz/pkg/internal/util/files"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/internal/util/report"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const catalogAppResource = "app.catalog.cattle.io"

// Minimal definition of object that only contains the fields that will be analyzed
type catalogAppList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []catalogApp `json:"items"`
}
type catalogApp struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Status            catalogAppStatus `json:"status,omitempty"`
}
type catalogAppStatus struct {
	Summary *catalogAppSummary `json:"summary,omitempty"`
}

type catalogAppSummary struct {
	Error         bool   `json:"error,omitempty"`
	State         string `json:"state,omitempty"`
	Transitioning bool   `json:"transitioning,omitempty"`
}

// AnalyzeCatalogApps - analyze the status of CatalogApp objects
func AnalyzeCatalogApps(clusterRoot string, namespace string, issueReporter *report.IssueReporter) error {
	resourceRoot := clusterRoot
	if len(namespace) != 0 {
		resourceRoot = filepath.Join(clusterRoot, namespace)
	}

	list := &catalogAppList{}
	err := files.UnmarshallFileInClusterRoot(resourceRoot, fmt.Sprintf("%s.json", catalogAppResource), list)
	if err != nil {
		return err
	}

	for _, catalogApp := range list.Items {
		err = analyzeCatalogApp(clusterRoot, catalogApp, issueReporter)
		if err != nil {
			return err
		}
	}

	return nil
}

// analyzeCatalogApp - analyze a single CatalogApp and report any issues
func analyzeCatalogApp(clusterRoot string, catalogApp catalogApp, issueReporter *report.IssueReporter) error {

	var messages []string

	summary := catalogApp.Status.Summary
	if summary != nil && summary.Error {
		message := fmt.Sprintf("Rancher %s resource %q in namespace %s is in state %s", catalogAppResource, catalogApp.Name, catalogApp.Namespace, catalogApp.Status.Summary.State)
		messages = append([]string{message}, messages...)
	}

	if len(messages) > 0 {
		issueReporter.AddKnownIssueMessagesFiles(report.RancherIssues, clusterRoot, messages, []string{})
	}

	return nil
}
