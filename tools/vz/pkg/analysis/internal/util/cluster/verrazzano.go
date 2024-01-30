// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Package cluster handles cluster analysis
package cluster

import (
	"strings"

	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/files"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/report"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
)

// TODO: Helpers to access this info as needed

// allNamespacesFound is a list of the namespaces found
var allNamespacesFound []string

// verrazzanoNamespacesFound is a list of the Verrazzano namespaces found
var verrazzanoNamespacesFound []string

// TODO: CRDs related to verrazzano
// TODO: Can we determine the underlying platform that is being used? This may generally help in terms
//       of the analysis (ie: message formatting), but it also is generally useful in terms of how we
//       provide action advice as well. Inspecting the nodes.json seems like the a good place to determine this

// verrazzanoDeployments related to verrazzano
var verrazzanoDeployments = make(map[string]appsv1.Deployment)
var problematicVerrazzanoDeploymentNames = make([]string, 0)

var verrazzanoAnalysisFunctions = map[string]func(log *zap.SugaredLogger, clusterRoot string, issueReporter *report.IssueReporter) (err error){
	"Verrazzano Resource Status": AnalyzeVerrazzanoResource,
	"Installation status":        installationStatus,
}

// AnalyzeVerrazzano handles high level checking for Verrazzano itself. Note that we are not necessarily going to drill deeply here and
// we may actually handle scenarios as part of the other drill-downs separately
func AnalyzeVerrazzano(log *zap.SugaredLogger, clusterRoot string) (err error) {
	log.Debugf("AnalyzeVerrazzano called for %s", clusterRoot)

	var issueReporter = report.IssueReporter{
		PendingIssues: make(map[string]report.Issue),
	}

	// Call the Verrazzano analysis functions
	for functionName, function := range verrazzanoAnalysisFunctions {
		err := function(log, clusterRoot, &issueReporter)
		if err != nil {
			// Log the error and continue on
			log.Errorf("Error processing analysis function %s", functionName, err)
		}
	}
	issueReporter.Contribute(log, clusterRoot)
	return nil
}

// Determine the state of the Verrazzano Installation
func installationStatus(log *zap.SugaredLogger, clusterRoot string, issueReporter *report.IssueReporter) (err error) {
	// TODO: Is verrazzano:
	//      installed, installed-but-not-running, uninstalled-success-no-cruft, failed-install, failed-uninstall,
	//      uninstall-success-but-cruft-remaining, etc...
	// The intention is that we should at least give an Informational on what the state is.

	// Enumerate the namespaces that we found overall and the Verrazzano specific ones separately
	// Also look at the deployments in the Verrazzano related namespaces
	allNamespacesFound, err = files.FindNamespaces(log, clusterRoot)
	if err != nil {
		return err
	}

	for _, namespace := range allNamespacesFound {
		// These are Verrazzano owned namespaces
		if strings.Contains(namespace, "verrazzano") {
			verrazzanoNamespacesFound = append(verrazzanoNamespacesFound, namespace)
			deploymentList, err := GetDeploymentList(log, files.FormFilePathInNamespace(clusterRoot, namespace, constants.DeploymentsJSON))
			if err != nil {
				// Log the error and continue on
				log.Debugf("Error getting deployments in %s", namespace, err)
			}
			if deploymentList != nil && len(deploymentList.Items) > 0 {
				for i, deployment := range deploymentList.Items {
					verrazzanoDeployments[deployment.ObjectMeta.Name] = deployment
					if IsDeploymentProblematic(&deploymentList.Items[i]) {
						problematicVerrazzanoDeploymentNames = append(problematicVerrazzanoDeploymentNames, deployment.ObjectMeta.Name)
					}
				}
			}
		}
		// TBD: For now not enumerating out potentially related namespaces that could be here even
		// without Verrazzano (cattle, keycloak, etc...). Those will still be in the AllNamespacesFound if present
		// so until there is an explicit need to separate those, not doing that here (we could though)
	}

	// TODO: Inspect the verrazzano-install namespace platform operator logs. We should be able to glean state from the
	//       the logs here, and what the name of the install job resource to look for is.
	// TODO: Inspect the default namespace for a Verrazzano install job pod logs. Inspecting the logs should here should
	//       tell us whether an install/uninstall was done and what state it thinks it is in. NOTE, a user can name this
	//       how they want, so use the resource gleaned above on what to look for here.
	// TODO: Inspect the verrazzano-system namespace. The deployments/status here will tell us what we need to fan out
	//       and drill into
	// TODO: Inspect the verrazzano-mc namespace (TBD)

	// TODO: verrazzanoApiResourceMatches := files.SearchFile(log, files.FindFileInCluster(cluserRoot, "api_resources.out"), ".*verrazzano.*")
	// TODO: verrazzanoResources (json file)

	// Get more details on problematicVerrazzanoDeploymentNames and find a way to report
	return nil
}
