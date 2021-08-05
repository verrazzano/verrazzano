// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package web_test

import (
	"context"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"k8s.io/client-go/kubernetes"

	"github.com/hashicorp/go-retryablehttp"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"k8s.io/api/extensions/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	waitTimeout     = 3 * time.Minute
	pollingInterval = 5 * time.Second
)

var ingress *v1beta1.Ingress
var consoleUIConfigured bool = false

var _ = BeforeSuite(func() {
	var clientset *kubernetes.Clientset
	Eventually(func() (*kubernetes.Clientset, error) {
		var err error
		clientset, err = pkg.GetKubernetesClientset()
		return clientset, err
	}, waitTimeout, pollingInterval).ShouldNot(BeNil())
	Eventually(func() (*v1beta1.Ingress, error) {
		var err error
		ingress, err = clientset.ExtensionsV1beta1().Ingresses("verrazzano-system").Get(context.TODO(), "verrazzano-ingress", v1.GetOptions{})
		return ingress, err
	}, waitTimeout, pollingInterval).ShouldNot(BeNil())

	Expect(len(ingress.Spec.Rules)).To(Equal(1))

	// Determine if the console UI is configured
	for _, path := range ingress.Spec.Rules[0].HTTP.Paths {
		if path.Backend.ServiceName == "verrazzano-console" {
			consoleUIConfigured = true
		}
	}
})

var _ = Describe("Verrazzano Web UI", func() {
	When("the console UI is configured", func() {
		var serverURL string

		BeforeEach(func() {
			if !consoleUIConfigured {
				Skip("Skipping spec since console UI is not configured")
			}

			ingressRules := ingress.Spec.Rules
			serverURL = fmt.Sprintf("https://%s/", ingressRules[0].Host)
		})

		It("can be accessed", func() {
			Eventually(func() (*pkg.HTTPResponse, error) {
				return pkg.GetWebPage(serverURL, "")
			}, waitTimeout, pollingInterval).Should(And(pkg.HasStatus(http.StatusOK), pkg.BodyNotEmpty(), pkg.BodyDoesNotContain("404")))
		})

		It("has the correct SSL certificate", func() {
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
		})

		It("should return no Server header", func() {
			kubeconfigPath, err := pkg.GetKubeConfigPathFromEnv()
			Expect(err).ShouldNot(HaveOccurred())
			httpClient, err := pkg.GetVerrazzanoHTTPClient(kubeconfigPath)
			Expect(err).ShouldNot(HaveOccurred())
			req, err := retryablehttp.NewRequest("GET", serverURL, nil)
			Expect(err).ShouldNot(HaveOccurred())
			resp, err := httpClient.Do(req)
			Expect(err).ShouldNot(HaveOccurred())
			ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			// HTTP Server headers should never be returned.
			for headerName, headerValues := range resp.Header {
				Expect(strings.ToLower(headerName)).ToNot(Equal("server"), fmt.Sprintf("Unexpected Server header %v", headerValues))
			}
		})

		It("should not return CORS Access-Control-Allow-Origin header when no Origin header is provided", func() {
			kubeconfigPath, err := pkg.GetKubeConfigPathFromEnv()
			Expect(err).ShouldNot(HaveOccurred())
			httpClient, err := pkg.GetVerrazzanoHTTPClient(kubeconfigPath)
			Expect(err).ShouldNot(HaveOccurred())
			req, err := retryablehttp.NewRequest("GET", serverURL, nil)
			Expect(err).ShouldNot(HaveOccurred())
			resp, err := httpClient.Do(req)
			Expect(err).ShouldNot(HaveOccurred())
			ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			// HTTP Access-Control-Allow-Origin header should never be returned.
			for headerName, headerValues := range resp.Header {
				Expect(strings.ToLower(headerName)).ToNot(Equal("access-control-allow-origin"), fmt.Sprintf("Unexpected header %s:%v", headerName, headerValues))
			}
		})

		It("should not return CORS Access-Control-Allow-Origin header when Origin: * is provided", func() {
			kubeconfigPath, err := pkg.GetKubeConfigPathFromEnv()
			Expect(err).ShouldNot(HaveOccurred())
			httpClient, err := pkg.GetVerrazzanoHTTPClient(kubeconfigPath)
			Expect(err).ShouldNot(HaveOccurred())
			req, err := retryablehttp.NewRequest("GET", serverURL, nil)
			req.Header.Add("Origin", "*")
			Expect(err).ShouldNot(HaveOccurred())
			resp, err := httpClient.Do(req)
			Expect(err).ShouldNot(HaveOccurred())
			ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			// HTTP Access-Control-Allow-Origin header should never be returned.
			for headerName, headerValues := range resp.Header {
				Expect(strings.ToLower(headerName)).ToNot(Equal("access-control-allow-origin"), fmt.Sprintf("Unexpected header %s:%v", headerName, headerValues))
			}
		})

		It("should not return CORS Access-Control-Allow-Origin header when Origin: null is provided", func() {
			kubeconfigPath, err := pkg.GetKubeConfigPathFromEnv()
			Expect(err).ShouldNot(HaveOccurred())
			httpClient, err := pkg.GetVerrazzanoHTTPClient(kubeconfigPath)
			Expect(err).ShouldNot(HaveOccurred())
			req, err := retryablehttp.NewRequest("GET", serverURL, nil)
			req.Header.Add("Origin", "null")
			Expect(err).ShouldNot(HaveOccurred())
			resp, err := httpClient.Do(req)
			Expect(err).ShouldNot(HaveOccurred())
			ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			// HTTP Access-Control-Allow-Origin header should never be returned.
			for headerName, headerValues := range resp.Header {
				Expect(strings.ToLower(headerName)).ToNot(Equal("access-control-allow-origin"), fmt.Sprintf("Unexpected header %s:%v", headerName, headerValues))
			}
		})
	})
})
