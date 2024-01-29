// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Package cluster handles cluster analysis
package cluster

import (
	encjson "encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/files"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/report"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
)

// AnalyzeNamespaceRelatedIssues is the initial entry function for namespace related issues, and it returns an error.
// It checks to see whether the namespace being analyzed is in a state of terminating
func AnalyzeNamespaceRelatedIssues(log *zap.SugaredLogger, clusterRoot string) (err error) {
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
		namespaceFile := files.FindFileInNamespace(clusterRoot, namespace, constants.NamespaceJSON)
		namespaceObject, err := getNamespaceResource(log, namespaceFile)
		if err != nil {
			return err
		}
		if namespaceObject == nil {
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

// getNamespaceResource returns the namespace object that is in the namespace file
func getNamespaceResource(log *zap.SugaredLogger, path string) (namespaceObject *corev1.Namespace, err error) {
	namespaceResource := &corev1.Namespace{}
	file, err := os.Open(path)
	if err != nil {
		log.Debug("file %s not found", path)
		return nil, nil
	}
	defer file.Close()
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		log.Error("Failed reading namespace.json file %s", path)
		return nil, err
	}
	err = encjson.Unmarshal(fileBytes, &namespaceResource)
	if err != nil {
		log.Error("Failed to unmarshal namespace resource at %s", path)
		return nil, err
	}
	return namespaceResource, err
}

// isNamespaceCurrentlyInTerminatingStatus checks to see if that is the namespace currently has a status of terminating
func isNamespaceCurrentlyInTerminatingStatus(namespaceObject *corev1.Namespace, timeOfCapture *time.Time) (bool, []string) {
	var listOfMessagesFromRelevantConditions = []string{}
	if namespaceObject.Status.Phase != corev1.NamespaceTerminating {
		return false, listOfMessagesFromRelevantConditions
	}
	var deletionMessage string
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
func reportNamespaceInTerminatingStatusIssue(clusterRoot string, namespace corev1.Namespace, issueReporter *report.IssueReporter, namespaceFile string, messagesFromConditions []string) {
	files := []string{namespaceFile}
	message := []string{fmt.Sprintf("The namespace %s is currently in a state of terminating", namespace.ObjectMeta.Name)}
	message = append(message, messagesFromConditions...)
	for i, _ := range messagesFromConditions {
		if strings.Contains(messagesFromConditions[i], "has spent an undetermined amount of time in a state of deletion") {
			issueReporter.AddKnownIssueMessagesFiles(report.NamespaceCurrentlyInTerminatingStateUnknownDuration, clusterRoot, message, files)
			return
		}
	}
	issueReporter.AddKnownIssueMessagesFiles(report.NamespaceCurrentlyInTerminatingStateKnownDuration, clusterRoot, message, files)

}
