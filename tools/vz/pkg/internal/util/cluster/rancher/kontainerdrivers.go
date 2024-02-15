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

const kontainerDriverResource = "kontainerdriver.management.cattle.io"

// KontainerDriverList has the minimal definition of KontainerDriver object.
type KontainerDriverList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KontainerDriver `json:"items"`
}
type KontainerDriver struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Status            KontainerDriverStatus `json:"status,omitempty"`
}
type KontainerDriverStatus struct {
	Conditions []KontainerDriverCondition `json:"conditions,omitempty"`
}
type KontainerDriverCondition struct {
	Status corev1.ConditionStatus `json:"status"`
	Type   string                 `json:"type"`
}

// AnalyzeKontainerDrivers handles the checking of the status of KontainerDriver resources.
func AnalyzeKontainerDrivers(clusterRoot string, namespace string, issueReporter *report.IssueReporter) error {
	resourceRoot := clusterRoot
	if len(namespace) != 0 {
		resourceRoot = filepath.Join(clusterRoot, namespace)
	}

	kontainerDriverList := &KontainerDriverList{}
	err := files.UnmarshallFileInClusterRoot(resourceRoot, fmt.Sprintf("%s.json", kontainerDriverResource), kontainerDriverList)
	if err != nil {
		return fmt.Errorf("failed to unmarshal KontainerDriver list from cluster snapshot: %s", err)
	}

	// Analyze each KontainerDriver resource.
	for _, kontainerDriver := range kontainerDriverList.Items {
		reportKontainerDriverIssue(clusterRoot, kontainerDriver, issueReporter)
	}

	return nil
}

// reportKontainerDriverIssue will check the ociocneengine and oraclecontainerengine KontainerDriver resources and
// report any issues that are found with them.
func reportKontainerDriverIssue(clusterRoot string, driver KontainerDriver, issueReporter *report.IssueReporter) error {
	var messages []string
	if driver.Name == "ociocneengine" || driver.Name == "oraclecontainerengine" {
		for _, condition := range driver.Status.Conditions {
			switch condition.Type {
			case "Active", "Downloaded", "Installed":
				if condition.Status != "True" {
					messages = append(messages, fmt.Sprintf("\tcondition type \"%s\" has a status of \"%s\"", condition.Type, condition.Status))
				}
			}
		}

		if len(messages) != 0 {
			messages = append([]string{fmt.Sprintf("Rancher %s resource \"%s\" is not ready per it's resource status", kontainerDriverResource, driver.Name)}, messages...)
			issueReporter.AddKnownIssueMessagesFiles(report.RancherIssues, clusterRoot, messages, []string{})
		}
	}

	return nil
}
