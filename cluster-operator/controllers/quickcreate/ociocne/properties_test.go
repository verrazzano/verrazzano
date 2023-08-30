// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ociocne

import (
	"context"
	_ "embed"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestCreateAndApplyOCNETemplate(t *testing.T) {
	var tests = []struct {
		name  string
		patch []byte
	}{
		{
			"existing vcn",
			testExistingVCNPatch,
		},
		{
			"new vcn",
			testNewVCNPatch,
		},
	}

	cli := fake.NewClientBuilder().WithScheme(scheme).WithObjects(testOCNEConfigMap()).Build()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := testCreateCR(tt.patch)
			assert.NoError(t, err)
			ctx := context.TODO()
			p, err := NewProperties(ctx, cli, testLoader, testOCIClientGetter, q)
			assert.NoError(t, err)
			assert.NotNil(t, p)
			err = p.ApplyTemplate(cli, clusterTemplate, nodesTemplate, ocneTemplate, addonsTemplate)
			assert.NoError(t, err)
		})
	}
}
