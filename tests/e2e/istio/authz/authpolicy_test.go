// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package authz_test

import (
	"bufio"
	"fmt"
	"net/http"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
)

const fooNamespace string = "foo"
const barNamespace string = "bar"
const noIstioNamespace string = "noistio"

const vmiPromConfigName string = "vmi-system-prometheus-config"
const verrazzanoNamespace string = "verrazzano-system"
const prometheusConfigMapName string = "prometheus.yml"
const prometheusFooScrapeName string = "authpolicy-appconf_default_foo_springboot-frontend"
const prometheusBarScrapeName string = "authpolicy-appconf_default_bar_springboot-frontend"
const prometheusNoIstioScrapeName string = "authpolicy-appconf_default_noistio_springboot-frontend"
const prometheusJobName string = "job_name"
const prometheusHTTPSScheme string = "scheme: https"

var expectedPodsFoo = []string{"sleep-workload", "springboot-frontend-workload", "springboot-backend-workload"}
var expectedPodsBar = []string{"sleep-workload", "springboot-frontend-workload", "springboot-backend-workload"}
var waitTimeout = 10 * time.Minute
var pollingInterval = 30 * time.Second
var shortPollingInterval = 10 * time.Second

var _ = BeforeSuite(func() {
	deployFooApplication()
	deployBarApplication()
	deployNoIstioApplication()
})

var clusterDump = pkg.NewClusterDumpWrapper()
var _ = clusterDump.AfterEach(func() {}) // Dump cluster if spec fails
var _ = clusterDump.AfterSuite(func() {  // Dump cluster if aftersuite fails
	undeployFooApplication()
	undeployBarApplication()
	undeployNoIstioApplication()
})

func deployFooApplication() {
	pkg.Log(pkg.Info, "Deploy Auth Policy Application in foo namespace")

	pkg.Log(pkg.Info, "Create namespace")
	Eventually(func() (*v1.Namespace, error) {
		return pkg.CreateNamespace(fooNamespace, map[string]string{"verrazzano-managed": "true", "istio-injection": "enabled"})
	}, waitTimeout, shortPollingInterval).ShouldNot(BeNil())

	pkg.Log(pkg.Info, "Create AuthPolicy App resources")
	Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFile("testdata/istio/authz/foo/istio-securitytest-app.yaml")
	}, waitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	pkg.Log(pkg.Info, "Create Sleep Component")
	Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFile("testdata/istio/authz/foo/sleep-comp.yaml")
	}, waitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	pkg.Log(pkg.Info, "Create Backend Component")
	Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFile("testdata/istio/authz/foo/springboot-backend.yaml")
	}, waitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	pkg.Log(pkg.Info, "Create Frontend Component")
	Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFile("testdata/istio/authz/foo/springboot-frontend.yaml")
	}, waitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

}

func deployBarApplication() {
	pkg.Log(pkg.Info, "Deploy Auth Policy Application in bar namespace")

	pkg.Log(pkg.Info, "Create namespace")
	Eventually(func() (*v1.Namespace, error) {
		return pkg.CreateNamespace(barNamespace, map[string]string{"verrazzano-managed": "true", "istio-injection": "enabled"})
	}, waitTimeout, shortPollingInterval).ShouldNot(BeNil())

	pkg.Log(pkg.Info, "Create AuthPolicy App resources")
	Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFile("testdata/istio/authz/bar/istio-securitytest-app.yaml")
	}, waitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	pkg.Log(pkg.Info, "Create Sleep Component")
	Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFile("testdata/istio/authz/bar/sleep-comp.yaml")
	}, waitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	pkg.Log(pkg.Info, "Create Backend Component")
	Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFile("testdata/istio/authz/bar/springboot-backend.yaml")
	}, waitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	pkg.Log(pkg.Info, "Create Frontend Component")
	Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFile("testdata/istio/authz/bar/springboot-frontend.yaml")
	}, waitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

}

