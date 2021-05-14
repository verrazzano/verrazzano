// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package web_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = ginkgo.Describe("Verrazzano Web UI",
	func() {
		ingress, err := pkg.GetKubernetesClientset().ExtensionsV1beta1().Ingresses("verrazzano-system").Get(context.TODO(), "verrazzano-ingress", v1.GetOptions{})

		ginkgo.It("ingress exist", func() {
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(len(ingress.Spec.Rules)).To(gomega.Equal(1))
		})

		// Determine if the console UI is configured
		consoleUIConfigured := false
		for _, path := range ingress.Spec.Rules[0].HTTP.Paths {
			if path.Backend.ServiceName == "verrazzano-console" {
				consoleUIConfigured = true
			}
		}

		if consoleUIConfigured {

			var ingressRules = ingress.Spec.Rules
			serverURL := fmt.Sprintf("https://%s/", ingressRules[0].Host)

			pkg.Log(pkg.Info, "The Web UI's URL is "+serverURL)

			ginkgo.It("can be accessed", func() {
				rc, content := pkg.GetWebPageWithCABundle(serverURL, "")
				gomega.Expect(rc).To(gomega.Equal(200))
				gomega.Expect(content).To(gomega.Not(gomega.BeEmpty()))
				gomega.Expect(content).To(gomega.Not(gomega.ContainSubstring("404")))
			})

			ginkgo.It("has the correct SSL certificate",
				func() {
					certs, err := pkg.GetCertificates(serverURL)
					gomega.Expect(err).To(gomega.BeNil())
					// There will normally be several certs, but we only need to check the
					// first one -- might want to refactor the checks out into a pkg.IsCertValid()
					// function so we can use it from other test suites too??
					pkg.Log(pkg.Debug, "Issuer Common Name: "+certs[0].Issuer.CommonName)
					pkg.Log(pkg.Debug, "Subject Common Name: "+certs[0].Subject.CommonName)
					pkg.Log(pkg.Debug, "Not Before: "+certs[0].NotBefore.String())
					pkg.Log(pkg.Debug, "Not After: "+certs[0].NotAfter.String())
					gomega.Expect(time.Now().After(certs[0].NotBefore)).To(gomega.BeTrue())
					gomega.Expect(time.Now().Before(certs[0].NotAfter)).To(gomega.BeTrue())
				})

			// VZ-2603: Assertion disabled until VZ-2599 is complete.
			ginkgo.PIt("should return no Server header",
				func() {
					httpClient := pkg.GetVerrazzanoHTTPClient()
					req, err := retryablehttp.NewRequest("GET", serverURL, nil)
					gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("Unexpected error %v", err))
					resp, err := httpClient.Do(req)
					gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("Unexpected error %v", err))
					ioutil.ReadAll(resp.Body)
					resp.Body.Close()
					// HTTP Server headers should never be returned.
					for headerName, headerValues := range resp.Header {
						gomega.Expect(strings.ToLower(headerName)).ToNot(gomega.Equal("server"), fmt.Sprintf("Unexpected Server header %v", headerValues))
					}
				})

			ginkgo.It("should not return CORS Access-Control-Allow-Origin header when no Origin header is provided",
				func() {
					httpClient := pkg.GetVerrazzanoHTTPClient()
					req, err := retryablehttp.NewRequest("GET", serverURL, nil)
					gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("Unexpected error %v", err))
					resp, err := httpClient.Do(req)
					gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("Unexpected error %v", err))
					ioutil.ReadAll(resp.Body)
					resp.Body.Close()
					// HTTP Server headers should never be returned.
					for headerName, headerValues := range resp.Header {
						gomega.Expect(strings.ToLower(headerName)).ToNot(gomega.Equal("access-control-allow-origin"), fmt.Sprintf("Unexpected header %s:%v", headerName, headerValues))
					}
				})

			ginkgo.It("should not return CORS Access-Control-Allow-Origin header when Origin: * is provided",
				func() {
					httpClient := pkg.GetVerrazzanoHTTPClient()
					req, err := retryablehttp.NewRequest("GET", serverURL, nil)
					req.Header.Add("Origin", "*")
					gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("Unexpected error %v", err))
					resp, err := httpClient.Do(req)
					gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("Unexpected error %v", err))
					ioutil.ReadAll(resp.Body)
					resp.Body.Close()
					// HTTP Server headers should never be returned.
					for headerName, headerValues := range resp.Header {
						gomega.Expect(strings.ToLower(headerName)).ToNot(gomega.Equal("access-control-allow-origin"), fmt.Sprintf("Unexpected header %s:%v", headerName, headerValues))
					}
				})

			ginkgo.It("should not return CORS Access-Control-Allow-Origin header when Origin: null is provided",
				func() {
					httpClient := pkg.GetVerrazzanoHTTPClient()
					req, err := retryablehttp.NewRequest("GET", serverURL, nil)
					req.Header.Add("Origin", "null")
					gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("Unexpected error %v", err))
					resp, err := httpClient.Do(req)
					gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("Unexpected error %v", err))
					ioutil.ReadAll(resp.Body)
					resp.Body.Close()
					// HTTP Server headers should never be returned.
					for headerName, headerValues := range resp.Header {
						gomega.Expect(strings.ToLower(headerName)).ToNot(gomega.Equal("access-control-allow-origin"), fmt.Sprintf("Unexpected header %s:%v", headerName, headerValues))
					}
				})
		}
	})
