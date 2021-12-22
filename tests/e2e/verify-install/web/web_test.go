// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package web_test

import (
	"context"
	"crypto/x509"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"k8s.io/client-go/kubernetes"

	"github.com/hashicorp/go-retryablehttp"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	networkingv1 "k8s.io/api/networking/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	waitTimeout     = 3 * time.Minute
	pollingInterval = 5 * time.Second
)

var serverURL string
var isManagedClusterProfile bool
var isTestSupported bool
var _ = BeforeSuite(func() {
	var ingress *networkingv1.Ingress
	var clientset *kubernetes.Clientset
	isManagedClusterProfile = pkg.IsManagedClusterProfile()
	if isManagedClusterProfile {
		return
	}

	Eventually(func() (*kubernetes.Clientset, error) {
		var err error
		clientset, err = k8sutil.GetKubernetesClientset()
		return clientset, err
	}, waitTimeout, pollingInterval).ShouldNot(BeNil())
	Eventually(func() (*networkingv1.Ingress, error) {
		var err error
		ingress, err = clientset.NetworkingV1().Ingresses("verrazzano-system").Get(context.TODO(), "verrazzano-ingress", v1.GetOptions{})
		return ingress, err
	}, waitTimeout, pollingInterval).ShouldNot(BeNil())

	Expect(len(ingress.Spec.Rules)).To(Equal(1))
	ingressRules := ingress.Spec.Rules
	serverURL = fmt.Sprintf("https://%s/", ingressRules[0].Host)
	var err error
	isTestSupported, err = pkg.IsVerrazzanoMinVersion("1.1.0")
	if err != nil {
		Fail(err.Error())
	}
})

