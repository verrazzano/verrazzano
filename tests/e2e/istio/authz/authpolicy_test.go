// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package authz_test

import (
	"fmt"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"k8s.io/apimachinery/pkg/api/errors"
	"net/http"
	"time"

	"github.com/onsi/ginkgo"
)

const fooNamespace string = "foo"
const barNamespace string = "bar"
const noIstioNamespace string = "noistio"

const fooHostHeaderValue string = "authpolicy.foo.com"
const barHostHeaderValue string = "authpolicy.bar.com"
const noIstioHostHeaderValue string = "authpolicy.noistio.com"

var expectedPodsFoo = []string{"sleep-workload", "springboot-frontend-workload", "springboot-backend-workload"}
var expectedPodsBar = []string{"sleep-workload", "springboot-frontend-workload", "springboot-backend-workload"}
var expectedPodsNoIstio = []string{"sleep-workload", "springboot-frontend-workload", "springboot-backend-workload"}
var waitTimeout = 10 * time.Minute
var pollingInterval = 30 * time.Second
var shortPollingInterval = 10 * time.Second
var shortWaitTimeout = 5 * time.Minute

var _ = ginkgo.BeforeSuite(func() {
	deployFooApplication()
	deployBarApplication()
	deployNoIstioApplication()
})

var _ = ginkgo.AfterSuite(func() {
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
	if _, err := pkg.CreateNamespace(barNamespace, map[string]string{"verrazzano-managed": "true" , "istio-injection": "enabled"}); err != nil {
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
	if _, err := pkg.CreateNamespace(noIstioNamespace, map[string]string{"verrazzano-managed": "true" }); err != nil {
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
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue())
		})
	})


	ginkgo.Context("Deployment.", func() {
		ginkgo.It("and waiting for expected pods must be running", func() {
			gomega.Eventually(func() bool {
				return pkg.PodsRunning(barNamespace, expectedPodsBar)
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue())
		})
	})


	ginkgo.Context("Deployment.", func() {
		ginkgo.It("and waiting for expected pods must be running", func() {
			gomega.Eventually(func() bool {
				return pkg.PodsRunning(noIstioNamespace, expectedPodsBar)
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue())
		})
	})

	// Verify application in namespace foo is working
	// GIVEN authorization test app is deployed
	// WHEN the component and appconfig with ingress trait are created
	// THEN the application endpoint must be accessible
	ginkgo.It("Verify welcome page of Foo Spring Boot FrontEnd is working.", func() {
		gomega.Eventually(func() bool {
			ingress := pkg.Ingress()
			pkg.Log(pkg.Info, fmt.Sprintf("Ingress: %s", ingress))
			url := fmt.Sprintf("http://%s/", ingress)
			status, content := pkg.GetWebPageWithCABundle(url, fooHostHeaderValue)
			return gomega.Expect(status).To(gomega.Equal(200)) &&
			//	gomega.Expect(content).To(gomega.ContainSubstring("Greetings from Verrazzano Enterprise Container Platform"))
			// Because I am using the old Springbbot Container for the FrontEnd
			    gomega.Expect(content).To(gomega.ContainSubstring("Greetings from Verrazzano Enterprise Container Platform"))
		}, shortWaitTimeout, shortPollingInterval).Should(gomega.BeTrue())
	})


	// Verify application in namespace bar is working
	// GIVEN authorization test app is deployed
	// WHEN the component and appconfig with ingress trait are created
	// THEN the application endpoint must be accessible
	ginkgo.It("Verify welcome page of Bar Spring Boot FrontEnd is working.", func() {
		gomega.Eventually(func() bool {
			ingress := pkg.Ingress()
			pkg.Log(pkg.Info, fmt.Sprintf("Ingress: %s", ingress))
			url := fmt.Sprintf("http://%s/", ingress)
			status, content := pkg.GetWebPageWithCABundle(url, barHostHeaderValue)
			return gomega.Expect(status).To(gomega.Equal(200)) &&
				//	gomega.Expect(content).To(gomega.ContainSubstring("Greetings from Verrazzano Enterprise Container Platform"))
				// Because I am using the old Springbbot Container for the FrontEnd
				gomega.Expect(content).To(gomega.ContainSubstring("Greetings from Verrazzano Enterprise Container Platform"))
		}, shortWaitTimeout, shortPollingInterval).Should(gomega.BeTrue())
	})


	// Verify application in namespace bar is working
	// GIVEN authorization test app is deployed
	// WHEN the component and appconfig with ingress trait are created
	// THEN the application endpoint must be accessible
	ginkgo.It("Verify welcome page of NoIstio Spring Boot FrontEnd is working.", func() {
		gomega.Eventually(func() bool {
			ingress := pkg.Ingress()
			pkg.Log(pkg.Info, fmt.Sprintf("Ingress: %s", ingress))
			url := fmt.Sprintf("http://%s/", ingress)
			status, content := pkg.GetWebPageWithCABundle(url, noIstioHostHeaderValue)
			return gomega.Expect(status).To(gomega.Equal(200)) &&
				//	gomega.Expect(content).To(gomega.ContainSubstring("Greetings from Verrazzano Enterprise Container Platform"))
				// Because I am using the old Springbbot Container for the FrontEnd
				gomega.Expect(content).To(gomega.ContainSubstring("Greetings from Verrazzano Enterprise Container Platform"))
		}, shortWaitTimeout, shortPollingInterval).Should(gomega.BeTrue())
	})


	// Verify Frontend can call Backend in foo
	// GIVEN authorization test app is deployed
	// WHEN the component and appconfig with ingress trait are created
	// THEN the frontend should be able to successfully call the backend and get a 200 on that invocation
	// the http code is returned in the response body and captured in content
	ginkgo.It("Verify Foo Frontend can call Foo Backend.", func() {
		gomega.Eventually(func() bool {
			ingress := pkg.Ingress()
			pkg.Log(pkg.Info, fmt.Sprintf("Ingress: %s", ingress))
			url := fmt.Sprintf("http://%s/externalCall?inurl=http://springboot-backend-workload.foo:8080/", ingress)
			status, content := pkg.GetWebPageWithCABundle(url, fooHostHeaderValue)
			pkg.Log(pkg.Info, fmt.Sprintf("Frontend Http return code: %d, Backend Http return code : %s", status, content))
			return gomega.Expect(status).To(gomega.Equal(200)) &&
				   gomega.Expect(content).To(gomega.Equal("200"))
		}, shortWaitTimeout, shortPollingInterval).Should(gomega.BeTrue())
	})

	// Verify Frontend can call Backend in bar
	// GIVEN authorization test app is deployed
	// WHEN the component and appconfig with ingress trait are created
	// THEN the frontend should be able to successfully call the backend and get a 200 on that invocation
	// the http code is returned in the response body and captured in content
	ginkgo.It("Verify Bar Frontend can call Bar Backend.", func() {
		gomega.Eventually(func() bool {
			ingress := pkg.Ingress()
			pkg.Log(pkg.Info, fmt.Sprintf("Ingress: %s", ingress))
			url := fmt.Sprintf("http://%s/externalCall?inurl=http://springboot-backend-workload.bar:8080/", ingress)
			status, content := pkg.GetWebPageWithCABundle(url, barHostHeaderValue)
			pkg.Log(pkg.Info, fmt.Sprintf("Frontend Http return code: %d, Backend Http return code : %s", status, content))
			return gomega.Expect(status).To(gomega.Equal(200)) &&
				   gomega.Expect(content).To(gomega.Equal("200"))
		}, shortWaitTimeout, shortPollingInterval).Should(gomega.BeTrue())
	})


	// Verify Foo Frontend can't call bar Backend
	// GIVEN authorization test app is deployed
	// WHEN the component and appconfig with ingress trait are created
	// THEN the frontend should be able to successfully call the backend and get a 200 on that invocation
	// the http code is returned in the response body and captured in content
	ginkgo.It("Verify Foo Frontend canNOT call Bar Backend.", func() {
		gomega.Eventually(func() bool {
			ingress := pkg.Ingress()
			pkg.Log(pkg.Info, fmt.Sprintf("Ingress: %s", ingress))
			url := fmt.Sprintf("http://%s/externalCall?inurl=http://springboot-backend-workload.bar:8080/", ingress)
			status, content := pkg.GetWebPageWithCABundle(url, fooHostHeaderValue)
			pkg.Log(pkg.Info, fmt.Sprintf("Frontend Http return code: %d, Backend Http return code : %s", status, content))
			return gomega.Expect(status).To(gomega.Equal(200)) &&
				   gomega.Expect(content).To(gomega.Equal("403"))
		}, shortWaitTimeout, shortPollingInterval).Should(gomega.BeTrue())
	})


	// Verify Bar Frontend can't call Foo Backend
	// GIVEN authorization test app is deployed
	// WHEN the component and appconfig with ingress trait are created
	// THEN the frontend should be able to successfully call the backend and get a 200 on that invocation
	// the http code is returned in the response body and captured in content
	ginkgo.It("Verify Bar Frontend canNOT call Foo Backend.", func() {
		gomega.Eventually(func() bool {
			ingress := pkg.Ingress()
			pkg.Log(pkg.Info, fmt.Sprintf("Ingress: %s", ingress))
			url := fmt.Sprintf("http://%s/externalCall?inurl=http://springboot-backend-workload.foo:8080/", ingress)
			status, content := pkg.GetWebPageWithCABundle(url, barHostHeaderValue)
			pkg.Log(pkg.Info, fmt.Sprintf("Frontend Http return code: %d, Backend Http return code : %s", status, content))
			return gomega.Expect(status).To(gomega.Equal(200)) &&
				   gomega.Expect(content).To(gomega.Equal("403"))
		}, shortWaitTimeout, shortPollingInterval).Should(gomega.BeTrue())
	})

	// Verify Bar Frontend can call NoIstio Backend
	// GIVEN authorization test app is deployed
	// WHEN the component and appconfig with ingress trait are created
	// THEN the frontend should be able to successfully call the backend and get a 200 on that invocation
	// the http code is returned in the response body and captured in content
	ginkgo.It("Verify Bar Frontend can call NoIstio Backend.", func() {
		gomega.Eventually(func() bool {
			ingress := pkg.Ingress()
			pkg.Log(pkg.Info, fmt.Sprintf("Ingress: %s", ingress))
			url := fmt.Sprintf("http://%s/externalCall?inurl=http://springboot-backend-workload.noistio:8080/", ingress)
			status, content := pkg.GetWebPageWithCABundle(url, barHostHeaderValue)
			pkg.Log(pkg.Info, fmt.Sprintf("Frontend Http return code: %d, Backend Http return code : %s", status, content))
			return gomega.Expect(status).To(gomega.Equal(200)) &&
				gomega.Expect(content).To(gomega.Equal("200"))
		}, shortWaitTimeout, shortPollingInterval).Should(gomega.BeTrue())
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
			ingress := pkg.Ingress()
			pkg.Log(pkg.Info, fmt.Sprintf("Ingress: %s", ingress))
			url := fmt.Sprintf("http://%s/externalCall?inurl=http://springboot-backend-workload.bar:8080/", ingress)

			client := &http.Client{
			}
			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				pkg.Log(pkg.Error, fmt.Sprintf("Failed to create http request: %v", err))
			}
			req.Host = noIstioHostHeaderValue

			q := req.URL.Query()
			q.Add("inurl", "http://springboot-backend-workload.bar:8080/")
			req.URL.RawQuery = q.Encode()

			resp, err := client.Do(req)
			if err != nil {
				pkg.Log(pkg.Error, fmt.Sprintf("Failed to do http request: %v", err))
			}
			return gomega.Expect(resp.StatusCode).To(gomega.Equal(500))
		}, shortWaitTimeout, shortPollingInterval).Should(gomega.BeTrue())
	})

})
