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
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/vmi"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const verrazzanoNamespace string = "verrazzano-system"

func vmiIngressURLs() (map[string]string, error) {
	ingressList, err := pkg.GetKubernetesClientset().ExtensionsV1beta1().Ingresses(verrazzanoNamespace).List(context.TODO(), v1.ListOptions{})
	if err != nil {
		return nil, err
	}

	ingressURLs := make(map[string]string)

	for _, ingress := range ingressList.Items {
		var ingressRules []v1beta1.IngressRule = ingress.Spec.Rules
		if len(ingressRules) != 1 {
			return nil, fmt.Errorf("Expected ingress %s in namespace %s to have 1 ingress rule, but had %v",
				ingress.Name, ingress.Namespace, ingressRules)
		}
		ingressURLs[ingress.Name] = fmt.Sprintf("https://%s/", ingressRules[0].Host)
	}
	return ingressURLs, nil
}

func verrazzanoMonitoringInstanceCRD() (*apiextensionsv1beta1.CustomResourceDefinition, error) {
	crd, err := pkg.APIExtensionsClientSet().CustomResourceDefinitions().Get(context.TODO(), "verrazzanomonitoringinstances.verrazzano.io", v1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return crd, nil
}

func verrazzanoInstallerCRD() (*apiextensionsv1beta1.CustomResourceDefinition, error) {
	crd, err := pkg.APIExtensionsClientSet().CustomResourceDefinitions().Get(context.TODO(), "verrazzanos.install.verrazzano.io", v1.GetOptions{})
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

var _ = ginkgo.BeforeSuite(func() {
	var err error

	vzCRD, err = verrazzanoInstallerCRD()
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Error retrieving Verrazzano Installer CRD: %v", err))
	}

	ingressURLs, err = vmiIngressURLs()
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Error retrieving system VMI ingress URLs: %v", err))
	}

	volumeClaims, err = pkg.GetPersistentVolumes(verrazzanoNamespace)
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Error retrieving persistent volumes for verrazzano-system: %v", err))
	}

	vmiCRD, err = verrazzanoMonitoringInstanceCRD()
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Error retrieving system VMI CRD: %v", err))
	}

	creds, err = pkg.GetSystemVMICredentials()
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Error retrieving system VMI credentials: %v", err))
	}
	elastic = vmi.GetElastic("system")
})

