// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package loggingscope

import "github.com/verrazzano/verrazzano/application-operator/constants"

const (
	testNamespace    = "test-namespace"
	testAPIVersion   = "testv1"
	testScopeName    = "test-scope-name"
	testFluentdImage = "test-image:latest"
)

// createTestLoggingScope creates a test logging scope
func createTestLoggingScope(includeWorkload bool) *LoggingScope {
	scope := LoggingScope{}
	scope.ElasticSearchURL = testESURL
	scope.SecretName = testESSecret
	scope.FluentdImage = testFluentdImage
	scope.SecretNamespace = constants.VerrazzanoSystemNamespace
	return &scope
}

func updateLoggingScope(scope *LoggingScope) {
	scope.ElasticSearchURL = testESURLUpdate
	scope.SecretName = testESSecretUpdate
}
