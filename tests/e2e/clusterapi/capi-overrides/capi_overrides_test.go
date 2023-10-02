// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package capioverrides

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	capipkg "github.com/verrazzano/verrazzano/tests/e2e/pkg/clusterapi"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
)

const (
	waitTimeout            = 5 * time.Minute
	pollingInterval        = 10 * time.Second
	longWaitTimeout        = 20 * time.Minute
	coreURLFmt             = "https://github.com/verrazzano/cluster-api/releases/%s/core-components.yaml"
	ocneBootstrapURLFmt    = "https://github.com/verrazzano/cluster-api-provider-ocne/releases/%s/bootstrap-components.yaml"
	ocneControlPlaneURLFmt = "https://github.com/verrazzano/cluster-api-provider-ocne/releases/%s/control-plane-components.yaml"
	ociInfraURLFmt         = "https://github.com/oracle/cluster-api-provider-oci/releases/%s/infrastructure-components.yaml"

	verrazzanoCAPINamespace          = "verrazzano-capi"
	capiCMDeployment                 = "capi-controller-manager"
	capiOcneBootstrapCMDeployment    = "capi-ocne-bootstrap-controller-manager"
	capiOcneControlPlaneCMDeployment = "capi-ocne-control-plane-controller-manager"
	capiociCMDeployment              = "capoci-controller-manager"

	managerContainerName = "manager"

	capiComponentName = "cluster-api"

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
	Expect(bomDoc).ToNot(BeNil())
	Expect(ociComp).ToNot(BeNil())
	Expect(ocneComp).ToNot(BeNil())
	Expect(coreComp).ToNot(BeNil())

	t.Context("initial state", func() {
		// GIVEN the Cluster API is installed
		// WHEN we check the initial state of the Verrazzano install
		// THEN we successfully find it ready
		capipkg.WhenClusterAPIInstalledIt(t, "the VZ status is ready", func() {
			Eventually(isStatusReady, waitTimeout, pollingInterval).Should(BeTrue())
		})
	})

	t.Context("override global registry", func() {
		// GIVEN a CAPI environment
		// WHEN we override the global registry
		// THEN the overrides get successfully applied
		capipkg.WhenClusterAPIInstalledIt(t, "and wait for deployments to use it", func() {
			Eventually(func() error {
				return updateClusterAPIOverrides(fmt.Sprintf(globalRegistryOverride, bomDoc.Registry))
			}, waitTimeout, pollingInterval).Should(BeNil())
			Eventually(isGenerationMet, waitTimeout, pollingInterval).Should(BeTrue(), "%s lastReconciledGeneration did not meet %v", capiComponentName)
			Eventually(isStatusReady, waitTimeout, pollingInterval).Should(BeTrue(), "Did not reach Ready status")
		})
	})

	t.Context("override image tags", func() {
		// GIVEN a CAPI environment
		// WHEN we override the image tags
		// THEN the overrides get successfully applied
		capipkg.WhenClusterAPIInstalledIt(t, "and wait for deployments to use it", func() {
			ociImageTag := ociComp.SubComponents[0].Images[0].ImageTag
			ocneImageTag := ocneComp.SubComponents[0].Images[0].ImageTag
			coreImageTag := coreComp.SubComponents[0].Images[0].ImageTag
			Eventually(func() error {
				return updateClusterAPIOverrides(fmt.Sprintf(tagOverrides, ociImageTag, ocneImageTag, ocneImageTag, coreImageTag))
			}, waitTimeout, pollingInterval).Should(BeNil())
			Eventually(isGenerationMet, waitTimeout, pollingInterval).Should(BeTrue(), "%s lastReconciledGeneration did not meet %v", capiComponentName)
			Eventually(isStatusReady, waitTimeout, pollingInterval).Should(BeTrue(), "Did not reach Ready status")
		})
	})

	t.Context("override repository", func() {
		// GIVEN a CAPI environment
		// WHEN we override the registry/repository tags
		// THEN the overrides get successfully applied
		capipkg.WhenClusterAPIInstalledIt(t, "and wait for deployments to use it", func() {
			ociRepo := ociComp.SubComponents[0].Repository
			ocneRepo := ocneComp.SubComponents[0].Repository
			coreRepo := ocneComp.SubComponents[0].Repository
			registry := bomDoc.Registry
			Eventually(func() error {
				return updateClusterAPIOverrides(fmt.Sprintf(repoOverrides, registry, ociRepo, registry, ocneRepo, registry,
					ocneRepo, registry, coreRepo))
			}, waitTimeout, pollingInterval).Should(BeNil())
			Eventually(isGenerationMet, waitTimeout, pollingInterval).Should(BeTrue(), "%s lastReconciledGeneration did not meet %v", capiComponentName)
			Eventually(isStatusReady, waitTimeout, pollingInterval).Should(BeTrue(), "Did not reach Ready status")
		})
	})

	t.Context("override oci, core, ocneBootstrap and ocneControlPlane versions from BOM", func() {
		// GIVEN the CAPI environment is ready
		// WHEN we override versions
		// THEN the overrides get successfully applied
		capipkg.WhenClusterAPIInstalledIt(t, "and wait for reconcile to complete", func() {
			// Using the current actual versions from the BOM, these are expected to work but download
			// from the internet instead of from the container image.
			Eventually(func() error {
				return updateClusterAPIOverrides(fmt.Sprintf(versionOverrides, ociComp.Version, ocneComp.Version, ocneComp.Version, coreComp.Version))
			}, waitTimeout, pollingInterval).Should(BeNil())
			Eventually(isGenerationMet, longWaitTimeout, pollingInterval).Should(BeTrue(), "%s lastReconciledGeneration did not meet %v", capiComponentName)
			Eventually(isStatusReady, longWaitTimeout, pollingInterval).Should(BeTrue(), "Did not reach Ready status")

			_, err := pkg.ValidateDeploymentContainerImage(verrazzanoCAPINamespace, capiOcneControlPlaneCMDeployment, managerContainerName, ocneComp.Version)
			Expect(err).ShouldNot(HaveOccurred())
			_, err = pkg.ValidateDeploymentContainerImage(verrazzanoCAPINamespace, capiOcneBootstrapCMDeployment, managerContainerName, ocneComp.Version)
			Expect(err).ShouldNot(HaveOccurred())
			_, err = pkg.ValidateDeploymentContainerImage(verrazzanoCAPINamespace, capiociCMDeployment, managerContainerName, ociComp.Version)
			Expect(err).ShouldNot(HaveOccurred())
			_, err = pkg.ValidateDeploymentContainerImage(verrazzanoCAPINamespace, capiCMDeployment, managerContainerName, coreComp.Version)
			Expect(err).ShouldNot(HaveOccurred())
		})
	})

	t.Context("adhoc override oci, core, ocneBootstrap and ocneControlPlane versions", func() {
		// GIVEN the CAPI environment is ready
		// WHEN we override versions
		// THEN the overrides get successfully applied
		capipkg.WhenClusterAPIInstalledIt(t, "and wait for reconcile to complete", func() {
			// Using the current actual versions from the BOM, these are expected to work but download
			// from the internet instead of from the container image.
			Eventually(func() error {
				return updateClusterAPIOverrides(fmt.Sprintf(versionOverrides, "v0.11.0", "v1.6.1", "v1.6.1", "v1.4.2"))
			}, waitTimeout, pollingInterval).Should(BeNil())
			Eventually(isGenerationMet, longWaitTimeout, pollingInterval).Should(BeTrue(), "%s lastReconciledGeneration did not meet %v", capiComponentName)
			Eventually(isStatusReady, longWaitTimeout, pollingInterval).Should(BeTrue(), "Did not reach Ready status")

			_, err := pkg.ValidateDeploymentContainerImage(verrazzanoCAPINamespace, capiOcneControlPlaneCMDeployment, managerContainerName, "v1.6.1")
			Expect(err).ShouldNot(HaveOccurred())
			_, err = pkg.ValidateDeploymentContainerImage(verrazzanoCAPINamespace, capiOcneBootstrapCMDeployment, managerContainerName, "v1.6.1")
			Expect(err).ShouldNot(HaveOccurred())
			_, err = pkg.ValidateDeploymentContainerImage(verrazzanoCAPINamespace, capiociCMDeployment, managerContainerName, "v0.11.0")
			Expect(err).ShouldNot(HaveOccurred())
			_, err = pkg.ValidateDeploymentContainerImage(verrazzanoCAPINamespace, capiCMDeployment, managerContainerName, "v1.4.2")
			Expect(err).ShouldNot(HaveOccurred())
		})
	})

	t.Context("override URL of each provider", func() {
		// GIVEN the CAPI environment is ready
		// WHEN we override the URL of each provider
		// THEN the overrides get successfully applied
		capipkg.WhenClusterAPIInstalledIt(t, "and wait for reconcile to complete", func() {
			// Using the current actual versions from the BOM, these are expected to work but download
			// from the internet instead of from the container image.
			Eventually(func() error {
				return updateClusterAPIOverrides(fmt.Sprintf(urlOverrides,
					fmt.Sprintf(ociInfraURLFmt, "v0.11.0"),
					fmt.Sprintf(ocneBootstrapURLFmt, "v1.6.1"),
					fmt.Sprintf(ocneControlPlaneURLFmt, "v1.6.1"),
					fmt.Sprintf(coreURLFmt, "v1.4.2")))
			}, waitTimeout, pollingInterval).Should(BeNil())
			Eventually(isGenerationMet, waitTimeout, pollingInterval).Should(BeTrue(), "%s lastReconciledGeneration did not meet %v", capiComponentName)
			Eventually(isStatusReady, waitTimeout, pollingInterval).Should(BeTrue(), "Did not reach Ready status")

			_, err := pkg.ValidateDeploymentContainerImage(verrazzanoCAPINamespace, capiOcneControlPlaneCMDeployment, managerContainerName, "v1.6.1")
			Expect(err).ShouldNot(HaveOccurred())
			_, err = pkg.ValidateDeploymentContainerImage(verrazzanoCAPINamespace, capiOcneBootstrapCMDeployment, managerContainerName, "v1.6.1")
			Expect(err).ShouldNot(HaveOccurred())
			_, err = pkg.ValidateDeploymentContainerImage(verrazzanoCAPINamespace, capiociCMDeployment, managerContainerName, "v0.11.0")
			Expect(err).ShouldNot(HaveOccurred())
			_, err = pkg.ValidateDeploymentContainerImage(verrazzanoCAPINamespace, capiCMDeployment, managerContainerName, "v1.4.2")
			Expect(err).ShouldNot(HaveOccurred())
		})
	})

	t.Context("restore VZ to default values for clusterAPI", func() {
		// GIVEN the CAPI environment is ready
		// WHEN we remove the overrides
		// THEN the default values will get restored
		capipkg.WhenClusterAPIInstalledIt(t, "and wait for reconcile to complete", func() {
			Eventually(func() error {
				return updateClusterAPIOverrides("")
			}, waitTimeout, pollingInterval).Should(BeNil())
			Eventually(isGenerationMet, waitTimeout, pollingInterval).Should(BeTrue(), "%s lastReconciledGeneration did not meet %v", capiComponentName)
			Eventually(isStatusReady, waitTimeout, pollingInterval).Should(BeTrue(), "Did not reach Ready status")

			_, err := pkg.ValidateDeploymentContainerImage(verrazzanoCAPINamespace, capiOcneControlPlaneCMDeployment, managerContainerName, ocneComp.Version)
			Expect(err).ShouldNot(HaveOccurred())
			_, err = pkg.ValidateDeploymentContainerImage(verrazzanoCAPINamespace, capiOcneBootstrapCMDeployment, managerContainerName, ocneComp.Version)
			Expect(err).ShouldNot(HaveOccurred())
			_, err = pkg.ValidateDeploymentContainerImage(verrazzanoCAPINamespace, capiociCMDeployment, managerContainerName, ociComp.Version)
			Expect(err).ShouldNot(HaveOccurred())
			_, err = pkg.ValidateDeploymentContainerImage(verrazzanoCAPINamespace, capiCMDeployment, managerContainerName, coreComp.Version)
			Expect(err).ShouldNot(HaveOccurred())
		})
	})
})

