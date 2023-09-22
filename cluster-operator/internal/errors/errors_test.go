// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package errors

import (
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestErrorAggregator(t *testing.T) {
	const (
		e1 = "boom!"
		e2 = "kaboom!"
	)
	e := NewAggregator(";")
	assert.False(t, e.HasError())
	e.Add(errors.New(e1))
	assert.True(t, e.HasError())
	assert.Equal(t, e1, e.Error())
	e.Addf("warning: %s", e2)
	assert.True(t, e.HasError())
	assert.Equal(t, fmt.Sprintf("%s;warning: %s", e1, e2), e.Error())
}