func deployNoIstioApplication() {
	pkg.Log(pkg.Info, "Deploy Auth Policy Application in NoIstio namespace")

	pkg.Log(pkg.Info, "Create namespace")
	Eventually(func() (*v1.Namespace, error) {
		return pkg.CreateNamespace(noIstioNamespace, map[string]string{"verrazzano-managed": "true"})
	}, waitTimeout, shortPollingInterval).ShouldNot(BeNil())

	pkg.Log(pkg.Info, "Create AuthPolicy App resources")
	Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFile("testdata/istio/authz/noistio/istio-securitytest-app.yaml")
	}, waitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	pkg.Log(pkg.Info, "Create Sleep Component")
	Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFile("testdata/istio/authz/noistio/sleep-comp.yaml")
	}, waitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	pkg.Log(pkg.Info, "Create Backend Component")
	Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFile("testdata/istio/authz/noistio/springboot-backend.yaml")
	}, waitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	pkg.Log(pkg.Info, "Create Frontend Component")
	Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFile("testdata/istio/authz/noistio/springboot-frontend.yaml")
	}, waitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

}

func undeployFooApplication() {
	pkg.Log(pkg.Info, "Undeploy Auth Policy Application in foo namespace")
	pkg.Log(pkg.Info, "Delete application")
	Eventually(func() error {
		return pkg.DeleteResourceFromFile("testdata/istio/authz/foo/istio-securitytest-app.yaml")
	}, waitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	pkg.Log(pkg.Info, "Delete components")
	Eventually(func() error {
		return pkg.DeleteResourceFromFile("testdata/istio/authz/foo/sleep-comp.yaml")
	}, waitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	Eventually(func() error {
		return pkg.DeleteResourceFromFile("testdata/istio/authz/foo/springboot-backend.yaml")
	}, waitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	Eventually(func() error {
		return pkg.DeleteResourceFromFile("testdata/istio/authz/foo/springboot-frontend.yaml")
	}, waitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	Eventually(func() (bool, error) {
		return pkg.PodsNotRunning(fooNamespace, expectedPodsFoo)
	}, waitTimeout, shortPollingInterval).Should(BeTrue(), fmt.Sprintf("Pods in namespace %s stuck terminating!", fooNamespace))

	pkg.Log(pkg.Info, "Delete namespace")
	Eventually(func() error {
		return pkg.DeleteNamespace(fooNamespace)
	}, waitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	Eventually(func() bool {
		_, err := pkg.GetNamespace(fooNamespace)
		return err != nil && errors.IsNotFound(err)
	}, waitTimeout, shortPollingInterval).Should(BeTrue())
}

func undeployBarApplication() {
	pkg.Log(pkg.Info, "Undeploy Auth Policy Application in bar namespace")
	pkg.Log(pkg.Info, "Delete application")
	Eventually(func() error {
		return pkg.DeleteResourceFromFile("testdata/istio/authz/bar/istio-securitytest-app.yaml")
	}, waitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	pkg.Log(pkg.Info, "Delete components")
	Eventually(func() error {
		return pkg.DeleteResourceFromFile("testdata/istio/authz/bar/sleep-comp.yaml")
	}, waitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	Eventually(func() error {
		return pkg.DeleteResourceFromFile("testdata/istio/authz/bar/springboot-backend.yaml")
	}, waitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	Eventually(func() error {
		return pkg.DeleteResourceFromFile("testdata/istio/authz/bar/springboot-frontend.yaml")
	}, waitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	Eventually(func() (bool, error) {
		return pkg.PodsNotRunning(barNamespace, expectedPodsBar)
	}, waitTimeout, shortPollingInterval).Should(BeTrue(), fmt.Sprintf("Pods in namespace %s stuck terminating!", barNamespace))

	pkg.Log(pkg.Info, "Delete namespace")
	Eventually(func() error {
		return pkg.DeleteNamespace(barNamespace)
	}, waitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	Eventually(func() bool {
		_, err := pkg.GetNamespace(barNamespace)
		return err != nil && errors.IsNotFound(err)
	}, waitTimeout, shortPollingInterval).Should(BeTrue())
}

func undeployNoIstioApplication() {
	pkg.Log(pkg.Info, "Undeploy Auth Policy Application in noistio namespace")
	pkg.Log(pkg.Info, "Delete application")
	Eventually(func() error {
		return pkg.DeleteResourceFromFile("testdata/istio/authz/noistio/istio-securitytest-app.yaml")
	}, waitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	pkg.Log(pkg.Info, "Delete components")
	Eventually(func() error {
		return pkg.DeleteResourceFromFile("testdata/istio/authz/noistio/sleep-comp.yaml")
	}, waitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	Eventually(func() error {
		return pkg.DeleteResourceFromFile("testdata/istio/authz/noistio/springboot-backend.yaml")
	}, waitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	Eventually(func() error {
		return pkg.DeleteResourceFromFile("testdata/istio/authz/noistio/springboot-frontend.yaml")
	}, waitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	Eventually(func() (bool, error) {
		return pkg.PodsNotRunning(noIstioNamespace, expectedPodsBar)
	}, waitTimeout, shortPollingInterval).Should(BeTrue(), fmt.Sprintf("Pods in namespace %s stuck terminating!", noIstioNamespace))

	pkg.Log(pkg.Info, "Delete namespace")
	Eventually(func() error {
		return pkg.DeleteNamespace(noIstioNamespace)
	}, waitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	Eventually(func() bool {
		_, err := pkg.GetNamespace(noIstioNamespace)
		return err != nil && errors.IsNotFound(err)
	}, waitTimeout, shortPollingInterval).Should(BeTrue())
}

var _ = Describe("Verify AuthPolicy Applications", func() {
	// Verify springboot-workload pod is running
	// GIVEN springboot app is deployed
	// WHEN the component and appconfig are created
	// THEN the expected pod must be running in the test namespace
	Context("Deployment.", func() {
		It("and waiting for expected pods must be running", func() {
			Eventually(func() bool {
				return pkg.PodsRunning(fooNamespace, expectedPodsFoo)
			}, waitTimeout, pollingInterval).Should(BeTrue(), fmt.Sprintf("Auth Policy Application failed to start in %s", fooNamespace))
		})
	})

	Context("Deployment.", func() {
		It("and waiting for expected pods must be running", func() {
			Eventually(func() bool {
				return pkg.PodsRunning(barNamespace, expectedPodsBar)
			}, waitTimeout, pollingInterval).Should(BeTrue(), fmt.Sprintf("Auth Policy Application failed to start in %s", barNamespace))
		})
	})

	Context("Deployment.", func() {
		It("and waiting for expected pods must be running", func() {
			Eventually(func() bool {
				return pkg.PodsRunning(noIstioNamespace, expectedPodsBar)
			}, waitTimeout, pollingInterval).Should(BeTrue(), fmt.Sprintf("Auth Policy Application failed to start in %s", noIstioNamespace))
		})
	})

	var fooHost = ""
	var err error
	It("Get foo host from gateway.", func() {
		Eventually(func() (string, error) {
			fooHost, err = k8sutil.GetHostnameFromGateway(fooNamespace, "")
			return fooHost, err
		}, waitTimeout, shortPollingInterval).Should(Not(BeEmpty()), fmt.Sprintf("Failed to get host from gateway in %s", fooNamespace))
	})

	var barHost = ""
	It("Get bar host from gateway.", func() {
		Eventually(func() (string, error) {
			barHost, err = k8sutil.GetHostnameFromGateway(barNamespace, "")
			return barHost, err
		}, waitTimeout, shortPollingInterval).Should(Not(BeEmpty()), fmt.Sprintf("Failed to get host from gateway in %s", barNamespace))
	})

	var noIstioHost = ""
	It("Get noistio host from gateway.", func() {
		Eventually(func() (string, error) {
			noIstioHost, err = k8sutil.GetHostnameFromGateway(noIstioNamespace, "")
			return noIstioHost, err
		}, waitTimeout, shortPollingInterval).Should(Not(BeEmpty()), fmt.Sprintf("Failed to get host from gateway in %s", noIstioNamespace))
	})

	// Verify application in namespace foo is working
	// GIVEN authorization test app is deployed
	// WHEN the component and appconfig with ingress trait are created
	// THEN the application endpoint must be accessible
	It("Verify welcome page of Foo Spring Boot FrontEnd is working.", func() {
		Eventually(func() (*pkg.HTTPResponse, error) {
			pkg.Log(pkg.Info, fmt.Sprintf("Ingress: %s", fooHost))
			url := fmt.Sprintf("https://%s/", fooHost)
			return pkg.GetWebPage(url, fooHost)
		}, waitTimeout, shortPollingInterval).Should(And(pkg.HasStatus(http.StatusOK), pkg.BodyContains("Greetings from Verrazzano Enterprise Container Platform")))
	})

	// Verify application in namespace bar is working
	// GIVEN authorization test app is deployed
	// WHEN the component and appconfig with ingress trait are created
	// THEN the application endpoint must be accessible
	It("Verify welcome page of Bar Spring Boot FrontEnd is working.", func() {
		Eventually(func() (*pkg.HTTPResponse, error) {
			pkg.Log(pkg.Info, fmt.Sprintf("Ingress: %s", barHost))
			url := fmt.Sprintf("https://%s/", barHost)
			return pkg.GetWebPage(url, barHost)
		}, waitTimeout, shortPollingInterval).Should(And(pkg.HasStatus(http.StatusOK), pkg.BodyContains("Greetings from Verrazzano Enterprise Container Platform")))
	})

	// Verify application in namespace bar is working
	// GIVEN authorization test app is deployed
	// WHEN the component and appconfig with ingress trait are created
	// THEN the application endpoint must be accessible
	It("Verify welcome page of NoIstio Spring Boot FrontEnd is working.", func() {
		Eventually(func() (*pkg.HTTPResponse, error) {
			pkg.Log(pkg.Info, fmt.Sprintf("Ingress: %s", noIstioHost))
			url := fmt.Sprintf("https://%s/", noIstioHost)
			return pkg.GetWebPage(url, noIstioHost)
		}, waitTimeout, shortPollingInterval).Should(And(pkg.HasStatus(http.StatusOK), pkg.BodyContains("Greetings from Verrazzano Enterprise Container Platform")))
	})

	// Verify Frontend can call Backend in foo
	// GIVEN authorization test app is deployed
	// WHEN the component and appconfig with ingress trait are created
	// THEN the frontend should be able to successfully call the backend and get a 200 on that invocation
	// the http code is returned in the response body and captured in content
	It("Verify Foo Frontend can call Foo Backend.", func() {
		Eventually(func() (*pkg.HTTPResponse, error) {
			pkg.Log(pkg.Info, fmt.Sprintf("Ingress: %s", fooHost))
			url := fmt.Sprintf("https://%s/externalCall?inurl=http://springboot-backend-workload.foo:8080/", fooHost)
			return pkg.GetWebPage(url, fooHost)
		}, waitTimeout, shortPollingInterval).Should(And(pkg.HasStatus(http.StatusOK), pkg.BodyEquals("200")))
	})

	// Verify Frontend can call Backend in bar
	// GIVEN authorization test app is deployed
	// WHEN the component and appconfig with ingress trait are created
	// THEN the frontend should be able to successfully call the backend and get a 200 on that invocation
	// the http code is returned in the response body and captured in content
	It("Verify Bar Frontend can call Bar Backend.", func() {
		Eventually(func() (*pkg.HTTPResponse, error) {
			pkg.Log(pkg.Info, fmt.Sprintf("Ingress: %s", barHost))
			url := fmt.Sprintf("https://%s/externalCall?inurl=http://springboot-backend-workload.bar:8080/", barHost)
			return pkg.GetWebPage(url, barHost)
		}, waitTimeout, shortPollingInterval).Should(And(pkg.HasStatus(http.StatusOK), pkg.BodyEquals("200")))
	})

	// Verify Foo Frontend can't call bar Backend
	// GIVEN authorization test app is deployed
	// WHEN the component and appconfig with ingress trait are created
	// THEN the frontend should be able to successfully call the backend and get a 200 on that invocation
	// the http code is returned in the response body and captured in content
	It("Verify Foo Frontend canNOT call Bar Backend.", func() {
		Eventually(func() (*pkg.HTTPResponse, error) {
			pkg.Log(pkg.Info, fmt.Sprintf("Ingress: %s", fooHost))
			url := fmt.Sprintf("https://%s/externalCall?inurl=http://springboot-backend-workload.bar:8080/", fooHost)
			return pkg.GetWebPage(url, fooHost)
		}, waitTimeout, shortPollingInterval).Should(And(pkg.HasStatus(http.StatusOK), pkg.BodyEquals("403")))
	})

	// Verify Bar Frontend can't call Foo Backend
	// GIVEN authorization test app is deployed
	// WHEN the component and appconfig with ingress trait are created
	// THEN the frontend should be able to successfully call the backend and get a 200 on that invocation
	// the http code is returned in the response body and captured in content
	It("Verify Bar Frontend canNOT call Foo Backend.", func() {
		Eventually(func() (*pkg.HTTPResponse, error) {
			pkg.Log(pkg.Info, fmt.Sprintf("Ingress: %s", barHost))
			url := fmt.Sprintf("https://%s/externalCall?inurl=http://springboot-backend-workload.foo:8080/", barHost)
			return pkg.GetWebPage(url, barHost)
		}, waitTimeout, shortPollingInterval).Should(And(pkg.HasStatus(http.StatusOK), pkg.BodyEquals("403")))
	})

	// Verify Bar Frontend can call NoIstio Backend
	// GIVEN authorization test app is deployed
	// WHEN the component and appconfig with ingress trait are created
	// THEN the frontend should be able to successfully call the backend and get a 200 on that invocation
	// the http code is returned in the response body and captured in content
	It("Verify Bar Frontend can call NoIstio Backend.", func() {
		Eventually(func() (*pkg.HTTPResponse, error) {
			pkg.Log(pkg.Info, fmt.Sprintf("Ingress: %s", barHost))
			url := fmt.Sprintf("https://%s/externalCall?inurl=http://springboot-backend-workload.noistio:8080/", barHost)
			return pkg.GetWebPage(url, barHost)
		}, waitTimeout, shortPollingInterval).Should(And(pkg.HasStatus(http.StatusOK), pkg.BodyEquals("200")))
	})

	// Verify noistio Frontend can't call bar Backend
	// GIVEN authorization test app is deployed
	// WHEN the component and appconfig with ingress trait are created
	// THEN the frontend should be able to successfully call the backend and get a 200 on that invocation
	// the http code is returned in the response body and captured in content
	// *** This call should fail for a 500 because Non-Istio can't call Istio when MTLS is STRICT
	// If this should fail because the call succeeded, verify that peerauthentication exists in istio-system and is set to STRICT
	It("Verify NoIstio Frontend canNOT call Bar Backend.", func() {
		kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
		Expect(err).ShouldNot(HaveOccurred())
		Eventually(func() bool {
			pkg.Log(pkg.Info, fmt.Sprintf("Ingress: %s", noIstioHost))
			url := fmt.Sprintf("https://%s/externalCall?inurl=http://springboot-backend-workload.bar:8080/", noIstioHost)
			client, err := pkg.GetVerrazzanoNoRetryHTTPClient(kubeconfigPath)
			if err != nil {
				pkg.Log(pkg.Error, fmt.Sprintf("Failed to get client: %v", err))
				return false
			}
			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				pkg.Log(pkg.Error, fmt.Sprintf("Failed to create http request: %v", err))
				return false
			}
			req.Host = noIstioHost

			q := req.URL.Query()
			q.Add("inurl", "https://springboot-backend-workload.bar:8080/")
			req.URL.RawQuery = q.Encode()

			resp, err := client.Do(req)
			if err != nil {
				// could be a transient network error so log it and let the Eventually retry
				pkg.Log(pkg.Error, fmt.Sprintf("Failed to do http request: %v", err))
				return false
			}
			return resp.StatusCode == 500
		}, waitTimeout, shortPollingInterval).Should(BeTrue(), "Failed to Verify NoIstio Frontend canNOT call Bar Backend")
	})

})

var _ = Describe("Verify Auth Policy Prometheus Scrape Targets", func() {
	// Verify springboot-workload pod is running
	// GIVEN springboot app is deployed
	// WHEN the component and appconfig are created
	// THEN the expected pod must be running in the test namespace
	Context("Deployment.", func() {
		It("and waiting for expected pods must be running", func() {
			Eventually(func() bool {
				return pkg.PodsRunning(fooNamespace, expectedPodsFoo)
			}, waitTimeout, pollingInterval).Should(BeTrue(), fmt.Sprintf("Auth Policy Application failed to start in %s", fooNamespace))
		})
	})

	Context("Deployment.", func() {
		It("and waiting for expected pods must be running", func() {
			Eventually(func() bool {
				return pkg.PodsRunning(barNamespace, expectedPodsBar)
			}, waitTimeout, pollingInterval).Should(BeTrue(), fmt.Sprintf("Auth Policy Application failed to start in %s", barNamespace))
		})
	})

	Context("Deployment.", func() {
		It("and waiting for expected pods must be running", func() {
			Eventually(func() bool {
				return pkg.PodsRunning(noIstioNamespace, expectedPodsBar)
			}, waitTimeout, pollingInterval).Should(BeTrue(), fmt.Sprintf("Auth Policy Application failed to start in %s", noIstioNamespace))
		})
	})

	// Verify That Generated Prometheus Scrape Targets for authpolicy-appconf_default_foo_springboot-frontend is using https for scraping
	// GIVEN that springboot deployed to Istio namespace foo
	// WHEN the Prometheus scrape targets are created
	// THEN they should be created to use the https protocol
	It("Verify that Istio scrape target authpolicy-appconf_default_foo_springboot-frontend is using https for scraping.", func() {
		Eventually(func() bool {
			var httpsFound bool = false

			configMap, err := pkg.GetConfigMap(vmiPromConfigName, verrazzanoNamespace)
			if err != nil {
				return false
			}
			dataMap := configMap.Data
			v := dataMap[prometheusConfigMapName]
			rdr := strings.NewReader(v)
			scanner := bufio.NewScanner(rdr)
			for scanner.Scan() {
				currentString := scanner.Text()
				if strings.Contains(currentString, prometheusFooScrapeName) {
					for scanner.Scan() {
						innerString := scanner.Text()
						if strings.Contains(innerString, prometheusJobName) {
							break
						}
						if strings.Contains(innerString, prometheusHTTPSScheme) {
							httpsFound = true
							break
						}
					}
					if httpsFound {
						break
					}
				}
			}
			return httpsFound == true
		}, waitTimeout, shortPollingInterval).Should(BeTrue(), "Failed to Verify that Istio scrape target authpolicy-appconf_default_foo_springboot-frontend is using https for scraping")
	})

	// Verify That Generated Prometheus Scrape Targets for authpolicy-appconf_default_bar_springboot-frontend is using https for scraping
	// GIVEN that springboot deployed to Istio namespace bar
	// WHEN the Prometheus scrape targets are created
	// THEN they should be created to use the https protocol
	It("Verify that Istio scrape target authpolicy-appconf_default_bar_springboot-frontend is using https for scraping.", func() {
		Eventually(func() bool {
			var httpsFound bool = false

			configMap, err := pkg.GetConfigMap(vmiPromConfigName, verrazzanoNamespace)
			if err != nil {
				return false
			}
			dataMap := configMap.Data
			v := dataMap[prometheusConfigMapName]
			rdr := strings.NewReader(v)
			scanner := bufio.NewScanner(rdr)
			for scanner.Scan() {
				currentString := scanner.Text()
				if strings.Contains(currentString, prometheusBarScrapeName) {
					for scanner.Scan() {
						innerString := scanner.Text()
						if strings.Contains(innerString, prometheusJobName) {
							break
						}
						if strings.Contains(innerString, prometheusHTTPSScheme) {
							httpsFound = true
							break
						}
					}
					if httpsFound {
						break
					}
				}
			}
			return httpsFound == true
		}, waitTimeout, shortPollingInterval).Should(BeTrue(), "Failed to Verify that Istio scrape target authpolicy-appconf_default_bar_springboot-frontend is using https for scraping")
	})

	// Verify That Generated Prometheus Scrape Targets for authpolicy-appconf_default_noistio_springboot-frontend is using http for scraping
	// GIVEN that springboot deployed to namespace noistio
	// WHEN the Prometheus scrape targets are created
	// THEN they should be created to use the http protocol
	It("Verify that Istio scrape target authpolicy-appconf_default_noistio_springboot-frontend is using http for scraping.", func() {
		Eventually(func() bool {
			var httpsNotFound bool = true

			configMap, err := pkg.GetConfigMap(vmiPromConfigName, verrazzanoNamespace)
			if err != nil {
				return false
			}
			dataMap := configMap.Data
			v := dataMap[prometheusConfigMapName]
			rdr := strings.NewReader(v)
			scanner := bufio.NewScanner(rdr)
			for scanner.Scan() {
				currentString := scanner.Text()
				if strings.Contains(currentString, prometheusNoIstioScrapeName) {
					for scanner.Scan() {
						innerString := scanner.Text()
						if strings.Contains(innerString, prometheusJobName) {
							break
						}
						if strings.Contains(innerString, prometheusHTTPSScheme) {
							httpsNotFound = false
							break
						}
					}
					if httpsNotFound {
						break
					}
				}
			}
			return httpsNotFound == true
		}, waitTimeout, shortPollingInterval).Should(BeTrue(), "Failed to Verify that Istio scrape target authpolicy-appconf_default_noistio_springboot-frontend is using http for scraping")
	})

})
