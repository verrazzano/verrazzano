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
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
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
				rancherURL := pkg.EventuallyGetURLForIngress(t.Logs, api, "cattle-system", "rancher", "https")
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

				start = time.Now()
				t.Logs.Info("Verify Local AuthConfig")
				Eventually(func() (bool, error) {
					localAuthConfigData, err := k8sClient.Resource(gvkToGvr(rancher.GVKAuthConfig)).Get(context.Background(), rancher.AuthConfigLocal, v1.GetOptions{})
					if err != nil {
						t.Logs.Error(fmt.Sprintf("error getting local authConfig: %v", err))
						return false, err
					}

					authConfigAttributes := localAuthConfigData.UnstructuredContent()
					return authConfigAttributes[rancher.AuthConfigAttributeEnabled].(bool), nil
				}, waitTimeout, pollingInterval).Should(BeTrue(), "failed verifying local authconfig")
				metrics.Emit(t.Metrics.With("get_local_authconfig_state_elapsed_time", time.Since(start).Milliseconds()))

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
					}, waitTimeout, pollingInterval).Should(Equal(true), "rancher OCI driver not activated")
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
					}, waitTimeout, pollingInterval).Should(Equal(true), "rancher OKE driver not activated")
					metrics.Emit(t.Metrics.With("get_oke_driver_state_elapsed_time", time.Since(start).Milliseconds()))

					start = time.Now()
					t.Logs.Info("Verify Keycloak AuthConfig")
					keycloakURL := pkg.EventuallyGetURLForIngress(t.Logs, api, "keycloak", "keycloak", "https")
					Eventually(func() (bool, error) {
						authConfigData, err := k8sClient.Resource(gvkToGvr(rancher.GVKAuthConfig)).Get(context.Background(), common.AuthConfigKeycloak, v1.GetOptions{})
						if err != nil {
							t.Logs.Error(fmt.Sprintf("error getting keycloak oidc authConfig: %v", err))
							return false, err
						}

						authConfigAttributes := authConfigData.UnstructuredContent()
						if err = verifyAuthConfigAttribute(rancher.AuthConfigKeycloakAttributeAccessMode, authConfigAttributes[rancher.AuthConfigKeycloakAttributeAccessMode].(string), rancher.AuthConfigKeycloakAccessMode); err != nil {
							return false, err
						}

						if err = verifyAuthConfigAttribute(rancher.AuthConfigKeycloakAttributeClientID, authConfigAttributes[rancher.AuthConfigKeycloakAttributeClientID].(string), rancher.AuthConfigKeycloakClientIDRancher); err != nil {
							return false, err
						}

						if err = verifyAuthConfigAttribute(rancher.AuthConfigKeycloakAttributeGroupSearchEnabled, authConfigAttributes[rancher.AuthConfigKeycloakAttributeGroupSearchEnabled].(bool), true); err != nil {
							return false, err
						}

						if err = verifyAuthConfigAttribute(rancher.AuthConfigKeycloakAttributeAuthEndpoint, authConfigAttributes[rancher.AuthConfigKeycloakAttributeAuthEndpoint].(string), keycloakURL+rancher.AuthConfigKeycloakURLPathAuthEndPoint); err != nil {
							return false, err
						}

						if err = verifyAuthConfigAttribute(rancher.AuthConfigKeycloakAttributeRancherURL, authConfigAttributes[rancher.AuthConfigKeycloakAttributeRancherURL].(string), rancherURL+rancher.AuthConfigKeycloakURLPathVerifyAuth); err != nil {
							return false, err
						}

						authConfigClientSecret := authConfigAttributes[common.AuthConfigKeycloakAttributeClientSecret].(string)
						if authConfigClientSecret == "" {
							err = fmt.Errorf("keycloak auth config attribute %s not correctly configured, value is empty", common.AuthConfigKeycloakAttributeClientSecret)
							t.Logs.Error(err.Error())
							return false, err
						}

						return true, nil
					}, waitTimeout, pollingInterval).Should(Equal(true), "keycloak oidc authconfig not configured correctly")
					metrics.Emit(t.Metrics.With("get_kc_authconfig_state_elapsed_time", time.Since(start).Milliseconds()))

					start = time.Now()
					t.Logs.Info("Verify Verrazzano rancher user")
					Eventually(func() (bool, error) {
						userData, err := k8sClient.Resource(gvkToGvr(rancher.GVKUser)).Get(context.Background(), rancher.UserVerrazzano, v1.GetOptions{})
						if err != nil {
							t.Logs.Error(fmt.Sprintf("error getting rancher verrazzano user: %v", err))
							return false, err
						}

						userPrincipals, ok := userData.UnstructuredContent()[rancher.UserAttributePrincipalIDs].([]interface{})
						if !ok {
							err = fmt.Errorf("rancher verrazzano user configured incorrectly,principalIds empty")
							t.Logs.Error(err.Error())
							return false, err
						}

						for _, userPrincipal := range userPrincipals {
							if strings.Contains(userPrincipal.(string), rancher.UserPrincipalKeycloakPrefix) {
								return true, nil
							}
						}
						return false, fmt.Errorf("Verrazzano rancher user is not mapped in keycloak")
					}, waitTimeout, pollingInterval).Should(Equal(true), "verrazzano rancher user not correctly configured")
					metrics.Emit(t.Metrics.With("get_vz_rancher_user_elapsed_time", time.Since(start).Milliseconds()))

					start = time.Now()
					t.Logs.Info("Verify Verrazzano rancher user admin GlobalRoleBinding")
					Eventually(func() (bool, error) {
						grbData, err := k8sClient.Resource(gvkToGvr(rancher.GVKGlobalRoleBinding)).Get(context.Background(), rancher.GlobalRoleBindingVerrazzano, v1.GetOptions{})
						if err != nil {
							t.Logs.Error(fmt.Sprintf("error getting rancher verrazzano user global role binding: %v", err))
							return false, err
						}

						grbAttributes := grbData.UnstructuredContent()
						if grbAttributes[rancher.GlobalRoleBindingAttributeUserName].(string) != rancher.UserVerrazzano {
							return false, fmt.Errorf("verrazzano rancher user global role binding user is invalid")
						}

						if grbAttributes[rancher.GlobalRoleBindingAttributeRoleName].(string) != rancher.AdminRoleName {
							return false, fmt.Errorf("verrazzano rancher user global role binding role is invalid")
						}

						return true, nil
					}, waitTimeout, pollingInterval).Should(Equal(true), "verrazzano rancher user global role binding does not exist")
					metrics.Emit(t.Metrics.With("get_vz_rancher_user_grb_elapsed_time", time.Since(start).Milliseconds()))

					start = time.Now()
					t.Logs.Info("Verify ClusterRoleTemplateBindings are created for Keycloak groups")
					Eventually(func() (bool, error) {
						for _, grp := range rancher.GroupRolePairs {
							name := fmt.Sprintf("crtb-%s-%s", grp[rancher.ClusterRoleKey], grp[rancher.GroupKey])
							crtpData, err := k8sClient.Resource(gvkToGvr(rancher.GVKClusterRoleTemplateBinding)).Namespace(rancher.ClusterLocal).Get(context.Background(), name, v1.GetOptions{})
							if err != nil {
								return false, fmt.Errorf("error getting ClusterRoleTemplateBinding %s: %v", name, err)
							}

							crtpAttributes := crtpData.UnstructuredContent()
							if crtpAttributes[rancher.ClusterRoleTemplateBindingAttributeGroupPrincipalName].(string) != rancher.GroupPrincipalKeycloakPrefix+grp[rancher.GroupKey] {
								return false, fmt.Errorf("ClusterRoleTemplateBinding %s attribute %s is invalid, expected %s, got %s", name, rancher.ClusterRoleTemplateBindingAttributeGroupPrincipalName, crtpAttributes[rancher.ClusterRoleTemplateBindingAttributeGroupPrincipalName].(string), rancher.GroupPrincipalKeycloakPrefix+grp[rancher.GroupKey])
							}

							if crtpAttributes[rancher.ClusterRoleTemplateBindingAttributeRoleTemplateName].(string) != grp[rancher.ClusterRoleKey] {
								return false, fmt.Errorf("ClusterRoleTemplateBinding %s attribute %s is invalid, expected %s, got %s", name, rancher.ClusterRoleTemplateBindingAttributeRoleTemplateName, crtpAttributes[rancher.ClusterRoleTemplateBindingAttributeRoleTemplateName].(string), grp[rancher.ClusterRoleKey])
							}
						}

						return true, nil
					}, waitTimeout, pollingInterval).Should(Equal(true), "ClusterRoleTemplateBinding not found or incorrect")
					metrics.Emit(t.Metrics.With("get_crtb_elapsed_time", time.Since(start).Milliseconds()))

					start = time.Now()
					t.Logs.Info("Verify RoleTemplate are created for Keycloak groups ClusterRoleBindings")
					Eventually(func() (bool, error) {
						_, err := k8sClient.Resource(gvkToGvr(rancher.GVKRoleTemplate)).Get(context.Background(), rancher.VerrazzanoAdminRoleName, v1.GetOptions{})
						if err != nil {
							return false, fmt.Errorf("error getting RoleTemplate %s: %v", rancher.VerrazzanoAdminRoleName, err)
						}

						_, err = k8sClient.Resource(gvkToGvr(rancher.GVKRoleTemplate)).Get(context.Background(), rancher.VerrazzanoMonitorRoleName, v1.GetOptions{})
						if err != nil {
							return false, fmt.Errorf("error getting RoleTemplate %s: %v", rancher.VerrazzanoMonitorRoleName, err)
						}

						return true, nil
					}, waitTimeout, pollingInterval).Should(Equal(true), "RoleTemplate not found")
					metrics.Emit(t.Metrics.With("get_roletemplate_elapsed_time", time.Since(start).Milliseconds()))

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

func verifyAuthConfigAttribute(name string, actual interface{}, expected interface{}) error {
	if expected != actual {
		err := fmt.Errorf("keycloak auth config attribute %s not correctly configured, expected %v, actual %v", name, expected, actual)
		t.Logs.Error(err.Error())
		return err
	}
	return nil
}

var _ = t.AfterEach(func() {})
