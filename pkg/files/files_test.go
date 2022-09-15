// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package files

import (
	asserts "github.com/stretchr/testify/assert"
	"regexp"
	"testing"
)

func TestGetMatchingFiles(t *testing.T) {
	assert := asserts.New(t)

	files1, err1 := GetMatchingFiles("testdata", regexp.MustCompile("dummy"))
	assert.Nil(err1)
	assert.Equal(len(files1), 2)

	files2, err2 := GetMatchingFiles("testdata", regexp.MustCompile("k8s"))
	assert.Nil(err2)
	assert.Equal(len(files2), 2)

	files3, err3 := GetMatchingFiles("testdata", regexp.MustCompile("cluster"))
	assert.Nil(err3)
	assert.Equal(len(files3), 1)

	files4, err4 := GetMatchingFiles("testdata", regexp.MustCompile("non-existing-file-regex"))
	assert.Nil(err4)
	assert.Equal(len(files4), 0)

	files5, err5 := GetMatchingFiles("non-existing-directory", regexp.MustCompile("non-existing-file-regex"))
	assert.Nil(err5)
	assert.Empty(len(files5), 0)
}
