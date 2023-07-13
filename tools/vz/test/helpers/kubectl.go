// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"github.com/google/go-cmp/cmp"
	v12 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func VerifyLastAppliedConfigAnnotation(t *testing.T, object v1.ObjectMeta, expectedLastAppliedConfigAnnotation string) {
	actual := object.GetAnnotations()[v12.LastAppliedConfigAnnotation]
	if diff := cmp.Diff(actual, expectedLastAppliedConfigAnnotation); diff != "" {
		t.Errorf("expected %v\n, got %v instead", expectedLastAppliedConfigAnnotation, actual)
		t.Logf("Difference: %s", diff)
	}
}
