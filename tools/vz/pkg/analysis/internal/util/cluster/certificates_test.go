// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package cluster

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/log"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/report"
)

// TestAnalyzeCertificateIssues asserts that no errors are called when the Certificate Analysis function is called on a valid input
func TestAnalyzeCertificateIssues(t *testing.T) {
	logger := log.GetDebugEnabledLogger()
	assert.NoError(t, AnalyzeCertificateRelatedIssues(logger, "../../../test/cluster/testCertificateIssue"))
	report.ClearReports()
}