var _ = Describe("Verrazzano Web UI", func() {
	When("the console UI is configured", func() {
		It("can be accessed", func() {
			if !isManagedClusterProfile {
				Eventually(func() (*pkg.HTTPResponse, error) {
					return pkg.GetWebPage(serverURL, "")
				}, waitTimeout, pollingInterval).Should(And(pkg.HasStatus(http.StatusOK), pkg.BodyNotEmpty()))
			}
		})

		It("has the correct SSL certificate", func() {
			if !isManagedClusterProfile {
				var certs []*x509.Certificate
				Eventually(func() ([]*x509.Certificate, error) {
					var err error
					certs, err = pkg.GetCertificates(serverURL)
					return certs, err
				}, waitTimeout, pollingInterval).ShouldNot(BeNil())

				// There will normally be several certs, but we only need to check the
				// first one -- might want to refactor the checks out into a pkg.IsCertValid()
				// function so we can use it from other test suites too??
				pkg.Log(pkg.Debug, "Issuer Common Name: "+certs[0].Issuer.CommonName)
				pkg.Log(pkg.Debug, "Subject Common Name: "+certs[0].Subject.CommonName)
				pkg.Log(pkg.Debug, "Not Before: "+certs[0].NotBefore.String())
				pkg.Log(pkg.Debug, "Not After: "+certs[0].NotAfter.String())
				Expect(time.Now().After(certs[0].NotBefore)).To(BeTrue())
				Expect(time.Now().Before(certs[0].NotAfter)).To(BeTrue())
			}
		})

		It("should return no Server header", func() {
			if !isManagedClusterProfile {
				kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
				Expect(err).ShouldNot(HaveOccurred())
				httpClient, err := pkg.GetVerrazzanoHTTPClient(kubeconfigPath)
				Expect(err).ShouldNot(HaveOccurred())
				req, err := retryablehttp.NewRequest("GET", serverURL, nil)
				Expect(err).ShouldNot(HaveOccurred())
				// There should be no server header found and no errors should occur during the request
				Eventually(func() error {
					return pkg.CheckStatusAndResponseHeaderAbsent(httpClient, req, "server", 0)
				}).Should(BeNil())
			}
		})

		It("should not return CORS Access-Control-Allow-Origin header when no Origin header is provided", func() {
			if !isManagedClusterProfile {
				kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
				Expect(err).ShouldNot(HaveOccurred())
				httpClient, err := pkg.GetVerrazzanoHTTPClient(kubeconfigPath)
				Expect(err).ShouldNot(HaveOccurred())
				req, err := retryablehttp.NewRequest("GET", serverURL, nil)
				Expect(err).ShouldNot(HaveOccurred())
				// HTTP Access-Control-Allow-Origin header should never be returned.
				Eventually(func() error {
					return pkg.CheckStatusAndResponseHeaderAbsent(
						httpClient, req, "access-control-allow-origin", 0)
				}).Should(BeNil())
			}
		})

		It("should not return CORS Access-Control-Allow-Origin header when Origin: * is provided", func() {
			if !isManagedClusterProfile {
				kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
				Expect(err).ShouldNot(HaveOccurred())
				httpClient, err := pkg.GetVerrazzanoHTTPClient(kubeconfigPath)
				Expect(err).ShouldNot(HaveOccurred())
				req, err := retryablehttp.NewRequest("GET", serverURL, nil)
				req.Header.Add("Origin", "*")
				Expect(err).ShouldNot(HaveOccurred())
				Eventually(func() error {
					return pkg.CheckStatusAndResponseHeaderAbsent(
						httpClient, req, "access-control-allow-origin", 0)
				}).Should(BeNil())
			}
		})

		It("should not return CORS Access-Control-Allow-Origin header when Origin: null is provided", func() {
			if !isManagedClusterProfile {
				kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
				Expect(err).ShouldNot(HaveOccurred())
				httpClient, err := pkg.GetVerrazzanoHTTPClient(kubeconfigPath)
				Expect(err).ShouldNot(HaveOccurred())
				req, err := retryablehttp.NewRequest("GET", serverURL, nil)
				req.Header.Add("Origin", "null")
				Expect(err).ShouldNot(HaveOccurred())
				Eventually(func() error {
					return pkg.CheckStatusAndResponseHeaderAbsent(
						httpClient, req, "access-control-allow-origin", 0)
				}).Should(BeNil())
			}
		})

		It("can be logged out", func() {
			if !isManagedClusterProfile && isTestSupported {
				Eventually(func() (*pkg.HTTPResponse, error) {
					return pkg.GetWebPage(fmt.Sprintf("%s%s", serverURL, "_logout"), "")
				}, waitTimeout, pollingInterval).Should(And(pkg.HasStatus(http.StatusOK)))
			}
		})

		It("should not allow malformed requests", func() {
			if !isManagedClusterProfile && isTestSupported {
				kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
				Expect(err).ShouldNot(HaveOccurred())
				httpClient, err := pkg.GetVerrazzanoHTTPClient(kubeconfigPath)
				Expect(err).ShouldNot(HaveOccurred())
				body := []byte(`
				0
				POST /mal formed ZZZZ/9.7
				Q: W`)
				req, err := retryablehttp.NewRequest("POST", serverURL, body)
				Expect(err).ShouldNot(HaveOccurred())
				req.Header.Add("Content-Length", "36")
				req.Header.Add("Transfer-Encoding", "chunked")
				Eventually(func() error {
					return pkg.CheckStatusAndResponseHeaderAbsent(httpClient, req, "", 400)
				}).Should(BeNil())
			}
		})

		It("should not allow state changing requests without valid origin header", func() {
			if !isManagedClusterProfile && isTestSupported {
				kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
				Expect(err).ShouldNot(HaveOccurred())
				httpClient, err := pkg.GetVerrazzanoHTTPClient(kubeconfigPath)
				Expect(err).ShouldNot(HaveOccurred())
				req, err := retryablehttp.NewRequest("POST", serverURL, nil)
				Expect(err).ShouldNot(HaveOccurred())
				req.Header.Add("Origin", "https://invalid-origin")
				Eventually(func() error {
					return pkg.CheckStatusAndResponseHeaderAbsent(httpClient, req, "", 403)
				}).Should(BeNil())
			}
		})

		It("should allow non state changing requests without valid origin header but not populate Access-Control-Allow-Origin header", func() {
			if !isManagedClusterProfile && isTestSupported {
				kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
				Expect(err).ShouldNot(HaveOccurred())
				httpClient, err := pkg.GetVerrazzanoHTTPClient(kubeconfigPath)
				Expect(err).ShouldNot(HaveOccurred())
				req, err := retryablehttp.NewRequest("GET", serverURL, nil)
				Expect(err).ShouldNot(HaveOccurred())
				req.Header.Add("Origin", "https://invalid-origin")
				Eventually(func() error {
					return pkg.CheckStatusAndResponseHeaderAbsent(httpClient, req, "access-control-allow-origin", 200)
				}).Should(BeNil())
			}
		})

		It("should return 502 for invalid URL not present in ingress", func() {
			if !isManagedClusterProfile {
				kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
				Expect(err).ShouldNot(HaveOccurred())
				httpClient, err := pkg.GetVerrazzanoHTTPClient(kubeconfigPath)
				Expect(err).ShouldNot(HaveOccurred())
				// make a random string of numbers
				rand.Seed(time.Now().UnixNano())
				numstr := ""
				numints := rand.Perm(10)
				for _, val := range numints {
					numstr = numstr + strconv.Itoa(val)
				}
				invalidURL := strings.Replace(serverURL, "https://verrazzano.", fmt.Sprintf("https://%s.", numstr), 1)
				req, err := retryablehttp.NewRequest("GET", invalidURL, nil)
				Expect(err).ShouldNot(HaveOccurred())
				Eventually(func() error {
					return pkg.CheckStatusAndResponseHeaderAbsent(httpClient, req, "", 502)
				}).Should(BeNil())
			}
		})

	})
})
