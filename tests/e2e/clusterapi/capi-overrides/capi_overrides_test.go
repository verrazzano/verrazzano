// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package capioverrides

import (
	"context"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	capipkg "github.com/verrazzano/verrazzano/tests/e2e/pkg/clusterapi"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	appsv1 "k8s.io/api/apps/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
)

const (
	waitTimeout       = 5 * time.Minute
	pollingInterval   = 10 * time.Second
	imageTagOverride  = "v1.2.3"
	imageRepoOverride = "acme"

	coreURLFmt             = "https://github.com/verrazzano/cluster-api/releases/download/%s/core-components.yaml"
	ocneBootstrapURLFmt    = "https://github.com/verrazzano/cluster-api-provider-ocne/releases/download/%s/bootstrap-components.yaml"
	ocneControlPlaneURLFmt = "https://github.com/verrazzano/cluster-api-provider-ocne/releases/download/%s/control-plane-components.yaml"
	ociInfraURLFmt         = "https://github.com/oracle/cluster-api-provider-oci/releases/download/%s/infrastructure-components.yaml"

	// globalRegistryOverride - format string to override the registry to use for all providers
	globalRegistryOverride = `
{
  "global": {
    "registry": "%s"
  }
}
`
	// tagOverrides - format string to override the image tags of each provider
	tagOverrides = `
{
  "defaultProviders": {
    "oci": {
      "image": {
        "tag": "%s"
      }
    },
    "ocneBootstrap": {
      "image": {
        "tag": "%s"
      }
    },
    "ocneControlPlane": {
      "image": {
        "tag": "%s"
      }
    },
    "core": {
      "image": {
        "tag": "%s"
      }
    }
  }
}`

	// repoOverrides - format string to override the image registry and repository tags of each provider
	repoOverrides = `
{
  "defaultProviders": {
    "oci": {
      "image": {
        "registry": "%s",
        "repository": "%s"
      }
    },
    "ocneBootstrap": {
      "image": {
        "registry": "%s",
        "repository": "%s"
      }
    },
    "ocneControlPlane": {
      "image": {
        "registry": "%s",
        "repository": "%s"
      }
    },
    "core": {
      "image": {
        "registry": "%s",
        "repository": "%s"
      }
    }
  }
}`

	// versionOverrides - format string to override the version of each provider
	versionOverrides = `
{
  "defaultProviders": {
    "oci": {
      "version": "%s"
    },
    "ocneBootstrap": {
      "version": "%s"
    },
    "ocneControlPlane": {
      "version": "%s"
	   },
    "core": {
      "version": "%s"
    }
  }
}`

	// urlOverrides - format string to override the URL of each provider
	urlOverrides = `
{
  "defaultProviders": {
    "oci": {
      "url": "%s"
    },
    "ocneBootstrap": {
      "url": "%s"
    },
    "ocneControlPlane": {
      "url": "%s"
    },
    "core": {
      "url": "%s"
    }
  }
}`
)

var t = framework.NewTestFramework("capi_overrides")

