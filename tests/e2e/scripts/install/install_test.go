// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package installscript_test

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	vzClient "github.com/verrazzano/verrazzano/platform-operator/clients/verrazzano/clientset/versioned"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	verrazzanoSystemNamespace = "verrazzano-system"
	rancherNamespace          = "cattle-system"
	keycloakNamespace         = "keycloak"
	grafanaIngress            = "vmi-system-grafana"
	prometheusIngress         = "vmi-system-prometheus"
	elasticsearchIngress      = "vmi-system-es-ingest"
	kibanaIngress             = "vmi-system-kibana"
	kialiIngress              = "vms-system-kiali"
	verrazzanoIngress         = "verrazzano-ingress"
	keycloakIngress           = "keycloak"
	rancherIngress            = "rancher"
)

const (
	waitTimeout     = 5 * time.Minute
	pollingInterval = 5 * time.Second
)

var kubeConfigFromEnv = os.Getenv("KUBECONFIG")
var totalClusters, present = os.LookupEnv("CLUSTER_COUNT")

// This test checks that the console output at the end of an install does not show a
// user URL's that do not exist for that installation platform.  For example, a managed
// cluster install would not have console URLs.
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
					consoleUrls, err := getConsoleURLsFromResource()
					Expect(err).ShouldNot(HaveOccurred(), "There is an error getting console URLs from the installed verrazzano resource")

					// By default, install logs of the managed clusters do not contain the console URLs
					Expect(consoleUrls).To(BeEmpty())
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
	consoleUrls, err := getConsoleURLsFromResource()
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("There is an error getting console URLs from the installed verrazzano resource - %v", err))
		return false
	}
	expectedConsoleUrls, err := getExpectedConsoleURLs(kubeconfig)
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("There is an error getting console URLs from the API server - %v", err))
		return false
	}

	return pkg.SlicesContainSameStrings(consoleUrls, expectedConsoleUrls)
}

// Get the list of console URLs from the status block of the installed verrrazzano resource
func getConsoleURLsFromResource() ([]string, error) {
	var consoleUrls []string
	var client *vzClient.Clientset

	vzList, err := client.VerrazzanoV1alpha1().Verrazzanos("default").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("There is an error getting list of verrazzano resources - %v", err))
		return consoleUrls, err
	}

	if len(vzList.Items) > 1 {
		msg := "expected only one verrazzano resource to be found"
		pkg.Log(pkg.Error, msg)
		return consoleUrls, fmt.Errorf(msg)
	}

	if vzList.Items[0].Status.VerrazzanoInstance.ConsoleURL != nil {
		consoleUrls = append(consoleUrls, *vzList.Items[0].Status.VerrazzanoInstance.ConsoleURL)
	}
	if vzList.Items[0].Status.VerrazzanoInstance.GrafanaURL != nil {
		consoleUrls = append(consoleUrls, *vzList.Items[0].Status.VerrazzanoInstance.GrafanaURL)
	}
	if vzList.Items[0].Status.VerrazzanoInstance.ElasticURL != nil {
		consoleUrls = append(consoleUrls, *vzList.Items[0].Status.VerrazzanoInstance.ElasticURL)
	}
	if vzList.Items[0].Status.VerrazzanoInstance.KeyCloakURL != nil {
		consoleUrls = append(consoleUrls, *vzList.Items[0].Status.VerrazzanoInstance.KeyCloakURL)
	}
	if vzList.Items[0].Status.VerrazzanoInstance.KibanaURL != nil {
		consoleUrls = append(consoleUrls, *vzList.Items[0].Status.VerrazzanoInstance.KibanaURL)
	}
	if vzList.Items[0].Status.VerrazzanoInstance.KialiURL != nil {
		consoleUrls = append(consoleUrls, *vzList.Items[0].Status.VerrazzanoInstance.KialiURL)
	}
	if vzList.Items[0].Status.VerrazzanoInstance.PrometheusURL != nil {
		consoleUrls = append(consoleUrls, *vzList.Items[0].Status.VerrazzanoInstance.PrometheusURL)
	}
	if vzList.Items[0].Status.VerrazzanoInstance.RancherURL != nil {
		consoleUrls = append(consoleUrls, *vzList.Items[0].Status.VerrazzanoInstance.RancherURL)
	}

	return consoleUrls, nil
}

// Get the expected console URLs for the given cluster
func getExpectedConsoleURLs(kubeConfig string) ([]string, error) {
	api, err := pkg.GetAPIEndpoint(kubeConfig)
	if api == nil {
		return nil, err
	}
	ingress, err := api.GetIngress(keycloakNamespace, keycloakIngress)
	if err != nil {
		return nil, err
	}
	keycloakURL := fmt.Sprintf("https://%s", ingress.Spec.TLS[0].Hosts[0])
	ingress, err = api.GetIngress(rancherNamespace, rancherIngress)
	if err != nil {
		return nil, err
	}
	rancherURL := fmt.Sprintf("https://%s", ingress.Spec.TLS[0].Hosts[0])
	grafanaURL, err := getComponentURL(api, grafanaIngress)
	if err != nil {
		return nil, err
	}
	prometheusURL, err := getComponentURL(api, prometheusIngress)
	if err != nil {
		return nil, err
	}
	kibanaURL, err := getComponentURL(api, kibanaIngress)
	if err != nil {
		return nil, err
	}
	elasticsearchURL, err := getComponentURL(api, elasticsearchIngress)
	if err != nil {
		return nil, err
	}
	consoleURL, err := getComponentURL(api, verrazzanoIngress)
	if err != nil {
		return nil, err
	}
	kialiURL, err := getComponentURL(api, kialiIngress)
	if err != nil {
		return nil, err
	}

	var expectedUrls = []string{
		grafanaURL,
		prometheusURL,
		kibanaURL,
		elasticsearchURL,
		consoleURL,
		rancherURL,
		keycloakURL,
		kialiURL}
	return expectedUrls, nil
}

func getComponentURL(api *pkg.APIEndpoint, ingressName string) (string, error) {
	ingress, err := api.GetIngress(verrazzanoSystemNamespace, ingressName)
	if err != nil {
		return "", err
	}
	vmiComponentURL := fmt.Sprintf("https://%s", ingress.Spec.TLS[0].Hosts[0])
	return vmiComponentURL, nil
}
