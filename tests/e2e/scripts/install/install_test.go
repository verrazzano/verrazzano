// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package installscript_test

import (
	"bufio"
	"fmt"
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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

var installLogDir = os.Getenv("VERRAZZANO_INSTALL_LOGS_DIR")
var installLog = os.Getenv("VERRAZZANO_INSTALL_LOG")

var kubeConfigFromEnv = os.Getenv("KUBECONFIG")
var totalClusters, present = os.LookupEnv("CLUSTER_COUNT")

var _ = ginkgo.BeforeSuite(func() {
	if len(installLogDir) < 1 {
		ginkgo.Fail("Specify the directory containing the install logs using environment variable VERRAZZANO_INSTALL_LOGS_DIR")
	}
	if len(installLog) < 1 {
		ginkgo.Fail("Specify the install log file using environment variable VERRAZZANO_INSTALL_LOG")
	}
})

var _ = ginkgo.Describe("Verify Verrazzano install scripts", func() {

	ginkgo.Context("Verify Console URLs", func() {
		clusterCount, _ := strconv.Atoi(totalClusters)
		if present && clusterCount > 0 {
			ginkgo.It("Verify the expected console URLs are there in the mc log ", func() {
				// Validation for admin cluster
				gomega.Expect(validateConsoleUrlsCluster(kubeConfigFromEnv, "cluster-1")).To(gomega.BeTrue())

				// Validation for managed clusters
				for i := 2; i <= clusterCount; i++ {
					installLogForCluster := filepath.FromSlash(installLogDir + "/cluster-" + strconv.Itoa(i) + "/" + installLog)
					consoleUrls, err := getConsoleURLsFromLog(installLogForCluster)
					if err != nil {
						gomega.Expect(false).To(gomega.BeTrue(), "There is an error getting console URLs from the log file ", err)
					}
					// By default, install logs of the managed clusters do not contain the console URLs
					gomega.Expect(len(consoleUrls) == 0).To(gomega.BeTrue())
				}
			})
		} else {
			ginkgo.It("Verify the expected console URLs are there in the install log", func() {
				gomega.Expect(validateConsoleUrlsCluster(kubeConfigFromEnv, "")).To(gomega.BeTrue())
			})
		}
	})
})

// Validate the console URLs for the admin cluster and single cluster installation
func validateConsoleUrlsCluster(kubeconfig string, clusterPrefix string) bool {
	consoleUrls, err := getConsoleURLsFromLog(filepath.FromSlash(installLogDir + "/" + clusterPrefix + "/" + installLog))
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("There is an error getting console URLs from the log file - %v", err))
	}
	expectedConsoleUrls := getExpectedConsoleURLs(kubeconfig)
	return pkg.SlicesContainSameStrings(consoleUrls, expectedConsoleUrls)
}

// Get the list of console URLs from the install log, after the line containing "Installation Complete."
func getConsoleURLsFromLog(installLog string) ([]string, error) {
	var consoleUrls []string
	if _, err := os.Stat(installLog); err != nil {
		if os.IsNotExist(err) {
			fmt.Println("The value set for installLog doesn't exist.")
			return consoleUrls, err
		}
	}
	inFile, err := os.Open(installLog)

	if err != nil {
		fmt.Println("Error reading install log file ", err.Error())
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
func getExpectedConsoleURLs(kubeConfig string) []string {
	api := pkg.GetAPIEndpoint(kubeConfig)
	ingress := api.GetIngress(keycloakNamespace, keycloakIngress)
	keycloakURL := fmt.Sprintf("https://%s", ingress.Spec.TLS[0].Hosts[0])
	ingress = api.GetIngress(rancherNamespace, rancherIngress)
	rancherURL := fmt.Sprintf("https://%s", ingress.Spec.TLS[0].Hosts[0])
	grafanaURL := getComponentURL(api, grafanaIngress)
	prometheusURL := getComponentURL(api, prometheusIngress)
	kibanaURL := getComponentURL(api, kibanaIngress)
	elasticsearchURL := getComponentURL(api, elasticsearchIngress)
	consoleURL := getComponentURL(api, verrazzanoIngress)

	// Expected console URLs in the order in which they appear in the install log
	var expectedUrls = []string{
		"Grafana - " + grafanaURL,
		"Prometheus - " + prometheusURL,
		"Kibana - " + kibanaURL,
		"Elasticsearch - " + elasticsearchURL,
		"Verrazzano Console - " + consoleURL,
		"Rancher - " + rancherURL,
		"Keycloak - " + keycloakURL}
	return expectedUrls
}

func getComponentURL(api *pkg.APIEndpoint, ingressName string) string {
	ingress := api.GetIngress(verrazzanoSystemNamespace, ingressName)
	vmiComponentURL := fmt.Sprintf("https://%s", ingress.Spec.TLS[0].Hosts[0])
	return vmiComponentURL
}