var _ = t.Describe("Cluster API", Label("f:platform-lcm.install"), func() {
	var dynClient dynamic.Interface

	// Get dynamic client
	Eventually(func() (dynamic.Interface, error) {
		kubePath, err := k8sutil.GetKubeConfigLocation()
		if err != nil {
			return nil, err
		}
		dynClient, err = pkg.GetDynamicClientInCluster(kubePath)
		return dynClient, err
	}, waitTimeout, pollingInterval).ShouldNot(BeNil())

	// Get the components from the BOM to pick up their current versions
	bomDoc, ociComp, ocneComp, coreComp := getComponentsFromBom()

	t.Context("initial state", func() {
		// GIVEN the Cluster API is installed
		// WHEN we check the initial state of the Verrazzano install
		// THEN we successfully find it ready
		capipkg.WhenClusterAPIInstalledIt(t, "kontainerdrivers are active", func() {
			Eventually(isStatusReady, waitTimeout, pollingInterval).Should(BeTrue())
		})
	})

	t.Context("override global registry", func() {
		// GIVEN a CAPI environment
		// WHEN we override the global registry
		// THEN the overrides get successfully applied
		capipkg.WhenClusterAPIInstalledIt(t, "and wait for deployments to use it", func() {
			Eventually(func() bool {
				return updateClusterAPIOverrides(fmt.Sprintf(globalRegistryOverride, bomDoc.Registry)) == nil
			}, waitTimeout, pollingInterval).Should(BeTrue())
			Eventually(isStatusReconciling, waitTimeout, pollingInterval).Should(BeTrue())
			Eventually(isStatusReady, waitTimeout, pollingInterval).Should(BeTrue())
		})
	})

	t.Context("override image tags", func() {
		// GIVEN a CAPI environment
		// WHEN we override the image tags
		// THEN the overrides get successfully applied
		capipkg.WhenClusterAPIInstalledIt(t, "and wait for deployments to use it", func() {
			Eventually(func() bool {
				return updateClusterAPIOverrides(fmt.Sprintf(tagOverrides, imageTagOverride, imageTagOverride, imageTagOverride, imageTagOverride)) == nil
			}, waitTimeout, pollingInterval).Should(BeTrue())
			Eventually(isStatusReconciling, waitTimeout, pollingInterval).Should(BeTrue())
			// The CAPI pods are now in a broken state because the image tag does not exist.
			// Verify the deployments get updated to use the new value.
			Eventually(isImageTagUsed, waitTimeout, pollingInterval).Should(BeTrue())
		})
	})

	t.Context("override repository", func() {
		// GIVEN a CAPI environment
		// WHEN we override the registry/repository tags
		// THEN the overrides get successfully applied
		capipkg.WhenClusterAPIInstalledIt(t, "and wait for deployments to use it", func() {
			Eventually(func() bool {
				return updateClusterAPIOverrides(fmt.Sprintf(repoOverrides, bomDoc.Registry, imageRepoOverride,
					bomDoc.Registry, imageRepoOverride, bomDoc.Registry, imageRepoOverride, bomDoc.Registry, imageRepoOverride)) == nil
			}, waitTimeout, pollingInterval).Should(BeTrue())
			Eventually(isStatusReconciling, waitTimeout, pollingInterval).Should(BeTrue())
			// The CAPI pods are now in a broken state because the repository does not exist.
			// Verify the deployments get updated to use the new value.
			Eventually(isRepositoryUsed, waitTimeout, pollingInterval).Should(BeTrue())
		})
	})

	t.Context("override oci, core, ocneBootstrap and ocneControlPlane versions", func() {
		// GIVEN the CAPI environment is ready
		// WHEN we override versions
		// THEN the overrides get successfully applied
		capipkg.WhenClusterAPIInstalledIt(t, "and wait for reconcile to complete", func() {
			// Using the current actual versions from the BOM, these are expected to work but download
			// from the internet instead of from the container image.
			Eventually(func() bool {
				return updateClusterAPIOverrides(fmt.Sprintf(versionOverrides, ociComp.Version, ocneComp.Version, ocneComp.Version, coreComp.Version)) == nil
			}, waitTimeout, pollingInterval).Should(BeTrue())
			Eventually(isStatusReconciling, waitTimeout, pollingInterval).Should(BeTrue())
			Eventually(isStatusReady, waitTimeout, pollingInterval).Should(BeTrue())
		})
	})

	t.Context("override URL of each provider", func() {
		// GIVEN the CAPI environment is ready
		// WHEN we override the URL of each provider
		// THEN the overrides get successfully applied
		capipkg.WhenClusterAPIInstalledIt(t, "and wait for reconcile to complete", func() {
			// Using the current actual versions from the BOM, these are expected to work but download
			// from the internet instead of from the container image.
			Eventually(func() bool {
				return updateClusterAPIOverrides(fmt.Sprintf(urlOverrides,
					fmt.Sprintf(ociInfraURLFmt, ociComp.Version),
					fmt.Sprintf(ocneBootstrapURLFmt, ocneComp.Version),
					fmt.Sprintf(ocneControlPlaneURLFmt, ocneComp.Version),
					fmt.Sprintf(coreURLFmt, coreComp.Version))) == nil
			}, waitTimeout, pollingInterval).Should(BeTrue())
			Eventually(isStatusReconciling, waitTimeout, pollingInterval).Should(BeTrue())
			Eventually(isStatusReady, waitTimeout, pollingInterval).Should(BeTrue())
		})
	})

	t.Context("restore VZ to default values for clusterAPI", func() {
		// GIVEN the CAPI environment is ready
		// WHEN we remove the overrides
		// THEN the default values will get restored
		capipkg.WhenClusterAPIInstalledIt(t, "and wait for reconcile to complete", func() {
			Eventually(func() bool {
				return updateClusterAPIOverrides("") == nil
			}, waitTimeout, pollingInterval).Should(BeTrue())
			Eventually(isStatusReconciling, waitTimeout, pollingInterval).Should(BeTrue())
			Eventually(isStatusReady, waitTimeout, pollingInterval).Should(BeTrue())
		})
	})
})

