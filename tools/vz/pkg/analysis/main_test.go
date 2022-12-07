// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package analysis

import (
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/log"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/report"
	"testing"
)

// TestHandleMain Tests the handleMain function
// GIVEN a call to handleMain
// WHEN with valid/invalid inputs
// THEN exit codes returned are as expected
func TestHandleMain(t *testing.T) {
	// This is setting up the main.logger, do NOT set it as a var here (or you will get a nil reference running
	// the test)
	logger = log.GetDebugEnabledLogger()
	analyzerType = "cluster"
}

// TestAnalyzeBad Tests the main Analyze function
// GIVEN a call to Analyze
// WHEN with invalid inputs
// THEN errors are generated as expected
func TestExecuteAnalysisBadArgs(t *testing.T) {
	logger := log.GetDebugEnabledLogger()

	// Call the analyzer with an unknown type, give it a good cluster dump directory
	err := Analyze(logger, "badnamehere", "../test/cluster/image-pull-case1")
	assert.NotNil(t, err)
	// TODO: Check error message is what we expected here

}

// TestImagePullCase1 Tests that analysis of a cluster dump with image pull issues is handled
// GIVEN a call to analyze a cluster-snapshot
// WHEN the cluster-snapshot shows image pull issues
// THEN a report is generated with image pull issues identified
func TestImagePull(t *testing.T) {
	logger := log.GetDebugEnabledLogger()

	err := Analyze(logger, "cluster", "test/cluster/image-pull-case1")
	assert.Nil(t, err)

	reportedIssues := report.GetAllSourcesFilteredIssues(logger, true, 0, 0)
	assert.NotNil(t, reportedIssues)
	assert.True(t, len(reportedIssues) > 0)
	imagePullsFound := 0
	for _, issue := range reportedIssues {
		if issue.Type == report.ImagePullNotFound {
			imagePullsFound++
		}
	}
	assert.True(t, imagePullsFound > 0)
}

// TestInsufficientMemory Tests that analysis of a cluster dump with pods that failed due to insufficient memory
// GIVEN a call to analyze a cluster-snapshot
// WHEN the cluster-snapshot shows pods with insufficient memory problems
// THEN a report is generated with issues identified
func TestInsufficientMemory(t *testing.T) {
	logger := log.GetDebugEnabledLogger()

	err := Analyze(logger, "cluster", "test/cluster/insufficient-mem")
	assert.Nil(t, err)

	reportedIssues := report.GetAllSourcesFilteredIssues(logger, true, 0, 0)
	assert.NotNil(t, reportedIssues)
	assert.True(t, len(reportedIssues) > 0)
	issuesFound := 0
	for _, issue := range reportedIssues {
		if issue.Type == report.InsufficientMemory {
			issuesFound++
		}
	}
	assert.True(t, issuesFound > 0)
}

// TestProblemPodsNotReportedUninstall Tests that analysis of a cluster dump with pods that have unknown issues during
// uninstall, is handled
// GIVEN a call to analyze a cluster-snapshot
// WHEN the cluster-snapshot shows pods with problems that are not known issues
// THEN a report is generated with problem pod issues identified
func TestProblemPodsNotReportedUninstall(t *testing.T) {
	logger := log.GetDebugEnabledLogger()

	err := Analyze(logger, "cluster", "test/cluster/problem-pods")
	assert.Nil(t, err)

	reportedIssues := report.GetAllSourcesFilteredIssues(logger, true, 0, 0)
	assert.NotNil(t, reportedIssues)
	assert.True(t, len(reportedIssues) > 0)
	problemPodsFound := 0
	for _, issue := range reportedIssues {
		if issue.Type == report.PodProblemsNotReported {
			problemPodsFound++
		}
	}
	assert.True(t, problemPodsFound > 0)
}

// TestProblemPodsNotReportedInstall Tests that analysis of a cluster dump with pods that have unknown issues during
// install, is handled
// GIVEN a call to analyze a cluster-snapshot
// WHEN the cluster-snapshot shows pods with problems that are not known issues
// THEN a report is generated with problem pod issues identified
func TestProblemPodsNotReportedInstall(t *testing.T) {
	logger := log.GetDebugEnabledLogger()

	err := Analyze(logger, "cluster", "test/cluster/problem-pods-install")
	assert.Nil(t, err)

	reportedIssues := report.GetAllSourcesFilteredIssues(logger, true, 0, 0)
	assert.NotNil(t, reportedIssues)
	assert.True(t, len(reportedIssues) > 0)
	problemPodsFound := 0
	exceededLBLimit := 0
	for _, issue := range reportedIssues {
		if issue.Type == report.PodProblemsNotReported {
			problemPodsFound++
		} else if issue.Type == report.IngressLBLimitExceeded {
			exceededLBLimit++
		}

	}
	assert.True(t, problemPodsFound > 0)
	assert.True(t, exceededLBLimit > 0)
}

