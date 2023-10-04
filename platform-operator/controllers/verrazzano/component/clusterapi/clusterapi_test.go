// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusterapi

import (
	"os"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8scheme "k8s.io/client-go/kubernetes/scheme"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testBomFilePath = "../../testdata/test_bom.json"
)

// TestSetEnvVariables tests the setEnvVariables function
// GIVEN a call to setEnvVariables
//
//	WHEN all env variables are set to the correct values
//	THEN true is returned
func TestSetEnvVariables(t *testing.T) {
	err := setEnvVariables()
	assert.Equal(t, "false", os.Getenv(initOCIClientsOnStartup))
	assert.Equal(t, "true", os.Getenv(expClusterResourceSet))
	assert.Equal(t, "true", os.Getenv(expMachinePool))
	assert.Equal(t, "true", os.Getenv(clusterTopology))
	assert.NoError(t, err)
}

// TestApplyTemplate tests the applyTemplate function
// GIVEN a call to applyTemplate
//
//	WHEN the template input is supplied
//	THEN a buffer containing the contents of clusterctl.yaml is returned and all parameters replaced
func TestApplyTemplate(t *testing.T) {
	fakeClient := fake.NewClientBuilder().Build()
	compContext := spi.NewFakeContext(fakeClient, &v1alpha1.Verrazzano{}, nil, false)
	config.SetDefaultBomFilePath(testBomFilePath)
	overrides, err := createOverrides(compContext)
	assert.NoError(t, err)
	assert.NotNil(t, overrides)
	clusterctl, err := applyTemplate(compContext, clusterctlYamlTemplate, overrides)
	assert.NoError(t, err)
	assert.NotEmpty(t, clusterctl)
	assert.NotContains(t, clusterctl.String(), "{{.")
}

// TestCreateClusterctlYaml tests the createClusterctlYaml function
// GIVEN a call to createClusterctlYaml
//
//	WHEN overrides from the BOM are applied
//	THEN a clusterctl.yaml file is created
func TestCreateClusterctlYaml(t *testing.T) {
	fakeClient := fake.NewClientBuilder().Build()
	compContext := spi.NewFakeContext(fakeClient, &v1alpha1.Verrazzano{}, nil, false)
	config.SetDefaultBomFilePath(testBomFilePath)
	dir := os.TempDir() + "/" + time.Now().Format("20060102150405")
	setClusterAPIDir(dir)
	defer resetClusterAPIDir()
	defer os.RemoveAll(dir)
	err := createClusterctlYaml(compContext)
	assert.NoError(t, err)
	_, err = os.Stat(dir + "/clusterctl.yaml")
	assert.NoError(t, err)
}

// TestToSkipSettingUpgradeOptions tests the matchAndPrepareUpgradeOptions function
// GIVEN a call to matchAndPrepareUpgradeOptions
//
//	WHEN images are initialized from BOM with overrides and same images as in the Running Pods.
//	THEN upgrade options are not set since there is no change in the image
func TestToSkipSettingUpgradeOptions(t *testing.T) {
	const capiOverrides = `
{
  "global": {
    "registry": "ghcr.io"
  },
  "defaultProviders": {
    "ocneBootstrap": {
      "image": {
        "registry": "myreg.io",
        "tag": "v1.0"
      }
    },
    "ocneControlPlane": {
      "image": {
        "repository": "verrazzano",
        "registry": "ghcr.io",
        "tag": "1.0.0-1-20211215184123-0a1b633"
      }
    },
    "oci": {
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
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(&corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ComponentNamespace,
			Name:      controlPlaneOcneProvider + "-95d8c5d97-m6mbr",
			Labels: map[string]string{
				providerLabel: clusterAPIProvider,
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  clusterAPIOCNEControlPLaneControllerImage,
					Image: "ghcr.io/verrazzano/cluster-api-ocne-control-plane-controller:1.0.0-1-20211215184123-0a1b633",
				},
			},
		},
	}).Build()
	compContext := spi.NewFakeContext(fakeClient, vz, nil, false)
	config.SetDefaultBomFilePath(testBomFilePath)
	config.TestHelmConfigDir = TestHelmConfigDir
	overrides, err := createOverrides(compContext)
	assert.NoError(t, err)
	assert.NotNil(t, overrides)
	overridesContext := newOverridesContext(overrides)
	podMatcher := &PodMatcherClusterAPI{}
	applyUpgradeOptions, err := podMatcher.matchAndPrepareUpgradeOptions(compContext, overridesContext)
	assert.NoError(t, err)
	assert.False(t, isUpgradeOptionsNotEmpty(applyUpgradeOptions))
	assert.NoError(t, err)
}

// TestToSetUpgradeOptions tests the matchAndPrepareUpgradeOptions function
// GIVEN a call to matchAndPrepareUpgradeOptions
//
//	WHEN images are initialized from BOM with overrides
//	THEN upgrade options are set only for the outdated cluster API images.
//
// Ex: In this test cluster-api-ocne-control-plane-controller image is outdated
// Therefore, upgrade options is set with the updated cluster-api-ocne-control-plane-controller image.
func TestToSetUpgradeOptions(t *testing.T) {
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
        "repository": "verrazzano",
        "registry": "ghcr.io",
        "tag": "0.0.1-1-20211215184123-0a1b633"
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

	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      controlPlaneOcneProvider + "-95d8c5d97-m6mbr",
				Labels: map[string]string{
					providerLabel: clusterAPIProvider,
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  clusterAPIOCNEControlPLaneControllerImage,
						Image: "ghcr.io/verrazzano/cluster-api-ocne-control-plane-controller:0.0.1-20211215184123-older",
					},
				},
			},
		}).Build()

	compContext := spi.NewFakeContext(fakeClient, vz, nil, false)
	config.SetDefaultBomFilePath(testBomFilePath)
	config.TestHelmConfigDir = TestHelmConfigDir
	overrides, err := createOverrides(compContext)
	assert.NoError(t, err)
	assert.NotNil(t, overrides)
	overridesContext := newOverridesContext(overrides)
	podMatcher := &PodMatcherClusterAPI{}
	applyUpgradeOptions, err := podMatcher.matchAndPrepareUpgradeOptions(compContext, overridesContext)
	assert.NoError(t, err)
	assert.True(t, isUpgradeOptionsNotEmpty(applyUpgradeOptions))
	// Checking for specific upgrade option that contains component namespace, OCNE provider, and it's version.
	// Upgrade option is not empty only if there is a change in the image.
	expectedUpgradeOption := []string{"verrazzano-capi/ocne:v0.1.0"}
	assert.Equal(t, expectedUpgradeOption, applyUpgradeOptions.ControlPlaneProviders)
	assert.NoError(t, err)
}
