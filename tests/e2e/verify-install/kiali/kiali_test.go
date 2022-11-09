// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package kiali

import (
	"fmt"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/kiali"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
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
	httpClient = pkg.EventuallyVerrazzanoRetryableHTTPClient()
})

// 'It' Wrapper to only run spec if Kiali is supported on the current Verrazzano installation
func WhenKialiInstalledIt(description string, f interface{}) {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Fail(fmt.Sprintf("Failed to get default kubeconfig path: %s", err.Error()))
	}
	supported, err := pkg.IsVerrazzanoMinVersionEventually("1.1.0", kubeconfigPath)
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
				result, err := pkg.PodsRunning(kiali.ComponentNamespace, []string{pkg.KialiName})
				if err != nil {
					AbortSuite(fmt.Sprintf("Pod %v is not running in the namespace: %v, error: %v", pkg.KialiName, kiali.ComponentNamespace, err))
				}
				return result
			}
			Eventually(kialiPodsRunning, waitTimeout, pollingInterval).Should(BeTrue())
		})

		WhenKialiInstalledIt("should have a pod with affinity configured", func() {
			var pods []corev1.Pod
			var err error
			Eventually(func() bool {
				pods, err = pkg.GetPodsFromSelector(&v1.LabelSelector{MatchLabels: map[string]string{"app": "kiali"}}, constants.VerrazzanoSystemNamespace)
				if err != nil {
					t.Logs.Errorf("Failed to get Kiali pods: %v", err)
					return false
				}
				return true
			}, waitTimeout, pollingInterval)
			for _, pod := range pods {
				affinity := pod.Spec.Affinity
				Expect(affinity).ToNot(BeNil())
				Expect(affinity.PodAffinity).To(BeNil())
				Expect(affinity.NodeAffinity).To(BeNil())
				Expect(affinity.PodAntiAffinity).ToNot(BeNil())
				Expect(len(affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution)).To(Equal(1))
			}
		})

		t.Context("should", func() {
			var (
				kialiHost string
				creds     *pkg.UsernamePassword
			)

			BeforeEach(func() {
				kialiHost = pkg.EventuallyGetKialiHost(client)
				creds = pkg.EventuallyGetSystemVMICredentials()
			})

			WhenKialiInstalledIt("not allow unauthenticated logins", func() {
				Eventually(func() bool {
					unauthHTTPClient := pkg.EventuallyVerrazzanoRetryableHTTPClient()
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
