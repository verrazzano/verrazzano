// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package authz

import (
	"fmt"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/pkg/test/framework/metrics"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	testpkg "github.com/verrazzano/verrazzano/tests/e2e/pkg"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
)

const fooNamespace string = "foo"
const barNamespace string = "bar"
const noIstioNamespace string = "noistio"

var expectedPodsFoo = []string{"sleep-workload", "springboot-frontend-workload", "springboot-backend-workload"}
var expectedPodsBar = []string{"sleep-workload", "springboot-frontend-workload", "springboot-backend-workload"}
var waitTimeout = 15 * time.Minute
var pollingInterval = 30 * time.Second
var shortPollingInterval = 10 * time.Second

var t = framework.NewTestFramework("authz")

var _ = t.BeforeSuite(func() {
	start := time.Now()
	deployFooApplication()
	deployBarApplication()
	deployNoIstioApplication()
	beforeSuitePassed = true
	metrics.Emit(t.Metrics.With("deployment_elapsed_time", time.Since(start).Milliseconds()))
})

var failed = false
var beforeSuitePassed = false

var _ = t.AfterEach(func() {
	failed = failed || framework.VzCurrentGinkgoTestDescription().Failed()
})

var _ = t.AfterSuite(func() {
	if failed || !beforeSuitePassed {
		pkg.ExecuteBugReport(fooNamespace, barNamespace, noIstioNamespace)
	}
	start := time.Now()
	undeployFooApplication()
	undeployBarApplication()
	undeployNoIstioApplication()
	metrics.Emit(t.Metrics.With("undeployment_elapsed_time", time.Since(start).Milliseconds()))
})

func deployFooApplication() {
	t.Logs.Info("Deploy Auth Policy Application in foo namespace")

	t.Logs.Info("Create namespace")
	Eventually(func() (*v1.Namespace, error) {
		return pkg.CreateNamespace(fooNamespace, map[string]string{"verrazzano-managed": "true", "istio-injection": istioInjection})
	}, waitTimeout, shortPollingInterval).ShouldNot(BeNil())

	t.Logs.Info("Create AuthPolicy App resources")
	Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFile("testdata/istio/authz/foo/istio-securitytest-app.yaml")
	}, waitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	t.Logs.Info("Create Sleep Component")
	Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFile("testdata/istio/authz/foo/sleep-comp.yaml")
	}, waitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	t.Logs.Info("Create Backend Component")
	Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFile("testdata/istio/authz/foo/springboot-backend.yaml")
	}, waitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	t.Logs.Info("Create Frontend Component")
	Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFile("testdata/istio/authz/foo/springboot-frontend.yaml")
	}, waitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

}

func deployBarApplication() {
	t.Logs.Info("Deploy Auth Policy Application in bar namespace")

	t.Logs.Info("Create namespace")
	Eventually(func() (*v1.Namespace, error) {
		return pkg.CreateNamespace(barNamespace, map[string]string{"verrazzano-managed": "true", "istio-injection": istioInjection})
	}, waitTimeout, shortPollingInterval).ShouldNot(BeNil())

	t.Logs.Info("Create AuthPolicy App resources")
	Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFile("testdata/istio/authz/bar/istio-securitytest-app.yaml")
	}, waitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	t.Logs.Info("Create Sleep Component")
	Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFile("testdata/istio/authz/bar/sleep-comp.yaml")
	}, waitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	t.Logs.Info("Create Backend Component")
	Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFile("testdata/istio/authz/bar/springboot-backend.yaml")
	}, waitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	t.Logs.Info("Create Frontend Component")
	Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFile("testdata/istio/authz/bar/springboot-frontend.yaml")
	}, waitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

}

func deployNoIstioApplication() {
	t.Logs.Info("Deploy Auth Policy Application in NoIstio namespace")

	t.Logs.Info("Create namespace")
	Eventually(func() (*v1.Namespace, error) {
		return pkg.CreateNamespace(noIstioNamespace, map[string]string{"verrazzano-managed": "true"})
	}, waitTimeout, shortPollingInterval).ShouldNot(BeNil())

	t.Logs.Info("Create AuthPolicy App resources")
	Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFile("testdata/istio/authz/noistio/istio-securitytest-app.yaml")
	}, waitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	t.Logs.Info("Create Sleep Component")
	Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFile("testdata/istio/authz/noistio/sleep-comp.yaml")
	}, waitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	t.Logs.Info("Create Backend Component")
	Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFile("testdata/istio/authz/noistio/springboot-backend.yaml")
	}, waitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	t.Logs.Info("Create Frontend Component")
	Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFile("testdata/istio/authz/noistio/springboot-frontend.yaml")
	}, waitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

}

