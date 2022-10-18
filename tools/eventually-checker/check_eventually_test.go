// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package main

import (
	"bytes"
	"flag"
	"go/token"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestAnalyzePackages tests analyzing packages and finds bad calls inside gomega.Eventually.
// GIVEN test source code that calls gomega.Eventually and ginkgo.Fail and gomega.Expect
// WHEN the packages are analyzed
// THEN the checker correctly identifies Fail and Expect calls that are called by Eventually (directly or indirectly)
func TestAnalyzePackages(t *testing.T) {
	assert := assert.New(t)

	// load the packages from the unit test data directory
	fset, pkgs, err := loadPackages("./testdata")
	if err != nil {
		assert.NoError(err)
	}

	// should be two packages
	assert.Len(pkgs, 2)

	// analyze the packages
	for _, pkg := range pkgs {
		analyze(pkg.Syntax)
	}

	// check for bad calls, we should get 2
	results := checkForBadCalls()
	assert.Len(results, 8)

	for key, val := range results {
		// convert the failed call position to a string of the form "filename:row:column"
		failedCallPos := fset.PositionFor(key, true).String()
		if strings.HasSuffix(failedCallPos, "/internal/helper.go:12:2") {
			// expect this bad call from Eventually in main.go
			assert.Len(val, 1)
			eventuallyPos := fset.PositionFor(val[0], true).String()
			assert.True(strings.HasSuffix(eventuallyPos, "/main.go:32:3"))
		} else if strings.HasSuffix(failedCallPos, "/main.go:62:2") {
			// expect this bad call from Eventually in main.go
			assert.Len(val, 1)
			eventuallyPos := fset.PositionFor(val[0], true).String()
			assert.True(strings.HasSuffix(eventuallyPos, "/main.go:41:3"))
		} else if strings.HasSuffix(failedCallPos, "/main.go:17:2") {
			// expect this bad call from Eventually in main.go
			assert.Len(val, 1)
			eventuallyPos := fset.PositionFor(val[0], true).String()
			assert.True(strings.HasSuffix(eventuallyPos, "/main.go:49:3"))
		} else if strings.HasSuffix(failedCallPos, "/main.go:22:2") {
			// expect this bad call from Eventually in main.go
			assert.Len(val, 1)
			eventuallyPos := fset.PositionFor(val[0], true).String()
			assert.True(strings.HasSuffix(eventuallyPos, "/main.go:55:3"))
		} else if strings.HasSuffix(failedCallPos, "/main.go:91:4") {
			// expect this bad call from Eventually in main.go
			assert.Len(val, 1)
			eventuallyPos := fset.PositionFor(val[0], true).String()
			assert.True(strings.HasSuffix(eventuallyPos, "/main.go:89:3"))
		} else if strings.HasSuffix(failedCallPos, "/main.go:99:4") {
			// expect this bad call from Eventually in main.go
			assert.Len(val, 1)
			eventuallyPos := fset.PositionFor(val[0], true).String()
			assert.True(strings.HasSuffix(eventuallyPos, "/main.go:97:3"))
		} else if strings.HasSuffix(failedCallPos, "/helper.go:21:9") {
			// expect this bad call from Eventually in main.go
			assert.Len(val, 1)
			eventuallyPos := fset.PositionFor(val[0], true).String()
			assert.True(strings.HasSuffix(eventuallyPos, "/main.go:105:3"))
		} else if strings.HasSuffix(failedCallPos, "/helper.go:26:3") {
			// expect this bad call from Eventually in main.go
			assert.Len(val, 1)
			eventuallyPos := fset.PositionFor(val[0], true).String()
			assert.True(strings.HasSuffix(eventuallyPos, "/helper.go:25:2"))
		} else {
			t.Errorf("Found unexpected Fail/Expect call at: %s", failedCallPos)
		}
	}
}

// TestParseFlags tests command line flag parsing
// GIVEN command line arguments
// WHEN the command line arguments are parsed
// THEN validate that the flags are set correctly and positional arguments are correct
func TestParseFlags(t *testing.T) {
	assert := assert.New(t)
	path := "/unit/test/path"

	os.Args = []string{"cmd_exe", "-report", path}

	parseFlags()
	assert.True(reportOnly)
	assert.Equal(path, flag.Args()[0])

	// reset flag parsing
	flag.CommandLine = flag.NewFlagSet("", flag.ExitOnError)

	os.Args = []string{"cmd_exe", path}

	parseFlags()
	assert.False(reportOnly)
	assert.Equal(path, flag.Args()[0])
}

// TestDisplayResults tests output of checker results
// GIVEN test source code that calls gomega.Eventually and ginkgo.Fail and gomega.Expect
// WHEN the packages are analyzed
// AND results are displayed
// THEN the output contains the files and positions of the bad calls
func TestDisplayResults(t *testing.T) {
	assert := assert.New(t)

	// clear the maps from previous analysis run
	funcMap = make(map[string][]funcCall)
	eventuallyMap = make(map[token.Pos][]funcCall)

	fset, pkgs, err := loadPackages("./testdata")
	if err != nil {
		assert.NoError(err)
	}

	for _, pkg := range pkgs {
		analyze(pkg.Syntax)
	}

	results := checkForBadCalls()

	var b bytes.Buffer
	displayResults(results, fset, &b)
	assert.Contains(b.String(), "helper.go:12:2")
	assert.Contains(b.String(), "main.go:32:3")
	assert.Contains(b.String(), "main.go:62:2")
	assert.Contains(b.String(), "main.go:41:3")
	assert.Contains(b.String(), "main.go:17:2")
	assert.Contains(b.String(), "main.go:49:3")
	assert.Contains(b.String(), "main.go:22:2")
	assert.Contains(b.String(), "main.go:55:3")
}
