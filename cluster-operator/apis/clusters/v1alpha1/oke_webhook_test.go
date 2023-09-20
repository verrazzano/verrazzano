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
	//go:embed testdata/okequickcreate_valid.yaml
	testValidOKECR []byte
	//go:embed testdata/okequickcreate_invalid.yaml
	testInvalidOKECR []byte
)

func TestValidateCreateOKE(t *testing.T) {
	cli := fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()
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
			testValidOKECR,
			false,
		},
		{
			"error for invalid CR",
			testInvalidOKECR,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() { NewValidationContext = newValidationContext }()
			NewValidationContext = func() (*validationContext, error) {
				return testValidationContextWithOCIClient(cli, ociClient), nil
			}
			o := &OKEQuickCreate{}
			err := yaml.Unmarshal(tt.crBytes, o)
			assert.NoError(t, err)
			err = o.ValidateCreate()
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateUpdateOK(t *testing.T) {
	o := &OKEQuickCreate{}
	err := yaml.Unmarshal(testValidOCIOCNECR, o)
	assert.NoError(t, err)
	var tests = []struct {
		name     string
		modifier func(o *OKEQuickCreate)
		hasError bool
	}{
		{
			"no error when no update",
			func(o *OKEQuickCreate) {},
			false,
		},
		{
			"error when spec update",
			func(o *OKEQuickCreate) {
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
			err := o1.ValidateUpdate(o2)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