func isStatusReady() bool {
	return isStatusMet(v1beta1.VzStateReady)
}

// isGenerationMet - Return boolean indicating if expected status is met
func isGenerationMet() (bool, error) {
	// Get the VZ resource
	vz, err := pkg.GetVerrazzanoV1beta1()
	if err != nil {
		return false, err
	}
	componentStatus, found := vz.Status.Components[capiComponentName]
	if !found {
		return false, fmt.Errorf("did not find status for component %s", capiComponentName)
	}
	t.Logs.Debugf("VZ generation: %v, %s generation: %v", vz.Generation, capiComponentName, componentStatus.LastReconciledGeneration)
	return componentStatus.LastReconciledGeneration == vz.Generation, nil
}

// isStatusMet - Return boolean indicating if expected status is met
func isStatusMet(state v1beta1.VzStateType) bool {
	// Get the VZ resource
	vz, err := pkg.GetVerrazzanoV1beta1()
	if err != nil {
		return false
	}
	return vz.Status.State == state
}

// updateClusterAPIOverrides - Update the VZ with the set of overrides pass for clusterAPI component
func updateClusterAPIOverrides(overrides string) error {
	// Get the VZ resource
	vz, err := pkg.GetVerrazzanoV1beta1()
	if err != nil {
		return err
	}

	// Get the client
	client, err := pkg.GetVerrazzanoClientset()
	if err != nil {
		return err
	}

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
		return nil, nil, nil, nil
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
	return bomDoc, ociComp, capiComp, coreComp
}
