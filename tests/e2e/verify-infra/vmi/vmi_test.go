// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vmi_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/verrazzano/verrazzano/tests/e2e/pkg/vmi"

	"github.com/hashicorp/go-retryablehttp"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"

	//v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func vmiIngressURLs() (map[string]string, error) {
	ingressList, err := pkg.GetKubernetesClientset().ExtensionsV1beta1().Ingresses("verrazzano-system").List(context.TODO(), v1.ListOptions{})
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
	crd, err := pkg.ApiExtensionsClientSet().CustomResourceDefinitions().Get(context.TODO(), "verrazzanomonitoringinstances.verrazzano.io", v1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return crd, nil
}

var (
	creds                  *pkg.UsernamePassword
	vmiCRD                 *apiextensionsv1beta1.CustomResourceDefinition
	ingressURLs            map[string]string
	elastic                *vmi.Elastic
	sysVmiHttpClient       *retryablehttp.Client
	waitTimeout            = 10 * time.Minute
	pollingInterval        = 30 * time.Second
	elasticWaitTimeout     = 2 * time.Minute
	elasticPollingInterval = 5 * time.Second
)

var _ = ginkgo.BeforeSuite(func() {
	var err error

	//env = NewVerrazzanoEnvironmentFromConfig(GetTestConfig())

	ingressURLs, err = vmiIngressURLs()
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Error retrieving system VMI ingress URLs: %v", err))
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

	sysVmiHttpClient = pkg.GetSystemVmiHttpClient()
})

var _ = ginkgo.Describe("VMI", func() {

	ginkgo.It("api server should be accessible", func() {
		assertIngressURL("vmi-system-api")
	})

	ginkgo.It("Elasticsearch endpoint should be accessible", func() {
		elasticPodsRunning := func() bool {
			return pkg.PodsRunning("verrazzano-system", []string{"vmi-system-es-master"})
		}
		gomega.Eventually(elasticPodsRunning, waitTimeout, pollingInterval).Should(gomega.BeTrue(), "pods did not all show up")
		gomega.Eventually(elasticTlsSecret, elasticWaitTimeout, elasticPollingInterval).Should(gomega.BeTrue(), "tls-secret did not show up")
		//gomega.Eventually(elasticCertificate, elasticWaitTimeout, elasticPollingInterval).Should(gomega.BeTrue(), "certificate did not show up")
		gomega.Eventually(elasticIngress, elasticWaitTimeout, elasticPollingInterval).Should(gomega.BeTrue(), "ingress did not show up")
		assertIngressURL("vmi-system-es-ingest")
		pkg.Concurrently(
			func() {
				gomega.Eventually(elasticConnected, elasticWaitTimeout, elasticPollingInterval).Should(gomega.BeTrue(), "never connected")
			},
			func() {
				gomega.Eventually(elasticIndicesCreated, elasticWaitTimeout, elasticPollingInterval).Should(gomega.BeTrue(), "indices never created")
			},
		)
	})

	ginkgo.It("Elasticsearch filebeat Index should be accessible", func() {
		gomega.Eventually(func() bool {
			return pkg.LogRecordFound("vmo-local-filebeat-"+time.Now().Format("2006.01.02"),
				time.Now().Add(-24*time.Hour),
				map[string]string{
					"beat.version":      "6.8.3"})
		}, 5*time.Minute, 10*time.Second).Should(gomega.BeTrue(), "Expected to find a filebeat log record")
	})

	ginkgo.It("Elasticsearch journalbeat Index should be accessible", func() {
		gomega.Eventually(func() bool {
			return pkg.LogRecordFound("vmo-local-journalbeat-"+time.Now().Format("2006.01.02"),
				time.Now().Add(-24*time.Hour),
				map[string]string{
					"beat.version":      "6.8.3"})
		}, 5*time.Minute, 10*time.Second).Should(gomega.BeTrue(), "Expected to find a journalbeat log record")
	})

	ginkgo.It("Kibana endpoint should be accessible", func() {
		assertIngressURL("vmi-system-kibana")
	})

	ginkgo.It("Prometheus endpoint should be accessible", func() {
		assertIngressURL("vmi-system-prometheus")

	})

	ginkgo.It("Prometheus push gateway should be accessible", func() {
		assertIngressURL("vmi-system-prometheus-gw")
	})

	ginkgo.It("Grafana endpoint should be accessible", func() {
		gomega.Expect(ingressURLs).To(gomega.HaveKey("vmi-system-grafana"), "Ingress vmi-system-grafana not found")
		sysVmiHttpClient.HTTPClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
		url := ingressURLs["vmi-system-grafana"]
		resp, err := sysVmiHttpClient.Get(url)
		gomega.Expect(err).NotTo(gomega.HaveOccurred(), "GET %s", url)
		gomega.Expect(resp.StatusCode).To(gomega.Equal(http.StatusFound), "GET %s", url)
		gomega.Expect(resp.Header.Get("location")).To(gomega.Equal("/login"))
	})

	ginkgo.It("System Node Exporter dashboard should be installed in Grafana", func() {
		browseGrafanaDashboard("Host%20Metrics")
	})

	ginkgo.It("Default dashboard should be installed in System Grafana for shared VMI", func() {
		fmt.Println("Running acceptance test for shared VMI ...")
		DefaultDashboards := []string{
			"WebLogic%20Server%20Dashboard",
			"Coherence%20Elastic%20Data%20Summary%20Dashboard",
			"Coherence%20Persistence%20Summary%20Dashboard",
			"Coherence%20Cache%20Details%20Dashboard",
			"Coherence%20Members%20Summary%20Dashboard",
			"Coherence%20Kubernetes%20Summary%20Dashboard",
			"Coherence%20Dashboard%20Main",
			"Coherence%20Caches%20Summary%20Dashboard",
			"Coherence%20Service%20Details%20Dashboard",
			"Coherence%20Proxy%20Servers%20Summary%20Dashboard",
			"Coherence%20Federation%20Details%20Dashboard",
			"Coherence%20Federation%20Summary%20Dashboard",
			"Coherence%20Services%20Summary%20Dashboard",
			"Coherence%20HTTP%20Servers%20Summary%20Dashboard",
			"Coherence%20Proxy%20Server%20Detail%20Dashboard",
			"Coherence%20Alerts%20Dashboard",
			"Coherence%20Member%20Details%20Dashboard",
			"Coherence%20Machines%20Summary%20Dashboard",
		}
		for _, value := range DefaultDashboards {
			browseGrafanaDashboard(value)
		}
	})
})

