// Copyright (c) 2021, 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package cluster

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/log"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/report"
	corev1 "k8s.io/api/core/v1"
)

// TODO: Add more tests

func TestPodConditionMessage(t *testing.T) {
	ns := "test"
	var tests = []struct {
		name      string
		condition corev1.PodCondition
		message   string
	}{
		{
			"pod-no-message-nor-reason",
			corev1.PodCondition{
				Type:   corev1.PodInitialized,
				Status: corev1.ConditionFalse,
			},
			"Namespace test, Pod pod-no-message-nor-reason, ConditionType Initialized, Status False",
		},
		{
			"pod-with-message-and-reason",
			corev1.PodCondition{
				Type:    corev1.ContainersReady,
				Status:  corev1.ConditionTrue,
				Message: "foo",
				Reason:  "bar",
			},
			"Namespace test, Pod pod-with-message-and-reason, ConditionType ContainersReady, Status True, Reason bar, Message foo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := podConditionMessage(tt.name, ns, tt.condition)
			assert.NoError(t, err)
			assert.Equal(t, tt.message, msg)
		})
	}
}

// Analyze Pod Issues with variety of cluster roots
// Expect No Error for each analysis
func TestAnalyzePodIssues(t *testing.T) {
	logger := log.GetDebugEnabledLogger()
	assert.NoError(t, AnalyzePodIssues(logger, "../../../test/cluster/problem-pods/cluster-snapshot"))
	assert.NoError(t, AnalyzePodIssues(logger, "../../../test/cluster/pending-pods/cluster-snapshot"))
	assert.NoError(t, AnalyzePodIssues(logger, "../../../test/cluster/problem-pods-install/cluster-snapshot"))
	assert.NoError(t, AnalyzePodIssues(logger, "../../../test/cluster/insufficient-mem/cluster-snapshot"))
	assert.NoError(t, AnalyzePodIssues(logger, "../../../test/cluster/pod-waiting-for-readiness-gates/cluster-snapshot"))
}

// TestPodReadinessGateIssue tests whether the relevant issue is reported when a pod does not have its readiness gates ready
// GIVEN a call to analyze pod related issues in a cluster-snapshot
// WHEN a valid input is provided that contains a pod whose readiness gates are not ready
// THEN the function does not generate an error and adds the correct issue
func TestPodReadinessGatesIssue(t *testing.T) {
	report.ClearReports()
	logger := log.GetDebugEnabledLogger()
	assert.NoError(t, AnalyzePodIssues(logger, "../../../test/cluster/pod-waiting-for-readiness-gates/cluster-snapshot"))
	reportedIssues := report.GetAllSourcesFilteredIssues(logger, true, 0, 0)
	assert.True(t, len(reportedIssues) == 1)
	assert.True(t, reportedIssues[0].Type == report.PodWaitingOnReadinessGates)
	report.ClearReports()
}

// TestPodHangingOnDeletionIssue tests whether the relevant issue is reported when a pod has been in a state of deletion for an extended period of time
// GIVEN a call to analyze pod related issues in a cluster-snapshot
// WHEN a valid input is provided that has a pod has been terminating for a long time
// THEN the function does not generate an error and adds the correct issue
func TestPodHangingOnDeletionIssue(t *testing.T) {
	report.ClearReports()
	logger := log.GetDebugEnabledLogger()
	assert.NoError(t, AnalyzePodIssues(logger, "../../../test/cluster/pod-hanging-on-deletion/cluster-snapshot"))
	reportedIssues := report.GetAllSourcesFilteredIssues(logger, true, 0, 0)
	assert.True(t, len(reportedIssues) == 1)
	assert.True(t, reportedIssues[0].Type == report.PodHangingOnDeletion)
	report.ClearReports()
}
