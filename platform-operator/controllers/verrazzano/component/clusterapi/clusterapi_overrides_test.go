// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusterapi

import (
	"testing"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1 "k8s.io/api/core/v1"

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
	"registry": "air-gap",
	"imagePullSecrets": [
	  {
		"name": "secret1"
	  }
	]
  },
  "defaultProviders": {
    "oci": {
      "image": {
        "registry": "air-gap-2"
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
	assert.Equal(t, corev1.PullIfNotPresent, overrides.Global.PullPolicy)
	assert.Equal(t, "secret1", overrides.Global.ImagePullSecrets[0].Name)

	bootstrapImage := overrides.DefaultProviders.OCNE.Bootstrap.Image
	assert.Equal(t, "verrazzano", bootstrapImage.Repository)
	assert.Equal(t, "v0.1.0-20230427222244-4ef1141", bootstrapImage.Tag)
	assert.Equal(t, "", bootstrapImage.Registry)
	assert.Equal(t, corev1.PullPolicy(""), bootstrapImage.PullPolicy)

	controlPlaneImage := overrides.DefaultProviders.OCNE.ControlPlane.Image
	assert.Equal(t, "verrazzano", controlPlaneImage.Repository)
	assert.Equal(t, "v0.1.0-20230427222244-4ef1141", controlPlaneImage.Tag)
	assert.Equal(t, "", controlPlaneImage.Registry)
	assert.Equal(t, corev1.PullPolicy(""), controlPlaneImage.PullPolicy)

	coreImage := overrides.DefaultProviders.Core.Image
	assert.Equal(t, "verrazzano", coreImage.Repository)
	assert.Equal(t, "v1.3.3-20230427222746-876fe3dc9", coreImage.Tag)
	assert.Equal(t, "", coreImage.Registry)
	assert.Equal(t, corev1.PullPolicy(""), coreImage.PullPolicy)

	ociImage := overrides.DefaultProviders.OCI.Image
	assert.Equal(t, "oracle", ociImage.Repository)
	assert.Equal(t, "v0.8.1", ociImage.Tag)
	assert.Equal(t, "air-gap-2", ociImage.Registry)
	assert.Equal(t, corev1.PullPolicy(""), ociImage.PullPolicy)
}
