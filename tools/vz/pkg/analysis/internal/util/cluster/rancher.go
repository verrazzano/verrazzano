// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cluster

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	corev1 "k8s.io/api/core/v1"

	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/files"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/report"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type rancherClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []rancherCluster `json:"items"`
}

// Minimal definition of cluster object that only contains the fields that will be analyzed
type rancherCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              rancherClusterSpec   `json:"spec"`
	Status            rancherClusterStatus `json:"status,omitempty"`
}

type rancherClusterSpec struct {
	DisplayName string `json:"displayName,omitempty"`
}
type rancherClusterStatus struct {
	Conditions []clusterCondition `json:"conditions,omitempty"`
}

type clusterCondition struct {
	Status corev1.ConditionStatus `json:"status"`
	Type   string                 `json:"type"`
	Reason string                 `json:"reason,omitempty"`
}

// AnalyzeRancher handles the checking of the status of Rancher resources.
func AnalyzeRancher(log *zap.SugaredLogger, clusterRoot string) error {
	log.Debugf("AnalyzeRancher called for %s", clusterRoot)

	var issueReporter = report.IssueReporter{
		PendingIssues: make(map[string]report.Issue),
	}

	return analyzeClusters(log, clusterRoot, &issueReporter)
}

// analyzeClusters - analyze the status of Rancher clusters
func analyzeClusters(log *zap.SugaredLogger, clusterRoot string, issueReporter *report.IssueReporter) error {
	clusterPath := files.FindFileInClusterRoot(clusterRoot, "default/cluster.json")

	// Parse the json into local struct
	file, err := os.Open(clusterPath)
	if err != nil {
		log.Debugf("file %s not found", clusterPath)
		return err
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		log.Debugf("Failed reading Json file %s", clusterPath)
		return err
	}
	clusterList := &rancherClusterList{}
	err = json.Unmarshal(fileBytes, &clusterList)
	if err != nil {
		log.Debugf("Failed to unmarshal Rancher Cluster list at %s", clusterPath)
		return err
	}

	for _, cluster := range clusterList.Items {
		err = reportClusterIssue(log, clusterRoot, cluster, issueReporter)
		if err != nil {
			return err
		}
	}

	issueReporter.Contribute(log, clusterRoot)
	return nil
}

// reportClusterIssue - analyze a single Rancher cluster and report any issues
func reportClusterIssue(log *zap.SugaredLogger, clusterRoot string, cluster rancherCluster, issueReporter *report.IssueReporter) error {

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
			if condition.Status != corev1.ConditionTrue {
				var message string
				if len(condition.Reason) == 0 {
					message = fmt.Sprintf("Rancher cluster resource %q, displayed as %s, %s", cluster.Name, cluster.Spec.DisplayName, subMessage)
				} else {
					message = fmt.Sprintf("Rancher cluster resource %q, displayed as %s, %s - reason is %s", cluster.Name, cluster.Spec.DisplayName, subMessage, condition.Reason)
				}
				messages = append([]string{message}, messages...)
			}
		}
	}

	issueReporter.AddKnownIssueMessagesFiles(report.KontainerDriverNotReady, clusterRoot, messages, []string{})

	return nil
}
