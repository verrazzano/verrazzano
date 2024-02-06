// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package cluster

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/files"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/report"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// AnalyzeMySQLRelatedIssues is the initial entry function for mySQL related issues, and it returns an error.
// It checks to see whether an innoDBCluster is in a state of terminating, and reports an issue based on the length of its termination
func AnalyzeMySQLRelatedIssues(log *zap.SugaredLogger, clusterRoot string) (err error) {
	allNamespacesFound, err = files.FindNamespaces(log, clusterRoot)
	if err != nil {
		return err
	}
	var issueReporter = report.IssueReporter{
		PendingIssues: make(map[string]report.Issue),
	}
	timeOfCapture, err := files.GetTimeOfCapture(log, clusterRoot)
	if err != nil {
		return err
	}
	for _, namespace := range allNamespacesFound {
		if namespace != "keycloak" {
			continue
		}
		innoDBClusterFile := files.FormFilePathInNamespace(clusterRoot, namespace, constants.InnoDBClusterJSON)
		innoDBResourceList, err := getInnoDBClusterResources(log, innoDBClusterFile)
		if err != nil {
			return err
		}
		if innoDBResourceList == nil {
			continue
		}
		for _, item := range innoDBResourceList.Items {
			isTerminating, message := isInnoDBClusterCurrentlyInTerminatingStatus(&item, timeOfCapture)
			if isTerminating {
				reportInnoDBClustersInTerminatingStatusIssue(clusterRoot, &issueReporter, innoDBClusterFile, message)

			}
		}

	}

	issueReporter.Contribute(log, clusterRoot)
	return nil
}

// getInnoDBClusterResource returns the InnoDBCluster list that is in the inno-db-cluster.json file
func getInnoDBClusterResources(log *zap.SugaredLogger, path string) (innoDBClusterObject *unstructured.UnstructuredList, err error) {
	resourceToReturn := unstructured.UnstructuredList{}
	file, err := os.Open(path)
	defer file.Close()
	if err != nil {
		log.Debug("file %s not found", path)
		return nil, nil
	}
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		log.Error("Failed reading namespace.json file %s", path)
		return nil, err
	}
	err = resourceToReturn.UnmarshalJSON(fileBytes)
	if err != nil {
		log.Error("Failed to unmarshal namespace resource at %s", path)
		return nil, err
	}
	return &resourceToReturn, err
}

// isNamespaceCurrentlyInTerminatingStatus checks to see if that is the namespace currently has a status of terminating
func isInnoDBClusterCurrentlyInTerminatingStatus(innoDBClusterResource *unstructured.Unstructured, timeOfCapture *time.Time) (bool, string) {
	var deletionMessage string
	deletionTimestamp := innoDBClusterResource.GetDeletionTimestamp()
	if deletionTimestamp == nil {
		return false, deletionMessage
	}
	if deletionTimestamp != nil && timeOfCapture != nil {
		diff := timeOfCapture.Sub(deletionTimestamp.Time)
		deletionMessage = "The InnoDBCluster " + innoDBClusterResource.GetName() + " has spent " + fmt.Sprint(int(diff.Minutes())) + " minutes and " + fmt.Sprint(int(diff.Seconds())%60) + " seconds deleting"
	} else {
		deletionMessage = "The InnoDBCluster " + innoDBClusterResource.GetName() + " has spent an undetermined amount of time in a state of deletion"
	}
	return true, deletionMessage
}
func reportInnoDBClustersInTerminatingStatusIssue(clusterRoot string, issueReporter *report.IssueReporter, InnoDBClusterFile string, message string) {
	files := []string{InnoDBClusterFile}
	messageList := []string{message}
	if strings.Contains(message, "unknown") {
		issueReporter.AddKnownIssueMessagesFiles(report.InnoDBClusterResourceCurrentlyInTerminatingStatusUnknownDuration, clusterRoot, messageList, files)
	} else {
		issueReporter.AddKnownIssueMessagesFiles(report.InnoDBClusterResourceCurrentlyInTerminatingStatusKnownDuration, clusterRoot, messageList, files)
	}

}
