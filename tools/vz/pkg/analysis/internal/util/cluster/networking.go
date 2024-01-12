// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package cluster

import (
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/files"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/report"
	"go.uber.org/zap"
	"regexp"
)

func AnalyzeNetworkingIssues(log *zap.SugaredLogger, clusterRoot string) (err error) {
	if err != nil {
		return err
	}
	var issueReporter = report.IssueReporter{
		PendingIssues: make(map[string]report.Issue),
	}

	resultOfTCPKeepIdleHasOccured, listOfFiles, err := determineIfTCPKeepIdleIssueHasOccurred(log, clusterRoot)
	if err != nil {
		return err
	}
	if resultOfTCPKeepIdleHasOccured {
		reportTCPKeepIdleHasOccuredIssue(clusterRoot, &issueReporter, listOfFiles)
	}
	issueReporter.Contribute(log, clusterRoot)

	return nil
}

// reportVZClientHangingIssue reports when a VZ Client issue has occurred due to certificate approval
func reportTCPKeepIdleHasOccuredIssue(clusterRoot string, issueReporter *report.IssueReporter, listOfFilesWhereErrorIsFoundInPodLogs []string) {
	files := listOfFilesWhereErrorIsFoundInPodLogs
	initialMessageString := "Issues regarding TCP Keep Idle have occurred in the istio-system namespace in these files: "
	for _, filename := range files {
		initialMessageString = initialMessageString + filename + ", "
	}
	// This removes the unnecessary ", " that would occur at the end of the string
	sanitizedMessageString := initialMessageString[0 : len(initialMessageString)-2]
	message := []string{sanitizedMessageString}
	issueReporter.AddKnownIssueMessagesFiles(report.TCPKeepIdleIssues, clusterRoot, message, files)
}
func determineIfTCPKeepIdleIssueHasOccurred(log *zap.SugaredLogger, clusterRoot string) (bool, []string, error) {
	// Check for files in the cluster-snapshot that match this pattern, this is the pattern for a pod log in the istio-system namespace
	listOfIstioPodsWithTCPKeepIdleIssues := []string{}
	istioPodLogRegExp := regexp.MustCompile("istio-system/.*/logs.txt")
	regexpExpressionForError := regexp.MustCompile("Setting IPPROTO_TCP/TCP_KEEPIDLE option on socket failed")
	listOfMatches, err := files.FindFilesAndSearch(log, clusterRoot, istioPodLogRegExp, regexpExpressionForError, nil)
	if err != nil {
		return false, nil, err
	}
	// This generates a map of unique filenames that contain the error
	var uniqueFileNames = make(map[string]bool)
	for i := range listOfMatches {
		uniqueFileNames[listOfMatches[i].FileName] = true
	}
	if len(listOfMatches) == 0 {
		return false, nil, nil
	}
	// If a match/matches are found, iterate through the unique filenames and add it to the list of unique filenames
	for filename := range uniqueFileNames {
		listOfIstioPodsWithTCPKeepIdleIssues = append(listOfIstioPodsWithTCPKeepIdleIssues, filename)
	}
	return true, listOfIstioPodsWithTCPKeepIdleIssues, nil
}
