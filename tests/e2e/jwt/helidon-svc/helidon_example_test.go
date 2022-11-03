// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helidonsvc

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/verrazzano/verrazzano/pkg/k8s/resource"

	"github.com/hashicorp/go-retryablehttp"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework/metrics"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	longWaitTimeout          = 20 * time.Minute
	longPollingInterval      = 20 * time.Second
	shortPollingInterval     = 10 * time.Second
	shortWaitTimeout         = 5 * time.Minute
	imagePullWaitTimeout     = 40 * time.Minute
	imagePullPollingInterval = 30 * time.Second
	skipVerifications        = "Skip Verifications"
	nodeExporterJobName      = "node-exporter"
)

const (
	helidonComponentYaml = "testdata/jwt/helidon-svc/hello-helidon-svc-comps.yaml"
	helidonAppYaml       = "testdata/jwt/helidon-svc/hello-helidon-svc-app.yaml"
)

var (
	t                  = framework.NewTestFramework("helidon")
	generatedNamespace = pkg.GenerateNamespace("hello-helidon-svc")
	//yamlApplier              = k8sutil.YAMLApplier{}
	expectedPodsHelloHelidon = []string{"hello-helidon-svc-deployment"}
)

var _ = t.BeforeSuite(func() {
	if !skipDeploy {
		start := time.Now()
		deployHelloHelidonApplication(namespace, "", istioInjection)
		metrics.Emit(t.Metrics.With("deployment_elapsed_time", time.Since(start).Milliseconds()))
	}

	Eventually(func() bool {
		return pkg.ContainerImagePullWait(namespace, expectedPodsHelloHelidon)
	}, imagePullWaitTimeout, imagePullPollingInterval).Should(BeTrue())
	// Verify hello-helidon-deployment pod is running
	// GIVEN OAM hello-helidon app is deployed
	// WHEN the component and appconfig are created
	// THEN the expected pod must be running in the test namespace
	if !skipVerify {
		Eventually(helloHelidonPodsRunning, longWaitTimeout, longPollingInterval).Should(BeTrue())
	}
	beforeSuitePassed = true
})

var failed = false
var beforeSuitePassed = false

var _ = t.AfterEach(func() {
	failed = failed || CurrentSpecReport().Failed()
})

var _ = t.AfterSuite(func() {
	if failed || !beforeSuitePassed {
		pkg.ExecuteBugReport(namespace)
	}
	if !skipUndeploy {
		start := time.Now()
		undeployHelloHelidonApplication(namespace)
		metrics.Emit(t.Metrics.With("undeployment_elapsed_time", time.Since(start).Milliseconds()))
	}
})

