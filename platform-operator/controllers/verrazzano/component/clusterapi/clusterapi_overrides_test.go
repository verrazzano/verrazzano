// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusterapi

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	OciImageVersion   = "v0.8.1"
	CoreImageTag      = "v1.3.3-20230427222746-876fe3dc9"
	TestHelmConfigDir = "../../../../helm_config"
)

// TestBomOverrides tests getting the override values for the Cluster API component from the BOM
// GIVEN a call to createOverrides
//
//	WHEN no user overrides have been specified
//	THEN check expected values returned from the BOM
func TestBomOverrides(t *testing.T) {
	config.SetDefaultBomFilePath(testBomFilePath)

	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects().Build()
	compContext := spi.NewFakeContext(fakeClient, &v1alpha1.Verrazzano{}, nil, false)
	config.TestHelmConfigDir = TestHelmConfigDir

	overrides, err := createOverrides(compContext)
	assert.NoError(t, err)
	assert.NotNil(t, overrides)

	// Check that expected values are loaded into the struct
	assert.Equal(t, "ghcr.io", overrides.Global.Registry)

	bootstrap := overrides.DefaultProviders.OCNEBootstrap
	assert.Equal(t, "verrazzano", bootstrap.Image.Repository)
	assert.Equal(t, "v0.1.0-20230427222244-4ef1141", bootstrap.Image.Tag)
	assert.Equal(t, "", bootstrap.Image.Registry)
	assert.Equal(t, "v0.1.0", bootstrap.Image.BomVersion)
	assert.Equal(t, "", bootstrap.Version)
	assert.Equal(t, "", bootstrap.URL)

	controlPlane := overrides.DefaultProviders.OCNEControlPlane
	assert.Equal(t, "verrazzano", controlPlane.Image.Repository)
	assert.Equal(t, "v0.1.0-20230427222244-4ef1141", controlPlane.Image.Tag)
	assert.Equal(t, "", controlPlane.Image.Registry)
	assert.Equal(t, "v0.1.0", controlPlane.Image.BomVersion)
	assert.Equal(t, "", controlPlane.Version)
	assert.Equal(t, "", controlPlane.URL)

	core := overrides.DefaultProviders.Core
	assert.Equal(t, "verrazzano", core.Image.Repository)
	assert.Equal(t, CoreImageTag, core.Image.Tag)
	assert.Equal(t, "", core.Image.Registry)
	assert.Equal(t, "v1.3.3", core.Image.BomVersion)
	assert.Equal(t, "", core.Version)
	assert.Equal(t, "", core.URL)

	oci := overrides.DefaultProviders.OCI
	assert.Equal(t, "oracle", oci.Image.Repository)
	assert.Equal(t, OciImageVersion, oci.Image.Tag)
	assert.Equal(t, "", oci.Image.Registry)
	assert.Equal(t, OciImageVersion, oci.Image.BomVersion)
	assert.Equal(t, "", oci.Version)
	assert.Equal(t, "", oci.URL)
}

