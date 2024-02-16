// Copyright (c) 2021, 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package cluster

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/internal/util/log"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/internal/util/report"
)

// Analyze Verrazzano Resources with variety of cluster root
// Expect No Error for each analysis
func TestAnalyzeVerrazzanoResource(t *testing.T) {
	var issueReporter = report.IssueReporter{
		PendingIssues: make(map[string]report.Issue),
	}
	logger := log.GetDebugEnabledLogger()
	assert.NoError(t, AnalyzeVerrazzanoResource(logger, "../../../test/cluster/problem-pods-install/cluster-snapshot", &issueReporter))
	assert.NoError(t, AnalyzeVerrazzanoResource(logger, "../../../test/cluster/istio-loadbalancer-creation-issue/cluster-snapshot", &issueReporter))
	assert.NoError(t, AnalyzeVerrazzanoResource(logger, "../../../test/cluster/ingress-install-unknown/cluster-snapshot", &issueReporter))
	assert.NoError(t, AnalyzeVerrazzanoResource(logger, "../../../test/cluster/image-pull-case/cluster-snapshot", &issueReporter))
	assert.NoError(t, AnalyzeVerrazzanoResource(logger, "../../../test/cluster/ingress-invalid-shape/cluster-snapshot", &issueReporter))
	assert.NoError(t, AnalyzeVerrazzanoResource(logger, "../../../test/cluster/ingress-ip-not-found/cluster-snapshot", &issueReporter))
	assert.NoError(t, AnalyzeVerrazzanoResource(logger, "../../../test/cluster/ingress-lb-limit/cluster-snapshot", &issueReporter))
	assert.NoError(t, AnalyzeVerrazzanoResource(logger, "../../../test/cluster/ingress-oci-limit/cluster-snapshot", &issueReporter))
	assert.NoError(t, AnalyzeVerrazzanoResource(logger, "../../../test/cluster/install-unknown/cluster-snapshot", &issueReporter))
	assert.NoError(t, AnalyzeVerrazzanoResource(logger, "../../../test/cluster/insufficient-mem/cluster-snapshot", &issueReporter))
	assert.NoError(t, AnalyzeVerrazzanoResource(logger, "../../../test/cluster/istio-ingress-ip-not-found/cluster-snapshot", &issueReporter))
	assert.NoError(t, AnalyzeVerrazzanoResource(logger, "../../../test/cluster/pending-pods/cluster-snapshot", &issueReporter))
	assert.NoError(t, AnalyzeVerrazzanoResource(logger, "../../../test/cluster/external-dns-issue/cluster-snapshot", &issueReporter))
}