var _ = t.Describe("Hello Helidon OAM App test", Label("f:app-lcm.oam",
	"f:app-lcm.helidon-workload"), func() {
	var host = ""
	var err error
	// Get the host from the Istio gateway resource.
	// GIVEN the Istio gateway for the hello-helidon namespace
	// WHEN GetHostnameFromGateway is called
	// THEN return the host name found in the gateway.
	t.BeforeEach(func() {
		Eventually(func() (string, error) {
			host, err = k8sutil.GetHostnameFromGateway(namespace, "")
			return host, err
		}, shortWaitTimeout, shortPollingInterval).Should(Not(BeEmpty()))
	})

	// Verify Hello Helidon app is working
	// GIVEN OAM hello-helidon app is deployed
	// WHEN the component and appconfig with ingress trait are created
	// THEN the application endpoint must be accessible
	t.Describe("for Ingress.", Label("f:mesh.ingress"), func() {
		t.It("Access /greet App Url w/o token and get RBAC denial", func() {
			if skipVerify {
				Skip(skipVerifications)
			}
			url := fmt.Sprintf("https://%s/greet", host)
			Eventually(func() bool {
				return appEndpointAccess(url, host, "", false)
			}, longWaitTimeout, longPollingInterval).Should(BeTrue())
		})

		t.It("Access /greet App Url with valid token", func() {
			if skipVerify {
				Skip(skipVerifications)
			}
			kc, err := pkg.NewKeycloakAdminRESTClient()
			Expect(err).To(BeNil())
			password := pkg.GetRequiredEnvVarOrFail("REALM_USER_PASSWORD")
			realmName := pkg.GetRequiredEnvVarOrFail("REALM_NAME")
			// check for realm
			_, err = kc.GetRealm(realmName)
			Expect(err).To(BeNil())
			var token string
			token, err = kc.GetToken(realmName, "testuser", password, "appsclient", t.Logs)
			Expect(err).To(BeNil())
			t.Logs.Debugf("Obtained token: %v", token)
			url := fmt.Sprintf("https://%s/greet", host)
			Eventually(func() bool {
				return appEndpointAccess(url, host, token, true)
			}, longWaitTimeout, longPollingInterval).Should(BeTrue())
		})
	})

	// Verify Prometheus scraped metrics
	// GIVEN OAM hello-helidon app is deployed
	// WHEN the component and appconfig without metrics-trait(using default) are created
	// THEN the application metrics must be accessible
	t.Describe("for Metrics.", Label("f:observability.monitoring.prom"), FlakeAttempts(5), func() {
		t.It("Retrieve Prometheus scraped metrics", func() {
			if skipVerify {
				Skip(skipVerifications)
			}
			pkg.Concurrently(
				func() {
					Eventually(appMetricsExists, longWaitTimeout, longPollingInterval).Should(BeTrue())
				},
				func() {
					Eventually(appComponentMetricsExists, longWaitTimeout, longPollingInterval).Should(BeTrue())
				},
				func() {
					Eventually(appConfigMetricsExists, longWaitTimeout, longPollingInterval).Should(BeTrue())
				},
				func() {
					Eventually(nodeExporterProcsRunning, longWaitTimeout, longPollingInterval).Should(BeTrue())
				},
				func() {
					Eventually(nodeExporterDiskIoNow, longWaitTimeout, longPollingInterval).Should(BeTrue())
				},
			)
		})
	})

	t.Context("Logging.", Label("f:observability.logging.es"), FlakeAttempts(5), func() {

		indexName, err := pkg.GetOpenSearchAppIndex(namespace)
		Expect(err).To(BeNil())
		// GIVEN an application with logging enabled
		// WHEN the Elasticsearch index is retrieved
		// THEN verify that it is found
		t.It("Verify Elasticsearch index exists", func() {
			if skipVerify {
				Skip(skipVerifications)
			}
			Eventually(func() bool {
				return pkg.LogIndexFound(indexName)
			}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find log index for hello helidon")
		})

		// GIVEN an application with logging enabled
		// WHEN the log records are retrieved from the Elasticsearch index
		// THEN verify that at least one recent log record is found
		t.It("Verify recent Elasticsearch log record exists", func() {
			if skipVerify {
				Skip(skipVerifications)
			}
			Eventually(func() bool {
				return pkg.LogRecordFound(indexName, time.Now().Add(-24*time.Hour), map[string]string{
					"kubernetes.labels.app_oam_dev\\/name": "hello-helidon-svc-application",
					"kubernetes.container_name":            "hello-helidon-container",
				})
			}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find a recent log record")
			Eventually(func() bool {
				return pkg.LogRecordFound(indexName, time.Now().Add(-24*time.Hour), map[string]string{
					"kubernetes.labels.app_oam_dev\\/component": "hello-helidon-deploy-component",
					"kubernetes.labels.app_oam_dev\\/name":      "hello-helidon-svc-application",
					"kubernetes.container_name":                 "hello-helidon-container",
				})
			}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find a recent log record")
		})
	})

})

// DeployHelloHelidonApplication deploys the Hello Helidon example application. It accepts an optional
// OCI Log ID that is added as an annotation on the namespace to test the OCI Logging service integration.
func deployHelloHelidonApplication(namespace string, ociLogID string, istioInjection string) {
	pkg.Log(pkg.Info, "Deploy Hello Helidon Application")
	pkg.Log(pkg.Info, fmt.Sprintf("Create namespace %s", namespace))
	Eventually(func() (*v1.Namespace, error) {
		nsLabels := map[string]string{
			"verrazzano-managed": "true",
			"istio-injection":    istioInjection}

		var annotations map[string]string
		if len(ociLogID) > 0 {
			annotations = make(map[string]string)
			annotations["verrazzano.io/oci-log-id"] = ociLogID
		}

		return pkg.CreateNamespaceWithAnnotations(namespace, nsLabels, annotations)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil(), fmt.Sprintf("Failed to create namespace %s", namespace))

	pkg.Log(pkg.Info, "Create Hello Helidon component resource")
	Eventually(func() error {
		file, err := pkg.FindTestDataFile(helidonComponentYaml)
		if err != nil {
			return err
		}
		return resource.CreateOrUpdateResourceFromFileInGeneratedNamespace(file, namespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred(), "Failed to create hello-helidon component resource")

	pkg.Log(pkg.Info, "Create Hello Helidon application resource")
	Eventually(func() error {
		file, err := pkg.FindTestDataFile(helidonAppYaml)
		if err != nil {
			return err
		}
		return resource.CreateOrUpdateResourceFromFileInGeneratedNamespace(file, namespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred(), "Failed to create hello-helidon application resource")
}

// undeployHelloHelidonApplication undeploys the Hello Helidon example application.
func undeployHelloHelidonApplication(namespace string) {
	pkg.Log(pkg.Info, "Undeploy Hello Helidon Application")
	if exists, _ := pkg.DoesNamespaceExist(namespace); exists {
		pkg.Log(pkg.Info, "Delete Hello Helidon application")
		Eventually(func() error {
			file, err := pkg.FindTestDataFile(helidonAppYaml)
			if err != nil {
				return err
			}
			return resource.DeleteResourceFromFileInGeneratedNamespace(file, namespace)
		}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred(), "Failed to create hello-helidon application resource")

		pkg.Log(pkg.Info, "Delete Hello Helidon components")
		Eventually(func() error {
			file, err := pkg.FindTestDataFile(helidonComponentYaml)
			if err != nil {
				return err
			}
			return resource.DeleteResourceFromFileInGeneratedNamespace(file, namespace)
		}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred(), "Failed to create hello-helidon component resource")

		pkg.Log(pkg.Info, "Wait for application pods to terminate")
		Eventually(func() bool {
			podsTerminated, _ := pkg.PodsNotRunning(namespace, expectedPodsHelloHelidon)
			return podsTerminated
		}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())

		pkg.Log(pkg.Info, fmt.Sprintf("Delete namespace %s", namespace))
		Eventually(func() error {
			return pkg.DeleteNamespace(namespace)
		}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred(), fmt.Sprintf("Failed to deleted namespace %s", namespace))

		pkg.Log(pkg.Info, "Wait for namespace finalizer to be removed")
		Eventually(func() bool {
			return pkg.CheckNamespaceFinalizerRemoved(namespace)
		}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())

		pkg.Log(pkg.Info, "Wait for namespace to be deleted")
		Eventually(func() bool {
			_, err := pkg.GetNamespace(namespace)
			return err != nil && errors.IsNotFound(err)
		}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())
	}
}

func helloHelidonPodsRunning() bool {
	result, err := pkg.PodsRunning(namespace, expectedPodsHelloHelidon)
	if err != nil {
		AbortSuite(fmt.Sprintf("One or more pods are not running in the namespace: %v, error: %v", namespace, err))
	}
	return result
}

func appEndpointAccess(url string, hostname string, token string, requestShouldSucceed bool) bool {
	req, err := retryablehttp.NewRequest("GET", url, nil)
	if err != nil {
		t.Logs.Errorf("Unexpected error=%v", err)
		return false
	}

	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		t.Logs.Errorf("Unexpected error=%v", err)
		return false
	}

	httpClient, err := pkg.GetVerrazzanoHTTPClient(kubeconfigPath)
	if err != nil {
		t.Logs.Errorf("Unexpected error=%v", err)
		return false
	}

	if len(token) > 0 {
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %v", token))
	}

	req.Host = hostname
	resp, err := httpClient.Do(req)
	if err != nil {
		t.Logs.Errorf("Unexpected error=%v", err)
		return false
	}
	bodyRaw, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Logs.Errorf("Unexpected error=%v", err)
		return false
	}
	if requestShouldSucceed {
		if resp.StatusCode != http.StatusOK {
			t.Logs.Errorf("Unexpected status code=%v", resp.StatusCode)
			return false
		}
		// HTTP Server headers should never be returned.
		for headerName, headerValues := range resp.Header {
			if strings.EqualFold(headerName, "Server") {
				t.Logs.Errorf("Unexpected Server header=%v", headerValues)
				return false
			}
		}
		bodyStr := string(bodyRaw)
		if !strings.Contains(bodyStr, "Hello World") {
			t.Logs.Errorf("Unexpected response body=%v", bodyStr)
			return false
		}
	} else {
		if resp.StatusCode == http.StatusOK {
			t.Logs.Errorf("Unexpected status code=%v", resp.StatusCode)
			return false
		}
	}
	return true
}

func appMetricsExists() bool {
	return pkg.MetricsExist("base_jvm_uptime_seconds", "app", "hello-helidon-svc-application")
}

func appComponentMetricsExists() bool {
	return pkg.MetricsExist("vendor_requests_count_total", "app_oam_dev_name", "hello-helidon-svc-application")
}

func appConfigMetricsExists() bool {
	return pkg.MetricsExist("vendor_requests_count_total", "app_oam_dev_component", "hello-helidon-deploy-component")
}

func nodeExporterProcsRunning() bool {
	return pkg.MetricsExist("node_procs_running", "job", nodeExporterJobName)
}

func nodeExporterDiskIoNow() bool {
	return pkg.MetricsExist("node_disk_io_now", "job", nodeExporterJobName)
}
