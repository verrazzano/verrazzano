// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package cluster

import (
	"fmt"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/files"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/report"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"go.uber.org/zap"
	"io"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"os"
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
		issueFound, messageList := isNamespaceCurrentlyInTerminatingStatus(namespaceObject, timeOfCapture)
		if issueFound {
			reportNamespaceInTerminatingStatusIssue(clusterRoot, *namespaceObject, &issueReporter, namespaceFile, messageList)
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
func isInnoDBClusterCurrentlyInTerminatingStatus(innoDBClusterList *unstructured.UnstructuredList, timeOfCapture *time.Time) (bool, []string) {
	var listOfMessagesFromRelevantConditions = []string{}
	for _, item := range innoDBClusterList.Items {
		deletionTimestamp := item.GetDeletionTimestamp()
		if deletionTimestamp == nil {
			return false, listOfMessagesFromRelevantConditions
		}
		var deletionMessage string
		if deletionTimestamp != nil && timeOfCapture != nil {
			diff := timeOfCapture.Sub(deletionTimestamp.Time)
			deletionMessage = "The namespace " + item.GetName() + " has spent " + fmt.Sprint(int(diff.Minutes())) + " minutes and " + fmt.Sprint(int(diff.Seconds())%60) + " seconds deleting"
		} else {
			deletionMessage = "The namespace " + item.GetName() + " has spent an undetermined amount of time in a state of deletion"
		}
	}
	if namespaceObject.DeletionTimestamp != nil && timeOfCapture != nil {
		diff := timeOfCapture.Sub(namespaceObject.DeletionTimestamp.Time)
		deletionMessage = "The namespace " + namespaceObject.Name + " has spent " + fmt.Sprint(int(diff.Minutes())) + " minutes and " + fmt.Sprint(int(diff.Seconds())%60) + " seconds deleting"
	} else {
		deletionMessage = "The namespace " + namespaceObject.Name + " has spent an undetermined amount of time in a state of deletion"
	}
	listOfMessagesFromRelevantConditions = append(listOfMessagesFromRelevantConditions, deletionMessage)
	namespaceConditions := namespaceObject.Status.Conditions
	if namespaceConditions == nil {
		return true, listOfMessagesFromRelevantConditions
	}
	for i := range namespaceConditions {
		if namespaceConditions[i].Type == corev1.NamespaceFinalizersRemaining || namespaceConditions[i].Type == corev1.NamespaceContentRemaining {
			listOfMessagesFromRelevantConditions = append(listOfMessagesFromRelevantConditions, namespaceConditions[i].Message)
		}
	}
	return true, listOfMessagesFromRelevantConditions
}
