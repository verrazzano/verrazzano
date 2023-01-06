// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package cluster

import (
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/log"
	corev1 "k8s.io/api/core/v1"
	"testing"
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
}
