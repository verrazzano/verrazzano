// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusterapi

import (
	"testing"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestGetCapiOverrides tests getting the override values for the Cluster API component
// GIVEN a call to getCapiOverrides
//
//	WHEN all env variables are set to the correct values
//	THEN true is returned
func TestGetCapiOverrides(t *testing.T) {
	config.SetDefaultBomFilePath(testBomFilePath)

	const capiOverrides = `
{
  "global": {
     "registry": "air-gap"
  },
  "defaultProviders": {
    "ocneBootstrap": {
      "image": {
        "tag": "v1.0"
      }
    },
    "ocneControlPlane": {
      "image": {
        "tag": "v1.1"
      }
    },
    "oci": {
      "image": {
        "repository": "repo-2",
        "registry": "air-gap-2"
      }
    },
    "core": {
      "image": {
        "repository": "repo"
      }
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

	overrides, err := getCapiOverrides(compContext)
	assert.NoError(t, err)
	assert.NotNil(t, overrides)

	// Check that expected values are loaded into the struct
	assert.Equal(t, "air-gap", overrides.Global.Registry)

	bootstrapImage := overrides.DefaultProviders.OCNEBootstrap.Image
	assert.Equal(t, "", bootstrapImage.Repository)
	assert.Equal(t, "v1.0", bootstrapImage.Tag)
	assert.Equal(t, "", bootstrapImage.Registry)

	controlPlaneImage := overrides.DefaultProviders.OCNEControlPlane.Image
	assert.Equal(t, "", controlPlaneImage.Repository)
	assert.Equal(t, "v1.1", controlPlaneImage.Tag)
	assert.Equal(t, "", controlPlaneImage.Registry)

	coreImage := overrides.DefaultProviders.Core.Image
	assert.Equal(t, "repo", coreImage.Repository)
	assert.Equal(t, "", coreImage.Tag)
	assert.Equal(t, "", coreImage.Registry)

	ociImage := overrides.DefaultProviders.OCI.Image
	assert.Equal(t, "repo-2", ociImage.Repository)
	assert.Equal(t, "", ociImage.Tag)
	assert.Equal(t, "air-gap-2", ociImage.Registry)
}

// TestCreateTemplateInput tests getting the override values for the Cluster API component
// GIVEN a call to createTemplateInput
//
//	WHEN all env variables are set to the correct values
//	THEN true is returned
func TestCreateTemplateInput(t *testing.T) {
	config.SetDefaultBomFilePath(testBomFilePath)

	const capiOverrides = `
{
  "global": {
  },
  "defaultProviders": {
    "ocneBootstrap": {
      "image": {
        "tag": "v1.0"
      }
    },
    "oci": {
      "image": {
        "repository": "repo",
        "registry": "air-gap-2"
      }
    },
    "core": {
      "image": {
      }
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

	// Check that expected values are loaded into the struct
	assert.Equal(t, "ghcr.io", templateInput.Global.Registry)

	bootstrapImage := templateInput.OCNEBootstrap.Image
	assert.Equal(t, "", bootstrapImage.Registry)
	assert.Equal(t, "verrazzano", bootstrapImage.Repository)
	assert.Equal(t, "v1.0", bootstrapImage.Tag)

	controlPlaneImage := templateInput.OCNEControlPlane.Image
	assert.Equal(t, "", controlPlaneImage.Registry)
	assert.Equal(t, "verrazzano", controlPlaneImage.Repository)
	assert.Equal(t, "v1.0", controlPlaneImage.Tag)

	coreImage := templateInput.Core.Image
	assert.Equal(t, "", coreImage.Registry)
	assert.Equal(t, "verrazzano", coreImage.Repository)
	assert.Equal(t, "v1.3.3-20230427222746-876fe3dc9", coreImage.Tag)

	ociImage := templateInput.OCI.Image
	assert.Equal(t, "air-gap-2", ociImage.Registry)
	assert.Equal(t, "repo", ociImage.Repository)
	assert.Equal(t, "v0.8.1", ociImage.Tag)
}
