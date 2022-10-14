// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package kiali

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	networking "k8s.io/api/networking/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	systemNamespace = "verrazzano-system"
	kiali           = "vmi-system-kiali"
	waitTimeout     = 15 * time.Minute
	pollingInterval = 10 * time.Second
)

var (
	client     *kubernetes.Clientset
	httpClient *retryablehttp.Client
	kialiErr   error
)

var t = framework.NewTestFramework("kiali")

var _ = t.BeforeSuite(func() {
	client, kialiErr = k8sutil.GetKubernetesClientset()
	Expect(kialiErr).ToNot(HaveOccurred())
	httpClient, kialiErr = pkg.GetVerrazzanoRetryableHTTPClient()
	Expect(kialiErr).ToNot(HaveOccurred())
})

// 'It' Wrapper to only run spec if Kiali is supported on the current Verrazzano installation
func WhenKialiInstalledIt(description string, f interface{}) {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Fail(fmt.Sprintf("Failed to get default kubeconfig path: %s", err.Error()))
	}
	supported, err := pkg.IsVerrazzanoMinVersion("1.1.0", kubeconfigPath)
	if err != nil {
		Fail(err.Error())
	}
	// Kiali only installed when VZ > 1.1.0 and not a managed cluster
	if supported && !pkg.IsManagedClusterProfile() {
		t.It(description, f)
	} else {
		t.Logs.Infof("Skipping check '%v', Kiali is not supported", description)
	}
}

var _ = t.AfterEach(func() {})

var _ = t.Describe("Kiali", Label("f:platform-lcm.install"), func() {

	t.Context("after successful installation", func() {

		WhenKialiInstalledIt("should have a running pod", func() {
			kialiPodsRunning := func() bool {
				result, err := pkg.PodsRunning(systemNamespace, []string{kiali})
				if err != nil {
					AbortSuite(fmt.Sprintf("Pod %v is not running in the namespace: %v, error: %v", kiali, systemNamespace, err))
				}
				return result
			}
			Eventually(kialiPodsRunning, waitTimeout, pollingInterval).Should(BeTrue())
		})

		t.Context("should", func() {
			var (
				ingress   *networking.Ingress
				kialiHost string
				creds     *pkg.UsernamePassword
				ingError  error
			)

			BeforeEach(func() {
				Eventually(func() (*networking.Ingress, error) {
					var err error
					ingress, err = client.NetworkingV1().Ingresses(systemNamespace).Get(context.TODO(), kiali, v1.GetOptions{})
					return ingress, err
				}, waitTimeout, pollingInterval).ShouldNot(BeNil())
				rules := ingress.Spec.Rules
				Expect(len(rules)).To(Equal(1))
				Expect(rules[0].Host).To(ContainSubstring("kiali.vmi.system"))
				kialiHost = fmt.Sprintf("https://%s", ingress.Spec.Rules[0].Host)
				Eventually(func() (*pkg.UsernamePassword, error) {
					creds, ingError = pkg.GetSystemVMICredentials()
					return creds, ingError
				}, waitTimeout, pollingInterval).ShouldNot(BeNil())
			})

			WhenKialiInstalledIt("not allow unauthenticated logins", func() {
				Eventually(func() bool {
					unauthHTTPClient, err := pkg.GetVerrazzanoRetryableHTTPClient()
					if err != nil {
						return false
					}
					return pkg.AssertOauthURLAccessibleAndUnauthorized(unauthHTTPClient, kialiHost)
				}, waitTimeout, pollingInterval).Should(BeTrue())
			})

			WhenKialiInstalledIt("allow basic authentication", func() {
				Eventually(func() bool {
					return pkg.AssertURLAccessibleAndAuthorized(httpClient, kialiHost, creds)
				}, waitTimeout, pollingInterval).Should(BeTrue())
			})

			WhenKialiInstalledIt("allow bearer authentication", func() {
				Eventually(func() bool {
					return pkg.AssertBearerAuthorized(httpClient, kialiHost)
				}, waitTimeout, pollingInterval).Should(BeTrue())
			})
		})
	})
})
