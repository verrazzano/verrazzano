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
	var message string
	for _, condition := range cluster.Status.Conditions {
		switch condition.Type {
		case "Ready":
			if condition.Reason == "" {
				condition.Reason = "Not Given"
			}
			message = fmt.Sprintf("Rancher cluster resource %q, displayed as %s, is not ready, reason is %s", cluster.Name, cluster.Spec.DisplayName, condition.Reason)
		case "Provisioning":
			message = fmt.Sprintf("Rancher cluster resource %q, displayed as %s, is not provisioning", cluster.Name, cluster.Spec.DisplayName)
		case "Provisioned":
			message = fmt.Sprintf("Rancher cluster resource %q, displayed as %s, is not provisioned", cluster.Name, cluster.Spec.DisplayName)
		case "Waiting":
			message = fmt.Sprintf("Rancher cluster resource %q, displayed as %s, is waiting", cluster.Name, cluster.Spec.DisplayName)
		case "Connected":
			message = fmt.Sprintf("Rancher cluster resource %q, displayed as %s, is not connected", cluster.Name, cluster.Spec.DisplayName)
		case "RKESecretsMigrated":
			message = fmt.Sprintf("Rancher cluster resource %q, displayed as %s, RKE secrets not migrated", cluster.Name, cluster.Spec.DisplayName)
		case "SecretsMigrated":
			message = fmt.Sprintf("Rancher cluster resource %q, displayed as %s, secrets not migrated", cluster.Name, cluster.Spec.DisplayName)
		case "NoMemoryPressure":
			message = fmt.Sprintf("Rancher cluster resource %q, displayed as %s, has memory pressure", cluster.Name, cluster.Spec.DisplayName)
		case "NoDiskPressure":
			message = fmt.Sprintf("Rancher cluster resource %q, displayed as %s, has disk pressure", cluster.Name, cluster.Spec.DisplayName)
		case "SystemAccountCreated":
			message = fmt.Sprintf("Rancher cluster resource %q, displayed as %s, system account not created", cluster.Name, cluster.Spec.DisplayName)
		case "SystemProjectCreated":
			message = fmt.Sprintf("Rancher cluster resource %q, displayed as %s, system project not created", cluster.Name, cluster.Spec.DisplayName)
		case "DefaultProjectCreated":
			message = fmt.Sprintf("Rancher cluster resource %q, displayed as %s, default project not created", cluster.Name, cluster.Spec.DisplayName)
		case "GlobalAdminsSynced":
			message = fmt.Sprintf("Rancher cluster resource %q, displayed as %s, global admins not synced", cluster.Name, cluster.Spec.DisplayName)
		case "ServiceAccountMigrated":
			message = fmt.Sprintf("Rancher cluster resource %q, displayed as %s, service account not migrated", cluster.Name, cluster.Spec.DisplayName)
		case "ServiceAccountSecretsMigrated":
			message = fmt.Sprintf("Rancher cluster resource %q, displayed as %s, service account secrets not migrated", cluster.Name, cluster.Spec.DisplayName)
		case "AgentDeployed":
			message = fmt.Sprintf("Rancher cluster resource %q, displayed as %s, agent not deployed", cluster.Name, cluster.Spec.DisplayName)
		case "CreatorMadeOwner":
			message = fmt.Sprintf("Rancher cluster resource %q, displayed as %s, creator not made owner", cluster.Name, cluster.Spec.DisplayName)
		default:
			fmt.Println(condition)
		}
		if condition.Status != corev1.ConditionTrue {
			messages = append([]string{message}, messages...)
		}
	}

	issueReporter.AddKnownIssueMessagesFiles(report.KontainerDriverNotReady, clusterRoot, messages, []string{})

	return nil
}
