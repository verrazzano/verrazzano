// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package authz_test

import (
	"bufio"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
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

var _ = ginkgo.BeforeSuite(func() {
	deployFooApplication()
	deployBarApplication()
	deployNoIstioApplication()
})

var failed = false
var _ = ginkgo.AfterEach(func() {
	failed = failed || ginkgo.CurrentGinkgoTestDescription().Failed
})

var _ = ginkgo.AfterSuite(func() {
	if failed {
		pkg.ExecuteClusterDumpWithEnvVarConfig()
	}
	undeployFooApplication()
	undeployBarApplication()
	undeployNoIstioApplication()
})

func deployFooApplication() {
	pkg.Log(pkg.Info, "Deploy Auth Policy Application in foo namespace")

	pkg.Log(pkg.Info, "Create namespace")
	if _, err := pkg.CreateNamespace(fooNamespace, map[string]string{"verrazzano-managed": "true", "istio-injection": "enabled"}); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create namespace: %v", err))
	}

	pkg.Log(pkg.Info, "Create AuthPolicy App resources")
	if err := pkg.CreateOrUpdateResourceFromFile("testdata/istio/authz/foo/istio-securitytest-app.yaml"); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create AuthPolicy application resources: %v", err))
	}

	pkg.Log(pkg.Info, "Create Sleep Component")
	if err := pkg.CreateOrUpdateResourceFromFile("testdata/istio/authz/foo/sleep-comp.yaml"); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create AuthPolicy Sleep component resources: %v", err))
	}

	pkg.Log(pkg.Info, "Create Backend Component")
	if err := pkg.CreateOrUpdateResourceFromFile("testdata/istio/authz/foo/springboot-backend.yaml"); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create AuthPolicy BackEnd component resources: %v", err))
	}

	pkg.Log(pkg.Info, "Create Frontend Component")
	if err := pkg.CreateOrUpdateResourceFromFile("testdata/istio/authz/foo/springboot-frontend.yaml"); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create AuthPolicy FrontEnd component resources: %v", err))
	}

}

func deployBarApplication() {
	pkg.Log(pkg.Info, "Deploy Auth Policy Application in bar namespace")

	pkg.Log(pkg.Info, "Create namespace")
	if _, err := pkg.CreateNamespace(barNamespace, map[string]string{"verrazzano-managed": "true", "istio-injection": "enabled"}); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create namespace: %v", err))
	}

	pkg.Log(pkg.Info, "Create AuthPolicy App resources")
	if err := pkg.CreateOrUpdateResourceFromFile("testdata/istio/authz/bar/istio-securitytest-app.yaml"); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create AuthPolicy application resources: %v", err))
	}

	pkg.Log(pkg.Info, "Create Sleep Component")
	if err := pkg.CreateOrUpdateResourceFromFile("testdata/istio/authz/bar/sleep-comp.yaml"); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create AuthPolicy Sleep component resources: %v", err))
	}

	pkg.Log(pkg.Info, "Create Backend Component")
	if err := pkg.CreateOrUpdateResourceFromFile("testdata/istio/authz/bar/springboot-backend.yaml"); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create AuthPolicy BackEnd component resources: %v", err))
	}

	pkg.Log(pkg.Info, "Create Frontend Component")
	if err := pkg.CreateOrUpdateResourceFromFile("testdata/istio/authz/bar/springboot-frontend.yaml"); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create AuthPolicy FrontEnd component resources: %v", err))
	}

}

