// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cluster

import (
	"fmt"
	"strings"

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

	var err error
	var errors []string
	analyzers := []func(clusterRoot string, issueReporter *report.IssueReporter) error{
		rancher.AnalyzeClusterRepos, rancher.AnalyzeCatalogs, rancher.AnalyzeProvisioningClusters,
		rancher.AnalyzeKontainerDrivers, rancher.AnalyzeBundleDeployments, rancher.AnalyzeManagementClusters,
		rancher.AnalyzeBundles, rancher.AnalyzeClusterGroups, rancher.AnalyzeClusterRegistrations,
		rancher.AnalyzeFleetClusters, rancher.AnalyzeNodes,
	}

	for _, analyze := range analyzers {
		if err = analyze(clusterRoot, &issueReporter); err != nil {
			errors = append(errors, err.Error())
		}
	}

	issueReporter.Contribute(log, clusterRoot)

	if len(errors) > 0 {
		return fmt.Errorf("Errors analyzing Rancher: %s", fmt.Sprintf(strings.Join(errors[:], ",")))
	}

	return nil
}