// TestUserOverrides tests getting the override values for the Cluster API component
// GIVEN a set of user overrides in the VZ custom resource
//
//	WHEN the user supplies overrides for ClusterAPI
//	THEN verify they are applied correctly
func TestUserOverrides(t *testing.T) {
	config.SetDefaultBomFilePath(testBomFilePath)

	const capiOverrides = `
{
  "global": {
    "registry": "myreg.io"
  },
  "defaultProviders": {
    "ocneBootstrap": {
      "image": {
        "tag": "v1.0"
      }
    },
    "ocneControlPlane": {
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
      "version": "v1.1"
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
	config.TestHelmConfigDir = TestHelmConfigDir

	overrides, err := createOverrides(compContext)
	assert.NoError(t, err)
	assert.NotNil(t, overrides)

	// Check that expected values are loaded into the struct
	assert.Equal(t, "myreg.io", overrides.Global.Registry)

	bootstrap := overrides.DefaultProviders.OCNEBootstrap
	assert.Equal(t, "", bootstrap.Image.Registry)
	assert.Equal(t, "verrazzano", bootstrap.Image.Repository)
	assert.Equal(t, "v1.0", bootstrap.Image.Tag)
	assert.Equal(t, "", bootstrap.Version)
	assert.Equal(t, "", bootstrap.URL)

	controlPlane := overrides.DefaultProviders.OCNEControlPlane
	assert.Equal(t, "", controlPlane.Image.Registry)
	assert.Equal(t, "verrazzano", controlPlane.Image.Repository)
	assert.Equal(t, "v1.0", controlPlane.Image.Tag)
	assert.Equal(t, "", controlPlane.Version)
	assert.Equal(t, "", controlPlane.URL)

	core := overrides.DefaultProviders.Core
	assert.Equal(t, "", core.Image.Registry)
	assert.Equal(t, "verrazzano", core.Image.Repository)
	assert.Equal(t, CoreImageTag, core.Image.Tag)
	assert.Equal(t, "v1.1", core.Version)
	assert.Equal(t, "", core.URL)

	oci := overrides.DefaultProviders.OCI
	assert.Equal(t, "air-gap-2", oci.Image.Registry)
	assert.Equal(t, "repo", oci.Image.Repository)
	assert.Equal(t, "v0.8.1", oci.Image.Tag)
	assert.Equal(t, "", oci.Version)
	assert.Equal(t, "", oci.URL)
}

// TestOverridesInterfaceDefault tests the OverridesInterface
// GIVEN version overrides for each capi component
//
//	WHEN the user supplies a version override for each component
//	THEN verify the OverridesInterface returns the expected values
func TestOverridesInterfaceDefault(t *testing.T) {
	config.SetDefaultBomFilePath(testBomFilePath)

	const capiOverrides = `
{
  "global": {
    "registry": "myreg.io"
  },
  "defaultProviders": {
    "ocneBootstrap": {
      "version": "v1.6.1"
    },
    "ocneControlPlane": {
      "version": "v1.6.1"
    },
    "oci": {
      "version": "v0.9.0"
     },
    "core": {
      "version": "v1.4.2"
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
	config.TestHelmConfigDir = TestHelmConfigDir

	overrides, err := createOverrides(compContext)
	assert.NoError(t, err)
	assert.NotNil(t, overrides)
	tc := newOverridesContext(overrides)

	assert.Equal(t, "https://github.com/verrazzano/cluster-api-provider-ocne/releases/v1.6.1/bootstrap-components.yaml", tc.GetOCNEBootstrapURL())
	assert.Equal(t, "https://github.com/verrazzano/cluster-api-provider-ocne/releases/v1.6.1/control-plane-components.yaml", tc.GetOCNEControlPlaneURL())
	assert.Equal(t, "https://github.com/verrazzano/cluster-api/releases/v1.4.2/core-components.yaml", tc.GetClusterAPIURL())
	assert.Equal(t, "https://github.com/oracle/cluster-api-provider-oci/releases/v0.9.0/infrastructure-components.yaml", tc.GetOCIURL())

}

// TestOverridesInterface tests the OverridesInterface
// GIVEN a set OverridesInput
//
//	WHEN the user supplies a OverridesInput
//	THEN verify the OverridesInterface returns the expected values
func TestOverridesInterface(t *testing.T) {
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
        "repository": "oci-repo",
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
	config.TestHelmConfigDir = TestHelmConfigDir

	overrides, err := createOverrides(compContext)
	assert.NoError(t, err)
	assert.NotNil(t, overrides)
	tc := newOverridesContext(overrides)

	assert.Equal(t, "ghcr.io", tc.GetGlobalRegistry())

	assert.Equal(t, "myreg.io/verrazzano", tc.GetOCNEBootstrapRepository())
	assert.Equal(t, "v1.0", tc.GetOCNEBootstrapTag())
	assert.Equal(t, "/test/bootstrap.yaml", tc.GetOCNEBootstrapURL())

	assert.Equal(t, "ghcr.io/verrazzano", tc.GetOCNEControlPlaneRepository())
	assert.Equal(t, "v1.1", tc.GetOCNEControlPlaneTag())
	assert.Equal(t, "/verrazzano/capi/control-plane-ocne/v0.1.0/control-plane-components.yaml", tc.GetOCNEControlPlaneURL())

	assert.Equal(t, "ghcr.io/verrazzano", tc.GetClusterAPIRepository())
	assert.Equal(t, CoreImageTag, tc.GetClusterAPITag())
	assert.Equal(t, "/verrazzano/capi/cluster-api/v1.3.3/core-components.yaml", tc.GetClusterAPIURL())

	assert.Equal(t, "myreg2.io/oci-repo", tc.GetOCIRepository())
	assert.Equal(t, "v0.8.2", tc.GetOCITag())
	assert.Equal(t, "https://github.com/oci-repo/cluster-api-provider-oci/releases/v0.8.2/infrastructure-components.yaml", tc.GetOCIURL())

}

// TestOverridesPrivateRegistry tests the OverridesInterface
// GIVEN a set OverridesInput
//
//	WHEN the user sets private registry image override variables
//	THEN verify the OverridesInterface returns the expected values
func TestOverridesPrivateRegistry(t *testing.T) {
	const privateRegistry = "myreg.io"
	const privateRepo = "private-repo"
	config.SetDefaultBomFilePath(testBomFilePath)
	os.Setenv(vzconst.RegistryOverrideEnvVar, privateRegistry)
	defer func() { os.Unsetenv(vzconst.RegistryOverrideEnvVar) }()
	os.Setenv(vzconst.ImageRepoOverrideEnvVar, privateRepo)
	defer func() { os.Unsetenv(vzconst.ImageRepoOverrideEnvVar) }()

	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects().Build()
	compContext := spi.NewFakeContext(fakeClient, &v1alpha1.Verrazzano{}, nil, false)
	config.TestHelmConfigDir = TestHelmConfigDir

	overrides, err := createOverrides(compContext)
	assert.NoError(t, err)
	assert.NotNil(t, overrides)
	tc := newOverridesContext(overrides)

	expectedRepo := fmt.Sprintf("%s/%s/verrazzano", privateRegistry, privateRepo)
	assert.Equal(t, expectedRepo, tc.GetOCNEBootstrapRepository())
	assert.Equal(t, expectedRepo, tc.GetOCNEControlPlaneRepository())
	assert.Equal(t, expectedRepo, tc.GetClusterAPIRepository())
	expectedRepo = fmt.Sprintf("%s/%s/oracle", privateRegistry, privateRepo)
	assert.Equal(t, expectedRepo, tc.GetOCIRepository())
}