// TestLBIpNotSet Tests that analysis of a cluster dump where LB issue occurred with no IP set is handled
// GIVEN a call to analyze a cluster-snapshot
// WHEN the cluster-snapshot shows pods with problems that are not known issues
// THEN a report is generated with problem pod issues identified
// Note: With the latest changes to platform operator and analysis tool, the issue is reported differently.
// Commenting the test for now, and added a new test TestLBIpNotFound
//func TestLBIpNotSet(t *testing.T) {
//	logger := log.GetDebugEnabledLogger()

//	err := Analyze(logger, "cluster", "test/cluster/lb-ipnotset")
//	assert.Nil(t, err)

//	reportedIssues := report.GetAllSourcesFilteredIssues(logger, true, 0, 0)
//	assert.NotNil(t, reportedIssues)
//	assert.True(t, len(reportedIssues) > 0)
//	problemsFound := 0
//	for _, issue := range reportedIssues {
//		if issue.Type == report.IngressNoLoadBalancerIP {
//			problemsFound++
//		}
//	}
//	assert.True(t, problemsFound > 0)
//}

// TestLBIpNotFound Tests that analysis of a cluster dump where no IP was found for load balancer
// GIVEN a call to analyze a cluster-snapshot
// WHEN the cluster-snapshot shows pods with problems that are not known issues
// THEN a report is generated with problem pod issues identified
func TestLBIpNotFound(t *testing.T) {
	logger := log.GetDebugEnabledLogger()

	err := Analyze(logger, "cluster", "test/cluster/ingress-ip-not-found")
	assert.Nil(t, err)

	reportedIssues := report.GetAllSourcesFilteredIssues(logger, true, 0, 0)
	assert.NotNil(t, reportedIssues)
	assert.True(t, len(reportedIssues) > 0)
	problemsFound := 0
	for _, issue := range reportedIssues {
		if issue.Type == report.IngressNoIPFound {
			problemsFound++
		}
	}
	assert.True(t, problemsFound > 0)
}

// TestIstioLBIpNotFound Tests that analysis of a cluster dump where no Istio Gateway IP was found
// GIVEN a call to analyze a cluster-snapshot
// WHEN the cluster-snapshot shows services with external IP problems
// THEN a report is generated with issues identified
func TestIstioLBIpNotFound(t *testing.T) {
	logger := log.GetDebugEnabledLogger()

	err := Analyze(logger, "cluster", "test/cluster/istio-ingress-ip-not-found")
	assert.Nil(t, err)

	reportedIssues := report.GetAllSourcesFilteredIssues(logger, true, 0, 0)
	assert.NotNil(t, reportedIssues)
	assert.True(t, len(reportedIssues) > 0)
	problemsFound := 0
	for _, issue := range reportedIssues {
		if issue.Type == report.IstioIngressNoIP {
			problemsFound++
		}
	}
	assert.True(t, problemsFound > 0)
}

// TODO: Enable this test once there is a cluster dump for this use case
// TestIngressInstall Tests that analysis of a cluster dump where Ingress install failed without more info handled
// GIVEN a call to analyze a cluster-snapshot
// WHEN the cluster-snapshot shows pods with problems that are not known issues
// THEN a report is generated with problem pod issues identified
// func TestIngressInstall(t *testing.T) {
//	logger := log.GetDebugEnabledLogger()

//	err := Analyze(logger, "cluster", "test/cluster/ingress-install-unknown")
//	assert.Nil(t, err)

//	reportedIssues := report.GetAllSourcesFilteredIssues(logger, true, 0, 0)
//	assert.NotNil(t, reportedIssues)
//	assert.True(t, len(reportedIssues) > 0)
//	problemsFound := 0
//	for _, issue := range reportedIssues {
//		if issue.Type == report.IngressInstallFailure {
//			problemsFound++
//		}
//	}
//	assert.True(t, problemsFound > 0)
//}

