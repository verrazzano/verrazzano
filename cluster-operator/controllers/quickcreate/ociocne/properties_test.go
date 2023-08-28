// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ociocne

import (
	"context"
	_ "embed"
	"github.com/stretchr/testify/assert"
	vmcv1alpha1 "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/cluster-operator/controllers/quickcreate/controller/oci"
	ocifake "github.com/verrazzano/verrazzano/cluster-operator/controllers/quickcreate/controller/oci/fake"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"
	"testing"
)

var (
	scheme *runtime.Scheme
	//go:embed testdata/base.yaml
	testBase []byte
	//go:embed testdata/existing-vcn-patch.yaml
	testExistingVCN  []byte
	testOCNEVersions = "../controller/ocne/testdata/ocne-versions.yaml"
)

func init() {
	scheme = runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = vmcv1alpha1.AddToScheme(scheme)
}

func TestCreateAndApplyOCNETemplate(t *testing.T) {
	var tests = []struct {
		name  string
		patch []byte
	}{
		{
			"existing vcn",
			testExistingVCN,
		},
	}

	cm := &corev1.ConfigMap{}
	b, _ := os.ReadFile(testOCNEVersions)
	_ = yaml.Unmarshal(b, cm)
	cli := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cm).Build()
	loader := &ocifake.CredentialsLoaderImpl{
		Credentials: &oci.Credentials{
			Region:  "",
			Tenancy: "a",
			User:    "b",
			PrivateKey: `abc
def
ghi
`,
			Fingerprint:          "d",
			Passphrase:           "e",
			UseInstancePrincipal: "false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := testCreateCR(tt.patch)
			assert.NoError(t, err)
			p, err := NewProperties(context.TODO(), cli, loader, q)
			assert.NoError(t, err)
			assert.NotNil(t, p)
		})
	}
}

func testCreateCR(patch []byte) (*vmcv1alpha1.OCNEOCIQuickCreate, error) {
	baseCR := &vmcv1alpha1.OCNEOCIQuickCreate{}
	patchCR := &vmcv1alpha1.OCNEOCIQuickCreate{}
	if err := yaml.Unmarshal(testBase, baseCR); err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(patch, patchCR); err != nil {
		return nil, err
	}
	baseCR.Spec = patchCR.Spec
	return baseCR, nil
}
