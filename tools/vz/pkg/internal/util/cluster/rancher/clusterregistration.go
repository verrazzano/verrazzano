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

const clusterRegistrationResource = "clusterregistration.fleet.cattle.io"

// Minimal definition of object that only contains the fields that will be analyzed
type clusterRegistrationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []clusterRegistration `json:"items"`
}
type clusterRegistration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Status            clusterRegistrationStatus `json:"status,omitempty"`
}
type clusterRegistrationStatus struct {
	ClusterName string `json:"clusterName,omitempty"`
	Granted     bool   `json:"granted,omitempty"`
}

// AnalyzeClusterRegistrations - analyze the status of ClusterRegistration objects
func AnalyzeClusterRegistrations(clusterRoot string, namespace string, issueReporter *report.IssueReporter) error {
	resourceRoot := clusterRoot
	if len(namespace) != 0 {
		resourceRoot = filepath.Join(clusterRoot, namespace)
	}

	list := &clusterRegistrationList{}
	err := files.UnmarshallFileInClusterRoot(resourceRoot, fmt.Sprintf("%s.json", clusterRegistrationResource), list)
	if err != nil {
		return err
	}

	for _, clusterRegistration := range list.Items {
		err = analyzeClusterRegistration(clusterRoot, clusterRegistration, issueReporter)
		if err != nil {
			return err
		}
	}

	return nil
}

// analyzeClusterRegistration - analyze a single ClusterRegistration and report any issues
func analyzeClusterRegistration(clusterRoot string, clusterRegistration clusterRegistration, issueReporter *report.IssueReporter) error {
	var messages []string

	if !clusterRegistration.Status.Granted {
		message := fmt.Sprintf("Rancher %s resource %q in namespace %s is not granted for cluster %s", clusterRegistrationResource, clusterRegistration.Name, clusterRegistration.Namespace, clusterRegistration.Status.ClusterName)
		messages = append([]string{message}, messages...)
	}

	if len(messages) > 0 {
		issueReporter.AddKnownIssueMessagesFiles(report.RancherIssues, clusterRoot, messages, []string{})
	}

	return nil
}
