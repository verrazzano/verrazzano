// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	_ "embed"
	"github.com/oracle/oci-go-sdk/v53/core"
	"github.com/stretchr/testify/assert"
	ocifake "github.com/verrazzano/verrazzano/cluster-operator/controllers/quickcreate/controller/oci/fake"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"
	"testing"
)

var (
	//go:embed testdata/ociocnequickcreate_valid.yaml
	testValidOCIOCNECR []byte
	//go:embed testdata/ociocnequickcreate_invalid.yaml
	testInvalidOCIOCNECR []byte
	testID               = "test"
)

func TestValidateCreateOCNEOCI(t *testing.T) {
	cm, err := testOCNEConfigMap()
	assert.NoError(t, err)
	cli := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(cm).Build()
	ociClient := &ocifake.ClientImpl{
		VCN: &core.Vcn{
			Id: &testID,
		},
	}

	var tests = []struct {
		name     string
		crBytes  []byte
		hasError bool
	}{
		{
			"no error for valid CR",
			testValidOCIOCNECR,
			false,
		},
		{
			"error for invalid CR",
			testInvalidOCIOCNECR,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() { NewValidationContext = newValidationContext }()
			NewValidationContext = func() (*validationContext, error) {
				return testValidationContextWithOCIClient(cli, ociClient), nil
			}
			o := &OCNEOCIQuickCreate{}
			err := yaml.Unmarshal(tt.crBytes, o)
			assert.NoError(t, err)
			_, err = o.ValidateCreate()
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateUpdateOCNEOCI(t *testing.T) {
	o := &OCNEOCIQuickCreate{}
	err := yaml.Unmarshal(testValidOCIOCNECR, o)
	assert.NoError(t, err)
	var tests = []struct {
		name     string
		modifier func(o *OCNEOCIQuickCreate)
		hasError bool
	}{
		{
			"no error when no update",
			func(o *OCNEOCIQuickCreate) {},
			false,
		},
		{
			"error when spec update",
			func(o *OCNEOCIQuickCreate) {
				o.Spec.IdentityRef.Name = "foo"
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o1 := o.DeepCopy()
			o2 := o.DeepCopy()
			tt.modifier(o2)
			_, err := o1.ValidateUpdate(o2)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
