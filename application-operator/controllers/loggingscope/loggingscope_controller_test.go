// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package loggingscope

import (
	oamrt "github.com/crossplane/crossplane-runtime/apis/core/v1alpha1"
	oamcore "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
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
func createTestLoggingScope(includeWorkload bool) *vzapi.LoggingScope {
	scope := vzapi.LoggingScope{}
	scope.TypeMeta = k8smeta.TypeMeta{
		APIVersion: vzapi.GroupVersion.Identifier(),
		Kind:       vzapi.LoggingScopeKind}
	scope.ObjectMeta = k8smeta.ObjectMeta{
		Namespace: testNamespace,
		Name:      testScopeName}
	scope.Spec.ElasticSearchURL = testESURL
	scope.Spec.SecretName = testESSecret
	scope.Spec.FluentdImage = testFluentdImage
	if includeWorkload {
		scope.Spec.WorkloadReferences = []oamrt.TypedReference{{
			APIVersion: oamcore.SchemeGroupVersion.Identifier(),
			Kind:       testKind,
			Name:       workloadName}}
	}

	return &scope
}

func updateLoggingScope(scope *vzapi.LoggingScope) {
	scope.Spec.ElasticSearchURL = testESURLUpdate
	scope.Spec.SecretName = testESSecretUpdate
}
