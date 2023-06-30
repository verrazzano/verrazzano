// Copyright (c) 2023 Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package cluster

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/log"
)

// This tests asserts for no errors ask about the no report for other unit tests
func TestAnalyzeCertificateIssues(t *testing.T) {
	logger := log.GetDebugEnabledLogger()
	assert.NoError(t, AnalyzeCertificateRelatedIsssues(logger, "../../../test/cluster/testCertificateIssue"))
}
