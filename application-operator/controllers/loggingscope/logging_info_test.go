// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package loggingscope

import (
	k8smeta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testKind         = "test-type"
	workloadUID      = "test-workload-uid"
	workloadName     = "test-workload-name"
	testNamespace    = "test-namespace"
	testAPIVersion   = "testv1"
	testScopeName    = "test-scope-name"
	testFluentdImage = "test-image:latest"
)

// createTestLoggingScope creates a test logging scope
func createTestLoggingScope(includeWorkload bool) *LoggingInfo {
	scope := LoggingInfo{}
	scope.ObjectMeta = k8smeta.ObjectMeta{
		Namespace: testNamespace,
		Name:      testScopeName}
	scope.Spec.ElasticSearchURL = testESURL
	scope.Spec.SecretName = testESSecret
	scope.Spec.FluentdImage = testFluentdImage

	return &scope
}

func updateLoggingScope(scope *LoggingInfo) {
	scope.Spec.ElasticSearchURL = testESURLUpdate
	scope.Spec.SecretName = testESSecretUpdate
}
