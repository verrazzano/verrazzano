// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package delegates

import (
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNotReadyErrorf(t *testing.T) {
	err := NotReadyErrorf("test %s", "a")
	fmt.Println(err)
}

func TestIsNotReadyError(t *testing.T) {
	err := NotReadyErrorf("test")
	assert.True(t, IsNotReadyError(err))
	assert.False(t, IsNotReadyError(errors.New("foo")))
}
