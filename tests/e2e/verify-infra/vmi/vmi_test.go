// Copyright (c) 2020, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vmi_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Jeffail/gabs/v2"
	"github.com/hashicorp/go-retryablehttp"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	vzalpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	cmcommon "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/grafana"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/keycloak"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/kiali"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/nginx"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/opensearch"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/opensearchdashboards"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/prometheus/operator"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/thanos"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/vmi"
	corev1 "k8s.io/api/core/v1"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	verrazzanoNamespace = "verrazzano-system"
	esMasterPrefix      = "elasticsearch-master-vmi-system-es-master"
	esMaster0           = esMasterPrefix + "-0"
	esMaster1           = esMasterPrefix + "-1"
	esMaster2           = esMasterPrefix + "-2"
	esData              = "vmi-system-es-data"
	esData1             = esData + "-1"
	esData2             = esData + "-2"
)

var (
	opensearchIngress = "vmi-system-os-ingest"
	osdIngress        = "vmi-system-osd"
	prometheusIngress = "vmi-system-prometheus"
	grafanaIngress    = "vmi-system-grafana"
)

var t = framework.NewTestFramework("vmi")

func getIngressURLs() (map[string]string, error) {
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
	elastic                *vmi.Opensearch
	waitTimeout            = 1 * time.Minute
	pollingInterval        = 5 * time.Second
	elasticWaitTimeout     = 1 * time.Minute
	elasticPollingInterval = 5 * time.Second

	vzMonitoringVolumeClaims map[string]*corev1.PersistentVolumeClaim
)

var beforeSuite = t.BeforeSuiteFunc(func() {
	var err error
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		t.Fail(fmt.Sprintf("Failed to get default kubeconfig path: %s", err.Error()))
	}
	vz, err := pkg.GetVerrazzanoInstallResourceInCluster(kubeconfigPath)
	if err != nil {
		t.Fail(fmt.Sprintf("Failed to get installed Verrazzano resource in the cluster: %v", err))
	}
	if ingressEnabled(vz) {
		httpClient = pkg.EventuallyVerrazzanoRetryableHTTPClient()
	}

	Eventually(func() (*apiextv1.CustomResourceDefinition, error) {
		vzCRD, err = verrazzanoInstallerCRD()
		return vzCRD, err
	}, waitTimeout, pollingInterval).ShouldNot(BeNil())

	Eventually(func() error {
		ingressURLs, err = getIngressURLs()
		return err
	}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())

	Eventually(func() error {
		volumeClaims, err = pkg.GetPersistentVolumeClaims(verrazzanoNamespace)
		return err
	}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())

	Eventually(func() error {
		vzMonitoringVolumeClaims, err = pkg.GetPersistentVolumeClaims(constants.VerrazzanoMonitoringNamespace)
		return err
	}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())

	elastic = vmi.GetOpensearch("system")
	if verrazzanoSecretRequired(vz) {
		creds = pkg.EventuallyGetSystemVMICredentials()
	}

})

var _ = BeforeSuite(beforeSuite)

var _ = t.AfterEach(func() {})

