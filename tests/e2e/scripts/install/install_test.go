// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// +build unstable_test

package installscript_test

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

const (
	installComplete           = "Installation Complete."
	verrazzanoSystemNamespace = "verrazzano-system"
	rancherNamespace          = "cattle-system"
	keycloakNamespace         = "keycloak"
	grafanaIngress            = "vmi-system-grafana"
	prometheusIngress         = "vmi-system-prometheus"
	elasticsearchIngress      = "vmi-system-es-ingest"
	kibanaIngress             = "vmi-system-kibana"
	verrazzanoIngress         = "verrazzano-ingress"
	keycloakIngress           = "keycloak"
	rancherIngress            = "rancher"
	noOfLinesToRead           = 20
)

const (
	waitTimeout     = 5 * time.Minute
	pollingInterval = 5 * time.Second
)

var installLogDir = os.Getenv("VERRAZZANO_INSTALL_LOGS_DIR")
var installLog = os.Getenv("VERRAZZANO_INSTALL_LOG")

var kubeConfigFromEnv = os.Getenv("KUBECONFIG")
var totalClusters, present = os.LookupEnv("CLUSTER_COUNT")

var _ = BeforeSuite(func() {
	if len(installLogDir) < 1 {
		Fail("Specify the directory containing the install logs using environment variable VERRAZZANO_INSTALL_LOGS_DIR")
	}
	if len(installLog) < 1 {
		Fail("Specify the install log file using environment variable VERRAZZANO_INSTALL_LOG")
	}
})

var _ = Describe("Verify Verrazzano install scripts", func() {

	Context("Verify Console URLs in the install log", func() {
		clusterCount, _ := strconv.Atoi(totalClusters)
		if present && clusterCount > 0 {
			It("Verify the expected console URLs are there in the install logs for the managed cluster(s)", func() {
				// Validation for admin cluster
				Eventually(func() bool {
					return validateConsoleUrlsCluster(kubeConfigFromEnv, "cluster-1")
				}, waitTimeout, pollingInterval).Should(BeTrue())

				// Validation for managed clusters
				for i := 2; i <= clusterCount; i++ {
					installLogForCluster := filepath.FromSlash(installLogDir + "/cluster-" + strconv.Itoa(i) + "/" + installLog)
					consoleUrls, err := getConsoleURLsFromLog(installLogForCluster)
					Expect(err).ShouldNot(HaveOccurred(), "There is an error getting console URLs from the log file")

					// By default, install logs of the managed clusters do not contain the console URLs
					Expect(consoleUrls).To(BeEmpty())
				}
			})
		} else {
			It("Verify the expected console URLs are there in the install log", func() {
				Eventually(func() bool {
					return validateConsoleUrlsCluster(kubeConfigFromEnv, "")
				}, waitTimeout, pollingInterval).Should(BeTrue())
			})
		}
	})
})

// Validate the console URLs for the admin cluster and single cluster installation
func validateConsoleUrlsCluster(kubeconfig string, clusterPrefix string) bool {
	consoleUrls, err := getConsoleURLsFromLog(filepath.FromSlash(installLogDir + "/" + clusterPrefix + "/" + installLog))
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("There is an error getting console URLs from the log file - %v", err))
		return false
	}
	expectedConsoleUrls, err := getExpectedConsoleURLs(kubeconfig)
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("There is an error getting console URLs from the API server - %v", err))
		return false
	}

	return pkg.SlicesContainSameStrings(consoleUrls, expectedConsoleUrls)
}

// Get the list of console URLs from the install log, after the line containing "Installation Complete."
func getConsoleURLsFromLog(installLog string) ([]string, error) {
	var consoleUrls []string
	if _, err := os.Stat(installLog); err != nil {
		if os.IsNotExist(err) {
			pkg.Log(pkg.Error, "The value set for installLog doesn't exist.")
		}
		return consoleUrls, err
	}
	inFile, err := os.Open(installLog)

	if err != nil {
		return consoleUrls, err
	}
	defer inFile.Close()
	rdr := bufio.NewReader(inFile)
	scanner := bufio.NewScanner(rdr)
	var line, startLine, endLine int

	for scanner.Scan() {
		line++
		currentLine := scanner.Text()
		if currentLine == installComplete {
			startLine = line
			endLine = startLine + noOfLinesToRead
			for scanner.Scan() {
				startLine++
				innerString := scanner.Text()
				if strings.Contains(innerString, "- https://") {
					consoleUrls = append(consoleUrls, innerString)
				}
				if startLine == endLine {
					break
				}
			}
			break
		}
	}
	return consoleUrls, nil
}

// Get the expected console URLs in the install log for the given cluster
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

	// Expected console URLs in the order in which they appear in the install log
	var expectedUrls = []string{
		"Grafana - " + grafanaURL,
		"Prometheus - " + prometheusURL,
		"Kibana - " + kibanaURL,
		"Elasticsearch - " + elasticsearchURL,
		"Verrazzano Console - " + consoleURL,
		"Rancher - " + rancherURL,
		"Keycloak - " + keycloakURL}
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
