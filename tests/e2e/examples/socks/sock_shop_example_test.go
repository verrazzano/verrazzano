// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package socks

import (
	b64 "encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework/metrics"
	"k8s.io/apimachinery/pkg/api/errors"

	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/verrazzano/verrazzano/pkg/k8s/resource"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	v1 "k8s.io/api/core/v1"
)

const (
	shortWaitTimeout         = 7 * time.Minute
	shortPollingInterval     = 10 * time.Second
	waitTimeout              = 10 * time.Minute
	longWaitTimeout          = 20 * time.Minute
	pollingInterval          = 30 * time.Second
	imagePullWaitTimeout     = 40 * time.Minute
	imagePullPollingInterval = 30 * time.Second
	sockshopAppName          = "sockshop-appconf"
	sampleSpringMetric       = "http_server_requests_seconds_count"
	sampleMicronautMetric    = "process_start_time_seconds"
	oamComponent             = "app_oam_dev_component"
)

var sockShop SockShop
var username, password string

var (
	t                  = framework.NewTestFramework("socks")
	generatedNamespace = pkg.GenerateNamespace("sockshop")
	clusterDump        = pkg.NewClusterDumpWrapper(generatedNamespace)
)

// creates the sockshop namespace and applies the components and application yaml
var _ = clusterDump.BeforeSuite(func() {
	username = "username" + strconv.FormatInt(time.Now().Unix(), 10)
	password = b64.StdEncoding.EncodeToString([]byte(time.Now().String()))
	sockShop = NewSockShop(username, password, pkg.Ingress())

	variant := getVariant()
	t.Logs.Infof("*** Socks shop test is running against variant: %s\n", variant)

	if !skipDeploy {
		start := time.Now()
		// deploy the application here
		Eventually(func() (*v1.Namespace, error) {
			return pkg.CreateNamespace(namespace, map[string]string{"verrazzano-managed": "true"})
		}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

		Eventually(func() error {
			file, err := pkg.FindTestDataFile("examples/sock-shop/" + variant + "/sock-shop-comp.yaml")
			if err != nil {
				return err
			}
			return resource.CreateOrUpdateResourceFromFileInGeneratedNamespace(file, namespace)
		}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

		Eventually(func() error {
			file, err := pkg.FindTestDataFile("examples/sock-shop/" + variant + "/sock-shop-app.yaml")
			if err != nil {
				return err
			}
			return resource.CreateOrUpdateResourceFromFileInGeneratedNamespace(file, namespace)
		}, shortWaitTimeout, shortPollingInterval, "Failed to create Sock Shop application resource").ShouldNot(HaveOccurred())
		metrics.Emit(t.Metrics.With("deployment_elapsed_time", time.Since(start).Milliseconds()))
	}

	Eventually(func() bool {
		return pkg.ContainerImagePullWait(namespace, expectedPods)
	}, imagePullWaitTimeout, imagePullPollingInterval).Should(BeTrue())
	// checks that all pods are up and running
	Eventually(sockshopPodsRunning, longWaitTimeout, pollingInterval).Should(BeTrue())
})

// the list of expected pods
var expectedPods = []string{
	"carts-coh-0",
	"catalog-coh-0",
	"orders-coh-0",
	"payment-coh-0",
	"shipping-coh-0",
	"users-coh-0"}

// user registration template
const registerTemp = `{
  "username":"%v",
  "password":"%v",
  "email":"foo@oracle.com",
  "firstName":"foo",
  "lastName":"coo"
}`

var _ = t.AfterEach(func() {})

var _ = t.Describe("Sock Shop test", Label("f:app-lcm.oam",
	"f:app-lcm.helidon-workload",
	"f:app-lcm.spring-workload",
	"f:app-lcm.coherence-workload"), func() {

	var hostname = ""
	var err error
	t.BeforeEach(func() {
		Eventually(func() (string, error) {
			hostname, err = k8sutil.GetHostnameFromGateway(namespace, "")
			return hostname, err
		}, waitTimeout, shortPollingInterval).Should(Not(BeEmpty()))
	})

	sockShop.SetHostHeader(hostname)

	t.It("SockShop application configuration exists", func() {
		Eventually(func() bool {
			appConfig, err := pkg.GetAppConfig(namespace, sockshopAppName)
			if err != nil {
				return false
			}
			return appConfig != nil
		}, waitTimeout, pollingInterval).Should(BeTrue(), "Failed to get the application configuration for Sockshop")
	})

	t.It("SockShop can be accessed and user can be registered", func() {
		Eventually(func() (bool, error) {
			return sockShop.RegisterUser(fmt.Sprintf(registerTemp, username, password), hostname)
		}, waitTimeout, pollingInterval).Should(BeTrue(), "Failed to register SockShop User")
	})

	t.It("SockShop can log in with default user", func() {
		kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
		Expect(err).ShouldNot(HaveOccurred())
		Eventually(func() (*pkg.HTTPResponse, error) {
			url := fmt.Sprintf("https://%v/login", hostname)
			return pkg.GetWebPageWithBasicAuth(url, hostname, username, password, kubeconfigPath)
		}, waitTimeout, pollingInterval).Should(pkg.HasStatus(http.StatusOK))

	})

	t.It("SockShop can add item to cart", func() {
		// get the catalog
		var response *pkg.HTTPResponse
		Eventually(func() (*pkg.HTTPResponse, error) {
			var err error
			response, err = sockShop.GetCatalogItems(hostname)
			return response, err
		}, shortWaitTimeout, shortPollingInterval).Should(And(pkg.HasStatus(200), pkg.BodyContains("/catalogue/")))

		var catalogItems []CatalogItem
		json.Unmarshal(response.Body, &catalogItems)
		Expect(catalogItems).ShouldNot(BeEmpty(), "Catalog should not be empty")

		// add items to the cart from the catalog
		Eventually(func() (*pkg.HTTPResponse, error) {
			return sockShop.AddToCart(catalogItems[0], hostname)
		}, shortWaitTimeout, shortPollingInterval).Should(pkg.HasStatus(201))

		Eventually(func() (*pkg.HTTPResponse, error) {
			return sockShop.AddToCart(catalogItems[0], hostname)
		}, shortWaitTimeout, shortPollingInterval).Should(pkg.HasStatus(201))

		Eventually(func() (*pkg.HTTPResponse, error) {
			return sockShop.AddToCart(catalogItems[0], hostname)
		}, shortWaitTimeout, shortPollingInterval).Should(pkg.HasStatus(201))

		Eventually(func() (*pkg.HTTPResponse, error) {
			return sockShop.AddToCart(catalogItems[1], hostname)
		}, shortWaitTimeout, shortPollingInterval).Should(pkg.HasStatus(201))

		Eventually(func() (*pkg.HTTPResponse, error) {
			return sockShop.AddToCart(catalogItems[2], hostname)
		}, shortWaitTimeout, shortPollingInterval).Should(pkg.HasStatus(201))

		Eventually(func() (*pkg.HTTPResponse, error) {
			return sockShop.AddToCart(catalogItems[2], hostname)
		}, shortWaitTimeout, shortPollingInterval).Should(pkg.HasStatus(201))

		// get the cart
		Eventually(func() (*pkg.HTTPResponse, error) {
			var err error
			response, err = sockShop.GetCartItems(hostname)
			return response, err
		}, shortWaitTimeout, shortPollingInterval).Should(pkg.HasStatus(200))

		var cartItems []CartItem
		json.Unmarshal(response.Body, &cartItems)
		Expect(cartItems).ShouldNot(BeEmpty(), "Cart should not be empty")

		// make sure the right items and quantities are in the cart
		sockShop.CheckCart(cartItems, catalogItems[0], 3)
		sockShop.CheckCart(cartItems, catalogItems[1], 1)
		sockShop.CheckCart(cartItems, catalogItems[2], 2)
	})

	t.It("SockShop can delete all cart items", func() {
		var response *pkg.HTTPResponse
		// get the cart
		Eventually(func() (*pkg.HTTPResponse, error) {
			var err error
			response, err = sockShop.GetCartItems(hostname)
			return response, err
		}, shortWaitTimeout, shortPollingInterval).Should(pkg.HasStatus(200))

		var cartItems []CartItem
		json.Unmarshal(response.Body, &cartItems)
		Expect(cartItems).ShouldNot(BeEmpty(), "Cart should not be empty")

		// delete each item
		for _, item := range cartItems {
			Eventually(func() (*pkg.HTTPResponse, error) {
				return sockShop.DeleteCartItem(item, hostname)
			}, shortWaitTimeout, shortPollingInterval).Should(pkg.HasStatus(202))
		}

		// get the cart again - this time the cart should be empty
		Eventually(func() (*pkg.HTTPResponse, error) {
			var err error
			response, err = sockShop.GetCartItems(hostname)
			return response, err
		}, shortWaitTimeout, shortPollingInterval).Should(pkg.HasStatus(200))

		json.Unmarshal(response.Body, &cartItems)
		Expect(cartItems).Should(BeEmpty(), "Cart should be empty")
	})

	// INFO: Front-End will not allow for complete implementation of this test
	t.It("SockShop can change address", func() {
		var response *pkg.HTTPResponse
		Eventually(func() (*pkg.HTTPResponse, error) {
			var err error
			response, err = sockShop.ChangeAddress(username, hostname)
			return response, err
		}, shortWaitTimeout, shortPollingInterval).Should(pkg.HasStatus(200))

		sockShop.CheckAddress(response, username)
	})

	// INFO: Front-End will not allow for complete implementation of this test
	t.It("SockShop can change payment", func() {
		Eventually(func() (*pkg.HTTPResponse, error) {
			return sockShop.ChangePayment(hostname)
		}, shortWaitTimeout, shortPollingInterval).Should(pkg.HasStatus(200))
	})

	PIt("SockShop can retrieve orders", func() {
		//TODO
	})

	t.It("Verify '/catalogue' UI endpoint is working.", Label("f:mesh.ingress"), func() {
		Eventually(func() (*pkg.HTTPResponse, error) {
			url := fmt.Sprintf("https://%s/catalogue", hostname)
			return pkg.GetWebPage(url, hostname)
		}, shortWaitTimeout, shortPollingInterval).Should(And(pkg.HasStatus(http.StatusOK), pkg.BodyContains("For all those leg lovers out there.")))
	})

	// Verify Prometheus scraped metrics
	t.Context("Metrics", Label("f:observability.monitoring.prom"), func() {
		kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
		if err != nil {
			Expect(err).To(BeNil(), fmt.Sprintf("Failed to get default kubeconfig path: %s", err.Error()))
		}
		// Coherence metric fix available only from 1.3.0
		if ok, _ := pkg.IsVerrazzanoMinVersion("1.3.0", kubeconfigPath); ok {
			t.It("Retrieve Prometheus scraped metrics", func() {
				if getVariant() == "helidon" {
					pkg.Concurrently(
						func() {
							Eventually(appMetricExists, waitTimeout, pollingInterval).Should(BeTrue())
						},
						func() {
							Eventually(coherenceMetricExists, waitTimeout, pollingInterval).Should(BeTrue())
						},
						func() {
							Eventually(appComponentMetricExists, waitTimeout, pollingInterval).Should(BeTrue())
						},
						func() {
							Eventually(appConfigMetricExists, waitTimeout, pollingInterval).Should(BeTrue())
						},
					)
				} else if getVariant() == "spring" {
					pkg.Concurrently(
						func() {
							Eventually(func() bool {
								return springMetricExists("carts")
							}, waitTimeout, pollingInterval).Should(BeTrue())
						},
						func() {
							Eventually(func() bool {
								return springMetricExists("catalog")
							}, waitTimeout, pollingInterval).Should(BeTrue())
						},
						func() {
							Eventually(func() bool {
								return springMetricExists("orders")
							}, waitTimeout, pollingInterval).Should(BeTrue())
						},
						func() {
							Eventually(func() bool {
								return springMetricExists("payment")
							}, waitTimeout, pollingInterval).Should(BeTrue())
						},
						func() {
							Eventually(func() bool {
								return springMetricExists("shipping")
							}, waitTimeout, pollingInterval).Should(BeTrue())
						},
						func() {
							Eventually(func() bool {
								return springMetricExists("users")
							}, waitTimeout, pollingInterval).Should(BeTrue())
						},
						func() {
							Eventually(coherenceMetricExists, waitTimeout, pollingInterval).Should(BeTrue())
						},
					)
				} else if getVariant() == "micronaut" {
					pkg.Concurrently(
						func() {
							Eventually(func() bool {
								return micronautMetricExists("carts")
							}, waitTimeout, pollingInterval).Should(BeTrue())
						},
						func() {
							Eventually(func() bool {
								return micronautMetricExists("catalog")
							}, waitTimeout, pollingInterval).Should(BeTrue())
						},
						func() {
							Eventually(func() bool {
								return micronautMetricExists("orders")
							}, waitTimeout, pollingInterval).Should(BeTrue())
						},
						func() {
							Eventually(func() bool {
								return micronautMetricExists("payment")
							}, waitTimeout, pollingInterval).Should(BeTrue())
						},
						func() {
							Eventually(func() bool {
								return micronautMetricExists("shipping")
							}, waitTimeout, pollingInterval).Should(BeTrue())
						},
						func() {
							Eventually(func() bool {
								return micronautMetricExists("users")
							}, waitTimeout, pollingInterval).Should(BeTrue())
						},
						func() {
							Eventually(coherenceMetricExists, waitTimeout, pollingInterval).Should(BeTrue())
						},
					)
				}
			})
		}
	})
})

var _ = clusterDump.AfterEach(func() {})

// undeploys the application, components, and namespace
var _ = clusterDump.AfterSuite(func() {
	if !skipUndeploy {
		start := time.Now()
		variant := getVariant()
		t.Logs.Info("Undeploy Sock Shop application")
		t.Logs.Info("Delete application")

		Eventually(func() error {
			file, err := pkg.FindTestDataFile("examples/sock-shop/" + variant + "/sock-shop-app.yaml")
			if err != nil {
				return err
			}
			return resource.DeleteResourceFromFileInGeneratedNamespace(file, namespace)
		}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

		t.Logs.Info("Delete components")
		Eventually(func() error {
			file, err := pkg.FindTestDataFile("examples/sock-shop/" + variant + "/sock-shop-comp.yaml")
			if err != nil {
				return err
			}
			return resource.DeleteResourceFromFileInGeneratedNamespace(file, namespace)
		}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

		t.Logs.Info("Wait for sockshop application to be deleted")
		Eventually(func() bool {
			_, err := pkg.GetAppConfig(namespace, sockshopAppName)
			if err != nil && errors.IsNotFound(err) {
				return true
			}
			if err != nil {
				t.Logs.Infof("Error getting sockshop appconfig: %v\n", err.Error())
				pkg.Log(pkg.Info, fmt.Sprintf("Error getting sockshop appconfig: %v\n", err.Error()))
			}
			return false
		}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())

		t.Logs.Info("Delete namespace")
		Eventually(func() error {
			return pkg.DeleteNamespace(namespace)
		}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

		t.Logs.Info("Wait for namespace finalizer to be removed")
		Eventually(func() bool {
			return pkg.CheckNamespaceFinalizerRemoved(namespace)
		}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())

		t.Logs.Info("Wait for sockshop namespace to be deleted")
		Eventually(func() bool {
			_, err := pkg.GetNamespace(namespace)
			if err != nil && errors.IsNotFound(err) {
				return true
			}
			if err != nil {
				t.Logs.Infof("Error getting sockshop namespace: %v\n", err.Error())
				pkg.Log(pkg.Info, fmt.Sprintf("Error getting sockshop namespace: %v\n", err.Error()))
			}
			return false
		}, longWaitTimeout, pollingInterval).Should(BeTrue())
		metrics.Emit(t.Metrics.With("undeployment_elapsed_time", time.Since(start).Milliseconds()))
	}
})

// sockshopPodsRunning checks whether the application pods are ready
func sockshopPodsRunning() bool {
	result, err := pkg.PodsRunning(namespace, expectedPods)
	if err != nil {
		AbortSuite(fmt.Sprintf("One or more pods are not running in the namespace: %v, error: %v", namespace, err))
	}
	return result
}

// appMetricExists checks whether app related metrics are available
func appMetricExists() bool {
	return pkg.MetricsExist("base_jvm_uptime_seconds", "app", "carts-coh")
}

func coherenceMetricExists() bool {
	return pkg.MetricsExist("vendor:coherence_service_messages_local", "role", "Orders")
}

// appComponentMetricExists checks whether component related metrics are available
func appComponentMetricExists() bool {
	return pkg.MetricsExist("vendor_requests_count_total", "app_oam_dev_name", sockshopAppName)
}

// appConfigMetricExists checks whether config metrics are available
func appConfigMetricExists() bool {
	return pkg.MetricsExist("vendor_requests_count_total", "app_oam_dev_component", "orders")
}

// springMetricExists checks whether sample Spring metrics is available for a given component
func springMetricExists(comp string) bool {
	return pkg.MetricsExist(sampleSpringMetric, oamComponent, comp)
}

// micronautMetricExists checks whether sample Micronaut metrics is available for a given component
func micronautMetricExists(comp string) bool {
	return pkg.MetricsExist(sampleMicronautMetric, oamComponent, comp)
}

// getVariant returns the variant of the sock shop application being tested
func getVariant() string {
	// read the variant from the environment - if not specified, default to "helidon"
	variant := os.Getenv("SOCKS_SHOP_VARIANT")
	if variant != "helidon" && variant != "micronaut" && variant != "spring" {
		variant = "helidon"
	}

	return variant
}
