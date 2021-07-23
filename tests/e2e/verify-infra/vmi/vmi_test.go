// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vmi_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/vmi"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const verrazzanoNamespace string = "verrazzano-system"

func vmiIngressURLs() (map[string]string, error) {
	clientset, err := pkg.GetKubernetesClientset()
	if err != nil {
		return nil, err
	}
	ingressList, err := clientset.ExtensionsV1beta1().Ingresses(verrazzanoNamespace).List(context.TODO(), v1.ListOptions{})
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

func verrazzanoMonitoringInstanceCRD() (*apiextensionsv1beta1.CustomResourceDefinition, error) {
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

func verrazzanoInstallerCRD() (*apiextensionsv1beta1.CustomResourceDefinition, error) {
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
	creds                  *pkg.UsernamePassword
	vmiCRD                 *apiextensionsv1beta1.CustomResourceDefinition
	vzCRD                  *apiextensionsv1beta1.CustomResourceDefinition
	ingressURLs            map[string]string
	volumeClaims           map[string]*corev1.PersistentVolumeClaim
	elastic                *vmi.Elastic
	waitTimeout            = 10 * time.Minute
	pollingInterval        = 5 * time.Second
	elasticWaitTimeout     = 2 * time.Minute
	elasticPollingInterval = 5 * time.Second
)

var savedProfile v1alpha1.ProfileType

var _ = BeforeSuite(func() {
	var err error

	Eventually(func() (*v1alpha1.ProfileType, error) {
		var profile *v1alpha1.ProfileType
		profile, err = pkg.GetVerrazzanoProfile()
		if profile != nil {
			savedProfile = *profile
		}
		return profile, err
	}, waitTimeout, pollingInterval).ShouldNot(BeNil())

	Eventually(func() (*apiextensionsv1beta1.CustomResourceDefinition, error) {
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

	Eventually(func() (*apiextensionsv1beta1.CustomResourceDefinition, error) {
		vmiCRD, err = verrazzanoMonitoringInstanceCRD()
		return vmiCRD, err
	}, waitTimeout, pollingInterval).ShouldNot(BeNil())

	Eventually(func() (*pkg.UsernamePassword, error) {
		creds, err = pkg.GetSystemVMICredentials()
		return creds, err
	}, waitTimeout, pollingInterval).ShouldNot(BeNil())

	elastic = vmi.GetElastic("system")
})

var _ = Describe("VMI", func() {

	if savedProfile == v1alpha1.ManagedCluster {
		It("Elasticsearch should NOT be present", func() {
			// Verify ES not present
			Expect(pkg.PodsNotRunning(verrazzanoNamespace, []string{"vmi-system-es"})).To(BeTrue())
			Expect(elasticTLSSecret()).To(BeTrue())
			Expect(elastic.CheckIngress()).To(BeFalse())
			Expect(ingressURLs).NotTo(HaveKey("vmi-system-es-ingest"), fmt.Sprintf("Ingress %s not found", "vmi-system-grafana"))

			// Verify Kibana not present
			Expect(pkg.PodsNotRunning(verrazzanoNamespace, []string{"vmi-system-kibana"})).To(BeTrue())
			Expect(ingressURLs).NotTo(HaveKey("vmi-system-kibana"), fmt.Sprintf("Ingress %s not found", "vmi-system-grafana"))

			// Verify Grafana not present
			Expect(pkg.PodsNotRunning(verrazzanoNamespace, []string{"vmi-system-grafana"})).To(BeTrue())
			Expect(ingressURLs).NotTo(HaveKey("vmi-system-grafana"), fmt.Sprintf("Ingress %s not found", "vmi-system-grafana"))
		})
	} else {
		It("Elasticsearch endpoint should be accessible", func() {
			elasticPodsRunning := func() bool {
				return pkg.PodsRunning(verrazzanoNamespace, []string{"vmi-system-es-master"})
			}
			Eventually(elasticPodsRunning, waitTimeout, pollingInterval).Should(BeTrue(), "pods did not all show up")
			Eventually(elasticTLSSecret, elasticWaitTimeout, elasticPollingInterval).Should(BeTrue(), "tls-secret did not show up")
			//Eventually(elasticCertificate, elasticWaitTimeout, elasticPollingInterval).Should(BeTrue(), "certificate did not show up")
			Eventually(elasticIngress, elasticWaitTimeout, elasticPollingInterval).Should(BeTrue(), "ingress did not show up")
			Expect(ingressURLs).To(HaveKey("vmi-system-es-ingest"), "Ingress vmi-system-es-ingest not found")
			assertOidcIngressByName("vmi-system-es-ingest")
			Eventually(elasticConnected, elasticWaitTimeout, elasticPollingInterval).Should(BeTrue(), "never connected")
			Eventually(elasticIndicesCreated, elasticWaitTimeout, elasticPollingInterval).Should(BeTrue(), "indices never created")
		})

		It("Elasticsearch verrazzano-system Index should be accessible", func() {
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

		It("Elasticsearch systemd journal Index should be accessible", func() {
			Eventually(func() bool {
				return pkg.FindAnyLog("verrazzano-systemd-journal",
					[]pkg.Match{
						{Key: "tag", Value: "systemd"},
						{Key: "TRANSPORT", Value: "journal"},
						{Key: "cluster_name", Value: constants.MCLocalCluster}},
					[]pkg.Match{})
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find a systemd log record")
		})

		It("Kibana endpoint should be accessible", func() {
			kibanaPodsRunning := func() bool {
				return pkg.PodsRunning(verrazzanoNamespace, []string{"vmi-system-kibana"})
			}
			Eventually(kibanaPodsRunning, waitTimeout, pollingInterval).Should(BeTrue(), "kibana pods did not all show up")
			Expect(ingressURLs).To(HaveKey("vmi-system-kibana"), "Ingress vmi-system-kibana not found")
			assertOidcIngressByName("vmi-system-kibana")
		})

		It("Prometheus endpoint should be accessible", func() {
			assertOidcIngressByName("vmi-system-prometheus")
		})

		It("Grafana endpoint should be accessible", func() {
			Expect(ingressURLs).To(HaveKey("vmi-system-grafana"), "Ingress vmi-system-grafana not found")
			assertOidcIngressByName("vmi-system-grafana")
		})

		It("Default dashboard should be installed in System Grafana for shared VMI", func() {
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
	}

	It("Verify the instance info endpoint URLs", func() {
		if savedProfile != v1alpha1.ManagedCluster {
			assertInstanceInfoURLs()
		}
	})

	size := "50Gi"
	if savedProfile == v1alpha1.Dev {
		It("Check persistent volumes for dev profile", func() {
			Expect(len(volumeClaims)).To(Equal(0))
		})
	} else if savedProfile == v1alpha1.ManagedCluster {
		It("Check persistent volumes for managed cluster profile", func() {
			Expect(len(volumeClaims)).To(Equal(1))
			assertPersistentVolume("vmi-system-prometheus", size)
		})
	} else if savedProfile == v1alpha1.Prod {
		It("Check persistent volumes for prod cluster profile", func() {
			Expect(len(volumeClaims)).To(Equal(7))
			assertPersistentVolume("vmi-system-prometheus", size)
			assertPersistentVolume("vmi-system-grafana", size)
			assertPersistentVolume("elasticsearch-master-vmi-system-es-master-0", size)
			assertPersistentVolume("elasticsearch-master-vmi-system-es-master-1", size)
			assertPersistentVolume("elasticsearch-master-vmi-system-es-master-2", size)
			assertPersistentVolume("vmi-system-es-data", size)
			assertPersistentVolume("vmi-system-es-data-1", size)
		})
	}
})

func assertPersistentVolume(key string, size string) {
	Expect(volumeClaims).To(HaveKey(key))
	pvc := volumeClaims[key]
	Expect(pvc.Spec.Resources.Requests.Storage().String()).To(Equal(size))
}

func assertURLAccessibleAndAuthorized(url string) bool {
	vmiHTTPClient, err := pkg.GetSystemVmiHTTPClient()
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Error getting HTTP client: %v", err))
		return false
	}
	return pkg.AssertURLAccessibleAndAuthorized(vmiHTTPClient, url, creds)
}

func assertBearerAuthorized(url string) bool {
	vmiHTTPClient, err := pkg.GetSystemVmiHTTPClient()
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Error getting HTTP client: %v", err))
		return false
	}
	api, err := pkg.GetAPIEndpoint(pkg.GetKubeConfigPathFromEnv())
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Error getting API endpoint: %v", err))
		return false
	}
	req, _ := retryablehttp.NewRequest("GET", url, nil)
	if api.AccessToken != "" {
		bearer := fmt.Sprintf("Bearer %v", api.AccessToken)
		req.Header.Set("Authorization", bearer)
	}
	resp, err := vmiHTTPClient.Do(req)
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed making request: %v", err))
		return false
	}
	resp.Body.Close()
	pkg.Log(pkg.Info, fmt.Sprintf("assertBearerAuthorized %v Response:%v Error:%v", url, resp.StatusCode, err))
	return resp.StatusCode == http.StatusOK
}

func assertOauthURLAccessibleAndUnauthorized(url string) bool {
	vmiHTTPClient, err := pkg.GetSystemVmiHTTPClient()
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Error getting HTTP client: %v", err))
		return false
	}
	vmiHTTPClient.HTTPClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		pkg.Log(pkg.Info, fmt.Sprintf("oidcUnauthorized req: %v \nvia: %v\n", req, via))
		return http.ErrUseLastResponse
	}
	resp, err := vmiHTTPClient.Get(url)
	if err != nil || resp == nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed making request: %v", err))
		return false
	}
	location, err := resp.Location()
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Error getting location from response: %v, error: %v", resp, err))
		return false
	}
	Expect(location).NotTo(BeNil())
	pkg.Log(pkg.Info, fmt.Sprintf("oidcUnauthorized %v StatusCode:%v host:%v", url, resp.StatusCode, location.Host))
	return resp.StatusCode == 302 && strings.Contains(location.Host, "keycloak")
}

func assertOidcIngressByName(key string) {
	Expect(ingressURLs).To(HaveKey(key), fmt.Sprintf("Ingress %s not found", key))
	url := ingressURLs[key]
	assertOidcIngress(url)
}

func assertOidcIngress(url string) {
	assertUnAuthorized := assertOauthURLAccessibleAndUnauthorized
	assertBasicAuth := assertURLAccessibleAndAuthorized
	assertBearerAuth := assertBearerAuthorized
	pkg.Concurrently(
		func() {
			Eventually(func() bool { return assertUnAuthorized(url) }, waitTimeout, pollingInterval).Should(BeTrue())
		},
		func() {
			Eventually(func() bool { return assertBasicAuth(url) }, waitTimeout, pollingInterval).Should(BeTrue())
		},
		func() {
			Eventually(func() bool { return assertBearerAuth(url) }, waitTimeout, pollingInterval).Should(BeTrue())
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
		vmiHTTPClient, err := pkg.GetSystemVmiHTTPClient()
		if err != nil {
			pkg.Log(pkg.Error, fmt.Sprintf("Error getting HTTP client: %v", err))
			return false
		}
		vmiHTTPClient.HTTPClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}

		req, err := retryablehttp.NewRequest("GET", searchURL, nil)
		if err != nil {
			pkg.Log(pkg.Error, fmt.Sprintf("Error creating HTTP request: %v", err))
			return false
		}
		req.SetBasicAuth(creds.Username, creds.Password)
		resp, err := vmiHTTPClient.Do(req)
		if err != nil {
			pkg.Log(pkg.Error, fmt.Sprintf("Error making HTTP request: %v", err))
			return false
		}
		if resp.StatusCode != http.StatusOK {
			pkg.Log(pkg.Error, fmt.Sprintf("Unexpected HTTP status code: %d", resp.StatusCode))
			return false
		}
		// assert that there is a single item in response
		defer resp.Body.Close()
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			pkg.Log(pkg.Error, fmt.Sprintf("Unable to read body from response: %v", err))
			return false
		}
		var response []map[string]interface{}
		if err := json.Unmarshal(bodyBytes, &response); err != nil {
			pkg.Log(pkg.Error, fmt.Sprintf("Error unmarshaling response body: %v", err))
			return false
		}
		if len(response) != 1 {
			pkg.Log(pkg.Error, fmt.Sprintf("Unexpected response length: %d", len(response)))
			return false
		}
		return true
	}
	Eventually(searchDashboard, waitTimeout, pollingInterval).Should(BeTrue())
}

func assertInstanceInfoURLs() {
	cr, err := pkg.GetVerrazzanoInstallResourceInCluster(pkg.GetKubeConfigPathFromEnv())
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
