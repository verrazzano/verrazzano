package vmi_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gitlab-odx.oracledx.com/verrazzano/verrazzano-acceptance-test-suite/util"
	. "gitlab-odx.oracledx.com/verrazzano/verrazzano-acceptance-test-suite/util"

	//v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func vmiIngressURLs(env util.VerrazzanoEnvironment) (map[string]string, error) {
	ingressList, err := env.ManagementCluster.ClientSet().ExtensionsV1beta1().Ingresses("verrazzano-system").List(v1.ListOptions{})
	if err != nil {
		return nil, err
	}

	ingressURLs := make(map[string]string)

	for _, ingress := range ingressList.Items {
		// Currently there is a mismatch between the ingress rule hostname and TLS hostname, TLS name seems incorrect
		/*
			var tls []v1beta1.IngressTLS = ingress.Spec.TLS
			if len(tls) != 1 {
				return nil, fmt.Errorf("Expected ingress %s in namespace %s to have 1 TLS entry, but had %v",
					ingress.Name, ingress.Namespace, tls)
			}
			var hostnames []string = tls[0].Hosts
			if len(hostnames) != 1 {
				return nil, fmt.Errorf("Expected ingress %s in namespace %s to have 1 TLS with 1 hostname multiple hostnames: %v",
					ingress.Name, ingress.Namespace, hostnames)
			}
			ingressURLs[ingress.Name] = fmt.Sprintf("https://%s/", hostnames[0])
		*/
		var ingressRules []v1beta1.IngressRule = ingress.Spec.Rules
		if len(ingressRules) != 1 {
			return nil, fmt.Errorf("Expected ingress %s in namespace %s to have 1 ingress rule, but had %v",
				ingress.Name, ingress.Namespace, ingressRules)
		}
		ingressURLs[ingress.Name] = fmt.Sprintf("https://%s/", ingressRules[0].Host)
	}
	return ingressURLs, nil
}

