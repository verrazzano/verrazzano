// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vmi_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/vmi"
	corev1 "k8s.io/api/core/v1"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const verrazzanoNamespace string = "verrazzano-system"

var t = framework.NewTestFramework("vmi")

func vmiIngressURLs() (map[string]string, error) {
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		return nil, err
	}
	ingressList, err := clientset.NetworkingV1().Ingresses(verrazzanoNamespace).List(context.TODO(), v1.ListOptions{})
	if err != nil {
		return nil, err
	}

	ingressURLs := make(map[string]string)

	for _, ingress := range ingressList.Items {
		var ingressRules = ingress.Spec.Rules
		if len(ingressRules) != 1 {
			return nil, fmt.Errorf("expected ingress %s in namespace %s to have 1 ingress rule, but had %v",
				ingress.Name, ingress.Namespace, ingressRules)
		}
		ingressURLs[ingress.Name] = fmt.Sprintf("https://%s/", ingressRules[0].Host)
	}
	return ingressURLs, nil
}

func verrazzanoMonitoringInstanceCRD() (*apiextv1.CustomResourceDefinition, error) {
	client, err := pkg.APIExtensionsClientSet()
	if err != nil {
		return nil, err
	}
	crd, err := client.CustomResourceDefinitions().Get(context.TODO(), "verrazzanomonitoringinstances.verrazzano.io", v1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return crd, nil
}

func verrazzanoInstallerCRD() (*apiextv1.CustomResourceDefinition, error) {
	client, err := pkg.APIExtensionsClientSet()
	if err != nil {
		return nil, err
	}
	crd, err := client.CustomResourceDefinitions().Get(context.TODO(), "verrazzanos.install.verrazzano.io", v1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return crd, nil
}

var (
	httpClient             *retryablehttp.Client
	creds                  *pkg.UsernamePassword
	vmiCRD                 *apiextv1.CustomResourceDefinition
	vzCRD                  *apiextv1.CustomResourceDefinition
	ingressURLs            map[string]string
	volumeClaims           map[string]*corev1.PersistentVolumeClaim
	elastic                *vmi.Elastic
	waitTimeout            = 10 * time.Minute
	pollingInterval        = 5 * time.Second
	elasticWaitTimeout     = 2 * time.Minute
	elasticPollingInterval = 5 * time.Second
)

var _ = t.BeforeSuite(func() {
	var err error

	httpClient, err = pkg.GetVerrazzanoRetryableHTTPClient()
	Expect(err).ToNot(HaveOccurred())

	Eventually(func() (*apiextv1.CustomResourceDefinition, error) {
		vzCRD, err = verrazzanoInstallerCRD()
		return vzCRD, err
	}, waitTimeout, pollingInterval).ShouldNot(BeNil())

	Eventually(func() (map[string]string, error) {
		ingressURLs, err = vmiIngressURLs()
		return ingressURLs, err
	}, waitTimeout, pollingInterval).ShouldNot(BeEmpty())

	Eventually(func() (map[string]*corev1.PersistentVolumeClaim, error) {
		volumeClaims, err = pkg.GetPersistentVolumes(verrazzanoNamespace)
		return volumeClaims, err
	}, waitTimeout, pollingInterval).ShouldNot(BeNil())

	Eventually(func() (*apiextv1.CustomResourceDefinition, error) {
		vmiCRD, err = verrazzanoMonitoringInstanceCRD()
		return vmiCRD, err
	}, waitTimeout, pollingInterval).ShouldNot(BeNil())

	Eventually(func() (*pkg.UsernamePassword, error) {
		creds, err = pkg.GetSystemVMICredentials()
		return creds, err
	}, waitTimeout, pollingInterval).ShouldNot(BeNil())

	elastic = vmi.GetElastic("system")
})

var _ = t.AfterEach(func() {})

var _ = t.Describe("VMI", Label("f:infra-lcm"), func() {

	isManagedClusterProfile := pkg.IsManagedClusterProfile()
	if isManagedClusterProfile {
		t.It("Elasticsearch should NOT be present", func() {
			// Verify ES not present
			Eventually(func() (bool, error) {
				return pkg.PodsNotRunning(verrazzanoNamespace, []string{"vmi-system-es"})
			}, waitTimeout, pollingInterval).Should(BeTrue())
			Expect(elasticTLSSecret()).To(BeTrue())
			Expect(elastic.CheckIngress()).To(BeFalse())
			Expect(ingressURLs).NotTo(HaveKey("vmi-system-es-ingest"), fmt.Sprintf("Ingress %s not found", "vmi-system-grafana"))

			// Verify Kibana not present
			Eventually(func() (bool, error) {
				return pkg.PodsNotRunning(verrazzanoNamespace, []string{"vmi-system-kibana"})
			}, waitTimeout, pollingInterval).Should(BeTrue())
			Expect(ingressURLs).NotTo(HaveKey("vmi-system-kibana"), fmt.Sprintf("Ingress %s not found", "vmi-system-grafana"))

			// Verify Grafana not present
			Eventually(func() (bool, error) {
				return pkg.PodsNotRunning(verrazzanoNamespace, []string{"vmi-system-grafana"})
			}, waitTimeout, pollingInterval).Should(BeTrue())
			Expect(ingressURLs).NotTo(HaveKey("vmi-system-grafana"), fmt.Sprintf("Ingress %s not found", "vmi-system-grafana"))
		})
	} else {
		t.It("Elasticsearch endpoint should be accessible", Label("f:mesh.ingress"), func() {
			elasticPodsRunning := func() bool {
				result, err := pkg.PodsRunning(verrazzanoNamespace, []string{"vmi-system-es-master"})
				if err != nil {
					AbortSuite(fmt.Sprintf("Pod %v is not running in the namespace: %v, error: %v", "vmi-system-es-master", verrazzanoNamespace, err))
				}
				return result
			}
			Eventually(elasticPodsRunning, waitTimeout, pollingInterval).Should(BeTrue(), "pods did not all show up")
			Eventually(elasticTLSSecret, elasticWaitTimeout, elasticPollingInterval).Should(BeTrue(), "tls-secret did not show up")
			// Eventually(elasticCertificate, elasticWaitTimeout, elasticPollingInterval).Should(BeTrue(), "certificate did not show up")
			Eventually(elasticIngress, elasticWaitTimeout, elasticPollingInterval).Should(BeTrue(), "ingress did not show up")
			Expect(ingressURLs).To(HaveKey("vmi-system-es-ingest"), "Ingress vmi-system-es-ingest not found")
			assertOidcIngressByName("vmi-system-es-ingest")
			Eventually(elasticConnected, elasticWaitTimeout, elasticPollingInterval).Should(BeTrue(), "never connected")
			Eventually(elasticIndicesCreated, elasticWaitTimeout, elasticPollingInterval).Should(BeTrue(), "indices never created")
		})

		t.It("Elasticsearch verrazzano-system Index should be accessible", Label("f:observability.logging.es"),
			func() {
				indexName := "verrazzano-namespace-verrazzano-system"
				pkg.Concurrently(
					func() {
						Eventually(func() bool {
							return pkg.LogRecordFound(indexName,
								time.Now().Add(-24*time.Hour), map[string]string{
									"kubernetes.container_name": "verrazzano-monitoring-operator",
									"caller":                    "controller",
									"cluster_name":              constants.MCLocalCluster,
								})
						}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find a verrazzano-monitoring-operator log record")
					},
					func() {
						Eventually(func() bool {
							return pkg.LogRecordFound(indexName,
								time.Now().Add(-24*time.Hour), map[string]string{
									"kubernetes.container_name": "verrazzano-application-operator",
									"caller":                    "controller",
									"cluster_name":              constants.MCLocalCluster,
								})
						}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find a verrazzano-application-operator log record")
					},
				)
			})

		t.It("Elasticsearch health should be green", func() {
			Eventually(elasticHealth, elasticWaitTimeout, elasticPollingInterval).Should(BeTrue(), "cluster health status not green")
			Eventually(elasticIndicesHealth, elasticWaitTimeout, elasticPollingInterval).Should(BeTrue(), "indices health status not green")
		})

		t.It("Elasticsearch systemd journal Index should be accessible", Label("f:observability.logging.es"),
			func() {
				Eventually(func() bool {
					return pkg.FindAnyLog("verrazzano-systemd-journal",
						[]pkg.Match{
							{Key: "tag", Value: "systemd"},
							{Key: "TRANSPORT", Value: "journal"},
							{Key: "cluster_name", Value: constants.MCLocalCluster}},
						[]pkg.Match{})
				}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find a systemd log record")
			})

		t.It("Kibana endpoint should be accessible", Label("f:mesh.ingress",
			"f:observability.logging.kibana"), func() {
			kibanaPodsRunning := func() bool {
				result, err := pkg.PodsRunning(verrazzanoNamespace, []string{"vmi-system-kibana"})
				if err != nil {
					AbortSuite(fmt.Sprintf("Pod %v is not running in the namespace: %v, error: %v", "vmi-system-kibana", verrazzanoNamespace, err))
				}
				return result
			}
			Eventually(kibanaPodsRunning, waitTimeout, pollingInterval).Should(BeTrue(), "kibana pods did not all show up")
			Expect(ingressURLs).To(HaveKey("vmi-system-kibana"), "Ingress vmi-system-kibana not found")
			assertOidcIngressByName("vmi-system-kibana")
		})

		t.It("Prometheus endpoint should be accessible", Label("f:mesh.ingress",
			"f:observability.monitoring.prom"), func() {
			assertOidcIngressByName("vmi-system-prometheus")
		})

		t.It("Grafana endpoint should be accessible", Label("f:mesh.ingress",
			"f:observability.monitoring.graf"), func() {
			Expect(ingressURLs).To(HaveKey("vmi-system-grafana"), "Ingress vmi-system-grafana not found")
			assertOidcIngressByName("vmi-system-grafana")
		})

		t.It("Default dashboard should be installed in System Grafana for shared VMI",
			Label("f:observability.monitoring.graf"), func() {
				pkg.Concurrently(
					func() { assertDashboard("Host%20Metrics") },
					func() { assertDashboard("WebLogic%20Server%20Dashboard") },
					func() { assertDashboard("Coherence%20Elastic%20Data%20Summary%20Dashboard") },
					func() { assertDashboard("Coherence%20Persistence%20Summary%20Dashboard") },
					func() { assertDashboard("Coherence%20Cache%20Details%20Dashboard") },
					func() { assertDashboard("Coherence%20Members%20Summary%20Dashboard") },
					func() { assertDashboard("Coherence%20Kubernetes%20Summary%20Dashboard") },
					func() { assertDashboard("Coherence%20Dashboard%20Main") },
					func() { assertDashboard("Coherence%20Caches%20Summary%20Dashboard") },
					func() { assertDashboard("Coherence%20Service%20Details%20Dashboard") },
					func() { assertDashboard("Coherence%20Proxy%20Servers%20Summary%20Dashboard") },
					func() { assertDashboard("Coherence%20Federation%20Details%20Dashboard") },
					func() { assertDashboard("Coherence%20Federation%20Summary%20Dashboard") },
					func() { assertDashboard("Coherence%20Services%20Summary%20Dashboard") },
					func() { assertDashboard("Coherence%20HTTP%20Servers%20Summary%20Dashboard") },
					func() { assertDashboard("Coherence%20Proxy%20Server%20Detail%20Dashboard") },
					func() { assertDashboard("Coherence%20Alerts%20Dashboard") },
					func() { assertDashboard("Coherence%20Member%20Details%20Dashboard") },
					func() { assertDashboard("Coherence%20Machines%20Summary%20Dashboard") },
				)
			})

		t.It("Grafana should have the verrazzano user with admin privileges", func() {
			Eventually(assertAdminRole, waitTimeout, pollingInterval).Should(BeTrue())

		})
	}

	t.It("Verify the instance info endpoint URLs", Label("f:mesh.ingress"), func() {
		if !isManagedClusterProfile {
			assertInstanceInfoURLs()
		}
	})

	size := "50Gi"
	// If there are persistence overrides at the global level, that will cause persistent
	// volumes to be created for the VMI components that use them (ES, Kibana, and Prometheus)
	// At some point we may need to check for individual VMI overrides.
	kubeconfigPath, _ := k8sutil.GetKubeConfigLocation()
	override, _ := pkg.GetEffectiveVMIPersistenceOverride(kubeconfigPath)
	if override != nil {
		size = override.Spec.Resources.Requests.Storage().String()
	}

	if pkg.IsDevProfile() {
		t.It("Check persistent volumes for dev profile", func() {
			expectedVolumes := 0
			if override != nil {
				expectedVolumes = 3
			}
			Expect(len(volumeClaims)).To(Equal(expectedVolumes))
			if expectedVolumes > 0 {
				assertPersistentVolume("vmi-system-prometheus", size)
				assertPersistentVolume("vmi-system-grafana", size)
				assertPersistentVolume("elasticsearch-master-vmi-system-es-master-0", size)
			}
		})
	} else if isManagedClusterProfile {
		t.It("Check persistent volumes for managed cluster profile", func() {
			Expect(len(volumeClaims)).To(Equal(1))
			assertPersistentVolume("vmi-system-prometheus", size)
		})
	} else if pkg.IsProdProfile() {
		t.It("Check persistent volumes for prod cluster profile", func() {
			Expect(len(volumeClaims)).To(Equal(8))
			assertPersistentVolume("vmi-system-prometheus", size)
			assertPersistentVolume("vmi-system-grafana", size)
			assertPersistentVolume("elasticsearch-master-vmi-system-es-master-0", size)
			assertPersistentVolume("elasticsearch-master-vmi-system-es-master-1", size)
			assertPersistentVolume("elasticsearch-master-vmi-system-es-master-2", size)
			assertPersistentVolume("vmi-system-es-data", size)
			assertPersistentVolume("vmi-system-es-data-1", size)
			assertPersistentVolume("vmi-system-es-data-2", size)
		})
	}
})

func assertPersistentVolume(key string, size string) {
	Expect(volumeClaims).To(HaveKey(key))
	pvc := volumeClaims[key]
	Expect(pvc.Spec.Resources.Requests.Storage().String()).To(Equal(size))
}

func assertOidcIngressByName(key string) {
	Expect(ingressURLs).To(HaveKey(key), fmt.Sprintf("Ingress %s not found", key))
	url := ingressURLs[key]
	assertOidcIngress(url)
}

func assertOidcIngress(url string) {
	unauthHTTPClient, err := pkg.GetVerrazzanoRetryableHTTPClient()
	Expect(err).ToNot(HaveOccurred())
	pkg.Concurrently(
		func() {
			Eventually(func() bool {
				return pkg.AssertOauthURLAccessibleAndUnauthorized(unauthHTTPClient, url)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		},
		func() {
			Eventually(func() bool {
				return pkg.AssertURLAccessibleAndAuthorized(httpClient, url, creds)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		},
		func() {
			Eventually(func() bool {
				return pkg.AssertBearerAuthorized(httpClient, url)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		},
	)
}

func elasticIndicesCreated() bool {
	b, _ := ContainElements(".kibana_1").Match(elastic.ListIndices())
	return b
}

func elasticConnected() bool {
	return elastic.Connect()
}

func elasticHealth() bool {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		t.Logs.Errorf("Failed to get default kubeconfig path: %s", err.Error())
		return false
	}
	return elastic.CheckHealth(kubeconfigPath)
}

func elasticIndicesHealth() bool {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		t.Logs.Errorf("Failed to get default kubeconfig path: %s", err.Error())
		return false
	}
	return elastic.CheckIndicesHealth(kubeconfigPath)
}

func elasticTLSSecret() bool {
	return elastic.CheckTLSSecret()
}

func elasticIngress() bool {
	return elastic.CheckIngress()
}

func assertDashboard(url string) {
	searchURL := fmt.Sprintf("%sapi/search?query=%s", ingressURLs["vmi-system-grafana"], url)
	fmt.Println("Grafana URL in browseGrafanaDashboard ", searchURL)

	searchDashboard := func() bool {
		vmiHTTPClient, err := pkg.GetVerrazzanoRetryableHTTPClient()
		if err != nil {
			t.Logs.Errorf("Error getting HTTP client: %v", err)
			return false
		}
		vmiHTTPClient.HTTPClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}

		req, err := retryablehttp.NewRequest("GET", searchURL, nil)
		if err != nil {
			t.Logs.Errorf("Error creating HTTP request: %v", err)
			return false
		}
		req.SetBasicAuth(creds.Username, creds.Password)
		resp, err := vmiHTTPClient.Do(req)
		if err != nil {
			t.Logs.Errorf("Error making HTTP request: %v", err)
			return false
		}
		if resp.StatusCode != http.StatusOK {
			t.Logs.Errorf("Unexpected HTTP status code: %d", resp.StatusCode)
			return false
		}
		// assert that there is a single item in response
		defer resp.Body.Close()
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Logs.Errorf("Unable to read body from response: %v", err)
			return false
		}
		var response []map[string]interface{}
		if err := json.Unmarshal(bodyBytes, &response); err != nil {
			t.Logs.Errorf("Error unmarshaling response body: %v", err)
			return false
		}
		if len(response) != 1 {
			t.Logs.Errorf("Unexpected response length: %d", len(response))
			return false
		}
		return true
	}
	Eventually(searchDashboard, waitTimeout, pollingInterval).Should(BeTrue())
}

func assertAdminRole() bool {
	searchURL := fmt.Sprintf("%sapi/users", ingressURLs["vmi-system-grafana"])
	vmiHTTPClient, err := pkg.GetVerrazzanoRetryableHTTPClient()
	if err != nil {
		t.Logs.Errorf("Error getting HTTP client: %v", err)
		return false
	}
	vmiHTTPClient.HTTPClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	req, err := retryablehttp.NewRequest("GET", searchURL, nil)
	if err != nil {
		t.Logs.Errorf("Error creating HTTP request: %v", err)
		return false
	}
	req.SetBasicAuth(creds.Username, creds.Password)
	resp, err := vmiHTTPClient.Do(req)
	if err != nil {
		t.Logs.Errorf("Error making HTTP request: %v", err)
		return false
	}
	if resp.StatusCode != http.StatusOK {
		t.Logs.Errorf("Unexpected HTTP status code: %d", resp.StatusCode)
		return false
	}
	// assert that there is a single item in response
	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Logs.Errorf("Unable to read body from response: %v", err)
		return false
	}
	var response []map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		t.Logs.Errorf("Error unmarshaling response body: %v", err)
		return false
	}
	if len(response) != 1 {
		t.Logs.Errorf("Unexpected response length: %d", len(response))
		return false
	}
	return true
}

func assertInstanceInfoURLs() {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	Expect(err).To(BeNil())
	cr, err := pkg.GetVerrazzanoInstallResourceInCluster(kubeconfigPath)
	Expect(err).To(BeNil())
	instanceInfo := cr.Status.VerrazzanoInstance
	switch cr.Spec.Profile {
	case v1alpha1.ManagedCluster:
		Expect(instanceInfo.GrafanaURL).To(BeNil())
		Expect(instanceInfo.ElasticURL).To(BeNil())
		Expect(instanceInfo.KibanaURL).To(BeNil())
	default:
		Expect(instanceInfo.GrafanaURL).NotTo(BeNil())
		Expect(instanceInfo.ElasticURL).NotTo(BeNil())
		Expect(instanceInfo.KibanaURL).NotTo(BeNil())
		if instanceInfo.ElasticURL != nil {
			assertOidcIngress(*instanceInfo.ElasticURL)
		}
		if instanceInfo.KibanaURL != nil {
			assertOidcIngress(*instanceInfo.KibanaURL)
		}
		if instanceInfo.GrafanaURL != nil {
			assertOidcIngress(*instanceInfo.GrafanaURL)
		}
	}
	Expect(instanceInfo.PrometheusURL).NotTo(BeNil())
	if instanceInfo.PrometheusURL != nil {
		assertOidcIngress(*instanceInfo.PrometheusURL)
	}
}
