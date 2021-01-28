// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package sock_shop

import (
	"context"
	b64 "encoding/base64"
	"fmt"
	"github.com/hashicorp/go-retryablehttp"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"net/http"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/util"
)

const (
	waitTimeout     = 10 * time.Minute
	pollingInterval = 30 * time.Second
)

var sockShop SockShop
var username, password string

// creates the oam-sockshop namespace and applies the components and application yaml
var _ = BeforeSuite(func() {
	username = "username" + strconv.FormatInt(time.Now().Unix(), 10)
	password = b64.StdEncoding.EncodeToString([]byte(time.Now().String()))
	sockShop = NewSockShop(username, password, util.Ingress(), "oam-sockshop.example.com")

	// deploy the application here
	if _, err := util.CreateNamespace("oam-sockshop", map[string]string{"verrazzano-managed": "true"}); err != nil {
		Fail(fmt.Sprintf("Failed to create namespace: %v", err))
	}
	if err := util.CreateOrUpdateResourceFromFile("examples/sock-shop/sock-shop-comp.yaml"); err != nil {
		Fail(fmt.Sprintf("Failed to create Sock Shop component resources: %v", err))
	}
	if err := util.CreateOrUpdateResourceFromFile("examples/sock-shop/sock-shop-app.yaml"); err != nil {
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
		util.Concurrently(
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

	It("SockShop can be accessed and user can be registered", func() {
		sockShop.RegisterUser(fmt.Sprintf(registerTemp, username, password))
	})
	It("SockShop can log in with default user", func() {
		sockShop.Cookies = login(username, password)
	})

	It("SockShop can access Calatogue and choose item", func() {
		webpage := sockShop.ConnectToCatalog()
		sockShop.VerifyCatalogItems(webpage)
	})

	It("SockShop can add item to cart", func() {
		cat := sockShop.GetCatalogItem()

		sockShop.AddToCart(cat.Item[0])
		sockShop.AddToCart(cat.Item[0])
		sockShop.AddToCart(cat.Item[0])
		sockShop.AddToCart(cat.Item[1])
		sockShop.AddToCart(cat.Item[2])
		sockShop.AddToCart(cat.Item[2])

		sockShop.CheckCart(cat.Item[0], 3)
		sockShop.CheckCart(cat.Item[1], 1)
		sockShop.CheckCart(cat.Item[2], 2)
	})

	It("SockShop can delete all cart items", func() {
		cartItems := sockShop.GetCartItems()
		sockShop.DeleteCartItems(cartItems)
		cartItems = sockShop.GetCartItems()

		sockShop.CheckCartEmpty()
	})

	// INFO: Front-End will not allow for complete implementation of this test
	It("SockShop can change address", func() {
		sockShop.ChangeAddress(username)
	})

	// INFO: Front-End will not allow for complete implementation of this test
	It("SockShop can change payment", func() {
		sockShop.ChangePayment()
	})

	It("SockShop can retrieve orders", func() {
		//https://jira.oraclecorp.com/jira/browse/VZ-1026
		cat := sockShop.GetCatalogItem()
		sockShop.AddToCart(cat.Item[0])
		sockShop.AddToCart(cat.Item[0])
		sockShop.AddToCart(cat.Item[1])
		sockShop.AddToCart(cat.Item[2])
		sockShop.AddToCart(cat.Item[2])
	})

	It("Verify '/catalogue' UI endpoint is working.", func() {
		Eventually(func() bool {
			ipAddress := util.Ingress()
			url := fmt.Sprintf("http://%s/catalogue", ipAddress)
			host := "oam-sockshop.example.com"
			status, content := util.GetWebPageWithCABundle(url, host)
			return Expect(status).To(Equal(200)) &&
				Expect(content).To(ContainSubstring("For all those leg lovers out there."))
		}, 3*time.Minute, 15*time.Second).Should(BeTrue())
	})

	Describe("Verify Prometheus scraped metrics", func() {
		It("Retrieve Prometheus scraped metrics", func() {
			util.Concurrently(
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
	err := util.DeleteResourceFromFile("examples/sock-shop/sock-shop-app.yaml")
	if err != nil {
		Fail(fmt.Sprintf("Could not delete sock shop applications: %v\n", err.Error()))
	}
	err = util.DeleteResourceFromFile("examples/sock-shop/sock-shop-comp.yaml")
	if err != nil {
		Fail(fmt.Sprintf("Could not delete sock shop components: %v\n", err.Error()))
	}
	err = util.DeleteNamespace("oam-sockshop")
	if err != nil {
		Fail(fmt.Sprintf("Could not delete sock shop namespace: %v\n", err.Error()))
	}
})

// isSockShopServiceReady checks if the service is ready
func isSockShopServiceReady(name string) bool {
	svc, err := util.GetKubernetesClientset().CoreV1().Services("oam-sockshop").Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		Fail(fmt.Sprintf("Could not get services %v in oam-sockshop: %v\n", name, err.Error()))
		return false
	}
	if len(svc.Spec.Ports) > 0 {
		return svc.Spec.Ports[0].Port == 80 && svc.Spec.Ports[0].TargetPort == intstr.FromInt(7001)
	}
	return false
}

// login logs in to the sockshop application
func login(username string, password string) []*http.Cookie {
	ingress := util.Ingress()
	url := fmt.Sprintf("http://%v/login", ingress)
	httpClient := retryablehttp.NewClient()
	req, _ := retryablehttp.NewRequest("GET", url, nil)
	req.SetBasicAuth(username, password)
	resp, err := httpClient.Do(req)
	if err != nil {
		util.Log(Error, err.Error())
		Fail("Could not log into " + url)
	}

	defer resp.Body.Close()

	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		util.Log(Error, err.Error())
		Fail("Could not read response body")
	}

	return resp.Cookies()
}

// sockshopPodsRunning checks whether the application pods are ready
func sockshopPodsRunning() bool {
	return util.PodsRunning("oam-sockshop", expectedPods)
}

// appMetricsExists checks whether app related metrics are available
func appMetricsExists() bool {
	return metricsExist("base_jvm_uptime_seconds", "cluster", "SockShop")
}

// appComponentMetricsExists checks whether component related metrics are available
func appComponentMetricsExists() bool {
	return metricsExist("vendor_requests_count_total", "app_oam_dev_name", "catalog-appconf")
}

// appConfigMetricsExists checks whether config metrics are available
func appConfigMetricsExists() bool {
	return metricsExist("vendor_requests_count_total", "app_oam_dev_component", "orders")
}

// findMetric parses a prometheus response to find a specified metric value
func findMetric(metrics []interface{}, key, value string) bool {
	for _, metric := range metrics {
		if util.Jq(metric, "metric", key) == value {
			return true
		}
	}
	return false
}

// metricsExist validates the availability of a specified metric
func metricsExist(metricsName, key, value string) bool {
	metrics := util.JTq(util.QueryMetric(metricsName), "data", "result").([]interface{})
	if metrics != nil {
		return findMetric(metrics, key, value)
	} else {
		return false
	}
}