func undeployFooApplication() {
	t.Logs.Info("Undeploy Auth Policy Application in foo namespace")

	t.Logs.Info("Delete application")
	Eventually(func() error {
		return pkg.DeleteResourceFromFile("testdata/istio/authz/foo/istio-securitytest-app.yaml")
	}, waitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	t.Logs.Info("Delete components")
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

	t.Logs.Info("Delete namespace")
	Eventually(func() error {
		return pkg.DeleteNamespace(fooNamespace)
	}, waitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	Eventually(func() bool {
		_, err := pkg.GetNamespace(fooNamespace)
		return err != nil && errors.IsNotFound(err)
	}, waitTimeout, shortPollingInterval).Should(BeTrue())
}

func undeployBarApplication() {
	t.Logs.Info("Undeploy Auth Policy Application in bar namespace")

	t.Logs.Info("Delete application")
	Eventually(func() error {
		return pkg.DeleteResourceFromFile("testdata/istio/authz/bar/istio-securitytest-app.yaml")
	}, waitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	t.Logs.Info("Delete components")
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

	t.Logs.Info("Delete namespace")
	Eventually(func() error {
		return pkg.DeleteNamespace(barNamespace)
	}, waitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	Eventually(func() bool {
		_, err := pkg.GetNamespace(barNamespace)
		return err != nil && errors.IsNotFound(err)
	}, waitTimeout, shortPollingInterval).Should(BeTrue())
}

func undeployNoIstioApplication() {
	t.Logs.Info("Undeploy Auth Policy Application in noistio namespace")

	t.Logs.Info("Delete application")
	Eventually(func() error {
		return pkg.DeleteResourceFromFile("testdata/istio/authz/noistio/istio-securitytest-app.yaml")
	}, waitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	t.Logs.Info("Delete components")
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

	t.Logs.Info("Delete namespace")
	Eventually(func() error {
		return pkg.DeleteNamespace(noIstioNamespace)
	}, waitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	Eventually(func() bool {
		_, err := pkg.GetNamespace(noIstioNamespace)
		return err != nil && errors.IsNotFound(err)
	}, waitTimeout, shortPollingInterval).Should(BeTrue())
}

var _ = t.Describe("AuthPolicy test,", Label("f:security.authpol",
	"f:app-lcm.spring-workload"), func() {
	// Verify springboot-workload pod is running
	// GIVEN springboot app is deployed
	// WHEN the component and appconfig are created
	// THEN the expected pod must be running in the test namespace
	t.Context("check app deployment", func() {
		t.It("in foo namespace", func() {
			Eventually(func() bool {
				return checkPodsRunning(fooNamespace, expectedPodsFoo)
			}, waitTimeout, pollingInterval).Should(BeTrue(), fmt.Sprintf("Auth Policy Application failed to start in %s", fooNamespace))
		})
	})

	t.Context("check app deployment", func() {
		t.It("in bar namespace", func() {
			Eventually(func() bool {
				return checkPodsRunning(barNamespace, expectedPodsBar)
			}, waitTimeout, pollingInterval).Should(BeTrue(), fmt.Sprintf("Auth Policy Application failed to start in %s", barNamespace))
		})
	})

	t.Context("check app deployment", func() {
		t.It("in noistio namespace", func() {
			Eventually(func() bool {
				return checkPodsRunning(noIstioNamespace, expectedPodsBar)
			}, waitTimeout, pollingInterval).Should(BeTrue(), fmt.Sprintf("Auth Policy Application failed to start in %s", noIstioNamespace))
		})
	})

	var fooHost = ""
	var barHost = ""
	var noIstioHost = ""

	var err error
	t.BeforeEach(func() {
		Eventually(func() (string, error) {
			fooHost, err = k8sutil.GetHostnameFromGateway(fooNamespace, "")
			return fooHost, err
		}, waitTimeout, shortPollingInterval).Should(Not(BeEmpty()), fmt.Sprintf("Failed to get host from gateway in %s", fooNamespace))

		Eventually(func() (string, error) {
			barHost, err = k8sutil.GetHostnameFromGateway(barNamespace, "")
			return barHost, err
		}, waitTimeout, shortPollingInterval).Should(Not(BeEmpty()), fmt.Sprintf("Failed to get host from gateway in %s", barNamespace))

		Eventually(func() (string, error) {
			noIstioHost, err = k8sutil.GetHostnameFromGateway(noIstioNamespace, "")
			return noIstioHost, err
		}, waitTimeout, shortPollingInterval).Should(Not(BeEmpty()), fmt.Sprintf("Failed to get host from gateway in %s", noIstioNamespace))
	})

	// Verify application in namespace foo is working
	// GIVEN authorization test app is deployed
	// WHEN the component and appconfig with ingress trait are created
	// THEN the application endpoint must be accessible
	t.It("Verify welcome page of Foo Spring Boot FrontEnd is working.", func() {
		Eventually(func() (*pkg.HTTPResponse, error) {
			t.Logs.Infof("Ingress: %s", fooHost)
			url := fmt.Sprintf("https://%s/", fooHost)
			return pkg.GetWebPage(url, fooHost)
		}, waitTimeout, shortPollingInterval).Should(And(pkg.HasStatus(http.StatusOK), pkg.BodyContains("Greetings from Verrazzano Enterprise Container Platform")))
	})

	// Verify application in namespace bar is working
	// GIVEN authorization test app is deployed
	// WHEN the component and appconfig with ingress trait are created
	// THEN the application endpoint must be accessible
	t.It("Verify welcome page of Bar Spring Boot FrontEnd is working.", func() {
		Eventually(func() (*pkg.HTTPResponse, error) {
			t.Logs.Infof("Ingress: %s", barHost)
			url := fmt.Sprintf("https://%s/", barHost)
			return pkg.GetWebPage(url, barHost)
		}, waitTimeout, shortPollingInterval).Should(And(pkg.HasStatus(http.StatusOK), pkg.BodyContains("Greetings from Verrazzano Enterprise Container Platform")))
	})

	// Verify application in namespace bar is working
	// GIVEN authorization test app is deployed
	// WHEN the component and appconfig with ingress trait are created
	// THEN the application endpoint must be accessible
	t.It("Verify welcome page of NoIstio Spring Boot FrontEnd is working.", func() {
		Eventually(func() (*pkg.HTTPResponse, error) {
			t.Logs.Infof("Ingress: %s", noIstioHost)
			url := fmt.Sprintf("https://%s/", noIstioHost)
			return pkg.GetWebPage(url, noIstioHost)
		}, waitTimeout, shortPollingInterval).Should(And(pkg.HasStatus(http.StatusOK), pkg.BodyContains("Greetings from Verrazzano Enterprise Container Platform")))
	})

	// Verify Frontend can call Backend in foo
	// GIVEN authorization test app is deployed
	// WHEN the component and appconfig with ingress trait are created
	// THEN the frontend should be able to successfully call the backend and get a 200 on that invocation
	// the http code is returned in the response body and captured in content
	t.It("Verify Foo Frontend can call Foo Backend.", func() {
		Eventually(func() (*pkg.HTTPResponse, error) {
			t.Logs.Infof("Ingress: %s", fooHost)
			url := fmt.Sprintf("https://%s/externalCall?inurl=http://springboot-backend-workload.foo:8080/", fooHost)
			return pkg.GetWebPage(url, fooHost)
		}, waitTimeout, shortPollingInterval).Should(And(pkg.HasStatus(http.StatusOK), pkg.BodyEquals("200")))
	})

	// Verify Frontend can call Backend in bar
	// GIVEN authorization test app is deployed
	// WHEN the component and appconfig with ingress trait are created
	// THEN the frontend should be able to successfully call the backend and get a 200 on that invocation
	// the http code is returned in the response body and captured in content
	t.It("Verify Bar Frontend can call Bar Backend.", func() {
		Eventually(func() (*pkg.HTTPResponse, error) {
			t.Logs.Infof("Ingress: %s", barHost)
			url := fmt.Sprintf("https://%s/externalCall?inurl=http://springboot-backend-workload.bar:8080/", barHost)
			return pkg.GetWebPage(url, barHost)
		}, waitTimeout, shortPollingInterval).Should(And(pkg.HasStatus(http.StatusOK), pkg.BodyEquals("200")))
	})

	// Verify Foo Frontend can't call bar Backend
	// GIVEN authorization test app is deployed
	// WHEN the component and appconfig with ingress trait are created
	// THEN the frontend should be able to successfully call the backend and get a 200 on that invocation
	// the http code is returned in the response body and captured in content
	t.It("Verify Foo Frontend canNOT call Bar Backend.", func() {
		Eventually(func() (*pkg.HTTPResponse, error) {
			t.Logs.Infof("Ingress: %s", fooHost)
			url := fmt.Sprintf("https://%s/externalCall?inurl=http://springboot-backend-workload.bar:8080/", fooHost)
			return pkg.GetWebPage(url, fooHost)
		}, waitTimeout, shortPollingInterval).Should(And(pkg.HasStatus(http.StatusOK), pkg.BodyEquals("403")))
	})

	// Verify Bar Frontend can't call Foo Backend
	// GIVEN authorization test app is deployed
	// WHEN the component and appconfig with ingress trait are created
	// THEN the frontend should be able to successfully call the backend and get a 200 on that invocation
	// the http code is returned in the response body and captured in content
	t.It("Verify Bar Frontend canNOT call Foo Backend.", func() {
		Eventually(func() (*pkg.HTTPResponse, error) {
			t.Logs.Infof("Ingress: %s", barHost)
			url := fmt.Sprintf("https://%s/externalCall?inurl=http://springboot-backend-workload.foo:8080/", barHost)
			return pkg.GetWebPage(url, barHost)
		}, waitTimeout, shortPollingInterval).Should(And(pkg.HasStatus(http.StatusOK), pkg.BodyEquals("403")))
	})

	// Verify Bar Frontend can call NoIstio Backend
	// GIVEN authorization test app is deployed
	// WHEN the component and appconfig with ingress trait are created
	// THEN the frontend should be able to successfully call the backend and get a 200 on that invocation
	// the http code is returned in the response body and captured in content
	t.It("Verify Bar Frontend can call NoIstio Backend.", func() {
		Eventually(func() (*pkg.HTTPResponse, error) {
			t.Logs.Infof("Ingress: %s", barHost)
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
	t.It("Verify NoIstio Frontend canNOT call Bar Backend.", func() {
		kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
		Expect(err).ShouldNot(HaveOccurred())
		Eventually(func() bool {
			t.Logs.Infof("Ingress: %s", noIstioHost)
			url := fmt.Sprintf("https://%s/externalCall?inurl=http://springboot-backend-workload.bar:8080/", noIstioHost)
			client, err := pkg.GetVerrazzanoNoRetryHTTPClient(kubeconfigPath)
			if err != nil {
				t.Logs.Errorf("Failed to get client: %v", err)
				return false
			}
			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				t.Logs.Errorf("Failed to create http request: %v", err)
				return false
			}
			req.Host = noIstioHost

			q := req.URL.Query()
			q.Add("inurl", "https://springboot-backend-workload.bar:8080/")
			req.URL.RawQuery = q.Encode()

			resp, err := client.Do(req)
			if err != nil {
				// could be a transient network error so log it and let the Eventually retry
				t.Logs.Errorf("Failed to do http request: %v", err)
				return false
			}
			return resp.StatusCode == 500
		}, waitTimeout, shortPollingInterval).Should(BeTrue(), "Failed to Verify NoIstio Frontend canNOT call Bar Backend")
	})

})

var _ = t.Describe("Verify Auth Policy Prometheus Scrape Targets", func() {
	// Verify springboot-workload pod is running
	// GIVEN springboot app is deployed
	// WHEN the component and appconfig are created
	// THEN the expected pod must be running in the test namespace
	t.Context("Deployment.", func() {
		t.It("and waiting for expected pods must be running", func() {
			Eventually(func() bool {
				return checkPodsRunning(fooNamespace, expectedPodsFoo)
			}, waitTimeout, pollingInterval).Should(BeTrue(), fmt.Sprintf("Auth Policy Application failed to start in %s", fooNamespace))
		})
	})

	t.Context("Deployment.", func() {
		t.It("and waiting for expected pods must be running", func() {
			Eventually(func() bool {
				return checkPodsRunning(barNamespace, expectedPodsBar)
			}, waitTimeout, pollingInterval).Should(BeTrue(), fmt.Sprintf("Auth Policy Application failed to start in %s", barNamespace))
		})
	})

	t.Context("Deployment.", func() {
		t.It("and waiting for expected pods must be running", func() {
			Eventually(func() bool {
				return checkPodsRunning(noIstioNamespace, expectedPodsBar)
			}, waitTimeout, pollingInterval).Should(BeTrue(), fmt.Sprintf("Auth Policy Application failed to start in %s", noIstioNamespace))
		})
	})

	// Verify That Generated Prometheus Scrape Targets for authpolicy-appconf_default_foo_springboot-frontend is using https for scraping
	// GIVEN that springboot deployed to Istio namespace foo
	// WHEN the Prometheus ServiceMonitors are created
	// THEN they should be created to use the https protocol
	t.It("Verify that ServiceMonitor authpolicy-appconf-default-foo-springboot-frontend is using https for scraping.", func() {
		Eventually(func() bool {
			serviceMonitor, err := testpkg.GetServiceMonitor(fooNamespace, "authpolicy-appconf-default-foo-springboot-frontend")
			if err != nil {
				return false
			}
			return serviceMonitor.Spec.Endpoints[0].Scheme == "https"
		}, waitTimeout, shortPollingInterval).Should(BeTrue(), "Failed to Verify that ServiceMonitor authpolicy-appconf-default-foo-springboot-frontend is using https for scraping")
	})

	// Verify That Generated Prometheus Scrape Targets for authpolicy-appconf_default_bar_springboot-frontend is using https for scraping
	// GIVEN that springboot deployed to Istio namespace bar
	// WHEN the Prometheus ServiceMonitors are created
	// THEN they should be created to use the https protocol
	t.It("Verify that ServiceMonitor authpolicy-appconf-default-bar-springboot-frontend is using https for scraping.", func() {
		Eventually(func() bool {
			serviceMonitor, err := testpkg.GetServiceMonitor(barNamespace, "authpolicy-appconf-default-bar-springboot-frontend")
			if err != nil {
				return false
			}
			return serviceMonitor.Spec.Endpoints[0].Scheme == "https"
		}, waitTimeout, shortPollingInterval).Should(BeTrue(), "Failed to Verify that ServiceMonitor authpolicy-appconf-default-bar-springboot-frontend is using https for scraping")
	})

	// Verify That Generated Prometheus Scrape Targets for authpolicy-appconf_default_noistio_springboot-frontend is using http for scraping
	// GIVEN that springboot deployed to namespace noistio
	// WHEN the Prometheus ServiceMonitors are created
	// THEN they should be created to use the http protocol
	t.It("Verify that ServiceMonitor authpolicy-appconf-default-noistio-springboot-frontend is using http for scraping.", func() {
		Eventually(func() bool {
			serviceMonitor, err := testpkg.GetServiceMonitor(noIstioNamespace, "authpolicy-appconf-default-noistio-springboot-frontend")
			if err != nil {
				return false
			}
			return serviceMonitor.Spec.Endpoints[0].Scheme == "http"
		}, waitTimeout, shortPollingInterval).Should(BeTrue(), "Failed to Verify that ServiceMonitor authpolicy-appconf-default-noistio-springboot-frontend is using http for scraping")
	})

})

// checkPodsRunning checks whether the pods are ready in a given namespace
func checkPodsRunning(namespace string, expectedPods []string) bool {
	result, err := pkg.PodsRunning(namespace, expectedPods)
	if err != nil {
		AbortSuite(fmt.Sprintf("One or more pods are not running in the namespace: %v, error: %v", namespace, err))
	}
	return result
}
