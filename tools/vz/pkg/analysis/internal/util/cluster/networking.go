package cluster

import (
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/files"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/report"
	"go.uber.org/zap"
	"regexp"
	"strings"
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

	return nil
}

// reportVZClientHangingIssue reports when a VZ Client issue has occurred due to certificate approval
func reportTCPKeepIdleHasOccuredIssue(clusterRoot string, issueReporter *report.IssueReporter, listOfFilesWhereErrorIsFoundInPodLogs []string) {
	files := listOfFilesWhereErrorIsFoundInPodLogs
	initialMessageString := "Issues regarding TCP Keep Idle have occured In the istio-system namespace in these files: "
	for _, filename := range files {
		initialMessageString = initialMessageString + filename + ", "
	}
	// This removes the uncessary ", " that would occur at the end of the string
	sanitizedMessageString := initialMessageString[0 : len(initialMessageString)-3]
	message := []string{sanitizedMessageString}
	issueReporter.AddKnownIssueMessagesFiles(report.VZClientHangingIssueDueToLongCertificateApproval, clusterRoot, message, files)
}
func determineIfTCPKeepIdleIssueHasOccurred(log *zap.SugaredLogger, clusterRoot string) (bool, []string, error) {
	// Check for files in the cluster-snapshot that match this pattern, this is the pattern for a pod log in the istio-system namespace
	listOfIstioPodsWithTCPKeepIdleIssues := []string{}
	istioPodLogRegExp := regexp.MustCompile(`istio-system/*/logs.txt`)
	allIstioPodLogFiles, err := files.GetMatchingFiles(log, clusterRoot, istioPodLogRegExp)
	if err != nil {
		return false, nil, err
	}
	// If no istioPodlogs are in the namespace, return false, as this issue can only be seen through logs
	if len(allIstioPodLogFiles) == 0 {
		return false, nil, nil
	}
	// If pod logs are found, go through each IstioPodLog file and each of the messages in that file to find a match for that issue
	numberOfIstioPodsWithTCPKeepIdleIssues := 0
	for _, istioPodLogFile := range allIstioPodLogFiles {
		istioPodLogList, err := files.ConvertToLogMessage(istioPodLogFile)
		if err != nil {
			log.Error("Failed to convert files to the vpo message")
			return false, nil, err
		}
		for _, istioPodLog := range istioPodLogList {
			istioPodLogMessage := istioPodLog.Message
			if strings.Contains(istioPodLogMessage, "Setting IPPROTO_TCP/TCP_KEEPIDLE option on socket failed") {
				listOfIstioPodsWithTCPKeepIdleIssues = append(listOfIstioPodsWithTCPKeepIdleIssues, istioPodLogFile)
				numberOfIstioPodsWithTCPKeepIdleIssues = numberOfIstioPodsWithTCPKeepIdleIssues + 1
				break
			}
		}
		if numberOfIstioPodsWithTCPKeepIdleIssues > 0 {
			return true, listOfIstioPodsWithTCPKeepIdleIssues, nil
		}
	}
	return false, nil, nil
}
