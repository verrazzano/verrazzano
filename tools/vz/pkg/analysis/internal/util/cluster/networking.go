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
	initialMessageString := "Issues regarding TCP Keep Idle have occured in the istio-system namespace in these files: "
	for _, filename := range files {
		initialMessageString = initialMessageString + filename + ", "
	}
	// This removes the uncessary ", " that would occur at the end of the string
	sanitizedMessageString := initialMessageString[0 : len(initialMessageString)-2]
	message := []string{sanitizedMessageString}
	issueReporter.AddKnownIssueMessagesFiles(report.TCPKeepIdleIssues, clusterRoot, message, files)
}
func determineIfTCPKeepIdleIssueHasOccurred(log *zap.SugaredLogger, clusterRoot string) (bool, []string, error) {
	// Check for files in the cluster-snapshot that match this pattern, this is the pattern for a pod log in the istio-system namespace
	listOfIstioPodsWithTCPKeepIdleIssues := []string{}
	istioPodLogRegExp := regexp.MustCompile("istio-system/.*/logs.txt")
	allIstioPodLogFiles, err := files.GetMatchingFiles(log, clusterRoot, istioPodLogRegExp)
	if err != nil {
		return false, nil, err
	}
	// If no istioPodlogs are in the namespace, return false, as this issue can only be seen through logs
	if len(allIstioPodLogFiles) == 0 {
		return false, nil, nil
	}
	// If pod logs are found, go through each istioPodLog file and each of the messages in that file to find a match for that issue
	numberOfIstioPodsWithTCPKeepIdleIssues := 0
	regexpExpressionForError := regexp.MustCompile("Setting IPPROTO_TCP/TCP_KEEPIDLE option on socket failed")
	for _, istioPodLogFile := range allIstioPodLogFiles {
		istioPodLogMatches, err := files.SearchFile(log, istioPodLogFile, regexpExpressionForError, nil)
		if err != nil {
			log.Error("Failed to convert files to the vpo message")
			return false, nil, err
		}
		// If a match is found, add the file that it was found to the list of pods with the issue and increment the number of pods that have this issue
		if len(istioPodLogMatches) > 0 {
			listOfIstioPodsWithTCPKeepIdleIssues = append(listOfIstioPodsWithTCPKeepIdleIssues, istioPodLogFile)
			numberOfIstioPodsWithTCPKeepIdleIssues = numberOfIstioPodsWithTCPKeepIdleIssues + 1
		}
	}
	if numberOfIstioPodsWithTCPKeepIdleIssues > 0 {
		return true, listOfIstioPodsWithTCPKeepIdleIssues, nil
	}
	return false, nil, nil
}
