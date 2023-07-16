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

// Minimal definition that only contains the fields that will be analyzed
type managementClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []managementCluster `json:"items"`
}
type managementCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              managementClusterSpec `json:"spec"`
	Status            cattleStatus          `json:"status,omitempty"`
}
type managementClusterSpec struct {
	DisplayName string `json:"displayName,omitempty"`
}

// AnalyzeManagementClusters - analyze the status of Rancher management clusters resources
func AnalyzeManagementClusters(log *zap.SugaredLogger, clusterRoot string, issueReporter *report.IssueReporter) error {
	list := &managementClusterList{}
	err := files.UnmarshallFileInClusterRoot(clusterRoot, "cluster.management.cattle.io.json", list)
	if err != nil {
		return err
	}

	for _, cluster := range list.Items {
		err = analyzeManagementCluster(clusterRoot, cluster, issueReporter)
		if err != nil {
			return err
		}
	}

	issueReporter.Contribute(log, clusterRoot)
	return nil
}

// analyzeManagementCluster - analyze a single Rancher management cluster and report any issues
func analyzeManagementCluster(clusterRoot string, cluster managementCluster, issueReporter *report.IssueReporter) error {

	var messages []string
	var subMessage string
	for _, condition := range cluster.Status.Conditions {
		if condition.Status != corev1.ConditionTrue {
			switch condition.Type {
			case "Ready":
				subMessage = "is not ready"
			case "Provisioning":
				subMessage = "is not provisioning"
			case "Provisioned":
				subMessage = "is not provisioned"
			case "Waiting":
				subMessage = "is waiting"
			case "Connected":
				subMessage = "is not connected"
			case "RKESecretsMigrated":
				subMessage = "RKE secrets not migrated"
			case "SecretsMigrated":
				subMessage = "secrets not migrated"
			case "NoMemoryPressure":
				subMessage = "has memory pressure"
			case "NoDiskPressure":
				subMessage = "has disk pressure"
			case "SystemAccountCreated":
				subMessage = "system account not created"
			case "SystemProjectCreated":
				subMessage = "system project not created"
			case "DefaultProjectCreated":
				subMessage = "default project not created"
			case "GlobalAdminsSynced":
				subMessage = "global admins not synced"
			case "ServiceAccountMigrated":
				subMessage = "service account not migrated"
			case "ServiceAccountSecretsMigrated":
				subMessage = "service account secrets not migrated"
			case "AgentDeployed":
				subMessage = "agent not deployed"
			case "CreatorMadeOwner":
				subMessage = "creator not made owner"
			case "InitialRolesPopulated":
				subMessage = "initial roles not populated"
			case "BackingNamespaceCreated":
				subMessage = "backing namespace not created"
			}
			// Add a message for the issue
			var message string
			reason := ""
			msg := ""
			if len(condition.Reason) > 0 {
				reason = fmt.Sprintf(", reason is %q", condition.Reason)
			}
			if len(condition.Message) > 0 {
				msg = fmt.Sprintf(", message is %q", condition.Message)
			}
			message = fmt.Sprintf("Rancher management cluster resource %q (displayed as %s) %s%s%s", cluster.Name, cluster.Spec.DisplayName, subMessage, reason, msg)
			messages = append([]string{message}, messages...)
		}
	}

	if len(messages) > 0 {
		issueReporter.AddKnownIssueMessagesFiles(report.RancherIssues, clusterRoot, messages, []string{})
	}

	return nil
}
