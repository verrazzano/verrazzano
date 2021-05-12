// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package logging

const (
	testNamespace    = "test-namespace"
	testAPIVersion   = "testv1"
	testFluentdImage = "test-image:latest"
)

// createTestLogInfo creates a test logging info
func createTestLogInfo(includeWorkload bool) *LogInfo {
	scope := LogInfo{}
	scope.FluentdImage = testFluentdImage
	return &scope
}