// TestLBLimitExceeded Test that analysis of a cluster dump where Ingress install failed due to LoadBalancer service limit handled
// GIVEN a call to analyze a cluster-snapshot
// WHEN the cluster-snapshot shows pods with problems that are not known issues
// THEN a report is generated with problem pod issues identified
func TestLBLimitExceeded(t *testing.T) {
	logger := log.GetDebugEnabledLogger()

	err := Analyze(logger, "cluster", "test/cluster/ingress-lb-limit")
	assert.Nil(t, err)

	reportedIssues := report.GetAllSourcesFilteredIssues(logger, true, 0, 0)
	assert.NotNil(t, reportedIssues)
	assert.True(t, len(reportedIssues) > 0)
	problemsFound := 0
	for _, issue := range reportedIssues {
		if issue.Type == report.IngressLBLimitExceeded {
			problemsFound++
		}
	}
	assert.True(t, problemsFound > 0)
}

// TestOciIPLimitExceeded Tests that analysis of a cluster dump where Ingress install failed due to OCI limit handled
// GIVEN a call to analyze a cluster-snapshot
// WHEN the cluster-snapshot shows pods with problems that are not known issues
// THEN a report is generated with problem pod issues identified
func TestOciIPLimitExceeded(t *testing.T) {
	logger := log.GetDebugEnabledLogger()

	err := Analyze(logger, "cluster", "test/cluster/ingress-oci-limit")
	assert.Nil(t, err)

	reportedIssues := report.GetAllSourcesFilteredIssues(logger, true, 0, 0)
	assert.NotNil(t, reportedIssues)
	assert.True(t, len(reportedIssues) > 0)
	problemsFound := 0
	for _, issue := range reportedIssues {
		if issue.Type == report.IngressOciIPLimitExceeded {
			problemsFound++
		}
	}
	assert.True(t, problemsFound > 0)
}

// TestOciLBInvalidShape Tests that analysis of a cluster dump where an invalid shape specified for OCI load balancer
// GIVEN a call to analyze a cluster-snapshot
// WHEN the cluster-snapshot shows pods with problems that are not known issues
// THEN a report is generated with problem pod issues identified
func TestOciLBInvalidShape(t *testing.T) {
	logger := log.GetDebugEnabledLogger()

	err := Analyze(logger, "cluster", "test/cluster/ingress-invalid-shape")
	assert.Nil(t, err)

	reportedIssues := report.GetAllSourcesFilteredIssues(logger, true, 0, 0)
	assert.NotNil(t, reportedIssues)
	assert.True(t, len(reportedIssues) > 0)
	problemsFound := 0
	for _, issue := range reportedIssues {
		if issue.Type == report.IngressShapeInvalid {
			problemsFound++
		}
	}
	assert.True(t, problemsFound > 0)
}

// TestPendingPods that analysis of a cluster dump where pending pods only is handled
// GIVEN a call to analyze a cluster-snapshot
// WHEN the cluster-snapshot shows pods with problems that are not known issues
// THEN a report is generated with problem pod issues identified
func TestPendingPods(t *testing.T) {
	logger := log.GetDebugEnabledLogger()

	err := Analyze(logger, "cluster", "test/cluster/pending-pods")
	assert.Nil(t, err)

	reportedIssues := report.GetAllSourcesFilteredIssues(logger, true, 0, 0)
	assert.NotNil(t, reportedIssues)
	assert.True(t, len(reportedIssues) > 0)
	problemsFound := 0
	for _, issue := range reportedIssues {
		if issue.Type == report.PendingPods {
			problemsFound++
		}
	}
	assert.True(t, problemsFound > 0)
}

// TestUnknownInstall Tests that analysis of a cluster dump where install failed without more info handled
// GIVEN a call to analyze a cluster-snapshot
// WHEN the cluster-snapshot shows pods with problems that are not known issues
// THEN a report is generated with problem pod issues identified
// Commenting this test as there might not be an install issue like this now.
//func TestUnknownInstall(t *testing.T) {
//	logger := log.GetDebugEnabledLogger()

//	err := Analyze(logger, "cluster", "test/cluster/install-unknown")
//	assert.Nil(t, err)

//	reportedIssues := report.GetAllSourcesFilteredIssues(logger, true, 0, 0)
//	assert.NotNil(t, reportedIssues)
//	assert.True(t, len(reportedIssues) > 0)
//	problemsFound := 0
//	for _, issue := range reportedIssues {
//		if issue.Type == report.InstallFailure {
//			problemsFound++
//		}
//	}
//	assert.True(t, problemsFound > 0)
//}
