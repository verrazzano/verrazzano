// Copyright (c) 2023, 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cluster

import (
	"fmt"
	"os"
	"strings"

	"github.com/verrazzano/verrazzano/tools/vz/pkg/internal/util/cluster/rancher"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/internal/util/report"
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

	// First, process the cluster scoped resources.
	analyzers := []func(clusterRoot string, namespace string, issueReporter *report.IssueReporter) error{
		rancher.AnalyzeClusterRepos, rancher.AnalyzeCatalogs,
		rancher.AnalyzeKontainerDrivers, rancher.AnalyzeManagementClusters,
	}
	for _, analyze := range analyzers {
		if err = analyze(clusterRoot, "", &issueReporter); err != nil {
			errors = append(errors, err.Error())
		}
	}

	// Second, process the namespaced resources.
	namespaceAnalyzers := []func(clusterRoot string, namespace string, issueReporter *report.IssueReporter) error{
		rancher.AnalyzeProvisioningClusters, rancher.AnalyzeBundleDeployments,
		rancher.AnalyzeBundles, rancher.AnalyzeClusterGroups, rancher.AnalyzeClusterRegistrations,
		rancher.AnalyzeFleetClusters, rancher.AnalyzeCatalogApps, rancher.AnalyzeNodes,
		rancher.AnalyzeGitRepos, rancher.AnalyzeGitJobs, rancher.AnalyzeManagedCharts,
	}
	snapshotFiles, err := os.ReadDir(clusterRoot)
	if err != nil {
		return err
	}
	for _, f := range snapshotFiles {
		if f.IsDir() {
			for _, analyze := range namespaceAnalyzers {
				if err = analyze(clusterRoot, f.Name(), &issueReporter); err != nil {
					errors = append(errors, err.Error())
				}
			}
		}
	}

	issueReporter.Contribute(log, clusterRoot)

	if len(errors) > 0 {
		return fmt.Errorf("Errors analyzing Rancher: %s", fmt.Sprintf(strings.Join(errors[:], ",")))
	}

	return nil
}
