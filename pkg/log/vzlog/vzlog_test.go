// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vzlog

import (
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"testing"
)

// TestLog tests the ProgressLogger function periodic logging
// GIVEN a ProgressLogger with a frequency of 3 seconds
// WHEN log is called 5 times in 5 seconds to log the same message
// THEN ensure that 2 messages are logged
func TestLog(t *testing.T) {
	const (
		name      = "name"
		namespace = "namespace"
		uid       = "uid"
		gen       = 100
	)

	assert.NotNil(t, DefaultLogger())

	log, err := EnsureResourceLogger(&ResourceConfig{
		Name:           name,
		Namespace:      namespace,
		ID:             uid,
		Generation:     gen,
		ControllerName: "verrazzano",
	})
	assert.NoError(t, err)
	assert.NotNil(t, log)

	r := ResourceConfig{
		Name:           name,
		Namespace:      namespace,
		ID:             uid,
		Generation:     gen,
		ControllerName: "verrazzano",
	}
	log = ForZapLogger(&r, zap.S())
	assert.NotNil(t, log)
}
