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
type provisioningClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []provisioningCluster `json:"items"`
}
type provisioningCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Status            provisioningStatus `json:"status,omitempty"`
}
type provisioningStatus struct {
	Conditions []cattleCondition `json:"conditions,omitempty"`
	Ready      bool              `json:"ready,omitempty"`
}

// AnalyzeProvisioningClusters - analyze the status of Rancher provisioning clusters resources
func AnalyzeProvisioningClusters(log *zap.SugaredLogger, clusterRoot string, issueReporter *report.IssueReporter) error {
	list := &provisioningClusterList{}
	err := files.UnmarshallFileInClusterRoot(clusterRoot, "cluster.provisioning.cattle.io.json", list)
	if err != nil {
		return err
	}

	for _, cluster := range list.Items {
		err = analyzeProvisioningCluster(clusterRoot, cluster, issueReporter)
		if err != nil {
			return err
		}
	}

	issueReporter.Contribute(log, clusterRoot)
	return nil
}

// analyzeProvisioningCluster - analyze a single Rancher provisioning cluster and report any issues
func analyzeProvisioningCluster(clusterRoot string, cluster provisioningCluster, issueReporter *report.IssueReporter) error {

	var messages []string
	var subMessage string
	for _, condition := range cluster.Status.Conditions {
		if condition.Status != corev1.ConditionTrue {
			switch condition.Type {
			case "Waiting":
				subMessage = "is waiting"
			case "Created":
				subMessage = "is not created"
			case "Provisioned":
				subMessage = "is not provisioned"
			case "BackingNamespaceCreated":
				subMessage = "backing namespace not created"
			case "DefaultProjectCreated":
				subMessage = "default project not created"
			case "SystemProjectCreated":
				subMessage = "system project not created"
			case "InitialRolesPopulated":
				subMessage = "initial roles not populated"
			case "CreatorMadeOwner":
				subMessage = "creator not made owner"
			case "Connected":
				subMessage = "is not connected"
			case "NoDiskPressure":
				subMessage = "has disk pressure"
			case "NoMemoryPressure":
				subMessage = "has memory pressure"
			case "SecretsMigrated":
				subMessage = "secrets not migrated"
			case "ServiceAccountSecretsMigrated":
				subMessage = "service account secrets not migrated"
			case "RKESecretsMigrated":
				subMessage = "RKE secrets not migrated"
			case "SystemAccountCreated":
				subMessage = "system account not created"
			case "AgentDeployed":
				subMessage = "agent not deployed"
			case "Ready":
				subMessage = "is not ready"
			case "ServiceAccountMigrated":
				subMessage = "service account not migrated"
			case "GlobalAdminsSynced":
				subMessage = "global admins not synced"
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
			message = fmt.Sprintf("Rancher provisioning cluster resource %q in namespace %s %s%s%s", cluster.Name, cluster.Namespace, subMessage, reason, msg)
			messages = append([]string{message}, messages...)
		}
	}

	if len(messages) > 0 {
		issueReporter.AddKnownIssueMessagesFiles(report.RancherIssues, clusterRoot, messages, []string{})
	}

	return nil
}
