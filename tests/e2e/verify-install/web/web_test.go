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
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
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
				resp, err := httpClient.Do(req)
				Expect(err).ShouldNot(HaveOccurred())
				ioutil.ReadAll(resp.Body)
				resp.Body.Close()
				// HTTP Server headers should never be returned.
				for headerName, headerValues := range resp.Header {
					Expect(strings.ToLower(headerName)).ToNot(Equal("server"), fmt.Sprintf("Unexpected Server header %v", headerValues))
				}
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
				resp, err := httpClient.Do(req)
				Expect(err).ShouldNot(HaveOccurred())
				ioutil.ReadAll(resp.Body)
				resp.Body.Close()
				// HTTP Access-Control-Allow-Origin header should never be returned.
				for headerName, headerValues := range resp.Header {
					Expect(strings.ToLower(headerName)).ToNot(Equal("access-control-allow-origin"), fmt.Sprintf("Unexpected header %s:%v", headerName, headerValues))
				}
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
				resp, err := httpClient.Do(req)
				Expect(err).ShouldNot(HaveOccurred())
				ioutil.ReadAll(resp.Body)
				resp.Body.Close()
				// HTTP Access-Control-Allow-Origin header should never be returned.
				for headerName, headerValues := range resp.Header {
					Expect(strings.ToLower(headerName)).ToNot(Equal("access-control-allow-origin"), fmt.Sprintf("Unexpected header %s:%v", headerName, headerValues))
				}
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
				resp, err := httpClient.Do(req)
				Expect(err).ShouldNot(HaveOccurred())
				ioutil.ReadAll(resp.Body)
				resp.Body.Close()
				// HTTP Access-Control-Allow-Origin header should never be returned.
				for headerName, headerValues := range resp.Header {
					Expect(strings.ToLower(headerName)).ToNot(Equal("access-control-allow-origin"), fmt.Sprintf("Unexpected header %s:%v", headerName, headerValues))
				}
			}
		})

		It("can be logged out", func() {
			if !isManagedClusterProfile {
				kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
				Expect(err).ShouldNot(HaveOccurred())
				vz, err := pkg.GetVerrazzanoInstallResourceInCluster(kubeconfigPath)
				Expect(err).ShouldNot(HaveOccurred())
				if v1alpha1.ValidateVersionHigherOrEqual(fmt.Sprintf("v%s", vz.Status.Version), "v1.0.1") {
					Eventually(func() (*pkg.HTTPResponse, error) {
						return pkg.GetWebPage(fmt.Sprintf("%s%s", serverURL, "_logout"), "")
					}, waitTimeout, pollingInterval).Should(And(pkg.HasStatus(http.StatusOK)))
				}
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
				resp, err := httpClient.Do(req)
				Expect(err).ShouldNot(HaveOccurred())
				ioutil.ReadAll(resp.Body)
				resp.Body.Close()
				Expect(resp.StatusCode).To(Equal(400))
			}
		})

	})
})
