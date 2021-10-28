// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package kiali

import (
	"context"
	"fmt"
	"github.com/hashicorp/go-retryablehttp"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	networking "k8s.io/api/networking/v1"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"strings"
	"time"
)

const (
	systemNamespace = "verrazzano-system"
	kiali           = "vmi-system-kiali"
	waitTimeout     = 10 * time.Minute
	pollingInterval = 5 * time.Second
)

var _ = Describe("Kiali", func() {
	var (
		client     *kubernetes.Clientset
		err        error
		httpClient *retryablehttp.Client
	)

	BeforeSuite(func() {
		client, err = k8sutil.GetKubernetesClientset()
		Expect(err).ToNot(HaveOccurred())
		httpClient, err = pkg.GetSystemVmiHTTPClient()
		Expect(err).ToNot(HaveOccurred())

	})

	Context("Kiali installed successfully", func() {
		var (
			extClient *apiextv1.ApiextensionsV1Client
			err       error
		)

		BeforeEach(func() {
			extClient, err = pkg.APIExtensionsClientSet()
			Expect(err).ToNot(HaveOccurred())
		})

		It("should have a monitoring crd", func() {
			crd, err := extClient.CustomResourceDefinitions().Get(context.TODO(), "monitoringdashboards.monitoring.kiali.io", v1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(crd).ToNot(BeNil())
		})

		It("has a running pod", func() {
			kialiPodsRunning := func() bool {
				return pkg.PodsRunning(systemNamespace, []string{kiali})
			}
			Eventually(kialiPodsRunning, waitTimeout, pollingInterval).Should(BeTrue())
		})

		Context("Ingress accessibility", func() {
			var (
				ingress *networking.Ingress
			)

			BeforeEach(func() {
				ingress, err = client.NetworkingV1().
					Ingresses(systemNamespace).
					Get(context.TODO(), kiali, v1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
			})

			It("should have exactly one host", func() {
				rules := ingress.Spec.Rules
				Expect(len(rules)).To(Equal(1))
				Expect(rules[0].Host).To(ContainSubstring("kiali.vmi.system.default"))
			})

			It("should be reachable over HTTPS", func() {
				kialiHost := fmt.Sprintf("https://%s", ingress.Spec.Rules[0].Host)
				Eventually(func() bool {
					resp, err := httpClient.Get(kialiHost)
					if err != nil {
						return false
					}
					location, err := resp.Location()
					if err != nil {
						return false
					}
					return resp.StatusCode == 202 && strings.Contains(location.Host, "kiali")
				}, waitTimeout, pollingInterval).Should(BeTrue())
			})
		})
	})
})