func deployNoIstioApplication() {
	pkg.Log(pkg.Info, "Deploy Auth Policy Application in NoIstio namespace")

	pkg.Log(pkg.Info, "Create namespace")
	if _, err := pkg.CreateNamespace(noIstioNamespace, map[string]string{"verrazzano-managed": "true"}); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create namespace: %v", err))
	}

	pkg.Log(pkg.Info, "Create AuthPolicy App resources")
	if err := pkg.CreateOrUpdateResourceFromFile("testdata/istio/authz/noistio/istio-securitytest-app.yaml"); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create AuthPolicy application resources: %v", err))
	}

	pkg.Log(pkg.Info, "Create Sleep Component")
	if err := pkg.CreateOrUpdateResourceFromFile("testdata/istio/authz/noistio/sleep-comp.yaml"); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create AuthPolicy Sleep component resources: %v", err))
	}

	pkg.Log(pkg.Info, "Create Backend Component")
	if err := pkg.CreateOrUpdateResourceFromFile("testdata/istio/authz/noistio/springboot-backend.yaml"); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create AuthPolicy BackEnd component resources: %v", err))
	}

	pkg.Log(pkg.Info, "Create Frontend Component")
	if err := pkg.CreateOrUpdateResourceFromFile("testdata/istio/authz/noistio/springboot-frontend.yaml"); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create AuthPolicy FrontEnd component resources: %v", err))
	}

}

func undeployFooApplication() {
	pkg.Log(pkg.Info, "Undeploy Auth Policy Application in foo namespace")
	pkg.Log(pkg.Info, "Delete application")
	if err := pkg.DeleteResourceFromFile("testdata/istio/authz/foo/istio-securitytest-app.yaml"); err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to delete the application: %v", err))
	}
	pkg.Log(pkg.Info, "Delete components")
	if err := pkg.DeleteResourceFromFile("testdata/istio/authz/foo/sleep-comp.yaml"); err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to delete the component: %v", err))
	}
	if err := pkg.DeleteResourceFromFile("testdata/istio/authz/foo/springboot-backend.yaml"); err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to delete the component: %v", err))
	}
	if err := pkg.DeleteResourceFromFile("testdata/istio/authz/foo/springboot-frontend.yaml"); err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to delete the component: %v", err))
	}

	if noPods := pkg.PodsNotRunning(fooNamespace, expectedPodsFoo); noPods != true {
		pkg.Log(pkg.Error, fmt.Sprintf("Pods in namespace %s stuck terminating!", fooNamespace))
	}

	pkg.Log(pkg.Info, "Delete namespace")
	if err := pkg.DeleteNamespace(fooNamespace); err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to delete the namespace: %v", err))
	}
	gomega.Eventually(func() bool {
		ns, err := pkg.GetNamespace(fooNamespace)
		return ns == nil && err != nil && errors.IsNotFound(err)
	}, 3*time.Minute, 15*time.Second).Should(gomega.BeFalse())
}

func undeployBarApplication() {
	pkg.Log(pkg.Info, "Undeploy Auth Policy Application in bar namespace")
	pkg.Log(pkg.Info, "Delete application")
	if err := pkg.DeleteResourceFromFile("testdata/istio/authz/bar/istio-securitytest-app.yaml"); err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to delete the application: %v", err))
	}
	pkg.Log(pkg.Info, "Delete components")
	if err := pkg.DeleteResourceFromFile("testdata/istio/authz/bar/sleep-comp.yaml"); err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to delete the component: %v", err))
	}
	if err := pkg.DeleteResourceFromFile("testdata/istio/authz/bar/springboot-backend.yaml"); err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to delete the component: %v", err))
	}
	if err := pkg.DeleteResourceFromFile("testdata/istio/authz/bar/springboot-frontend.yaml"); err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to delete the component: %v", err))
	}

	if noPods := pkg.PodsNotRunning(barNamespace, expectedPodsBar); noPods != true {
		pkg.Log(pkg.Error, fmt.Sprintf("Pods in namespace %s stuck terminating!", barNamespace))
	}

	pkg.Log(pkg.Info, "Delete namespace")
	if err := pkg.DeleteNamespace(barNamespace); err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to delete the namespace: %v", err))
	}
	gomega.Eventually(func() bool {
		ns, err := pkg.GetNamespace(barNamespace)
		return ns == nil && err != nil && errors.IsNotFound(err)
	}, 3*time.Minute, 15*time.Second).Should(gomega.BeFalse())
}