var _ = ginkgo.Describe("VMI", func() {

	isManagedClusterProfile := pkg.IsManagedClusterProfile()
	if isManagedClusterProfile {
		ginkgo.It("Elasticsearch should NOT be present", func() {
			// Verify ES not present
			gomega.Expect(pkg.PodsNotRunning(verrazzanoNamespace, []string{"vmi-system-es"})).To(gomega.BeTrue())
			gomega.Expect(elasticTLSSecret()).To(gomega.BeTrue())
			gomega.Expect(elastic.CheckIngress()).To(gomega.BeFalse())
			gomega.Expect(ingressURLs).NotTo(gomega.HaveKey("vmi-system-es-ingest"), fmt.Sprintf("Ingress %s not found", "vmi-system-grafana"))

			// Verify Kibana not present
			gomega.Expect(pkg.PodsNotRunning(verrazzanoNamespace, []string{"vmi-system-kibana"})).To(gomega.BeTrue())
			gomega.Expect(ingressURLs).NotTo(gomega.HaveKey("vmi-system-kibana"), fmt.Sprintf("Ingress %s not found", "vmi-system-grafana"))

			// Verify Grafana not present
			gomega.Expect(pkg.PodsNotRunning(verrazzanoNamespace, []string{"vmi-system-grafana"})).To(gomega.BeTrue())
			gomega.Expect(ingressURLs).NotTo(gomega.HaveKey("vmi-system-grafana"), fmt.Sprintf("Ingress %s not found", "vmi-system-grafana"))
		})
	} else {
		ginkgo.It("Elasticsearch endpoint should be accessible", func() {
			elasticPodsRunning := func() bool {
				return pkg.PodsRunning(verrazzanoNamespace, []string{"vmi-system-es-master"})
			}
			gomega.Eventually(elasticPodsRunning, waitTimeout, pollingInterval).Should(gomega.BeTrue(), "pods did not all show up")
			gomega.Eventually(elasticTLSSecret, elasticWaitTimeout, elasticPollingInterval).Should(gomega.BeTrue(), "tls-secret did not show up")
			//gomega.Eventually(elasticCertificate, elasticWaitTimeout, elasticPollingInterval).Should(gomega.BeTrue(), "certificate did not show up")
			gomega.Eventually(elasticIngress, elasticWaitTimeout, elasticPollingInterval).Should(gomega.BeTrue(), "ingress did not show up")
			gomega.Expect(ingressURLs).To(gomega.HaveKey("vmi-system-es-ingest"), "Ingress vmi-system-es-ingest not found")
			assertOidcIngressByName("vmi-system-es-ingest")
			gomega.Eventually(elasticConnected, elasticWaitTimeout, elasticPollingInterval).Should(gomega.BeTrue(), "never connected")
			gomega.Eventually(elasticIndicesCreated, elasticWaitTimeout, elasticPollingInterval).Should(gomega.BeTrue(), "indices never created")
		})

		ginkgo.It("Elasticsearch verrazzano-system Index should be accessible", func() {
			indexName := "verrazzano-namespace-verrazzano-system"
			pkg.Concurrently(
				func() {
					gomega.Eventually(func() bool {
						return pkg.LogRecordFound(indexName,
							time.Now().Add(-24*time.Hour), map[string]string{
								"kubernetes.container_name": "verrazzano-monitoring-operator",
								"caller":                    "controller",
								"cluster_name":              constants.MCLocalCluster,
							})
					}, waitTimeout, pollingInterval).Should(gomega.BeTrue(), "Expected to find a verrazzano-monitoring-operator log record")
				},
				func() {
					gomega.Eventually(func() bool {
						return pkg.LogRecordFound(indexName,
							time.Now().Add(-24*time.Hour), map[string]string{
								"kubernetes.container_name": "verrazzano-application-operator",
								"caller":                    "controller",
								"cluster_name":              constants.MCLocalCluster,
							})
					}, waitTimeout, pollingInterval).Should(gomega.BeTrue(), "Expected to find a verrazzano-application-operator log record")
				},
			)
		})

		ginkgo.It("Elasticsearch systemd journal Index should be accessible", func() {
			gomega.Eventually(func() bool {
				return pkg.FindAnyLog("verrazzano-systemd-journal",
					[]pkg.Match{
						{Key: "tag", Value: "systemd"},
						{Key: "TRANSPORT", Value: "journal"},
						{Key: "cluster_name", Value: constants.MCLocalCluster}},
					[]pkg.Match{})
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue(), "Expected to find a systemd log record")
		})

		ginkgo.It("Kibana endpoint should be accessible", func() {
			kibanaPodsRunning := func() bool {
				return pkg.PodsRunning(verrazzanoNamespace, []string{"vmi-system-kibana"})
			}
			gomega.Eventually(kibanaPodsRunning, waitTimeout, pollingInterval).Should(gomega.BeTrue(), "kibana pods did not all show up")
			gomega.Expect(ingressURLs).To(gomega.HaveKey("vmi-system-kibana"), "Ingress vmi-system-kibana not found")
			assertOidcIngressByName("vmi-system-kibana")
		})

		ginkgo.It("Prometheus endpoint should be accessible", func() {
			assertOidcIngressByName("vmi-system-prometheus")
		})

		ginkgo.It("Grafana endpoint should be accessible", func() {
			gomega.Expect(ingressURLs).To(gomega.HaveKey("vmi-system-grafana"), "Ingress vmi-system-grafana not found")
			assertOidcIngressByName("vmi-system-grafana")
		})

		ginkgo.It("Default dashboard should be installed in System Grafana for shared VMI", func() {
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

	ginkgo.It("Verify the instance info endpoint URLs", func() {
		if !isManagedClusterProfile {
			assertInstanceInfoURLs()
		}
	})

	size := "50Gi"
	if pkg.IsDevProfile() {
		ginkgo.It("Check persistent volumes for dev profile", func() {
			gomega.Expect(len(volumeClaims)).To(gomega.Equal(0))
		})
	} else if isManagedClusterProfile {
		ginkgo.It("Check persistent volumes for managed cluster profile", func() {
			gomega.Expect(len(volumeClaims)).To(gomega.Equal(1))
			assertPersistentVolume("vmi-system-prometheus", size)
		})
	} else if pkg.IsProdProfile() {
		ginkgo.It("Check persistent volumes for prod cluster profile", func() {
			gomega.Expect(len(volumeClaims)).To(gomega.Equal(7))
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
	gomega.Expect(volumeClaims).To(gomega.HaveKey(key))
	pvc := volumeClaims[key]
	gomega.Expect(pvc.Spec.Resources.Requests.Storage().String()).To(gomega.Equal(size))
}

func assertURLAccessibleAndAuthorized(url string) bool {
	vmiHTTPClient := pkg.GetSystemVmiHTTPClient()
	return pkg.AssertURLAccessibleAndAuthorized(vmiHTTPClient, url, creds)
}

func assertBearerAuthorized(url string) bool {
	vmiHTTPClient := pkg.GetSystemVmiHTTPClient()
	api, err := pkg.GetAPIEndpoint(pkg.GetKubeConfigPathFromEnv())
	if err != nil {
		return false
	}
	req, _ := retryablehttp.NewRequest("GET", url, nil)
	if api.AccessToken != "" {
		bearer := fmt.Sprintf("Bearer %v", api.AccessToken)
		req.Header.Set("Authorization", bearer)
	}
	resp, err := vmiHTTPClient.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	pkg.Log(pkg.Info, fmt.Sprintf("assertBearerAuthorized %v Response:%v Error:%v", url, resp.StatusCode, err))
	return resp.StatusCode == http.StatusOK
}

func assertOauthURLAccessibleAndUnauthorized(url string) bool {
	vmiHTTPClient := pkg.GetSystemVmiHTTPClient()
	vmiHTTPClient.HTTPClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		pkg.Log(pkg.Info, fmt.Sprintf("oidcUnauthorized req: %v \nvia: %v\n", req, via))
		return http.ErrUseLastResponse
	}
	resp, err := vmiHTTPClient.Get(url)
	if err != nil || resp == nil {
		return false
	}
	location, _ := resp.Location()
	gomega.Expect(location).NotTo(gomega.BeNil())
	pkg.Log(pkg.Info, fmt.Sprintf("oidcUnauthorized %v StatusCode:%v host:%v", url, resp.StatusCode, location.Host))
	return resp.StatusCode == 302 && strings.Contains(location.Host, "keycloak")
}

func assertOidcIngressByName(key string) {
	gomega.Expect(ingressURLs).To(gomega.HaveKey(key), fmt.Sprintf("Ingress %s not found", key))
	url := ingressURLs[key]
	assertOidcIngress(url)
}

func assertOidcIngress(url string) {
	assertUnAuthorized := assertOauthURLAccessibleAndUnauthorized
	assertBasicAuth := assertURLAccessibleAndAuthorized
	assertBearerAuth := assertBearerAuthorized
	pkg.Concurrently(
		func() {
			gomega.Eventually(func() bool { return assertUnAuthorized(url) }, waitTimeout, pollingInterval).Should(gomega.BeTrue())
		},
		func() {
			gomega.Eventually(func() bool { return assertBasicAuth(url) }, waitTimeout, pollingInterval).Should(gomega.BeTrue())
		},
		func() {
			gomega.Eventually(func() bool { return assertBearerAuth(url) }, waitTimeout, pollingInterval).Should(gomega.BeTrue())
		},
	)
}

func elasticIndicesCreated() bool {
	b, _ := gomega.ContainElements(".kibana_1").Match(elastic.ListIndices())
	return b
}

func elasticConnected() bool {
	return elastic.Connect()
}

func elasticTLSSecret() bool {
	return elastic.CheckTLSSecret()
}

//func elasticCertificate() bool {
//	return elastic.CheckCertificate()
//}

func elasticIngress() bool {
	return elastic.CheckIngress()
}

func assertDashboard(url string) {
	searchURL := fmt.Sprintf("%sapi/search?query=%s", ingressURLs["vmi-system-grafana"], url)
	fmt.Println("Grafana URL in browseGrafanaDashboard ", searchURL)
	vmiHTTPClient := pkg.GetSystemVmiHTTPClient()
	vmiHTTPClient.HTTPClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	searchDashboard := func() bool {
		req, err := retryablehttp.NewRequest("GET", searchURL, nil)
		if err != nil {
			return false
		}
		req.SetBasicAuth(creds.Username, creds.Password)
		resp, err := vmiHTTPClient.Do(req)
		if err != nil {
			return false
		}
		if resp.StatusCode != http.StatusOK {
			return false
		}
		// assert that there is a single item in response
		defer resp.Body.Close()
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			ginkgo.Fail("Unable to read body from response " + err.Error())
		}
		var response []map[string]interface{}
		if err := json.Unmarshal(bodyBytes, &response); err != nil {
			return false
		}
		if len(response) != 1 {
			return false
		}
		return true
	}
	gomega.Eventually(searchDashboard, waitTimeout, pollingInterval).Should(gomega.BeTrue())
}

func assertInstanceInfoURLs() {
	cr := pkg.GetVerrazzanoInstallResourceInCluster(pkg.GetKubeConfigPathFromEnv())
	instanceInfo := cr.Status.VerrazzanoInstance
	switch cr.Spec.Profile {
	case v1alpha1.ManagedCluster:
		gomega.Expect(instanceInfo.GrafanaURL).To(gomega.BeNil())
		gomega.Expect(instanceInfo.ElasticURL).To(gomega.BeNil())
		gomega.Expect(instanceInfo.KibanaURL).To(gomega.BeNil())
	default:
		gomega.Expect(instanceInfo.GrafanaURL).NotTo(gomega.BeNil())
		gomega.Expect(instanceInfo.ElasticURL).NotTo(gomega.BeNil())
		gomega.Expect(instanceInfo.KibanaURL).NotTo(gomega.BeNil())
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
	gomega.Expect(instanceInfo.PrometheusURL).NotTo(gomega.BeNil())
	if instanceInfo.PrometheusURL != nil {
		assertOidcIngress(*instanceInfo.PrometheusURL)
	}
}
