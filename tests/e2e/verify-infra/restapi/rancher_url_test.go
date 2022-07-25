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

	"github.com/hashicorp/go-retryablehttp"
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

				var rancherURL string

				Eventually(func() error {
					api, err := pkg.GetAPIEndpoint(kubeconfigPath)
					if err != nil {
						return err
					}
					ingress, err := api.GetIngress("cattle-system", "rancher")
					if err != nil {
						return err
					}
					rancherURL = fmt.Sprintf("https://%s", ingress.Spec.Rules[0].Host)
					t.Logs.Info(fmt.Sprintf("Found ingress URL: %s", rancherURL))
					return nil
				}, waitTimeout, pollingInterval).Should(BeNil())

				Expect(rancherURL).NotTo(BeEmpty())
				var httpClient *retryablehttp.Client
				Eventually(func() error {
					httpClient, err = pkg.GetVerrazzanoHTTPClient(kubeconfigPath)
					if err != nil {
						t.Logs.Error(fmt.Sprintf("Error getting HTTP client: %v", err))
						return err
					}
					return nil
				}, waitTimeout, pollingInterval).Should(BeNil())
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
				Eventually(func() (string, error) {
					clusterData, err := k8sClient.Resource(gvkToGvr(rancher.GVKCluster)).Get(context.Background(), rancher.ClusterLocal, v1.GetOptions{})
					if err != nil {
						t.Logs.Error(fmt.Sprintf("Error getting local Cluster CR: %v", err))
						return "", err
					}
					conditions := clusterData.UnstructuredContent()["conditions"].([]map[string]string)
					for _, condition := range conditions {
						if condition["status"] == "True" && condition["type"] == "Ready" {
							return "active", nil
						}
					}
					return "inactive", fmt.Errorf("Cluster still not in active state")
				}, waitTimeout, pollingInterval).Should(Equal("active"), "rancher local cluster not in active state")
				metrics.Emit(t.Metrics.With("get_cluster_state_elapsed_time", time.Since(start).Milliseconds()))

				minVer14, err := pkg.IsVerrazzanoMinVersion("1.4.0", kubeconfigPath)
				Expect(err).ToNot(HaveOccurred())
				if minVer14 {
					start = time.Now()
					Eventually(func() (string, error) {
						ociDriverData, err := k8sClient.Resource(gvkToGvr(rancher.GVKNodeDriver)).Get(context.Background(), rancher.NodeDriverOCI, v1.GetOptions{})
						if err != nil {
							t.Logs.Error(fmt.Sprintf("Error getting OCI Driver CR: %v", err))
							return "", err
						}
						return ociDriverData.UnstructuredContent()["spec"].(map[string]interface{})["active"].(string), nil
					}, waitTimeout, pollingInterval).Should(Equal("active"), "rancher oci driver not activated")
					metrics.Emit(t.Metrics.With("get_oci_driver_state_elapsed_time", time.Since(start).Milliseconds()))

					start = time.Now()
					Eventually(func() (string, error) {
						okeDriverData, err := k8sClient.Resource(gvkToGvr(rancher.GVKKontainerDriver)).Get(context.Background(), rancher.KontainerDriverOKE, v1.GetOptions{})
						if err != nil {
							t.Logs.Error(fmt.Sprintf("Error getting OKE Driver CR: %v", err))
							return "", err
						}
						return okeDriverData.UnstructuredContent()["spec"].(map[string]interface{})["active"].(string), nil
					}, waitTimeout, pollingInterval).Should(Equal("active"), "rancher oke driver not activated")
					metrics.Emit(t.Metrics.With("get_oke_driver_state_elapsed_time", time.Since(start).Milliseconds()))
				}
			}
		})
	})
})

func gvkToGvr(gvk schema.GroupVersionKind) schema.GroupVersionResource {
	resource := strings.ToLower(gvk.Kind)
	if strings.HasSuffix(resource, "s") {
		resource = resource + "es"
	}
	return schema.GroupVersionResource{Group: gvk.Group,
		Version:  gvk.Version,
		Resource: resource,
	}
}

var _ = t.AfterEach(func() {})
