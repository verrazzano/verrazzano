// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package socks

import (
	"context"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	shortWaitTimeout     = 5 * time.Minute
	shortPollingInterval = 10 * time.Second
	waitTimeout          = 10 * time.Minute
	pollingInterval      = 30 * time.Second
)

var sockShop SockShop
var username, password string

// creates the sockshop namespace and applies the components and application yaml
var _ = BeforeSuite(func() {
	username = "username" + strconv.FormatInt(time.Now().Unix(), 10)
	password = b64.StdEncoding.EncodeToString([]byte(time.Now().String()))
	sockShop = NewSockShop(username, password, pkg.Ingress())

	// deploy the application here
	Eventually(func() (*v1.Namespace, error) {
		return pkg.CreateNamespace("sockshop", map[string]string{"verrazzano-managed": "true"})
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

	Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFile("examples/sock-shop/sock-shop-comp.yaml")
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFile("examples/sock-shop/sock-shop-app.yaml")
	}, shortWaitTimeout, shortPollingInterval, "Failed to create Sock Shop application resource").ShouldNot(HaveOccurred())
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
		// checks that all application services are up
		pkg.Concurrently(
			func() {
				Eventually(func() bool {
					return isSockShopServiceReady("catalogue")
				}, waitTimeout, pollingInterval).Should(BeTrue())
			},
			func() {
				Eventually(func() bool {
					return isSockShopServiceReady("carts")
				}, waitTimeout, pollingInterval).Should(BeTrue())
			},
			func() {
				Eventually(func() bool {
					return isSockShopServiceReady("orders")
				}, waitTimeout, pollingInterval).Should(BeTrue())
			},
			func() {
				Eventually(func() bool {
					return isSockShopServiceReady("payment-http")
				}, waitTimeout, pollingInterval).Should(BeTrue())
			},
			func() {
				Eventually(func() bool {
					return isSockShopServiceReady("shipping-http")
				}, waitTimeout, pollingInterval).Should(BeTrue())
			},
			func() {
				Eventually(func() bool {
					return isSockShopServiceReady("user")
				}, waitTimeout, pollingInterval).Should(BeTrue())
			})
	})

	var hostname = ""
	It("Get host from gateway.", func() {
		Eventually(func() string {
			hostname = pkg.GetHostnameFromGateway("sockshop", "")
			return hostname
		}, waitTimeout, shortPollingInterval).Should(Not(BeEmpty()))
	})

	sockShop.SetHostHeader(hostname)

	It("SockShop can be accessed and user can be registered", func() {
		Eventually(func() bool {
			return sockShop.RegisterUser(fmt.Sprintf(registerTemp, username, password), hostname)
		}, waitTimeout, pollingInterval).Should(BeTrue(), "Failed to register SockShop User")
	})

	It("SockShop can log in with default user", func() {
		Eventually(func() (*pkg.HTTPResponse, error) {
			url := fmt.Sprintf("https://%v/login", hostname)
			kubeconfigPath := pkg.GetKubeConfigPathFromEnv()
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
		//https://jira.oraclecorp.com/jira/browse/VZ-1026
	})

	It("Verify '/catalogue' UI endpoint is working.", func() {
		Eventually(func() (*pkg.HTTPResponse, error) {
			url := fmt.Sprintf("https://%s/catalogue", hostname)
			return pkg.GetWebPage(url, hostname)
		}, shortWaitTimeout, shortPollingInterval).Should(And(pkg.HasStatus(http.StatusOK), pkg.BodyContains("For all those leg lovers out there.")))
	})

	Describe("Verify Prometheus scraped metrics", func() {
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
	failed = failed || CurrentGinkgoTestDescription().Failed
})

// undeploys the application, components, and namespace
var _ = AfterSuite(func() {
	if failed {
		pkg.ExecuteClusterDumpWithEnvVarConfig()
	}

	Eventually(func() error {
		return pkg.DeleteNamespace("sockshop")
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	Eventually(func() bool {
		_, err := pkg.GetNamespace("sockshop")
		return err != nil && errors.IsNotFound(err)
	}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())
})

// isSockShopServiceReady checks if the service is ready
func isSockShopServiceReady(name string) bool {
	svc, err := pkg.GetKubernetesClientset().CoreV1().Services("sockshop").Get(context.TODO(), name, metav1.GetOptions{})
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
