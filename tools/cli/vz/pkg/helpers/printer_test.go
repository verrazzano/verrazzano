// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"strings"
	"testing"
)

var (
	headings = []string{"NAME", "STATUS", "AGE"}
	data     = [][]string{
		{"first", "Ready", "4d"},
		{"second", "Error", "23s"},
	}

	// note that this string has the three spoces at the end of the last column
	// and the newline at the end!!!
	expected = `NAME     STATUS   AGE   
first    Ready    4d    
second   Error    23s   
`
)

func TestFormatOutput(t *testing.T) {
	actual, err := formatOutput(headings, data)
	if err != nil {
		t.Error(err)
	}

	if strings.Compare(actual, expected) != 0 {
		t.Errorf("Actual output did not match expected output.\nExpected this:\n%v\n\nGot this:\n%v", expected, actual)
	}
}

func TestFormatStringSlice(t *testing.T) {
	expectedResult := "one,two,three"
	result := FormatStringSlice([]string{"one", "two", "three"})
	if strings.Compare(result, expectedResult) != 0 {
		t.Errorf("Result was incorrect.  Expected: %s  but got: %s", expectedResult, result)
	}
}