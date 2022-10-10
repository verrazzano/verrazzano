// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package register_test

import (
	"context"
	"fmt"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	vmcv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	vmcClient "github.com/verrazzano/verrazzano/platform-operator/clientset/versioned"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const waitTimeout = 10 * time.Minute
const pollingInterval = 10 * time.Second

const longWaitTimeout = 20 * time.Minute
const longPollingInterval = 30 * time.Second

const multiclusterNamespace = "verrazzano-mc"
const verrazzanoSystemNamespace = "verrazzano-system"

var managedClusterName = os.Getenv("MANAGED_CLUSTER_NAME")
var vmiEsIngressURL = getVmiEsIngressURL()
var adminKubeconfig = os.Getenv("ADMIN_KUBECONFIG")
var externalEsURL = pkg.GetExternalOpenSearchURL(adminKubeconfig)

var t = framework.NewTestFramework("register_test")

var _ = t.AfterSuite(func() {})
var _ = t.BeforeSuite(func() {})
var _ = t.AfterEach(func() {})

var _ = t.Describe("Multi Cluster Verify Register", Label("f:multicluster.register"), func() {
	t.Context("Admin Cluster", func() {
		t.BeforeEach(func() {
			os.Setenv(k8sutil.EnvVarTestKubeConfig, os.Getenv("ADMIN_KUBECONFIG"))
		})

		t.It("create VerrazzanoProject", func() {
			if minimalVerification {
				Skip("Skipping since not part of minimal verification")
			}
			// create a project
			Eventually(func() error {
				return pkg.CreateOrUpdateResourceFromFile(fmt.Sprintf("testdata/multicluster/verrazzanoproject-%s.yaml", managedClusterName))
			}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())

			Eventually(func() (bool, error) {
				return findVerrazzanoProject(fmt.Sprintf("project-%s", managedClusterName))
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find VerrazzanoProject")
		})

		// This test is part of the minimal verification.
		t.It("has the expected VerrazzanoManagedCluster", func() {
			var client *vmcClient.Clientset
			// If registration happend in VZ versions 1.4 and above on admin cluster, check the ManagedCARetrieved condition as well
			adminVersionAtRegistration := os.Getenv("ADMIN_VZ_VERSION_AT_REGISTRATION")
			pkg.Log(pkg.Info, fmt.Sprintf("Admin cluster VZ version at registration is '%s'", adminVersionAtRegistration))
			regVersion14 := false
			var err error
			if adminVersionAtRegistration != "" {
				regVersion14, err = pkg.IsMinVersion(adminVersionAtRegistration, "1.4.0")
				if err != nil {
					Fail(err.Error())
				}
			}
			curAdminVersion14, err := pkg.IsVerrazzanoMinVersion("1.4.0", adminKubeconfig)
			if err != nil {
				Fail(err.Error())
			}
			Eventually(func() (*vmcClient.Clientset, error) {
				var err error
				client, err = pkg.GetVerrazzanoManagedClusterClientset()
				return client, err
			}, waitTimeout, pollingInterval).ShouldNot(BeNil())
			Eventually(func() bool {
				vmc, err := client.ClustersV1alpha1().VerrazzanoManagedClusters(multiclusterNamespace).Get(context.TODO(), managedClusterName, metav1.GetOptions{})
				return err == nil &&
					vmcStatusCheckOkay(vmc, regVersion14) &&
					vmcRancherStatusCheckOkay(vmc, curAdminVersion14) &&
					vmc.Status.LastAgentConnectTime != nil &&
					vmc.Status.LastAgentConnectTime.After(time.Now().Add(-30*time.Minute))
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find VerrazzanoManagedCluster")
		})

		t.It("has the expected ServiceAccounts", func() {
			if minimalVerification {
				Skip("Skipping since not part of minimal verification")
			}
			pkg.Concurrently(
				func() {
					Eventually(func() (bool, error) {
						return pkg.DoesServiceAccountExist(multiclusterNamespace, fmt.Sprintf("verrazzano-cluster-%s", managedClusterName))
					}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find ServiceAccount")
				},
			)
		})

		t.It("no longer has a ClusterRoleBinding for a managed cluster", func() {
			if minimalVerification {
				Skip("Skipping since not part of minimal verification")
			}
			supported, err := pkg.IsVerrazzanoMinVersion("1.1.0", adminKubeconfig)
			if err != nil {
				Fail(err.Error())
			}
			if supported {
				Eventually(func() (bool, error) {
					return pkg.DoesClusterRoleBindingExist(fmt.Sprintf("verrazzano-cluster-%s", managedClusterName))
				}, waitTimeout, pollingInterval).Should(BeFalse(), "Expected not to find ClusterRoleBinding")
			} else {
				pkg.Log(pkg.Info, "Skipping check, Verrazzano minimum version is not V1.1.0")
			}
		})

		t.It("has a ClusterRoleBinding for a managed cluster", func() {
			if minimalVerification {
				Skip("Skipping since not part of minimal verification")
			}
			supported, err := pkg.IsVerrazzanoMinVersion("1.1.0", adminKubeconfig)
			if err != nil {
				Fail(err.Error())
			}
			if !supported {
				Eventually(func() (bool, error) {
					return pkg.DoesClusterRoleBindingExist(fmt.Sprintf("verrazzano-cluster-%s", managedClusterName))
				}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find ClusterRoleBinding")
			} else {
				pkg.Log(pkg.Info, "Skipping check, Verrazzano minimum version is not less than V1.1.0")
			}
		})

		t.It("has the expected secrets", func() {
			if minimalVerification {
				Skip("Skipping since not part of minimal verification")
			}
			pkg.Concurrently(
				func() {
					secretName := fmt.Sprintf("verrazzano-cluster-%s-manifest", managedClusterName)
					Eventually(func() bool {
						return findSecret(multiclusterNamespace, secretName)
					}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find secret "+secretName)
				},
				func() {
					secretName := fmt.Sprintf("verrazzano-cluster-%s-agent", managedClusterName)
					Eventually(func() bool {
						return findSecret(multiclusterNamespace, secretName)
					}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find secret "+secretName)
				},
				func() {
					secretName := fmt.Sprintf("verrazzano-cluster-%s-registration", managedClusterName)
					Eventually(func() bool {
						return findSecret(multiclusterNamespace, secretName)
					}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find secret "+secretName)
				},
			)
		})

		t.It("has the expected system logs from admin and managed cluster", func() {
			if minimalVerification {
				Skip("Skipping since not part of minimal verification")
			}
			verrazzanoIndex, err := pkg.GetOpenSearchSystemIndex("verrazzano-system")
			Expect(err).To(BeNil())
			systemdIndex, err := pkg.GetOpenSearchSystemIndex("systemd-journal")
			Expect(err).To(BeNil())
			pkg.Concurrently(
				func() {
					Eventually(func() bool {
						return pkg.FindLog(verrazzanoIndex,
							[]pkg.Match{
								{Key: "kubernetes.container_name", Value: "verrazzano-application-operator"},
								{Key: "cluster_name.keyword", Value: constants.DefaultClusterName}},
							[]pkg.Match{
								{Key: "cluster_name", Value: managedClusterName}})
					}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find a systemd log record")
				},
				func() {
					Eventually(func() bool {
						return pkg.FindLog(systemdIndex,
							[]pkg.Match{
								{Key: "tag", Value: "systemd"},
								{Key: "cluster_name.keyword", Value: constants.DefaultClusterName},
								{Key: "SYSTEMD_UNIT", Value: "kubelet.service"}},
							[]pkg.Match{
								{Key: "cluster_name", Value: managedClusterName}})
					}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find a systemd log record")
				},
				func() {
					Eventually(func() bool {
						return pkg.FindLog(verrazzanoIndex,
							[]pkg.Match{
								{Key: "kubernetes.container_name", Value: "verrazzano-application-operator"},
								{Key: "cluster_name.keyword", Value: managedClusterName}},
							[]pkg.Match{
								{Key: "cluster_name.keyword", Value: constants.DefaultClusterName}})
					}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find a systemd log record")
				},
				func() {
					Eventually(func() bool {
						return pkg.FindLog(systemdIndex,
							[]pkg.Match{
								{Key: "tag", Value: "systemd"},
								{Key: "cluster_name", Value: managedClusterName},
								{Key: "SYSTEMD_UNIT.keyword", Value: "kubelet.service"}},
							[]pkg.Match{
								{Key: "cluster_name", Value: constants.DefaultClusterName}})
					}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find a systemd log record")
				},
			)
		})

		t.It("has the expected metrics from managed cluster", func() {
			if minimalVerification {
				Skip("Skipping since not part of minimal verification")
			}
			clusterNameMetricsLabel := getClusterNameMetricLabel(adminKubeconfig)
			pkg.Log(pkg.Info, fmt.Sprintf("Looking for metric with label %s with value %s", clusterNameMetricsLabel, managedClusterName))
			Eventually(func() bool {
				return pkg.MetricsExist("up", clusterNameMetricsLabel, managedClusterName)
			}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find metrics from managed cluster")
		})

		t.It("Fluentd should point to the correct ES", func() {
			if minimalVerification {
				Skip("Skipping since not part of minimal verification")
			}
			supported, err := pkg.IsVerrazzanoMinVersion("1.3.0", adminKubeconfig)
			if err != nil {
				Fail(err.Error())
			}
			if pkg.UseExternalElasticsearch() {
				Eventually(func() bool {
					return pkg.AssertFluentdURLAndSecret(externalEsURL, "external-es-secret")
				}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected external ES in admin cluster fluentd Daemonset setting")
			} else {
				var secret string
				if supported {
					secret = pkg.VmiESInternalSecret
				} else {
					secret = pkg.VmiESLegacySecret
				}
				Eventually(func() bool {
					return pkg.AssertFluentdURLAndSecret("", secret)
				}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected VMI ES in admin cluster fluentd Daemonset setting")
			}
		})
	})

	t.Context("Managed Cluster", func() {
		t.BeforeEach(func() {
			os.Setenv(k8sutil.EnvVarTestKubeConfig, os.Getenv("MANAGED_KUBECONFIG"))
		})

		t.It("has the expected secrets", func() {
			if minimalVerification {
				Skip("Skipping since not part of minimal verification")
			}
			pkg.Concurrently(
				func() {
					Eventually(func() bool {
						return findSecret(verrazzanoSystemNamespace, "verrazzano-cluster-agent")
					}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find secret verrazzano-cluster-agent")
				},
				func() {
					Eventually(func() bool {
						return findSecret(verrazzanoSystemNamespace, "verrazzano-cluster-registration")
					}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find secret verrazzano-cluster-registration")
					assertRegistrationSecret()
				},
			)
		})

		t.It("has the expected VerrazzanoProject", func() {
			if minimalVerification {
				Skip("Skipping since not part of minimal verification")
			}
			Eventually(func() (bool, error) {
				return findVerrazzanoProject(fmt.Sprintf("project-%s", managedClusterName))
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find VerrazzanoProject")
		})

		t.It("has the expected namespace", func() {
			if minimalVerification {
				Skip("Skipping since not part of minimal verification")
			}
			Eventually(func() bool {
				return findNamespace(fmt.Sprintf("ns-%s", managedClusterName))
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find namespace")
		})

		t.It("has the expected RoleBindings", func() {
			if minimalVerification {
				Skip("Skipping since not part of minimal verification")
			}
			namespace := fmt.Sprintf("ns-%s", managedClusterName)
			pkg.Concurrently(
				func() {
					Eventually(func() (bool, error) {
						return pkg.DoesRoleBindingContainSubject(namespace, "verrazzano-project-admin", "User", "test-user")
					}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find role binding verrazzano-project-admin")
				},
				func() {
					Eventually(func() (bool, error) {
						return pkg.DoesRoleBindingContainSubject(namespace, "admin", "User", "test-user")
					}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find role binding admin")
				},
				func() {
					Eventually(func() (bool, error) {
						return pkg.DoesRoleBindingContainSubject(namespace, "verrazzano-project-monitor", "Group", "test-viewers")
					}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find role binding verrazzano-project-monitor")
				},
				func() {
					Eventually(func() (bool, error) {
						return pkg.DoesRoleBindingContainSubject(namespace, "view", "Group", "test-viewers")
					}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find role binding view")
				},
			)
		})

		t.It("Fluentd should point to the correct ES", func() {
			if minimalVerification {
				Skip("Skipping since not part of minimal verification")
			}
			if pkg.UseExternalElasticsearch() {
				Eventually(func() bool {
					return pkg.AssertFluentdURLAndSecret(externalEsURL, "verrazzano-cluster-registration")
				}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected external ES in managed cluster fluentd Daemonset setting")
			} else {
				Eventually(func() bool {
					return pkg.AssertFluentdURLAndSecret("", "verrazzano-cluster-registration")
				}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected VMI ES  in managed cluster fluentd Daemonset setting")
			}
		})
	})
})

func vmcRancherStatusCheckOkay(vmc *vmcv1alpha1.VerrazzanoManagedCluster, versionSupportsClusterID bool) bool {
	pkg.Log(pkg.Info, fmt.Sprintf("VMC %s has Rancher status %s and cluster id %s\n",
		vmc.Name, vmc.Status.RancherRegistration.Status, vmc.Status.RancherRegistration.ClusterID))
	clusterIDConditionMet := true
	if versionSupportsClusterID {
		// if this VZ version supports cluster id in rancher reg status, then it should be present
		clusterIDConditionMet = vmc.Status.RancherRegistration.ClusterID != ""
	}
	return vmc.Status.RancherRegistration.Status == vmcv1alpha1.RegistrationCompleted && clusterIDConditionMet
}

func vmcStatusCheckOkay(vmc *vmcv1alpha1.VerrazzanoManagedCluster, managedCAConditionSupported bool) bool {
	pkg.Log(pkg.Info, fmt.Sprintf("VMC %s has %d status conditions\n", vmc.Name, len(vmc.Status.Conditions)))
	readyConditionMet := false
	managedCAConditionMet := false
	for _, cond := range vmc.Status.Conditions {
		pkg.Log(pkg.Info, fmt.Sprintf("VMC %s has status condition %s with value %s with message %s\n", vmc.Name, cond.Type, cond.Status, cond.Message))
		if cond.Type == vmcv1alpha1.ConditionReady && cond.Status == v1.ConditionTrue {
			readyConditionMet = true
		}
		// If admin cluster VZ version at registration time supports it, check the ManagedCARetrieved condition as well
		if managedCAConditionSupported {
			pkg.Log(pkg.Info, "Checking for ManagedCARetrieved condition")
			if cond.Type == vmcv1alpha1.ConditionManagedCARetrieved && cond.Status == v1.ConditionTrue {
				managedCAConditionMet = true
			}
		} else {
			// In older versions of VZ, no need to check managed CA condition since the condition doesn't exist
			managedCAConditionMet = true
		}
	}
	return readyConditionMet && managedCAConditionMet
}

func findSecret(namespace, name string) bool {
	s, err := pkg.GetSecret(namespace, name)
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to get secret %s in namespace %s with error: %v", name, namespace, err))
		return false
	}
	return s != nil
}

func findNamespace(namespace string) bool {
	ns, err := pkg.GetNamespace(namespace)
	if err != nil {
		if !errors.IsNotFound(err) {
			pkg.Log(pkg.Error, fmt.Sprintf("Failed to get namespace %s with error: %v", namespace, err))
			return false
		}
		pkg.Log(pkg.Info, fmt.Sprintf("The namespace %q is not found", namespace))
		return false
	}
	labels := ns.GetObjectMeta().GetLabels()
	if labels[vzconst.VerrazzanoManagedLabelKey] != constants.LabelVerrazzanoManagedDefault {
		pkg.Log(pkg.Info, fmt.Sprintf("The namespace %q label %q is set to wrong value of %q", namespace, vzconst.VerrazzanoManagedLabelKey, labels[vzconst.VerrazzanoManagedLabelKey]))
		return false
	}
	if labels[constants.LabelIstioInjection] != constants.LabelIstioInjectionDefault {
		pkg.Log(pkg.Info, fmt.Sprintf("The namespace %q label %q is set to wrong value of %q", namespace, constants.LabelIstioInjection, labels[constants.LabelIstioInjection]))
		return false
	}
	return true
}

func findVerrazzanoProject(projectName string) (bool, error) {
	config, err := clientcmd.BuildConfigFromFlags("", os.Getenv(k8sutil.EnvVarTestKubeConfig))
	if err != nil {
		pkg.Log(pkg.Info, fmt.Sprintf("Failed to build config from %s with error: %v", os.Getenv(k8sutil.EnvVarTestKubeConfig), err))
		return false, err
	}

	scheme := runtime.NewScheme()
	_ = clustersv1alpha1.AddToScheme(scheme)
	_ = vmcv1alpha1.AddToScheme(scheme)

	clustersClient, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		pkg.Log(pkg.Info, fmt.Sprintf("Failed to get clusters client with error: %v", err))
		return false, err
	}

	projectList := clustersv1alpha1.VerrazzanoProjectList{}
	err = clustersClient.List(context.TODO(), &projectList, &client.ListOptions{Namespace: constants.VerrazzanoMultiClusterNamespace})
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to list VerrazzanoProject with error: %v", err))
		return false, err
	}
	for _, item := range projectList.Items {
		if item.Name == projectName && item.Namespace == multiclusterNamespace {
			return true, nil
		}
	}
	return false, nil
}

func assertRegistrationSecret() {
	regSecret, err := pkg.GetSecret(verrazzanoSystemNamespace, "verrazzano-cluster-registration")
	Expect(err).To(BeNil())
	Expect(regSecret).To(Not(BeNil()))
	if pkg.UseExternalElasticsearch() {
		Expect(string(regSecret.Data["es-url"])).To(Equal(externalEsURL))
		esSecret, err := pkg.GetSecretInCluster("verrazzano-system", "external-es-secret", os.Getenv("ADMIN_KUBECONFIG"))
		Expect(err).To(BeNil())
		Expect(regSecret.Data["username"]).To(Equal(esSecret.Data["username"]))
		Expect(regSecret.Data["password"]).To(Equal(esSecret.Data["password"]))
		Expect(regSecret.Data["es-ca-bundle"]).To(Equal(esSecret.Data["ca-bundle"]))
	} else {
		Expect(string(regSecret.Data["es-url"])).To(Equal(vmiEsIngressURL))
		vmiEsInternalSecret, err := pkg.GetSecretInCluster("verrazzano-system", "verrazzano-es-internal", os.Getenv("ADMIN_KUBECONFIG"))
		Expect(err).To(BeNil())
		Expect(regSecret.Data["username"]).To(Equal(vmiEsInternalSecret.Data["username"]))
		Expect(regSecret.Data["password"]).To(Equal(vmiEsInternalSecret.Data["password"]))
	}
}

func getVmiEsIngressURL() string {
	return fmt.Sprintf("%s:443", pkg.GetSystemOpenSearchIngressURL(adminKubeconfig))
}

func getClusterNameMetricLabel(kubeconfigPath string) string {
	// ignore error getting the metric label - we'll just use the default value returned
	lbl, err := pkg.GetClusterNameMetricLabel(kubeconfigPath)
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Error getting cluster name metric label: %s", err.Error()))
	}
	return lbl
}
