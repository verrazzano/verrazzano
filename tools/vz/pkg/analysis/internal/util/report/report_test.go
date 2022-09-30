// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package report

import (
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/log"
	"strings"
	"testing"
)

const errWas = "Err was"

// TestInvalidIssues Tests the helpers with invalid issues
// GIVEN a call to helper
// WHEN with an invalid issue
// THEN the appropriate error is returned
func TestInvalidIssues(t *testing.T) {
	logger := log.GetDebugEnabledLogger()

	// We start with a custom issue which is created without being populated. This is
	// invalid as there are a few fields which are required
	var invalidIssue = Issue{}

	// This will fail as there is no Type specified
	err := ContributeIssue(logger, invalidIssue)
	assert.NotNil(t, err)
	logger.Debugf(errWas, err)
	assert.True(t, strings.Contains(err.Error(), "Type"))

	// Next set the Type on the issue, it should then complain that no Source is specified
	invalidIssue.Type = "MyIssueType"
	err = ContributeIssue(logger, invalidIssue)
	assert.NotNil(t, err)
	logger.Debugf(errWas, err)
	assert.True(t, strings.Contains(err.Error(), "Source"))

	// Next set the Source on the issue, it should then complain that no Summary is specified
	invalidIssue.Source = "MyIssueSource"
	err = ContributeIssue(logger, invalidIssue)
	assert.NotNil(t, err)
	logger.Debugf(errWas, err)
	assert.True(t, strings.Contains(err.Error(), "Summary"))

	// Next set the summary but also set a confidence to a value out of range
	invalidIssue.Summary = "MyIssueSummary"
	invalidIssue.Confidence = 11
	err = ContributeIssue(logger, invalidIssue)
	assert.NotNil(t, err)
	logger.Debugf(errWas, err)
	assert.True(t, strings.Contains(err.Error(), "Confidence"))
}

// Add tests