func undeployNoIstioApplication() {
	pkg.Log(pkg.Info, "Undeploy Auth Policy Application in noistio namespace")
	pkg.Log(pkg.Info, "Delete application")
	if err := pkg.DeleteResourceFromFile("testdata/istio/authz/noistio/istio-securitytest-app.yaml"); err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to delete the application: %v", err))
	}
	pkg.Log(pkg.Info, "Delete components")
	if err := pkg.DeleteResourceFromFile("testdata/istio/authz/noistio/sleep-comp.yaml"); err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to delete the component: %v", err))
	}
	if err := pkg.DeleteResourceFromFile("testdata/istio/authz/noistio/springboot-backend.yaml"); err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to delete the component: %v", err))
	}
	if err := pkg.DeleteResourceFromFile("testdata/istio/authz/noistio/springboot-frontend.yaml"); err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to delete the component: %v", err))
	}

	if noPods := pkg.PodsNotRunning(noIstioNamespace, expectedPodsBar); noPods != true {
		pkg.Log(pkg.Error, fmt.Sprintf("Pods in namespace %s stuck terminating!", noIstioNamespace))
	}

	pkg.Log(pkg.Info, "Delete namespace")
	if err := pkg.DeleteNamespace(noIstioNamespace); err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to delete the namespace: %v", err))
	}
	gomega.Eventually(func() bool {
		ns, err := pkg.GetNamespace(noIstioNamespace)
		return ns == nil && err != nil && errors.IsNotFound(err)
	}, 3*time.Minute, 15*time.Second).Should(gomega.BeFalse())
}

