// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package installscript_test

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	waitTimeout     = 5 * time.Minute
	pollingInterval = 5 * time.Second
)

var kubeConfigFromEnv = os.Getenv("KUBECONFIG")
var totalClusters, present = os.LookupEnv("CLUSTER_COUNT")

// This test checks that the verrazzano install resource has the expected console URLs.
var _ = Describe("Verify Verrazzano install scripts", func() {

	Context("Verify Console URLs in the installed verrazzano resource", func() {
		clusterCount, _ := strconv.Atoi(totalClusters)
		if present && clusterCount > 0 {
			It("Verify the expected console URLs are there in the installed verrazzano resource for the managed cluster(s)", func() {
				// Validation for admin cluster
				Eventually(func() bool {
					return validateConsoleUrlsCluster(kubeConfigFromEnv)
				}, waitTimeout, pollingInterval).Should(BeTrue())

				// Validation for managed clusters
				for i := 2; i <= clusterCount; i++ {
					Eventually(func() bool {
						return validateConsoleUrlsCluster(strings.Replace(kubeConfigFromEnv, "1", strconv.Itoa(i), 1))
					}, waitTimeout, pollingInterval).Should(BeTrue())
				}
			})
		} else {
			It("Verify the expected console URLs are there in the installed verrazzano resource", func() {
				Eventually(func() bool {
					return validateConsoleUrlsCluster(kubeConfigFromEnv)
				}, waitTimeout, pollingInterval).Should(BeTrue())
			})
		}
	})
})

// Validate the console URLs for the admin cluster and single cluster installation
func validateConsoleUrlsCluster(kubeconfig string) bool {
	consoleUrls, err := getConsoleURLsFromResource(kubeconfig)
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("There is an error getting console URLs from the installed verrazzano resource - %v", err))
		return false
	}
	expectedConsoleUrls, err := getExpectedConsoleURLs(kubeconfig)
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("There is an error getting console URLs from ingress resources - %v", err))
		return false
	}

	return pkg.SlicesContainSameStrings(consoleUrls, expectedConsoleUrls)
}

// Get the list of console URLs from the status block of the installed verrazzano resource
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

	for _, ingress := range ingresses.Items {
		expectedUrls = append(expectedUrls, fmt.Sprintf("https://%s", ingress.Spec.Rules[0].Host))
	}

	return expectedUrls, nil
}
