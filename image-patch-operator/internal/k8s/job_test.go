// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package k8s

import (
	"testing"

	"github.com/stretchr/testify/assert"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestDeleteJob tests the deletion of a job
// GIVEN a fake job
// WHEN DeleteJob is called
// THEN the function should return success
func TestDeleteJob(t *testing.T) {
	const name = "test"
	const namespace = "testns"

	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
		},
	).Build()

	err := DeleteJob(client, name, namespace)
	assert.NoError(t, err, "Error deleting job")
}