var _ = t.Describe("VMI", Label("f:infra-lcm"), func() {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		AbortSuite(fmt.Sprintf("Failed to get default kubeconfig path: %s", err.Error()))
	}
	vz, err := pkg.GetVerrazzanoInstallResourceInCluster(kubeconfigPath)
	if err != nil {
		AbortSuite(fmt.Sprintf("Failed to get installed Verrazzano resource in the cluster: %v", err))
	}

	t.Context("Check that OpenSearch", func() {
		if vzcr.IsComponentStatusEnabled(vz, opensearch.ComponentName) {
			t.It("VMI is created successfully", func() {
				Eventually(func() (*apiextv1.CustomResourceDefinition, error) {
					vmiCRD, err = verrazzanoMonitoringInstanceCRD()
					return vmiCRD, err
				}, waitTimeout, pollingInterval).ShouldNot(BeNil())
			})

			if ingressEnabled(vz) {
				t.It("endpoint is accessible", Label("f:mesh.ingress"), func() {
					elasticPodsRunning := func() bool {
						result, err := pkg.PodsRunning(verrazzanoNamespace, []string{"vmi-system-es-master"})
						if err != nil {
							AbortSuite(fmt.Sprintf("Pod %v is not running in the namespace: %v, error: %v", "vmi-system-es-master", verrazzanoNamespace, err))
						}
						return result
					}
					Eventually(elasticPodsRunning, waitTimeout, pollingInterval).Should(BeTrue(), "pods did not all show up")
					Eventually(elasticIngress, elasticWaitTimeout, elasticPollingInterval).Should(BeTrue(), "ingress did not show up")
					Expect(ingressURLs).To(HaveKey(opensearchIngress), "Ingress vmi-system-os-ingest not found")
					Eventually(elasticConnected, elasticWaitTimeout, elasticPollingInterval).Should(BeTrue(), "never connected")
					Eventually(elasticIndicesCreated, elasticWaitTimeout, elasticPollingInterval).Should(BeTrue(), "indices never created")
					assertOidcIngressByName(opensearchIngress, vz, opensearch.ComponentName)
					Expect(vz.Status.VerrazzanoInstance.ElasticURL).ToNot(BeNil())
				})

				t.It("verrazzano-system Index is accessible", Label("f:observability.logging.es"),
					func() {
						if os.Getenv("TEST_ENV") != "LRE" {
							indexName, err := pkg.GetOpenSearchSystemIndex(verrazzanoNamespace)
							Expect(err).ShouldNot(HaveOccurred())
							pkg.Concurrently(
								func() {
									Eventually(func() bool {
										return pkg.FindLog(indexName,
											[]pkg.Match{
												{Key: "kubernetes.container_name", Value: "verrazzano-monitoring-operator"},
												{Key: "cluster_name", Value: constants.MCLocalCluster}},
											[]pkg.Match{})
									}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find a verrazzano-monitoring-operator log record")
								},
								func() {
									Eventually(func() bool {
										return pkg.FindLog(indexName,
											[]pkg.Match{
												{Key: "kubernetes.container_name", Value: "verrazzano-application-operator"},
												{Key: "cluster_name", Value: constants.MCLocalCluster}},
											[]pkg.Match{})
									}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find a verrazzano-application-operator log record")
								},
							)
						}
					})

				t.It("health is green", func() {
					Eventually(elasticHealth, elasticWaitTimeout, elasticPollingInterval).Should(BeTrue(), "cluster health status not green")
					Eventually(elasticIndicesHealth, elasticWaitTimeout, elasticPollingInterval).Should(BeTrue(), "indices health status not green")
				})

				t.It("systemd journal Index is accessible", Label("f:observability.logging.es"),
					func() {
						indexName, err := pkg.GetOpenSearchSystemIndex("systemd-journal")
						Expect(err).ShouldNot(HaveOccurred())
						Eventually(func() bool {
							return pkg.FindAnyLog(indexName,
								[]pkg.Match{
									{Key: "tag", Value: "systemd"},
									{Key: "TRANSPORT", Value: "journal"},
									{Key: "cluster_name", Value: constants.MCLocalCluster}},
								[]pkg.Match{})
						}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find a systemd log record")
					})
			}
		} else {
			t.It("is not running", func() {
				// Verify ES not present
				Eventually(func() (bool, error) {
					return pkg.PodsNotRunning(verrazzanoNamespace, []string{"vmi-system-es"})
				}, waitTimeout, pollingInterval).Should(BeTrue())
				Expect(elastic.CheckIngress()).To(BeFalse())
				Expect(ingressURLs).NotTo(HaveKey(opensearchIngress), fmt.Sprintf("Ingress %s should not exist", opensearchIngress))
				Expect(vz.Status.VerrazzanoInstance == nil || vz.Status.VerrazzanoInstance.ElasticURL == nil).To(BeTrue())
			})
		}
	})

	t.Context("Check that OpenSearch-Dashboards", func() {
		if vzcr.IsComponentStatusEnabled(vz, opensearchdashboards.ComponentName) {
			if ingressEnabled(vz) {
				t.It("endpoint is accessible", Label("f:mesh.ingress",
					"f:observability.logging.kibana"), func() {
					osdPodsRunning := func() bool {
						result, err := pkg.PodsRunning(verrazzanoNamespace, []string{"vmi-system-osd"})
						if err != nil {
							AbortSuite(fmt.Sprintf("Pod %v is not running in the namespace: %v, error: %v", "vmi-system-osd", verrazzanoNamespace, err))
						}
						return result
					}
					Eventually(osdPodsRunning, waitTimeout, pollingInterval).Should(BeTrue(), "osd pods did not all show up")
					Expect(ingressURLs).To(HaveKey("vmi-system-osd"), "Ingress vmi-system-osd not found")
					assertOidcIngressByName(osdIngress, vz, opensearchdashboards.ComponentName)
					Expect(vz.Status.VerrazzanoInstance.KibanaURL).ToNot(BeNil())
				})
			}
		} else {
			t.It("is not running", func() {

				Eventually(func() (bool, error) {
					return pkg.PodsNotRunning(verrazzanoNamespace, []string{"vmi-system-osd"})
				}, waitTimeout, pollingInterval).Should(BeTrue())
				Expect(ingressURLs).NotTo(HaveKey(osdIngress), fmt.Sprintf("Ingress %s should not exist", osdIngress))
				Expect(vz.Status.VerrazzanoInstance == nil || vz.Status.VerrazzanoInstance.KibanaURL == nil).To(BeTrue())
			})
		}
	})

	t.Context("Check that Prometheus", func() {
		const stsName = "prometheus-prometheus-operator-kube-p-prometheus"
		if vzcr.IsComponentStatusEnabled(vz, operator.ComponentName) {
			t.It("helm override for replicas is in effect", Label("f:observability.monitoring.prom"), func() {
				expectedReplicas, err := getExpectedPrometheusReplicaCount(kubeconfigPath)
				Expect(err).ToNot(HaveOccurred())

				// expect Prometheus statefulset to be configured for the expected number of replicas
				sts, err := pkg.GetStatefulSet(constants.VerrazzanoMonitoringNamespace, stsName)
				Expect(err).ToNot(HaveOccurred())
				Expect(sts.Spec.Replicas).ToNot(BeNil())
				Expect(*sts.Spec.Replicas).To(Equal(expectedReplicas))

				// expect the replicas to be ready
				Eventually(func() (int32, error) {
					sts, err := pkg.GetStatefulSet(constants.VerrazzanoMonitoringNamespace, stsName)
					if err != nil {
						return 0, err
					}
					return sts.Status.ReadyReplicas, nil
				}, waitTimeout, pollingInterval).Should(Equal(expectedReplicas),
					fmt.Sprintf("Statefulset %s in namespace %s does not have the expected number of ready replicas", stsName, constants.VerrazzanoMonitoringNamespace))
			})
			if ingressEnabled(vz) {
				t.It("endpoint is accessible", Label("f:mesh.ingress",
					"f:observability.monitoring.prom"), func() {
					assertOidcIngressByName(prometheusIngress, vz, operator.ComponentName)
					Expect(vz.Status.VerrazzanoInstance.PrometheusURL).ToNot(BeNil())
				})
			}

		} else {
			t.It("is not running", func() {
				Eventually(func() (bool, error) {
					return pkg.PodsNotRunning(verrazzanoNamespace, []string{stsName})
				}, waitTimeout, pollingInterval).Should(BeTrue())
				Expect(ingressURLs).NotTo(HaveKey(prometheusIngress), fmt.Sprintf("Ingress %s should not exist", prometheusIngress))
				Expect(vz.Status.VerrazzanoInstance == nil || vz.Status.VerrazzanoInstance.PrometheusURL == nil).To(BeTrue())
			})
		}
	})

	t.Context("Check that Grafana", func() {
		if vzcr.IsComponentStatusEnabled(vz, grafana.ComponentName) {
			if ingressEnabled(vz) {
				t.It("Grafana endpoint should be accessible", Label("f:mesh.ingress",
					"f:observability.monitoring.graf"), func() {
					Expect(ingressURLs).To(HaveKey("vmi-system-grafana"), "Ingress vmi-system-grafana not found")
					Expect(vz.Status.VerrazzanoInstance.GrafanaURL).ToNot(BeNil())
				})

				t.It("Default dashboard should be installed in System Grafana for shared VMI",
					Label("f:observability.monitoring.graf"), func() {
						pkg.Concurrently(
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
				t.ItMinimumVersion("Grafana should have the verrazzano user with admin privileges", "1.3.0", kubeconfigPath, func() {
					vz, err := pkg.GetVerrazzanoInstallResourceInCluster(kubeconfigPath)
					if err != nil {
						t.Logs.Errorf("Error getting Verrazzano resource: %v", err)
						Fail(err.Error())
					}
					if vz.Spec.Version != "" {
						t.Logs.Info("Skipping test because Verrazzano has been upgraded %s")
					} else {
						Eventually(assertAdminRole, waitTimeout, pollingInterval).Should(BeTrue())
					}
				})

				t.It("Grafana should have a default datasource present", func() {
					vz, err := pkg.GetVerrazzanoInstallResourceInCluster(kubeconfigPath)
					if err != nil {
						t.Logs.Errorf("Error getting Verrazzano resource: %v", err)
						Fail(err.Error())
					}
					name := "Prometheus"
					if vzcr.IsThanosEnabled(vz) {
						name = "Thanos"
					}
					Eventually(func() (bool, error) {
						return grafanaDefaultDatasourceExists(vz, name, kubeconfigPath)
					}).WithTimeout(waitTimeout).WithPolling(pollingInterval).Should(BeTrue())
				})
			}

		} else {
			t.It("is not running", func() {
				Eventually(func() (bool, error) {
					return pkg.PodsNotRunning(verrazzanoNamespace, []string{"vmi-system-grafana"})
				}, waitTimeout, pollingInterval).Should(BeTrue())
				Expect(ingressURLs).NotTo(HaveKey(grafanaIngress), fmt.Sprintf("Ingress %s should not exist", grafanaIngress))
				Expect(vz.Status.VerrazzanoInstance == nil || vz.Status.VerrazzanoInstance.GrafanaURL == nil).To(BeTrue())
			})
		}
	})

	t.Context("Check Storage", func() {
		size := "50Gi"
		// If there are persistence overrides at the global level, that will cause persistent
		// volumes to be created for the VMI components that use them (ES, Kibana, and Prometheus)
		// At some point we may need to check for individual VMI overrides.
		override, _ := pkg.GetEffectiveVMIPersistenceOverride(kubeconfigPath)
		if override != nil {
			size = override.Spec.Resources.Requests.Storage().String()
		}
		if pkg.IsDevProfile() {
			t.It("Check persistent volumes for dev profile", func() {
				if override != nil {
					minVer14, err := pkg.IsVerrazzanoMinVersion("1.4.0", kubeconfigPath)
					Expect(err).ToNot(HaveOccurred())

					expectedPromReplicas, err := getExpectedPrometheusReplicaCount(kubeconfigPath)
					Expect(err).ToNot(HaveOccurred())
					expectedThanosReplicas, err := getExpectedThanosReplicaCount(kubeconfigPath)
					Expect(err).ToNot(HaveOccurred())

					if minVer14 {
						Expect(len(volumeClaims)).To(Equal(2))
						assertPersistentVolume("vmi-system-grafana", size)
						assertPersistentVolume(esMaster0, size)

						Expect(len(vzMonitoringVolumeClaims)).To(Equal(int(expectedPromReplicas) + int(expectedThanosReplicas)))
						assertPrometheusVolume(size)
					} else {
						Expect(len(volumeClaims)).To(Equal(3))
						assertPersistentVolume("vmi-system-prometheus", size)
						assertPersistentVolume("vmi-system-grafana", size)
						assertPersistentVolume(esMaster0, size)
					}
				} else {
					Expect(len(volumeClaims)).To(Equal(0))
				}
			})
		} else if pkg.IsManagedClusterProfile() {
			t.It("Check persistent volumes for managed cluster profile", func() {
				minVer14, err := pkg.IsVerrazzanoMinVersion("1.4.0", kubeconfigPath)
				Expect(err).ToNot(HaveOccurred())

				expectedPromReplicas, err := getExpectedPrometheusReplicaCount(kubeconfigPath)
				Expect(err).ToNot(HaveOccurred())
				expectedThanosReplicas, err := getExpectedThanosReplicaCount(kubeconfigPath)
				Expect(err).ToNot(HaveOccurred())

				if minVer14 {
					Expect(len(volumeClaims)).To(Equal(0))
					Expect(len(vzMonitoringVolumeClaims)).To(Equal(int(expectedPromReplicas) + int(expectedThanosReplicas)))
					assertPrometheusVolume(size)
				} else {
					Expect(len(volumeClaims)).To(Equal(1))
					assertPersistentVolume("vmi-system-prometheus", size)
				}
			})
		} else if pkg.IsProdProfile() {
			t.It("Check persistent volumes for prod cluster profile", func() {
				minVer14, err := pkg.IsVerrazzanoMinVersion("1.4.0", kubeconfigPath)
				Expect(err).ToNot(HaveOccurred())

				expectedPromReplicas, err := getExpectedPrometheusReplicaCount(kubeconfigPath)
				Expect(err).ToNot(HaveOccurred())
				expectedThanosReplicas, err := getExpectedThanosReplicaCount(kubeconfigPath)
				Expect(err).ToNot(HaveOccurred())

				if minVer14 {
					Expect(len(volumeClaims)).To(Equal(7))
					Expect(len(vzMonitoringVolumeClaims)).To(Equal(int(expectedPromReplicas) + int(expectedThanosReplicas)))
					assertPrometheusVolume(size)
				} else {
					Expect(len(volumeClaims)).To(Equal(8))
					assertPersistentVolume("vmi-system-prometheus", size)
				}
				assertPersistentVolume("vmi-system-grafana", size)
				assertPersistentVolume(esMaster0, size)
				assertPersistentVolume(esMaster1, size)
				assertPersistentVolume(esMaster2, size)
				assertPersistentVolume(esData, size)
				assertPersistentVolume(esData1, size)
				assertPersistentVolume(esData2, size)
			})
		}
	})
})

func assertPersistentVolume(key string, size string) {
	Expect(volumeClaims).To(HaveKey(key))
	pvc := volumeClaims[key]
	Expect(pvc.Spec.Resources.Requests.Storage().String()).To(Equal(size))
}

func assertPrometheusVolume(size string) {
	// Prometheus Operator generates the name for the PVC so look for a PVC name that contains "prometheus"
	for key, pvc := range vzMonitoringVolumeClaims {
		if strings.Contains(key, "prometheus") {
			Expect(pvc.Spec.Resources.Requests.Storage().String()).To(Equal(size))
			return
		}
	}
	Fail("Expected to find Prometheus persistent volume claim")
}

func assertOidcIngressByName(key string, vz *vzalpha1.Verrazzano, componentName string) {
	if ingressEnabled(vz) {
		Expect(ingressURLs).To(HaveKey(key), fmt.Sprintf("Ingress %s not found", key))
		url := ingressURLs[key]
		assertOidcIngress(url)
	} else {
		t.Logs.Infof("Skipping checking ingress %s because ingress-nginx or cert-manager is not installed", key)
	}

}

func assertOidcIngress(url string) {
	unauthHTTPClient := pkg.EventuallyVerrazzanoRetryableHTTPClient()
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

func elasticIngress() bool {
	return elastic.CheckIngress()
}

func assertDashboard(url string) {
	searchURL := fmt.Sprintf("%sapi/search?query=%s", ingressURLs["vmi-system-grafana"], url)
	fmt.Println("Grafana URL in browseGrafanaDashboard ", searchURL)

	searchDashboard := func() bool {
		vmiHTTPClient := pkg.EventuallyVerrazzanoRetryableHTTPClient()
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
		bodyBytes, err := io.ReadAll(resp.Body)
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
	vmiHTTPClient := pkg.EventuallyVerrazzanoRetryableHTTPClient()
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
	bodyBytes, err := io.ReadAll(resp.Body)
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
	t.Logs.Infof("Grafana users: %s", response)
	return response[0]["login"] == "verrazzano" && response[0]["isAdmin"] == true
}

// getExpectedPrometheusReplicaCount returns the Prometheus replicas in the values overrides from the
// Prometheus Operator component in the Verrazzano CR. If there is no override for replicas then the
// default replica count of 1 is returned.
func getExpectedPrometheusReplicaCount(kubeconfig string) (int32, error) {
	vz, err := pkg.GetVerrazzanoInstallResourceInCluster(kubeconfig)
	if err != nil {
		return 0, err
	}
	if !vzcr.IsComponentStatusEnabled(vz, operator.ComponentName) {
		return 0, nil
	}
	var expectedReplicas int32 = 1
	if vz.Spec.Components.PrometheusOperator == nil {
		return expectedReplicas, nil
	}

	for _, override := range vz.Spec.Components.PrometheusOperator.InstallOverrides.ValueOverrides {
		if override.Values != nil {
			jsonString, err := gabs.ParseJSON(override.Values.Raw)
			if err != nil {
				return 0, err
			}
			if container := jsonString.Path("prometheus.prometheusSpec.replicas"); container != nil {
				if val, ok := container.Data().(float64); ok {
					expectedReplicas = int32(val)
					t.Logs.Infof("Found Prometheus replicas override in Verrazzano CR, replica count is: %d", expectedReplicas)
					break
				}
			}
		}
	}

	return expectedReplicas, nil
}

// getExpectedThanosReplicaCount returns the thanos replicas in the values overrides from the
// Thanos component in the Verrazzano CR. If there is no override for replicas then the
// default replica count of 1 is returned.
func getExpectedThanosReplicaCount(kubeconfig string) (int32, error) {
	vz, err := pkg.GetVerrazzanoInstallResourceInCluster(kubeconfig)
	if err != nil {
		return 0, err
	}
	if !vzcr.IsComponentStatusEnabled(vz, thanos.ComponentName) {
		return 0, nil
	}
	expectedReplicas := int32(0)
	if vz.Spec.Components.Thanos == nil {
		return expectedReplicas, nil
	}

	for _, override := range vz.Spec.Components.Thanos.InstallOverrides.ValueOverrides {
		if override.Values != nil {
			jsonString, err := gabs.ParseJSON(override.Values.Raw)
			if err != nil {
				return expectedReplicas, err
			}
			// check to see if storegateway is enabled and if so how many replicas it has
			if enabledContainer := jsonString.Path("storegateway.enabled"); enabledContainer != nil {
				if enabled, ok := enabledContainer.Data().(bool); ok && enabled {
					expectedReplicas = int32(1)
					if replicaContainer := jsonString.Path("storegateway.replicaCount"); replicaContainer != nil {
						if val, ok := replicaContainer.Data().(float64); ok {
							expectedReplicas = int32(val)
							t.Logs.Infof("Found Thanos storegateway replicas override in Verrazzano CR, replica count is: %d", expectedReplicas)
							break
						}
					}
				}
			}
		}
	}

	return expectedReplicas, nil
}

func ingressEnabled(vz *vzalpha1.Verrazzano) bool {
	return vzcr.IsComponentStatusEnabled(vz, nginx.ComponentName) &&
		vzcr.IsComponentStatusEnabled(vz, cmcommon.CertManagerComponentName) &&
		vzcr.IsComponentStatusEnabled(vz, keycloak.ComponentName)
}

func verrazzanoSecretRequired(vz *vzalpha1.Verrazzano) bool {
	return vzcr.IsComponentStatusEnabled(vz, opensearch.ComponentName) ||
		vzcr.IsComponentStatusEnabled(vz, opensearchdashboards.ComponentName) ||
		vzcr.IsComponentStatusEnabled(vz, operator.ComponentName) ||
		vzcr.IsComponentStatusEnabled(vz, grafana.ComponentName) ||
		vzcr.IsComponentStatusEnabled(vz, kiali.ComponentName)
}

func grafanaDefaultDatasourceExists(vz *vzalpha1.Verrazzano, name, kubeconfigPath string) (bool, error) {
	password, err := pkg.GetVerrazzanoPasswordInCluster(kubeconfigPath)
	if err != nil {
		t.Logs.Error("Failed to get the Verrazzano password from the cluster")
		return false, err
	}
	if vz == nil || vz.Status.VerrazzanoInstance == nil || vz.Status.VerrazzanoInstance.GrafanaURL == nil {
		t.Logs.Error("Grafana URL in the Verrazzano status is empty")
		return false, nil
	}
	resp, err := pkg.GetWebPageWithBasicAuth(*vz.Status.VerrazzanoInstance.GrafanaURL+"/api/datasources", "", "verrazzano", password, kubeconfigPath)
	if err != nil {
		t.Logs.Errorf("Failed to get Grafana datasources: %v", err)
		return false, err
	}

	var datasources []map[string]interface{}
	err = json.Unmarshal(resp.Body, &datasources)
	if err != nil {
		t.Logs.Errorf("Failed to unmarshal Grafana datasources: %v", err)
		return false, err
	}

	for _, source := range datasources {
		sourceName, ok := source["name"]
		if !ok {
			t.Logs.Errorf("Failed to find name for Grafana datasource")
			continue
		}

		nameStr, ok := sourceName.(string)
		if !ok {
			t.Logs.Errorf("Failed to convert name field to string")
			continue
		}
		if nameStr != name {
			continue
		}

		sourceDefault, ok := source["isDefault"]
		if !ok {
			t.Logs.Errorf("Failed to verify the datasource was the default")
			continue
		}
		defaultBool, ok := sourceDefault.(bool)
		if !ok {
			t.Logs.Errorf("Failed to convert default to bool")
			continue
		}
		return defaultBool, nil
	}

	t.Logs.Errorf("Failed to find Grafana datasource %s", name)
	return false, nil
}
