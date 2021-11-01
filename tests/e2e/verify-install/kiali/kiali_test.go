// Copyright (c) 2021, Oracle and/or its affiliates.
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
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
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
		client         *kubernetes.Clientset
		httpClient     *retryablehttp.Client
		kialiErr       error
		kialiSupported bool
	)

	BeforeSuite(func() {
		client, kialiErr = k8sutil.GetKubernetesClientset()
		Expect(kialiErr).ToNot(HaveOccurred())
		httpClient, kialiErr = pkg.GetSystemVmiHTTPClient()
		Expect(kialiErr).ToNot(HaveOccurred())
		kialiSupported, kialiErr = pkg.IsVerrazzanoMinVersion("1.1.0")
		Expect(kialiErr).ToNot(HaveOccurred())
	})

	// It Wrapper to only run spec if Kiali is supported on the current Verrazzano installation
	WhenKialiInstalledIt := func(description string, f interface{}) {
		if kialiSupported {
			It(description, f)
		} else {
			pkg.Log(pkg.Info, fmt.Sprintf("Skipping check '%v', Kiali is not supported", description))
		}
	}

	Context("Successful Install", func() {
		WhenKialiInstalledIt("should have a monitoring crd", func() {
			Eventually(func() bool {
				exists, err := pkg.DoesCRDExist("monitoringdashboards.monitoring.kiali.io")
				if err != nil {
					return false
				}
				return exists
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})

		WhenKialiInstalledIt("has a running pod", func() {
			kialiPodsRunning := func() bool {
				return pkg.PodsRunning(systemNamespace, []string{kiali})
			}
			Eventually(kialiPodsRunning, waitTimeout, pollingInterval).Should(BeTrue())
		})

		Context("Ingress", func() {
			var (
				ingress   *networking.Ingress
				kialiHost string
				creds     *pkg.UsernamePassword
				ingError  error
			)

			BeforeEach(func() {
				ingress, kialiErr = client.NetworkingV1().
					Ingresses(systemNamespace).
					Get(context.TODO(), kiali, v1.GetOptions{})
				Expect(kialiErr).ToNot(HaveOccurred())
				rules := ingress.Spec.Rules
				Expect(len(rules)).To(Equal(1))
				Expect(rules[0].Host).To(ContainSubstring("kiali.vmi.system.default"))
				kialiHost = fmt.Sprintf("https://%s", ingress.Spec.Rules[0].Host)
				Eventually(func() (*pkg.UsernamePassword, error) {
					creds, ingError = pkg.GetSystemVMICredentials()
					return creds, ingError
				}, waitTimeout, pollingInterval).ShouldNot(BeNil())
			})

			WhenKialiInstalledIt("should not allow unauthenticated logins", func() {
				Eventually(func() bool {
					unauthHTTPClient, err := pkg.GetSystemVmiHTTPClient()
					if err != nil {
						return false
					}
					return pkg.AssertOauthURLAccessibleAndUnauthorized(unauthHTTPClient, kialiHost)
				}, waitTimeout, pollingInterval).Should(BeTrue())
			})

			WhenKialiInstalledIt("should allow basic authentication", func() {
				Eventually(func() bool {
					return pkg.AssertURLAccessibleAndAuthorized(httpClient, kialiHost, creds)
				}, waitTimeout, pollingInterval).Should(BeTrue())
			})

			WhenKialiInstalledIt("should allow bearer authentication", func() {
				Eventually(func() bool {
					return pkg.AssertBearerAuthorized(httpClient, kialiHost)
				}, waitTimeout, pollingInterval).Should(BeTrue())
			})
		})
	})
})