func jq(node interface{}, path ...string) interface{} {
	for _, p := range path {
		node = node.(map[string]interface{})[p]
	}
	return node
}

func assertIngressURL(key string) {
	gomega.Expect(ingressURLs).To(gomega.HaveKey(key), fmt.Sprintf("Ingress %s not found", key))
	assertUnAuthorized := assertURLAccessibleAndUnauthorized(ingressURLs[key])
	assertAuthorized := assertURLAccessibleAndAuthorized(ingressURLs[key])
	pkg.Concurrently(
		func() {
			gomega.Eventually(assertUnAuthorized, waitTimeout, pollingInterval).Should(gomega.BeTrue())
		},
		func() {
			gomega.Eventually(assertAuthorized, waitTimeout, pollingInterval).Should(gomega.BeTrue())
		},
	)
}

func assertURLAccessibleAndAuthorized(url string) bool {
	sysVmiHttpClient.HTTPClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		pkg.Log(pkg.Info, fmt.Sprintf("assertURLAccessibleAndAuthorized redirect %v", req))
		return nil
	}
	return pkg.AssertURLAccessibleAndAuthorized(sysVmiHttpClient, url, creds)
}

func assertURLAccessibleAndUnauthorized(url string) bool {
	resp, err := sysVmiHttpClient.Get(url)
	if err != nil {
		return false
	}
	pkg.Log(pkg.Info, fmt.Sprintf("assertURLAccessibleAndUnauthorized %v Response:%v Error:%v", url, resp.StatusCode, err))
	return resp.StatusCode == http.StatusUnauthorized
}

func elasticIndicesCreated() bool {
	b, _ := gomega.ContainElements(".kibana_1").Match(elastic.ListIndices())
	return b
}

func elasticConnected() bool {
	if elastic.Connect() {
		return true
	} else {
		return false
	}
}

func elasticTlsSecret() bool {
	return elastic.CheckTlsSecret()
}

//func elasticCertificate() bool {
//	return elastic.CheckCertificate()
//}

func elasticIngress() bool {
	return elastic.CheckIngress()
}

func browseGrafanaDashboard(url string) error {
	searchURL := fmt.Sprintf("%sapi/search?query=%s", ingressURLs["vmi-system-grafana"], url)
	fmt.Println("Grafana URL in browseGrafanaDashboard ", searchURL)
	sysVmiHttpClient.HTTPClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	req, err := retryablehttp.NewRequest("GET", searchURL, nil)
	req.SetBasicAuth(creds.Username, creds.Password)
	resp, err := sysVmiHttpClient.Do(req)

	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "GET %s", searchURL)
	gomega.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK), "GET %s", searchURL)

	// assert that there is a single item in response
	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		ginkgo.Fail("Unable to read body from response " + err.Error())
	}
	var response []map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		ginkgo.Fail("Unable to unmarshal response: " + err.Error())
	}
	if len(response) != 1 {
		ginkgo.Fail(fmt.Sprintf("Expected a dashboard in response to system vmi dashboard query but received: %v", len(response)))
	}
	return nil
}
