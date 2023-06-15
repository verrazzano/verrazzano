// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusterapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestTemplateInterface tests the TemplateInterface
// GIVEN a set TemplateInput
//
//	WHEN the user supplies a TemplateInput
//	THEN verify the TemplateInterface returns the expected values
func TestTemplateInterface(t *testing.T) {
	config.SetDefaultBomFilePath(testBomFilePath)

	const capiOverrides = `
{
  "global": {
    "registry": "ghcr.io"
  },
  "defaultProviders": {
    "ocneBootstrap": {
      "url": "/test/bootstrap.yaml",
      "image": {
        "registry": "myreg.io",
        "tag": "v1.0"
      }
    },
    "ocneControlPlane": {
      "image": {
        "tag": "v1.1"
      }
    },
    "oci": {
      "version": "v0.8.2",
      "image": {
        "repository": "repo",
        "registry": "myreg2.io",
        "tag": "v0.8.2"
      }
    },
    "core": {
    }
  }
}`

	vz := &v1alpha1.Verrazzano{
		ObjectMeta: metav1.ObjectMeta{
			Name: "vz",
		},
		Spec: v1alpha1.VerrazzanoSpec{
			Components: v1alpha1.ComponentSpec{
				ClusterAPI: &v1alpha1.ClusterAPIComponent{
					InstallOverrides: v1alpha1.InstallOverrides{
						ValueOverrides: []v1alpha1.Overrides{
							{
								Values: &apiextensionsv1.JSON{
									Raw: []byte(capiOverrides),
								},
							},
						},
					},
				},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects().Build()
	compContext := spi.NewFakeContext(fakeClient, vz, nil, false)
	config.TestHelmConfigDir = "../../../../helm_config"

	templateInput, err := createTemplateInput(compContext)
	assert.NoError(t, err)
	assert.NotNil(t, templateInput)
	tc := newTemplateContext(templateInput)

	assert.Equal(t, "ghcr.io", tc.GetGlobalRegistry())

	assert.Equal(t, "myreg.io/verrazzano", tc.GetOCNEBootstrapRepository())
	assert.Equal(t, "v1.0", tc.GetOCNEBootstrapTag())
	assert.Equal(t, "/test/bootstrap.yaml", tc.GetOCNEBootstrapURL())

	assert.Equal(t, "ghcr.io/verrazzano", tc.GetOCNEControlPlaneRepository())
	assert.Equal(t, "v1.1", tc.GetOCNEControlPlaneTag())
	assert.Equal(t, "/verrazzano/capi/control-plane-ocne/v0.1.0/control-plane-components.yaml", tc.GetOCNEControlPlaneURL())

	assert.Equal(t, "ghcr.io/verrazzano", tc.GetClusterAPIRepository())
	assert.Equal(t, "v1.3.3-20230427222746-876fe3dc9", tc.GetClusterAPITag())
	assert.Equal(t, "/verrazzano/capi/cluster-api/v1.3.3/core-components.yaml", tc.GetClusterAPIURL())

	assert.Equal(t, "myreg2.io/repo", tc.GetOCIRepository())
	assert.Equal(t, "v0.8.2", tc.GetOCITag())
	assert.Equal(t, "https://github.com/verrazzano/capi/infrastructure-oci/v0.8.2/infrastructure-components.yaml", tc.GetOCIURL())

}
