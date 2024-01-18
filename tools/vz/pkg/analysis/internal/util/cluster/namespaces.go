// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Package cluster handles cluster analysis
package cluster

import (
	encjson "encoding/json"
	"fmt"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/files"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/report"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"go.uber.org/zap"
	"io"
	corev1 "k8s.io/api/core/v1"
	"os"
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
	for _, namespace := range allNamespacesFound {
		namespaceFile := files.FindFileInNamespace(clusterRoot, namespace, constants.NamespaceJSON)
		namespaceObject, err := getNamespaceResource(log, namespaceFile)
		if err != nil {
			return err
		}
		if isNamespaceCurrentlyInTerminatingStatus(namespaceObject) {
			reportNamespaceInTerminatingStatusIssue(clusterRoot, *namespaceObject, &issueReporter, namespaceFile)

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
		log.Error("Failed reading Certificates.json file %s", path)
		return nil, err
	}
	err = encjson.Unmarshal(fileBytes, &namespaceResource)
	if err != nil {
		log.Error("Failed to unmarshal CertificateList at %s", path)
		return nil, err
	}
	return namespaceResource, err
}

// isNamespaceCurrentlyInTerminatingStatus checks to see if that is the namespace currently has a status of terminating
func isNamespaceCurrentlyInTerminatingStatus(namespaceObject *corev1.Namespace) bool {
	return namespaceObject.Status.Phase == corev1.NamespaceTerminating
}
func reportNamespaceInTerminatingStatusIssue(clusterRoot string, namespace corev1.Namespace, issueReporter *report.IssueReporter, namespaceFile string) {
	files := []string{namespaceFile}
	message := []string{fmt.Sprintf("The namespace %s is currently in a state of terminating", namespace.ObjectMeta.Name)}
	issueReporter.AddKnownIssueMessagesFiles(report.VZClientHangingIssueDueToLongCertificateApproval, clusterRoot, message, files)

}
