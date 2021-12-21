// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package installscript_test

import (
	"context"
	"fmt"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"os"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	waitTimeout     = 5 * time.Minute
	pollingInterval = 5 * time.Second
)

var kubeConfigFromEnv = os.Getenv("KUBECONFIG")

// This test checks that the Verrazzano install resource has the expected console URLs.
var _ = Describe("Verify Verrazzano install scripts", func() {

	Context("Verify Console URLs in the installed Verrazzano resource", func() {
		It("Verify the expected console URLs are there in the installed Verrazzano resource", func() {
			// Validation for passed in cluster
			Eventually(func() bool {
				return validateConsoleUrlsCluster(kubeConfigFromEnv)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})
	})
})

// Validate the console URLs for the admin cluster and single cluster installation
func validateConsoleUrlsCluster(kubeconfig string) bool {
	consoleUrls, err := getConsoleURLsFromResource(kubeconfig)
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("There is an error getting console URLs from the installed Verrazzano resource - %v", err))
		return false
	}
	expectedConsoleUrls, err := getExpectedConsoleURLs(kubeconfig)
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("There is an error getting console URLs from ingress resources - %v", err))
		return false
	}
	pkg.Log(pkg.Info, fmt.Sprintf("Expected URLs based on ingresses: %v", expectedConsoleUrls))
	pkg.Log(pkg.Info, fmt.Sprintf("Actual URLs in Verrazzano resource: %v", consoleUrls))

	return pkg.SlicesContainSameStrings(consoleUrls, expectedConsoleUrls)
}

// Get the list of console URLs from the status block of the installed Verrazzano resource
func getConsoleURLsFromResource(kubeconfig string) ([]string, error) {
	var consoleUrls []string
	vz, err := pkg.GetVerrazzanoInstallResourceInCluster(kubeconfig)
	if err != nil {
		return consoleUrls, err
	}

	if vz.Status.VerrazzanoInstance.ConsoleURL != nil {
		consoleUrls = append(consoleUrls, *vz.Status.VerrazzanoInstance.ConsoleURL)
	}
	if vz.Status.VerrazzanoInstance.GrafanaURL != nil {
		consoleUrls = append(consoleUrls, *vz.Status.VerrazzanoInstance.GrafanaURL)
	}
	if vz.Status.VerrazzanoInstance.ElasticURL != nil {
		consoleUrls = append(consoleUrls, *vz.Status.VerrazzanoInstance.ElasticURL)
	}
	if vz.Status.VerrazzanoInstance.KeyCloakURL != nil {
		consoleUrls = append(consoleUrls, *vz.Status.VerrazzanoInstance.KeyCloakURL)
	}
	if vz.Status.VerrazzanoInstance.KibanaURL != nil {
		consoleUrls = append(consoleUrls, *vz.Status.VerrazzanoInstance.KibanaURL)
	}
	if vz.Status.VerrazzanoInstance.KialiURL != nil {
		consoleUrls = append(consoleUrls, *vz.Status.VerrazzanoInstance.KialiURL)
	}
	if vz.Status.VerrazzanoInstance.PrometheusURL != nil {
		consoleUrls = append(consoleUrls, *vz.Status.VerrazzanoInstance.PrometheusURL)
	}
	if vz.Status.VerrazzanoInstance.RancherURL != nil {
		consoleUrls = append(consoleUrls, *vz.Status.VerrazzanoInstance.RancherURL)
	}

	return consoleUrls, nil
}

// Get the expected console URLs for the given cluster from the ingress resources
func getExpectedConsoleURLs(kubeConfig string) ([]string, error) {
	var expectedUrls []string
	clientset, err := pkg.GetKubernetesClientsetForCluster(kubeConfig)
	if err != nil {
		return expectedUrls, err
	}
	ingresses, err := clientset.NetworkingV1().Ingresses("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return expectedUrls, err
	}

	consoleURLExpected, err := isConsoleURLExpected(kubeConfig)
	if err != nil {
		return expectedUrls, err
	}

	for _, ingress := range ingresses.Items {
		ingressHost := ingress.Spec.Rules[0].Host
		// If it's not the console ingress, or it is and the console is enabled, add it to the expected set of URLs
		if !isConsoleIngressHost(ingressHost) || consoleURLExpected {
			expectedUrls = append(expectedUrls, fmt.Sprintf("https://%s", ingressHost))
		}
	}

	return expectedUrls, nil
}

// isConsoleIngressHost - Returns true if the given ingress host is the one for the VZ UI console
func isConsoleIngressHost(ingressHost string) bool {
	return strings.HasPrefix(ingressHost, "verrazzano.")
}

// isConsoleURLExpected - Returns true in VZ < 1.2.0. For VZ >= 1.2.0, returns false only if explicitly disabled
// in the CR or when managed cluster profile is used
func isConsoleURLExpected(kubeconfigPath string) (bool, error) {
	isAtleastVz12, err := pkg.IsVerrazzanoMinVersion("1.2.0")
	if err != nil {
		return false, err
	}
	// Pre 1.2.0, the console URL was always present irrespective of whether console is enabled
	// This behavior changed in VZ 1.2.0
	if !isAtleastVz12 {
		return true, nil
	}

	// In 1.2.0 and later, the console URL will only be present in the VZ status instance info if the console is enabled
	vz, err := pkg.GetVerrazzanoInstallResourceInCluster(kubeconfigPath)
	if err != nil {
		return false, err
	}
	// Return the value of the Console enabled flag if present
	if vz != nil && vz.Spec.Components.Console != nil && vz.Spec.Components.Console.Enabled != nil {
		return *vz.Spec.Components.Console.Enabled, nil
	}
	// otherwise, expect console to be enabled for all profiles but managed-cluster
	return vz.Spec.Profile != vzapi.ManagedCluster, nil
}