func isStatusReconciling() bool {
	return isStatusMet(v1beta1.VzStateReconciling)
}

func isStatusReady() bool {
	return isStatusMet(v1beta1.VzStateReady)
}

// isStatusMet - Return boolean indicating if expected status is met
func isStatusMet(state v1beta1.VzStateType) bool {
	// Get the VZ resource
	vz, err := pkg.GetVerrazzanoV1beta1()
	Expect(err).ToNot(HaveOccurred())
	return vz.Status.State == state
}

// updateClusterAPIOverrides - Update the VZ with the set of overrides pass for clusterAPI component
func updateClusterAPIOverrides(overrides string) error {
	// Get the VZ resource
	vz, err := pkg.GetVerrazzanoV1beta1()
	Expect(err).ToNot(HaveOccurred())

	// Get the client
	client, err := pkg.GetVerrazzanoClientset()
	Expect(err).ToNot(HaveOccurred())

	// Update the VZ with the overrides
	if len(overrides) == 0 {
		// Restore the VZ to default values
		vz.Spec.Components.ClusterAPI = nil
	} else {
		vz.Spec.Components.ClusterAPI = &v1beta1.ClusterAPIComponent{}
		vz.Spec.Components.ClusterAPI.InstallOverrides = v1beta1.InstallOverrides{
			ValueOverrides: []v1beta1.Overrides{
				{
					Values: &apiextensionsv1.JSON{
						Raw: []byte(overrides),
					},
				},
			},
		}
	}

	_, err = client.VerrazzanoV1beta1().Verrazzanos(vz.Namespace).Update(context.TODO(), vz, metav1.UpdateOptions{})
	return err
}

// getComponentsFromBom - return some components from the BOM file
func getComponentsFromBom() (*bom.BomDoc, *bom.BomComponent, *bom.BomComponent, *bom.BomComponent) {
	// Get the BOM from the installed Platform Operator
	bomDoc, err := pkg.GetBOMDoc()
	if err != nil {
		AbortSuite(fmt.Sprintf("Failed to get BOM from platform operator: %v", err))
	}

	// Find the Rancher and CAPI components
	var ociComp *bom.BomComponent
	var capiComp *bom.BomComponent
	var coreComp *bom.BomComponent
	for i, component := range bomDoc.Components {
		switch component.Name {
		case "capi-oci":
			ociComp = &bomDoc.Components[i]
		case "capi-ocne":
			capiComp = &bomDoc.Components[i]
		case "capi-cluster-api":
			coreComp = &bomDoc.Components[i]
		}
	}
	Expect(ociComp).ToNot(BeNil())
	Expect(capiComp).ToNot(BeNil())
	Expect(coreComp).ToNot(BeNil())
	return bomDoc, ociComp, capiComp, coreComp
}

// isImageTagUsed - determine if the image tag override is being used in the CAPI deployments
func isImageTagUsed() bool {
	return isSubstringInDeploymentImages(fmt.Sprintf(":%s", imageTagOverride))
}

// isRepositoryUsed - determine if the image tag overrides for registry and repo are being used in the CAPI deployments
func isRepositoryUsed() bool {
	return isSubstringInDeploymentImages(fmt.Sprintf("%s/%s", "ghcr.io", imageRepoOverride))
}

func isSubstringInDeploymentImages(substring string) bool {
	var capiFound, ocneBootstrapFound, ocneControlFound, ociFound = false, false, false, false
	deployments := getCAPIDeployments()
	for _, deployment := range deployments {
		for _, container := range deployment.Spec.Template.Spec.Containers {
			if strings.Contains(container.Image, substring) {
				switch deployment.Name {
				case "capi-controller-manager":
					capiFound = true
				case "capi-ocne-bootstrap-controller-manager":
					ocneBootstrapFound = true
				case "capi-ocne-control-plane-controller-manager":
					ocneControlFound = true
				case "capoci-controller-manager":
					ociFound = true
				}
			}
		}
	}
	// Expect four deployments to match
	return capiFound && ocneBootstrapFound && ocneControlFound && ociFound
}

func getCAPIDeployments() []appsv1.Deployment {
	v1client, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		return []appsv1.Deployment{}
	}
	deployments, err := v1client.AppsV1().Deployments("verrazzano-capi").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return []appsv1.Deployment{}
	}
	return deployments.Items
}
