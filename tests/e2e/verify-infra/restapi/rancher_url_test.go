// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package restapi_test

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/rancher"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework/metrics"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
)

const (
	waitTimeout     = 5 * time.Minute
	pollingInterval = 5 * time.Second
)

var _ = t.Describe("rancher", Label("f:infra-lcm",
	"f:ui.console"), func() {

	t.Context("test to", func() {
		t.It("Verify rancher access and configuration", func() {
			if !pkg.IsManagedClusterProfile() {
				kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
				if err != nil {
					t.Logs.Error(fmt.Sprintf("Error getting kubeconfig: %v", err))
					t.Fail(err.Error())
				}

				start := time.Now()
				err = pkg.VerifyRancherAccess(t.Logs)
				if err != nil {
					t.Logs.Error(fmt.Sprintf("Error verifying access to Rancher: %v", err))
					t.Fail(err.Error())
				}

				metrics.Emit(t.Metrics.With("rancher_access_elapsed_time", time.Since(start).Milliseconds()))

				k8sClient, err := pkg.GetDynamicClientInCluster(kubeconfigPath)
				if err != nil {
					t.Logs.Error(fmt.Sprintf("Error getting K8S client: %v", err))
					t.Fail(err.Error())
				}

				start = time.Now()
				t.Logs.Info("Verify local cluster status")
				Eventually(func() (bool, error) {
					clusterData, err := k8sClient.Resource(pkg.GvkToGvr(rancher.GVKCluster)).Get(context.Background(), rancher.ClusterLocal, v1.GetOptions{})
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
					localAuthConfigData, err := k8sClient.Resource(pkg.GvkToGvr(common.GVKAuthConfig)).Get(context.Background(), rancher.AuthConfigLocal, v1.GetOptions{})
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
					t.Logs.Info("Verify OKE driver status")
					Eventually(func() (bool, error) {
						okeDriverData, err := k8sClient.Resource(pkg.GvkToGvr(rancher.GVKKontainerDriver)).Get(context.Background(), rancher.KontainerDriverOKE, v1.GetOptions{})
						if err != nil {
							t.Logs.Error(fmt.Sprintf("Error getting OKE Driver CR: %v", err))
							return false, err
						}
						return okeDriverData.UnstructuredContent()["spec"].(map[string]interface{})["active"].(bool), nil
					}, waitTimeout, pollingInterval).Should(Equal(true), "rancher OKE driver not activated")
					metrics.Emit(t.Metrics.With("get_oke_driver_state_elapsed_time", time.Since(start).Milliseconds()))

					start = time.Now()
					err = pkg.VerifyRancherKeycloakAuthConfig(t.Logs)
					if err != nil {
						t.Logs.Error(fmt.Sprintf("Error verifying Rancher/Keycloak integration: %v", err))
						t.Fail(err.Error())
					}

					metrics.Emit(t.Metrics.With("get_kc_authconfig_state_elapsed_time", time.Since(start).Milliseconds()))
					// Verify Rancher auth settings
					minVer20, err := pkg.IsVerrazzanoMinVersion("2.0.0", kubeconfigPath)
					Expect(err).ToNot(HaveOccurred())
					if minVer20 {
						verifySettingValue(rancher.SettingAuthResyncCron, rancher.SettingAuthResyncCronValue, k8sClient)
						verifySettingValue(rancher.SettingAuthMaxAge, rancher.SettingAuthMaxAgeValue, k8sClient)
						verifySettingValue(rancher.SettingAuthTTL, rancher.SettingAuthTTLValue, k8sClient)
						verifySettingValue(rancher.SettingKubeDefaultTokenTTL, rancher.SettingKubeDefaultTokenTTLValue, k8sClient)
					}

					start = time.Now()
					t.Logs.Info("Verify Verrazzano rancher user")
					var rancherUsername string
					Eventually(func() (bool, error) {
						userList, err := k8sClient.Resource(pkg.GvkToGvr(rancher.GVKUser)).List(context.Background(), v1.ListOptions{})
						if err != nil {
							t.Logs.Error(fmt.Sprintf("error getting rancher verrazzano user: %v", err))
							return false, err
						}
						for _, userData := range userList.Items {
							username, ok := userData.UnstructuredContent()[rancher.UserAttributeUserName].(string)
							if !ok || username != rancher.UsernameVerrazzano {
								continue
							}
							userPrincipals, ok := userData.UnstructuredContent()[rancher.UserAttributePrincipalIDs].([]interface{})
							if ok {
								switch reflect.TypeOf(userPrincipals).Kind() {
								case reflect.Slice:
									listOfPrincipleIDs := reflect.ValueOf(userPrincipals)
									for i := 0; i < listOfPrincipleIDs.Len(); i++ {
										principleID := listOfPrincipleIDs.Index(i).Interface().(string)
										if strings.Contains(principleID, rancher.UserPrincipalKeycloakPrefix) {
											rancherUsername = userData.GetName()
											return true, nil
										}
									}
								}
							}
						}
						return false, fmt.Errorf("Verrazzano rancher user is not mapped in keycloak")
					}, waitTimeout, pollingInterval).Should(Equal(true), "verrazzano rancher user not correctly configured")
					metrics.Emit(t.Metrics.With("get_vz_rancher_user_elapsed_time", time.Since(start).Milliseconds()))

					start = time.Now()
					t.Logs.Info("Verify Verrazzano rancher user admin GlobalRoleBinding")
					Eventually(func() (bool, error) {
						grbData, err := k8sClient.Resource(pkg.GvkToGvr(rancher.GVKGlobalRoleBinding)).Get(context.Background(), rancher.GlobalRoleBindingVerrazzanoPrefix+rancherUsername, v1.GetOptions{})
						if err != nil {
							t.Logs.Error(fmt.Sprintf("error getting rancher verrazzano user global role binding for rancher user %s : %v", rancherUsername, err))
							return false, err
						}

						grbAttributes := grbData.UnstructuredContent()
						if grbAttributes[rancher.GlobalRoleBindingAttributeUserName].(string) != rancherUsername {
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
							crtpData, err := k8sClient.Resource(pkg.GvkToGvr(rancher.GVKClusterRoleTemplateBinding)).Namespace(rancher.ClusterLocal).Get(context.Background(), name, v1.GetOptions{})
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
						_, err := k8sClient.Resource(pkg.GvkToGvr(rancher.GVKRoleTemplate)).Get(context.Background(), rancher.VerrazzanoAdminRoleName, v1.GetOptions{})
						if err != nil {
							return false, fmt.Errorf("error getting RoleTemplate %s: %v", rancher.VerrazzanoAdminRoleName, err)
						}

						_, err = k8sClient.Resource(pkg.GvkToGvr(rancher.GVKRoleTemplate)).Get(context.Background(), rancher.VerrazzanoMonitorRoleName, v1.GetOptions{})
						if err != nil {
							return false, fmt.Errorf("error getting RoleTemplate %s: %v", rancher.VerrazzanoMonitorRoleName, err)
						}

						return true, nil
					}, waitTimeout, pollingInterval).Should(Equal(true), "RoleTemplate not found")
					metrics.Emit(t.Metrics.With("get_roletemplate_elapsed_time", time.Since(start).Milliseconds()))
					verifySettingValue(rancher.SettingUIPL, rancher.SettingUIPLValueVerrazzano, k8sClient)
					// VZ-11418
					//verifyUILogoSetting(rancher.SettingUILogoLight, rancher.SettingUILogoLightFile, k8sClient)
					//verifyUILogoSetting(rancher.SettingUILogoDark, rancher.SettingUILogoDarkFile, k8sClient)

				}

				minVer15, err := pkg.IsVerrazzanoMinVersion("1.5.0", kubeconfigPath)
				Expect(err).ToNot(HaveOccurred())
				if minVer15 {
					verifySettingValue(rancher.SettingUIPrimaryColor, rancher.SettingUIPrimaryColorValue, k8sClient)
					verifySettingValue(rancher.SettingUILinkColor, rancher.SettingUILinkColorValue, k8sClient)
					verifySettingValue(rancher.SettingUIBrand, rancher.SettingUIBrandValue, k8sClient)
				}
			}
		})
	})
})

var _ = t.AfterEach(func() {})

// verifyUILogoSetting verifies the value of ui logo related rancher setting
// GIVEN a Verrazzano installation with ui settings (ui-logo-dark and ui-logo-light) populated
// AND corresponding actual logo files present in specified path in a running rancher pod
//
//	WHEN value of the base64 encoded logo file is extracted from the setting CR specified by settingName
//	AND compared with base64 encoded value of corresponding actual logo file present in running rancher pod
//	THEN both the values are expected to be equal, otherwise the test scenario is deemed to have failed.
/*func verifyUILogoSetting(settingName string, logoFilename string, dynamicClient dynamic.Interface) {
	start := time.Now()
	t.Logs.Infof("Verify %s setting", settingName)
	Eventually(func() (bool, error) {
		clusterData, err := dynamicClient.Resource(pkg.GvkToGvr(common.GVKSetting)).Get(context.Background(), settingName, v1.GetOptions{})
		if err != nil {
			t.Logs.Error(fmt.Sprintf("Error getting %s setting: %v", settingName, err))
			return false, err
		}

		value := clusterData.UnstructuredContent()["value"].(string)
		logoSVG := strings.Split(value, rancher.SettingUILogoValueprefix)[1]
		// Strip out any extra carriage returns
		logoSVG = strings.ReplaceAll(logoSVG, "\r\r", "\r")
		cfg, err := k8sutil.GetKubeConfig()
		if err != nil {
			t.Logs.Error(fmt.Sprintf("Error getting client config to verify value of %s setting: %v", settingName, err))
			return false, err
		}

		c, err := client.New(cfg, client.Options{})
		if err != nil {
			t.Logs.Error(fmt.Sprintf("Error getting client to verify value of %s setting: %v", settingName, err))
			return false, err
		}

		pod, err := k8sutil.GetRunningPodForLabel(c, "app=rancher", "cattle-system")
		if err != nil {
			t.Logs.Error(fmt.Sprintf("Error getting running rancher pod to verify value of %s setting: %v", settingName, err))
			return false, err
		}

		k8sClient, err := kubernetes.NewForConfig(cfg)
		if err != nil {
			t.Logs.Error(fmt.Sprintf("Error getting kube client to verify value of %s setting: %v", settingName, err))
			return false, err
		}

		logoCommand := []string{"/bin/sh", "-c", fmt.Sprintf("cat %s/%s | base64", rancher.SettingUILogoFolder, logoFilename)}
		var stdout, stderr string
		stdout, _, err = k8sutil.ExecPod(k8sClient, cfg, pod, "rancher", logoCommand)
		if err != nil || strings.Contains(stdout, "No such file or directory") {
			// Try pre-Rancher 2.7.5 location
			logoCommand = []string{"/bin/sh", "-c", fmt.Sprintf("cat %s/%s | base64", rancher.SettingUILogoFolderBeforeRancher275, logoFilename)}
			stdout, stderr, err = k8sutil.ExecPod(k8sClient, cfg, pod, "rancher", logoCommand)
			if err != nil {
				t.Logs.Error(fmt.Sprintf("Error executing command in rancher pod to verify value of %s setting: %v, stderr: %v", settingName, err, stderr))
				return false, err
			}
			break
		}

		// Strip out any extra carriage returns
		stdout = strings.ReplaceAll(stdout, "\r\r", "\r")
		if stdout != logoSVG {
			t.Logs.Errorf("Setting %s unstructured %v", logoFilename, clusterData.UnstructuredContent())
			t.Logs.Errorf("Got %s for Rancher UI logo path, expected %s", stdout, logoSVG)
			return false, nil
		}
		return true, nil
	}, waitTimeout, pollingInterval).Should(Equal(true))
	metrics.Emit(t.Metrics.With("get_ui_setting_elapsed_time", time.Since(start).Milliseconds()))

}
*/
// verifySettingValue verifies the value of a rancher setting
// GIVEN a Verrazzano installation with setting specified by settingName populated
//
//	WHEN value field of the setting CR specified by settingName is extracted
//	AND compared with input expectedValue
//	THEN both the values are expected to be equal, otherwise the test scenario is deemed to have failed.
func verifySettingValue(settingName string, expectedValue string, k8sClient dynamic.Interface) {
	start := time.Now()
	t.Logs.Infof("Verify %s setting", settingName)
	Eventually(func() (bool, error) {
		clusterData, err := k8sClient.Resource(pkg.GvkToGvr(common.GVKSetting)).Get(context.Background(), settingName, v1.GetOptions{})
		if err != nil {
			t.Logs.Errorf("Error getting %s setting: %v", settingName, err.Error())
			return false, err
		}
		value := clusterData.UnstructuredContent()["value"].(string)
		return expectedValue == value, nil
	}, waitTimeout, pollingInterval).Should(Equal(true), fmt.Sprintf("rancher %s setting not updated", settingName))
	metrics.Emit(t.Metrics.With(fmt.Sprintf("get_%s_setting_elapsed_time", strings.ReplaceAll(settingName, "-", "")), time.Since(start).Milliseconds()))
}