func verrazzanoMonitoringInstanceCRD(env util.VerrazzanoEnvironment) (*apiextensionsv1beta1.CustomResourceDefinition, error) {
	crd, err := env.ManagementCluster.ApiExtensionsClientSet().CustomResourceDefinitions().Get("verrazzanomonitoringinstances.verrazzano.io", v1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return crd, nil
}

var (
	env                    VerrazzanoEnvironment
	creds                  *UsernamePassword
	vmiCRD                 *apiextensionsv1beta1.CustomResourceDefinition
	ingressURLs            map[string]string
	elastic                *Elastic
	sysVmiHttpClient       *retryablehttp.Client
	waitTimeout            = 10 * time.Minute
	pollingInterval        = 30 * time.Second
	elasticWaitTimeout     = 2 * time.Minute
	elasticPollingInterval = 5 * time.Second
)

var _ = BeforeSuite(func() {
	var err error

	env = NewVerrazzanoEnvironmentFromConfig(GetTestConfig())

	ingressURLs, err = vmiIngressURLs(env)
	if err != nil {
		Fail(fmt.Sprintf("Error retrieving system VMI ingress URLs: %v", err))
	}

	vmiCRD, err = verrazzanoMonitoringInstanceCRD(env)
	if err != nil {
		Fail(fmt.Sprintf("Error retrieving system VMI CRD: %v", err))
	}

	creds, err = GetSystemVMICredentials(env)
	if err != nil {
		Fail(fmt.Sprintf("Error retrieving system VMI credentials: %v", err))
	}
	elastic = env.GetElastic("system")

	sysVmiHttpClient = GetSystemVmiHttpClient()
})

var _ = Describe("VMI", func() {

	It("api server should be accessible", func() {
		assertIngressURL("vmi-system-api")
	})

	It("Elasticsearch endpoint should be accessible", func() {
		elasticPodsRunning := func() bool {
			return elastic.PodsRunning()
		}
		Eventually(elasticPodsRunning, waitTimeout, pollingInterval).Should(BeTrue())
		Eventually(elasticTlsSecret, elasticWaitTimeout, elasticPollingInterval).Should(BeTrue())
		Eventually(elasticCertificate, elasticWaitTimeout, elasticPollingInterval).Should(BeTrue())
		Eventually(elasticIngress, elasticWaitTimeout, elasticPollingInterval).Should(BeTrue())
		Eventually(elasticLookup, elasticWaitTimeout, elasticPollingInterval).Should(BeTrue())
		Concurrently(
			func() {
				Eventually(elasticConnected, elasticWaitTimeout, elasticPollingInterval).Should(BeTrue())
			},
			func() {
				Eventually(elasticIndicesCreated, elasticWaitTimeout, elasticPollingInterval).Should(BeTrue())
			},
		)
	})

    It("Elasticsearch filebeat Index should be accessible", func() {
		//Disabled for KiND. See VZ-1918
		if env.ManagementCluster.ClusterType == CLUSTER_TYPE_KIND {
			return
		}

		var filebeatIndexName string
		var filebeatIndex interface{}
		for name, esIndex := range elastic.GetIndices() {
			if strings.Contains(name, "filebeat") {
				filebeatIndexName = name
				filebeatIndex = esIndex
			}
		}
		Expect(filebeatIndexName).NotTo(Equal(""), "Found filebeatIndex")
		dynamicTemplates := jq(filebeatIndex, "mappings", "dynamic_templates").([]interface{})
		var messageField interface{}
		for _, dynamicTemp := range dynamicTemplates {
			found := dynamicTemp.(map[string]interface{})["message_field"]
			if found != nil {
				messageField = found
			}
		}
		messagePath := jq(messageField, "path_match")
		Expect(messagePath).To(Equal("log.message"), "log message path")
		messageType := jq(messageField, "mapping", "type")
		Expect(messageType).To(Equal("text"), "log message type")

		searchResult := elastic.Search(filebeatIndexName,
			Field{"kubernetes.namespace", "verrazzano-system"},
			Field{"kubernetes.container.name", "verrazzano-monitoring-operator"})
		hits := jq(searchResult, "hits", "hits")
		for _, hit := range hits.([]interface{}) {
			caller := jq(hit, "_source", "log", "caller")
			Expect(caller).NotTo(Equal(""), "caller field should be found on log message")
		}
	})

	It("Kibana endpoint should be accessible", func() {
		assertIngressURL("vmi-system-kibana")
	})

	It("Prometheus endpoint should be accessible", func() {
		assertIngressURL("vmi-system-prometheus")

	})

	It("Prometheus push gateway should be accessible", func() {
		assertIngressURL("vmi-system-prometheus-gw")
	})

	It("Grafana endpoint should be accessible", func() {
		Expect(ingressURLs).To(HaveKey("vmi-system-grafana"), "Ingress vmi-system-grafana not found")
		sysVmiHttpClient.HTTPClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
		url := ingressURLs["vmi-system-grafana"]
		resp, err := sysVmiHttpClient.Get(url)
		Expect(err).NotTo(HaveOccurred(), "GET %s", url)
		Expect(resp.StatusCode).To(Equal(http.StatusFound), "GET %s", url)
		Expect(resp.Header.Get("location")).To(Equal("/login"))
	})

	It("System Node Exporter dashboard should be installed in Grafana", func() {
		browseGrafanaDashboard("Host%20Metrics")
	})

	It("Default dashboard should be installed in System Grafana for shared VMI", func() {
		if env.IsUsingSharedVMI() {
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
	Expect(ingressURLs).To(HaveKey(key), fmt.Sprintf("Ingress %s not found", key))
	assertURLAccessibleAndUnauthorized(ingressURLs[key])
	assertURLAccessibleAndAuthorized(ingressURLs[key])
}
func assertURLAccessibleAndAuthorized(url string) {

	AssertURLAccessibleAndAuthorized(sysVmiHttpClient, url, creds)
}

func assertURLAccessibleAndUnauthorized(url string) {
	resp, err := sysVmiHttpClient.Get(url)
	Expect(err).NotTo(HaveOccurred(), "GET %s", url)
	Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized), "GET %s", url)
}

func elasticIndicesCreated() bool {
	b, _ := ContainElements(".kibana_1").Match(elastic.ListIndices())
	return b
}

func elasticConnected() bool {
	if elastic.Connect() {
		assertIngressURL("vmi-system-es-ingest")
		return true
	} else {
		return false
	}
}

func elasticLookup() bool {
	return elastic.LookupHost()
}

func elasticTlsSecret() bool {
	return elastic.CheckTlsSecret()
}

func elasticCertificate() bool {
	return elastic.CheckCertificate()
}

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

	Expect(err).NotTo(HaveOccurred(), "GET %s", searchURL)
	Expect(resp.StatusCode).To(Equal(http.StatusOK), "GET %s", searchURL)

	// assert that there is a single item in response
	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		Fail("Unable to read body from response " + err.Error())
	}
	var response []map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		Fail("Unable to unmarshal response: " + err.Error())
	}
	if len(response) != 1 {
		Fail(fmt.Sprintf("Expected a dashboard in response to system vmi dashboard query but received: %v", len(response)))
	}
	return nil
}
