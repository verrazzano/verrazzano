// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package restapi_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/verrazzano/verrazzano/pkg/test/framework/metrics"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/rancher"
	"k8s.io/apimachinery/pkg/runtime/schema"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = t.Describe("rancher", Label("f:infra-lcm",
	"f:ui.console"), func() {
	const (
		waitTimeout     = 5 * time.Minute
		pollingInterval = 5 * time.Second
	)

	t.Context("url test to", func() {
		t.It("Fetch rancher url", func() {
			if !pkg.IsManagedClusterProfile() {
				kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
				if err != nil {
					t.Logs.Error(fmt.Sprintf("Error getting kubeconfig: %v", err))
					t.Fail(err.Error())
				}

				api := pkg.EventuallyGetAPIEndpoint(kubeconfigPath)
				rancherURL := pkg.EventuallyGetRancherURL(t.Logs, api)
				httpClient := pkg.EventuallyVerrazzanoRetryableHTTPClient()
				var httpResponse *pkg.HTTPResponse

				Eventually(func() (*pkg.HTTPResponse, error) {
					httpResponse, err = pkg.GetWebPageWithClient(httpClient, rancherURL, "")
					return httpResponse, err
				}, waitTimeout, pollingInterval).Should(pkg.HasStatus(http.StatusOK))

				Expect(pkg.CheckNoServerHeader(httpResponse)).To(BeTrue(), "Found unexpected server header in response")

				k8sClient, err := pkg.GetDynamicClientInCluster(kubeconfigPath)
				if err != nil {
					t.Logs.Error(fmt.Sprintf("Error getting K8S client: %v", err))
					t.Fail(err.Error())
				}

				start := time.Now()
				t.Logs.Info("Verify local cluster status")
				Eventually(func() (bool, error) {
					clusterData, err := k8sClient.Resource(gvkToGvr(rancher.GVKCluster)).Get(context.Background(), rancher.ClusterLocal, v1.GetOptions{})
					if err != nil {
						t.Logs.Error(fmt.Sprintf("Error getting local Cluster CR: %v", err))
						return false, err
					}
					status := clusterData.UnstructuredContent()["status"].(map[string]interface{})
					conditions := status["conditions"].([]interface{})
					for _, condition := range conditions {
						conditionStage := condition.(map[string]interface{})
						if conditionStage["status"].(string) == "True" && conditionStage["type"].(string) == "Ready" {
							return true, nil
						}
					}
					return false, fmt.Errorf("Cluster still not in active state")
				}, waitTimeout, pollingInterval).Should(Equal(true), "rancher local cluster not in active state")
				metrics.Emit(t.Metrics.With("get_cluster_state_elapsed_time", time.Since(start).Milliseconds()))

				minVer14, err := pkg.IsVerrazzanoMinVersion("1.4.0", kubeconfigPath)
				Expect(err).ToNot(HaveOccurred())
				if minVer14 {
					start = time.Now()
					t.Logs.Info("Verify OCI driver status")
					Eventually(func() (bool, error) {
						ociDriverData, err := k8sClient.Resource(gvkToGvr(rancher.GVKNodeDriver)).Get(context.Background(), rancher.NodeDriverOCI, v1.GetOptions{})
						if err != nil {
							t.Logs.Error(fmt.Sprintf("Error getting OCI Driver CR: %v", err))
							return false, err
						}
						return ociDriverData.UnstructuredContent()["spec"].(map[string]interface{})["active"].(bool), nil
					}, waitTimeout, pollingInterval).Should(Equal(true), "rancher oci driver not activated")
					metrics.Emit(t.Metrics.With("get_oci_driver_state_elapsed_time", time.Since(start).Milliseconds()))

					start = time.Now()
					t.Logs.Info("Verify OKE driver status")
					Eventually(func() (bool, error) {
						okeDriverData, err := k8sClient.Resource(gvkToGvr(rancher.GVKKontainerDriver)).Get(context.Background(), rancher.KontainerDriverOKE, v1.GetOptions{})
						if err != nil {
							t.Logs.Error(fmt.Sprintf("Error getting OKE Driver CR: %v", err))
							return false, err
						}
						return okeDriverData.UnstructuredContent()["spec"].(map[string]interface{})["active"].(bool), nil
					}, waitTimeout, pollingInterval).Should(Equal(true), "rancher oke driver not activated")
					metrics.Emit(t.Metrics.With("get_oke_driver_state_elapsed_time", time.Since(start).Milliseconds()))

					start = time.Now()
					t.Logs.Info("Verify Keycloak AuthConfig")
					Eventually(func() (bool, error) {
						authConfigData, err := k8sClient.Resource(gvkToGvr(rancher.GVKAuthConfig)).Get(context.Background(), rancher.AuthConfigKeycloak, v1.GetOptions{})
						if err != nil {
							t.Logs.Error(fmt.Sprintf("error getting keycloak oidc authConfig: %v", err))
							return false, err
						}
						return authConfigData.UnstructuredContent()["enabled"].(bool), nil
					}, waitTimeout, pollingInterval).Should(Equal(true), "keycloak oidc authconfig not enabled")
					metrics.Emit(t.Metrics.With("get_kc_authconfig_state_elapsed_time", time.Since(start).Milliseconds()))
				}
			}
		})
	})
})

func gvkToGvr(gvk schema.GroupVersionKind) schema.GroupVersionResource {
	resource := strings.ToLower(gvk.Kind)
	if strings.HasSuffix(resource, "s") {
		resource = resource + "es"
	} else {
		resource = resource + "s"
	}

	return schema.GroupVersionResource{Group: gvk.Group,
		Version:  gvk.Version,
		Resource: resource,
	}
}

var _ = t.AfterEach(func() {})
