// Copyright (c) 2023, 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cluster

import (
	"fmt"
	"os"
	"strings"

	capi "github.com/verrazzano/verrazzano/tools/vz/pkg/internal/util/cluster/clusterapi"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/internal/util/report"
	"go.uber.org/zap"
)

// AnalyzeClusterAPI handles the checking of the status of Cluster API resources.
func AnalyzeClusterAPI(log *zap.SugaredLogger, clusterRoot string) error {
	log.Debugf("AnalyzeClusterAPI called for %s", clusterRoot)

	var issueReporter = report.IssueReporter{
		PendingIssues: make(map[string]report.Issue),
	}

	var err error
	var errors []string

	// First, process the cluster scoped resources.
	analyzers := []func(clusterRoot string, namespace string, issueReporter *report.IssueReporter) error{}
	for _, analyze := range analyzers {
		if err = analyze(clusterRoot, "", &issueReporter); err != nil {
			errors = append(errors, err.Error())
		}
	}

	// Second, process the namespaced resources.
	namespaceAnalyzers := []func(clusterRoot string, namespace string, issueReporter *report.IssueReporter) error{
		capi.AnalyzeClusters, capi.AnalyzeOCIClusters, capi.AnalyzeOCNEControlPlanes,
		capi.AnalyzeMachines, capi.AnalyzeMachineDeployments, capi.AnalyzeOCNEConfigs,
		capi.AnalyzeOCIMachines, capi.AnalyzeMachineSets, capi.AnalyzeClusterResourceSets,
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
		return fmt.Errorf("Errors analyzing Cluster API reources: %s", fmt.Sprintf(strings.Join(errors[:], ",")))
	}

	return nil
}
