// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package kubectlutil_test

import (
	"github.com/google/go-cmp/cmp"
	"github.com/verrazzano/verrazzano/pkg/kubectlutil"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestSetLastAppliedConfigurationAnnotation(t *testing.T) {
	vz := &v1beta1.Verrazzano{
		ObjectMeta: metav1.ObjectMeta{},
		Spec:       v1beta1.VerrazzanoSpec{},
	}

	err := kubectlutil.SetLastAppliedConfigurationAnnotation(vz)
	if err != nil {
		t.Errorf("expected no error, got error %v", err)
	}

	value, ok := vz.Annotations[v1.LastAppliedConfigAnnotation]
	if !ok {
		t.Errorf("expected "+v1.LastAppliedConfigAnnotation+" , not found on object %v", vz)
	}
	expected := "{\"metadata\":{\"creationTimestamp\":null},\"spec\":{\"components\":{},\"security\":{}},\"status\":{}}\n"
	if diff := cmp.Diff(expected, value); diff != "" {
		t.Errorf("expected %v, got %v instead", expected, value)
		t.Logf("Difference: %s", diff)
	}
}
