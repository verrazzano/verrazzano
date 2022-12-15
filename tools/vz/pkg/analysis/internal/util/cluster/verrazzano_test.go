// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package cluster

import (
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/log"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/report"
	"testing"
)

// Test Verrazzano Installation Status with a couple of cluster root
// Expect No Error for each installation status
func TestInstallationStatus(t *testing.T) {
	logger := log.GetDebugEnabledLogger()
	var issueReporter = report.IssueReporter{
		PendingIssues: make(map[string]report.Issue),
	}
	assert.NoError(t, installationStatus(logger, "../../../test/cluster/problem-pods-install/cluster-snapshot", &issueReporter))
	assert.NoError(t, installationStatus(logger, "../../../test/cluster/pending-pods/cluster-snapshot", &issueReporter))
}
