// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package socks

import (
	"context"
	b64 "encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

const (
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
	if _, err := pkg.CreateNamespace("sockshop", map[string]string{"verrazzano-managed": "true"}); err != nil {
		Fail(fmt.Sprintf("Failed to create namespace: %v", err))
	}
	if err := pkg.CreateOrUpdateResourceFromFile("examples/sock-shop/sock-shop-comp.yaml"); err != nil {
		Fail(fmt.Sprintf("Failed to create Sock Shop component resources: %v", err))
	}
	if err := pkg.CreateOrUpdateResourceFromFile("examples/sock-shop/sock-shop-app.yaml"); err != nil {
		Fail(fmt.Sprintf("Failed to create Sock Shop application resource: %v", err))
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
		// checks that all application services are up
		pkg.Concurrently(
			func() {
				Eventually(isSockShopServiceReady("catalogue"), waitTimeout, pollingInterval).Should(BeTrue())
			},
			func() {
				Eventually(isSockShopServiceReady("carts"), waitTimeout, pollingInterval).Should(BeTrue())
			},
			func() {
				Eventually(isSockShopServiceReady("orders"), waitTimeout, pollingInterval).Should(BeTrue())
			},
			func() {
				Eventually(isSockShopServiceReady("payment-http"), waitTimeout, pollingInterval).Should(BeTrue())
			},
			func() {
				Eventually(isSockShopServiceReady("shipping-http"), waitTimeout, pollingInterval).Should(BeTrue())
			},
			func() {
				Eventually(isSockShopServiceReady("user"), waitTimeout, pollingInterval).Should(BeTrue())
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
		sockShop.RegisterUser(fmt.Sprintf(registerTemp, username, password), hostname)
	})
	It("SockShop can log in with default user", func() {
		sockShop.Cookies = login(username, password)
	})

	It("SockShop can access Calatogue and choose item", func() {
		webpage := sockShop.ConnectToCatalog(hostname)
		sockShop.VerifyCatalogItems(webpage)
	})

	It("SockShop can add item to cart", func() {
		cat := sockShop.GetCatalogItem(hostname)

		sockShop.AddToCart(cat.Item[0], hostname)
		sockShop.AddToCart(cat.Item[0], hostname)
		sockShop.AddToCart(cat.Item[0], hostname)
		sockShop.AddToCart(cat.Item[1], hostname)
		sockShop.AddToCart(cat.Item[2], hostname)
		sockShop.AddToCart(cat.Item[2], hostname)

		sockShop.CheckCart(cat.Item[0], 3, hostname)
		sockShop.CheckCart(cat.Item[1], 1, hostname)
		sockShop.CheckCart(cat.Item[2], 2, hostname)
	})

	It("SockShop can delete all cart items", func() {
		cartItems := sockShop.GetCartItems(hostname)
		sockShop.DeleteCartItems(cartItems, hostname)
		//cartItems = sockShop.GetCartItems()

		sockShop.CheckCartEmpty(hostname)
	})

	// INFO: Front-End will not allow for complete implementation of this test
	It("SockShop can change address", func() {
		sockShop.ChangeAddress(username, hostname)
	})

	// INFO: Front-End will not allow for complete implementation of this test
	It("SockShop can change payment", func() {
		sockShop.ChangePayment(hostname)
	})

	It("SockShop can retrieve orders", func() {
		//https://jira.oraclecorp.com/jira/browse/VZ-1026
		cat := sockShop.GetCatalogItem(hostname)
		sockShop.AddToCart(cat.Item[0], hostname)
		sockShop.AddToCart(cat.Item[0], hostname)
		sockShop.AddToCart(cat.Item[1], hostname)
		sockShop.AddToCart(cat.Item[2], hostname)
		sockShop.AddToCart(cat.Item[2], hostname)
	})

	It("Verify '/catalogue' UI endpoint is working.", func() {
		Eventually(func() bool {
			url := fmt.Sprintf("https://%s/catalogue", hostname)
			status, content := pkg.GetWebPageWithCABundle(url, hostname)
			return Expect(status).To(Equal(200)) &&
				Expect(content).To(ContainSubstring("For all those leg lovers out there."))
		}, 3*time.Minute, 15*time.Second).Should(BeTrue())
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

// undeploys the application, components, and namespace
var _ = AfterSuite(func() {
	// undeploy the application here
	err := pkg.DeleteResourceFromFile("examples/sock-shop/sock-shop-app.yaml")
	if err != nil {
		Fail(fmt.Sprintf("Could not delete sock shop applications: %v\n", err.Error()))
	}
	err = pkg.DeleteResourceFromFile("examples/sock-shop/sock-shop-comp.yaml")
	if err != nil {
		Fail(fmt.Sprintf("Could not delete sock shop components: %v\n", err.Error()))
	}
	err = pkg.DeleteNamespace("sockshop")
	if err != nil {
		Fail(fmt.Sprintf("Could not delete sock shop namespace: %v\n", err.Error()))
	}
})

// isSockShopServiceReady checks if the service is ready
func isSockShopServiceReady(name string) bool {
	svc, err := pkg.GetKubernetesClientset().CoreV1().Services("sockshop").Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		Fail(fmt.Sprintf("Could not get services %v in sockshop: %v\n", name, err.Error()))
		return false
	}
	if len(svc.Spec.Ports) > 0 {
		return svc.Spec.Ports[0].Port == 80 && svc.Spec.Ports[0].TargetPort == intstr.FromInt(7001)
	}
	return false
}

// login logs in to the sockshop application
func login(username string, password string) []*http.Cookie {
	ingress := pkg.Ingress()
	url := fmt.Sprintf("http://%v/login", ingress)
	httpClient := retryablehttp.NewClient()
	req, _ := retryablehttp.NewRequest("GET", url, nil)
	req.SetBasicAuth(username, password)
	resp, err := httpClient.Do(req)
	if err != nil {
		pkg.Log(Error, err.Error())
		Fail("Could not log into " + url)
	}

	defer resp.Body.Close()

	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		pkg.Log(Error, err.Error())
		Fail("Could not read response body")
	}

	return resp.Cookies()
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
