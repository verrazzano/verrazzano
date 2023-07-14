// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cluster

import (
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/cluster/rancher"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/report"
	"go.uber.org/zap"
)

// AnalyzeRancher handles the checking of the status of Rancher resources.
func AnalyzeRancher(log *zap.SugaredLogger, clusterRoot string) error {
	log.Debugf("AnalyzeRancher called for %s", clusterRoot)

	var issueReporter = report.IssueReporter{
		PendingIssues: make(map[string]report.Issue),
	}

	err := rancher.AnalyzeClusterRepos(log, clusterRoot, &issueReporter)
	if err != nil {
		return err
	}
	err = rancher.AnalyzeKontainerDrivers(log, clusterRoot, &issueReporter)
	if err != nil {
		return err
	}
	err = rancher.AnalyzeRancherClusters(log, clusterRoot, &issueReporter)
	if err != nil {
		return err
	}

	issueReporter.Contribute(log, clusterRoot)

	return nil
}
