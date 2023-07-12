package cluster

import (
	"fmt"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/files"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/report"
	"go.uber.org/zap"
	"os"
)

// AnalyzeClusterAPIIssues handles the checking of cluster-api resources.
func AnalyzeClusterAPIIssues(log *zap.SugaredLogger, clusterRoot string) error {
	log.Debugf("AnalyzeClusterAPIIssues called for %s", clusterRoot)

	var issueReporter = report.IssueReporter{
		PendingIssues: make(map[string]report.Issue),
	}

	return analyzeClusterAPIIssues(log, clusterRoot, &issueReporter)
}

// analyzeClusterAPIIssues handles the checking of the status of KontainerDriver resources.
func analyzeClusterAPIIssues(log *zap.SugaredLogger, clusterRoot string, issueReporter *report.IssueReporter) error {
	namespaces, err := files.FindNamespaces(log, clusterRoot)
	if err != nil {
		// TODO: what to do about error handling...
		return err
	}

	for _, namespace := range namespaces {
		clusterPath := files.FindFileInNamespace(clusterRoot, namespace, "cluster.json")
		_, err := os.Stat(clusterPath)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			// TODO: what to do about error handling...
			msg := fmt.Sprintf("failed to access file %s: %s", clusterPath, err.Error())
			log.Errorf(msg)
			return fmt.Errorf(msg)
		}

		// check if this is the correct CAPI json file and handle multiple cluster resources in the same namespace
	}

	return nil
}