var _ = ginkgo.Describe("Verify AuthPolicy Applications", func() {
	// Verify springboot-workload pod is running
	// GIVEN springboot app is deployed
	// WHEN the component and appconfig are created
	// THEN the expected pod must be running in the test namespace
	ginkgo.Context("Deployment.", func() {
		ginkgo.It("and waiting for expected pods must be running", func() {
			gomega.Eventually(func() bool {
				return pkg.PodsRunning(fooNamespace, expectedPodsFoo)
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue(), fmt.Sprintf("Auth Policy Application failed to start in %s", fooNamespace))
		})
	})

	ginkgo.Context("Deployment.", func() {
		ginkgo.It("and waiting for expected pods must be running", func() {
			gomega.Eventually(func() bool {
				return pkg.PodsRunning(barNamespace, expectedPodsBar)
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue(), fmt.Sprintf("Auth Policy Application failed to start in %s", barNamespace))
		})
	})

	ginkgo.Context("Deployment.", func() {
		ginkgo.It("and waiting for expected pods must be running", func() {
			gomega.Eventually(func() bool {
				return pkg.PodsRunning(noIstioNamespace, expectedPodsBar)
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue(), fmt.Sprintf("Auth Policy Application failed to start in %s", noIstioNamespace))
		})
	})

	var fooHost = ""
	ginkgo.It("Get foo host from gateway.", func() {
		gomega.Eventually(func() string {
			fooHost = pkg.GetHostnameFromGateway(fooNamespace, "")
			return fooHost
		}, waitTimeout, shortPollingInterval).Should(gomega.Not(gomega.BeEmpty()), fmt.Sprintf("Failed to get host from gateway in %s", fooNamespace))
	})

	var barHost = ""
	ginkgo.It("Get bar host from gateway.", func() {
		gomega.Eventually(func() string {
			barHost = pkg.GetHostnameFromGateway(barNamespace, "")
			return barHost
		}, waitTimeout, shortPollingInterval).Should(gomega.Not(gomega.BeEmpty()), fmt.Sprintf("Failed to get host from gateway in %s", barNamespace))
	})

	var noIstioHost = ""
	ginkgo.It("Get noistio host from gateway.", func() {
		gomega.Eventually(func() string {
			noIstioHost = pkg.GetHostnameFromGateway(noIstioNamespace, "")
			return noIstioHost
		}, waitTimeout, shortPollingInterval).Should(gomega.Not(gomega.BeEmpty()), fmt.Sprintf("Failed to get host from gateway in %s", noIstioNamespace))
	})

	// Verify application in namespace foo is working
	// GIVEN authorization test app is deployed
	// WHEN the component and appconfig with ingress trait are created
	// THEN the application endpoint must be accessible
	ginkgo.It("Verify welcome page of Foo Spring Boot FrontEnd is working.", func() {
		gomega.Eventually(func() (*pkg.HTTPResponse, error) {
			pkg.Log(pkg.Info, fmt.Sprintf("Ingress: %s", fooHost))
			url := fmt.Sprintf("https://%s/", fooHost)
			return pkg.GetWebPage(url, fooHost)
		}, waitTimeout, shortPollingInterval).Should(gomega.And(pkg.HasStatus(http.StatusOK), pkg.BodyContains("Greetings from Verrazzano Enterprise Container Platform")))
	})

	// Verify application in namespace bar is working
	// GIVEN authorization test app is deployed
	// WHEN the component and appconfig with ingress trait are created
	// THEN the application endpoint must be accessible
	ginkgo.It("Verify welcome page of Bar Spring Boot FrontEnd is working.", func() {
		gomega.Eventually(func() (*pkg.HTTPResponse, error) {
			pkg.Log(pkg.Info, fmt.Sprintf("Ingress: %s", barHost))
			url := fmt.Sprintf("https://%s/", barHost)
			return pkg.GetWebPage(url, barHost)
		}, waitTimeout, shortPollingInterval).Should(gomega.And(pkg.HasStatus(http.StatusOK), pkg.BodyContains("Greetings from Verrazzano Enterprise Container Platform")))
	})

	// Verify application in namespace bar is working
	// GIVEN authorization test app is deployed
	// WHEN the component and appconfig with ingress trait are created
	// THEN the application endpoint must be accessible
	ginkgo.It("Verify welcome page of NoIstio Spring Boot FrontEnd is working.", func() {
		gomega.Eventually(func() (*pkg.HTTPResponse, error) {
			pkg.Log(pkg.Info, fmt.Sprintf("Ingress: %s", noIstioHost))
			url := fmt.Sprintf("https://%s/", noIstioHost)
			return pkg.GetWebPage(url, noIstioHost)
		}, waitTimeout, shortPollingInterval).Should(gomega.And(pkg.HasStatus(http.StatusOK), pkg.BodyContains("Greetings from Verrazzano Enterprise Container Platform")))
	})

	// Verify Frontend can call Backend in foo
	// GIVEN authorization test app is deployed
	// WHEN the component and appconfig with ingress trait are created
	// THEN the frontend should be able to successfully call the backend and get a 200 on that invocation
	// the http code is returned in the response body and captured in content
	ginkgo.It("Verify Foo Frontend can call Foo Backend.", func() {
		gomega.Eventually(func() (*pkg.HTTPResponse, error) {
			pkg.Log(pkg.Info, fmt.Sprintf("Ingress: %s", fooHost))
			url := fmt.Sprintf("https://%s/externalCall?inurl=http://springboot-backend-workload.foo:8080/", fooHost)
			return pkg.GetWebPage(url, fooHost)
		}, waitTimeout, shortPollingInterval).Should(gomega.And(pkg.HasStatus(http.StatusOK), pkg.BodyEquals("200")))
	})

	// Verify Frontend can call Backend in bar
	// GIVEN authorization test app is deployed
	// WHEN the component and appconfig with ingress trait are created
	// THEN the frontend should be able to successfully call the backend and get a 200 on that invocation
	// the http code is returned in the response body and captured in content
	ginkgo.It("Verify Bar Frontend can call Bar Backend.", func() {
		gomega.Eventually(func() (*pkg.HTTPResponse, error) {
			pkg.Log(pkg.Info, fmt.Sprintf("Ingress: %s", barHost))
			url := fmt.Sprintf("https://%s/externalCall?inurl=http://springboot-backend-workload.bar:8080/", barHost)
			return pkg.GetWebPage(url, barHost)
		}, waitTimeout, shortPollingInterval).Should(gomega.And(pkg.HasStatus(http.StatusOK), pkg.BodyEquals("200")))
	})

	// Verify Foo Frontend can't call bar Backend
	// GIVEN authorization test app is deployed
	// WHEN the component and appconfig with ingress trait are created
	// THEN the frontend should be able to successfully call the backend and get a 200 on that invocation
	// the http code is returned in the response body and captured in content
	ginkgo.It("Verify Foo Frontend canNOT call Bar Backend.", func() {
		gomega.Eventually(func() (*pkg.HTTPResponse, error) {
			pkg.Log(pkg.Info, fmt.Sprintf("Ingress: %s", fooHost))
			url := fmt.Sprintf("https://%s/externalCall?inurl=http://springboot-backend-workload.bar:8080/", fooHost)
			return pkg.GetWebPage(url, fooHost)
		}, waitTimeout, shortPollingInterval).Should(gomega.And(pkg.HasStatus(http.StatusOK), pkg.BodyEquals("403")))
	})

	// Verify Bar Frontend can't call Foo Backend
	// GIVEN authorization test app is deployed
	// WHEN the component and appconfig with ingress trait are created
	// THEN the frontend should be able to successfully call the backend and get a 200 on that invocation
	// the http code is returned in the response body and captured in content
	ginkgo.It("Verify Bar Frontend canNOT call Foo Backend.", func() {
		gomega.Eventually(func() (*pkg.HTTPResponse, error) {
			pkg.Log(pkg.Info, fmt.Sprintf("Ingress: %s", barHost))
			url := fmt.Sprintf("https://%s/externalCall?inurl=http://springboot-backend-workload.foo:8080/", barHost)
			return pkg.GetWebPage(url, barHost)
		}, waitTimeout, shortPollingInterval).Should(gomega.And(pkg.HasStatus(http.StatusOK), pkg.BodyEquals("403")))
	})

	// Verify Bar Frontend can call NoIstio Backend
	// GIVEN authorization test app is deployed
	// WHEN the component and appconfig with ingress trait are created
	// THEN the frontend should be able to successfully call the backend and get a 200 on that invocation
	// the http code is returned in the response body and captured in content
	ginkgo.It("Verify Bar Frontend can call NoIstio Backend.", func() {
		gomega.Eventually(func() (*pkg.HTTPResponse, error) {
			pkg.Log(pkg.Info, fmt.Sprintf("Ingress: %s", barHost))
			url := fmt.Sprintf("https://%s/externalCall?inurl=http://springboot-backend-workload.noistio:8080/", barHost)
			return pkg.GetWebPage(url, barHost)
		}, waitTimeout, shortPollingInterval).Should(gomega.And(pkg.HasStatus(http.StatusOK), pkg.BodyEquals("200")))
	})

	// Verify noistio Frontend can't call bar Backend
	// GIVEN authorization test app is deployed
	// WHEN the component and appconfig with ingress trait are created
	// THEN the frontend should be able to successfully call the backend and get a 200 on that invocation
	// the http code is returned in the response body and captured in content
	// *** This call should fail for a 500 because Non-Istio can't call Istio when MTLS is STRICT
	// If this should fail because the call succeeded, verify that peerauthentication exists in istio-system and is set to STRICT
	ginkgo.It("Verify NoIstio Frontend canNOT call Bar Backend.", func() {
		gomega.Eventually(func() bool {
			pkg.Log(pkg.Info, fmt.Sprintf("Ingress: %s", noIstioHost))
			url := fmt.Sprintf("https://%s/externalCall?inurl=http://springboot-backend-workload.bar:8080/", noIstioHost)

			kubeconfigPath := pkg.GetKubeConfigPathFromEnv()
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
		}, waitTimeout, shortPollingInterval).Should(gomega.BeTrue(), "Failed to Verify NoIstio Frontend canNOT call Bar Backend")
	})

})

var _ = ginkgo.Describe("Verify Auth Policy Prometheus Scrape Targets", func() {
	// Verify springboot-workload pod is running
	// GIVEN springboot app is deployed
	// WHEN the component and appconfig are created
	// THEN the expected pod must be running in the test namespace
	ginkgo.Context("Deployment.", func() {
		ginkgo.It("and waiting for expected pods must be running", func() {
			gomega.Eventually(func() bool {
				return pkg.PodsRunning(fooNamespace, expectedPodsFoo)
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue(), fmt.Sprintf("Auth Policy Application failed to start in %s", fooNamespace))
		})
	})

	ginkgo.Context("Deployment.", func() {
		ginkgo.It("and waiting for expected pods must be running", func() {
			gomega.Eventually(func() bool {
				return pkg.PodsRunning(barNamespace, expectedPodsBar)
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue(), fmt.Sprintf("Auth Policy Application failed to start in %s", barNamespace))
		})
	})

	ginkgo.Context("Deployment.", func() {
		ginkgo.It("and waiting for expected pods must be running", func() {
			gomega.Eventually(func() bool {
				return pkg.PodsRunning(noIstioNamespace, expectedPodsBar)
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue(), fmt.Sprintf("Auth Policy Application failed to start in %s", noIstioNamespace))
		})
	})

	// Verify That Generated Prometheus Scrape Targets for authpolicy-appconf_default_foo_springboot-frontend is using https for scraping
	// GIVEN that springboot deployed to Istio namespace foo
	// WHEN the Prometheus scrape targets are created
	// THEN they should be created to use the https protocol
	ginkgo.It("Verify that Istio scrape target authpolicy-appconf_default_foo_springboot-frontend is using https for scraping.", func() {
		gomega.Eventually(func() bool {
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
		}, waitTimeout, shortPollingInterval).Should(gomega.BeTrue(), "Failed to Verify that Istio scrape target authpolicy-appconf_default_foo_springboot-frontend is using https for scraping")
	})

	// Verify That Generated Prometheus Scrape Targets for authpolicy-appconf_default_bar_springboot-frontend is using https for scraping
	// GIVEN that springboot deployed to Istio namespace bar
	// WHEN the Prometheus scrape targets are created
	// THEN they should be created to use the https protocol
	ginkgo.It("Verify that Istio scrape target authpolicy-appconf_default_bar_springboot-frontend is using https for scraping.", func() {
		gomega.Eventually(func() bool {
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
		}, waitTimeout, shortPollingInterval).Should(gomega.BeTrue(), "Failed to Verify that Istio scrape target authpolicy-appconf_default_bar_springboot-frontend is using https for scraping")
	})

	// Verify That Generated Prometheus Scrape Targets for authpolicy-appconf_default_noistio_springboot-frontend is using http for scraping
	// GIVEN that springboot deployed to namespace noistio
	// WHEN the Prometheus scrape targets are created
	// THEN they should be created to use the http protocol
	ginkgo.It("Verify that Istio scrape target authpolicy-appconf_default_noistio_springboot-frontend is using http for scraping.", func() {
		gomega.Eventually(func() bool {
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
		}, waitTimeout, shortPollingInterval).Should(gomega.BeTrue(), "Failed to Verify that Istio scrape target authpolicy-appconf_default_noistio_springboot-frontend is using http for scraping")
	})

})
