// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package socks

import (
	"context"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"k8s.io/apimachinery/pkg/api/errors"
	"net/http"
	"os"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	shortWaitTimeout     = 5 * time.Minute
	shortPollingInterval = 10 * time.Second
	waitTimeout          = 10 * time.Minute
	pollingInterval      = 30 * time.Second
	sockshopAppName      = "sockshop-appconfig"
	sockshopNamespace    = "sockshop"
)

var sockShop SockShop
var username, password string

// creates the sockshop namespace and applies the components and application yaml
var _ = BeforeSuite(func() {
	username = "username" + strconv.FormatInt(time.Now().Unix(), 10)
	password = b64.StdEncoding.EncodeToString([]byte(time.Now().String()))
	sockShop = NewSockShop(username, password, pkg.Ingress())

	variant := getVariant()
	GinkgoWriter.Write([]byte(fmt.Sprintf("*** Socks shop test is running against variant: %s\n", variant)))

	if !skipDeploy {
		// deploy the application here
		Eventually(func() (*v1.Namespace, error) {
			return pkg.CreateNamespace("sockshop", map[string]string{"verrazzano-managed": "true"})
		}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

		Eventually(func() error {
			return pkg.CreateOrUpdateResourceFromFile("examples/sock-shop/" + variant + "/sock-shop-comp.yaml")
		}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

		Eventually(func() error {
			return pkg.CreateOrUpdateResourceFromFile("examples/sock-shop/" + variant + "/sock-shop-app.yaml")
		}, shortWaitTimeout, shortPollingInterval, "Failed to create Sock Shop application resource").ShouldNot(HaveOccurred())
	}
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

var _ = Describe("Sock Shop Application", func() {
	It("Verify application pods are running", func() {
		// checks that all pods are up and running
		Eventually(sockshopPodsRunning, waitTimeout, pollingInterval).Should(BeTrue())
	})

	var hostname = ""
	var err error
	It("Get host from gateway.", func() {
		Eventually(func() (string, error) {
			hostname, err = k8sutil.GetHostnameFromGateway("sockshop", "")
			return hostname, err
		}, waitTimeout, shortPollingInterval).Should(Not(BeEmpty()))
	})

	sockShop.SetHostHeader(hostname)

	It("SockShop can be accessed and user can be registered", func() {
		Eventually(func() bool {
			return sockShop.RegisterUser(fmt.Sprintf(registerTemp, username, password), hostname)
		}, waitTimeout, pollingInterval).Should(BeTrue(), "Failed to register SockShop User")
	})

	It("SockShop can log in with default user", func() {
		kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
		Expect(err).ShouldNot(HaveOccurred())
		Eventually(func() (*pkg.HTTPResponse, error) {
			url := fmt.Sprintf("https://%v/login", hostname)
			return pkg.GetWebPageWithBasicAuth(url, hostname, username, password, kubeconfigPath)
		}, waitTimeout, pollingInterval).Should(pkg.HasStatus(http.StatusOK))

	})

	It("SockShop can add item to cart", func() {
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

	It("SockShop can delete all cart items", func() {
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
	It("SockShop can change address", func() {
		var response *pkg.HTTPResponse
		Eventually(func() (*pkg.HTTPResponse, error) {
			var err error
			response, err = sockShop.ChangeAddress(username, hostname)
			return response, err
		}, shortWaitTimeout, shortPollingInterval).Should(pkg.HasStatus(200))

		sockShop.CheckAddress(response, username)
	})

	// INFO: Front-End will not allow for complete implementation of this test
	It("SockShop can change payment", func() {
		Eventually(func() (*pkg.HTTPResponse, error) {
			return sockShop.ChangePayment(hostname)
		}, shortWaitTimeout, shortPollingInterval).Should(pkg.HasStatus(200))
	})

	PIt("SockShop can retrieve orders", func() {
		//TODO
	})

	It("Verify '/catalogue' UI endpoint is working.", func() {
		Eventually(func() (*pkg.HTTPResponse, error) {
			url := fmt.Sprintf("https://%s/catalogue", hostname)
			return pkg.GetWebPage(url, hostname)
		}, shortWaitTimeout, shortPollingInterval).Should(And(pkg.HasStatus(http.StatusOK), pkg.BodyContains("For all those leg lovers out there.")))
	})

	// this is marked pending until VZ-3760 is fixed
	PDescribe("Verify Prometheus scraped metrics", func() {
		It("Retrieve Prometheus scraped metrics", func() {
			pkg.Concurrently(
				func() {
					Eventually(appMetricsExists, waitTimeout, pollingInterval).Should(BeTrue())
				},
				func() {
					Eventually(appComponentMetricsExists, waitTimeout, pollingInterval).Should(BeTrue())
				},
				func() {
					Eventually(appConfigMetricsExists, waitTimeout, pollingInterval).Should(BeTrue())
				},
			)
		})
	})

})

var failed = false
var _ = AfterEach(func() {
	failed = failed || CurrentSpecReport().Failed()
})

// undeploys the application, components, and namespace
var _ = AfterSuite(func() {
	if failed {
		pkg.ExecuteClusterDumpWithEnvVarConfig()
	}
	if !skipUndeploy {
		variant := getVariant()
		pkg.Log(pkg.Info, "Undeploy Sock Shop application")
		pkg.Log(pkg.Info, "Delete application")
		Eventually(func() error {
			return pkg.DeleteResourceFromFile("examples/sock-shop/" + variant + "/sock-shop-app.yaml")
		}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

		pkg.Log(pkg.Info, "Delete components")
		Eventually(func() error {
			return pkg.DeleteResourceFromFile("examples/sock-shop/" + variant + "/sock-shop-comp.yaml")
		}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

		pkg.Log(pkg.Info, "Wait for sockshop application to be deleted")
		Eventually(func() bool {
			_, err := pkg.GetAppConfig(sockshopNamespace, sockshopAppName)
			if err == nil {
				return false
			}
			if errors.IsNotFound(err) {
				return true
			}
			pkg.Log(pkg.Info, fmt.Sprintf("Error getting sockshop appconfig: %v\n", err.Error()))
			return false
		}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

		pkg.Log(pkg.Info, "Delete namespace")
		Eventually(func() error {
			return pkg.DeleteNamespace(sockshopNamespace)
		}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

		pkg.Log(pkg.Info, "Wait for sockshop namespace to be deleted")
		Eventually(func() bool {
			_, err := pkg.DoesNamespaceExist(sockshopNamespace)
			if err == nil {
				return false
			}
			if errors.IsNotFound(err) {
				return true
			}
			pkg.Log(pkg.Info, fmt.Sprintf("Error getting sockshop namespace: %v\n", err.Error()))
			return false
		}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())
	}
	if failed {
		pkg.ExecuteClusterDumpWithEnvVarConfig()
	}
})

// isSockShopServiceReady checks if the service is ready
func isSockShopServiceReady(name string) bool {
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		pkg.Log(pkg.Info, fmt.Sprintf("Could not get Kubernetes clientset: %v\n", err.Error()))
		return false
	}
	svc, err := clientset.CoreV1().Services("sockshop").Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		pkg.Log(pkg.Info, fmt.Sprintf("Could not get services %v in sockshop: %v\n", name, err.Error()))
		return false
	}
	if len(svc.Spec.Ports) > 0 {
		return svc.Spec.Ports[0].Port == 80 && svc.Spec.Ports[0].TargetPort == intstr.FromInt(7001)
	}
	return false
}

// sockshopPodsRunning checks whether the application pods are ready
func sockshopPodsRunning() bool {
	return pkg.PodsRunning("sockshop", expectedPods)
}

// appMetricsExists checks whether app related metrics are available
func appMetricsExists() bool {
	return pkg.MetricsExist("base_jvm_uptime_seconds", "cluster", "SockShop")
}

// appComponentMetricsExists checks whether component related metrics are available
func appComponentMetricsExists() bool {
	return pkg.MetricsExist("vendor_requests_count_total", "app_oam_dev_name", "sockshop-appconf")
}

// appConfigMetricsExists checks whether config metrics are available
func appConfigMetricsExists() bool {
	return pkg.MetricsExist("vendor_requests_count_total", "app_oam_dev_component", "orders")
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
