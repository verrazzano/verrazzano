// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ociocne

import (
	"context"
	"github.com/stretchr/testify/assert"
	vmcv1alpha1 "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/cluster-operator/controllers/quickcreate/controller/oci"
	ocifake "github.com/verrazzano/verrazzano/cluster-operator/controllers/quickcreate/controller/oci/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestCreateAndApplyOCNETemplate(t *testing.T) {
	cli := fake.NewClientBuilder().Build()
	loader := &ocifake.CredentialsLoaderImpl{
		Credentials: &oci.Credentials{},
	}
	ocne, err := NewOCNE(context.TODO(), cli, loader, &vmcv1alpha1.OCNEOCIQuickCreate{})
	assert.NoError(t, err)
	assert.NotNil(t, ocne.OCNEOCIQuickCreate)
	assert.NotNil(t, ocne.Credentials)
}
