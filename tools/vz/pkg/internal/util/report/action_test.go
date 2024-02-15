// Copyright (c) 2021, 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package report

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/internal/util/log"
)

func TestActionValidate(t *testing.T) {
	logger := log.GetDebugEnabledLogger()
	a := Action{
		Summary: "",
		Links:   []string{"l1"},
		Steps:   []string{"s1"},
	}
	assert.Error(t, a.Validate(logger))
	a.Summary = "S1"
	assert.NoError(t, a.Validate(logger))
}
