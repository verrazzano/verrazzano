// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package register_test

import (
	"context"
	"fmt"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	vmcv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	vmcClient "github.com/verrazzano/verrazzano/platform-operator/clients/clusters/clientset/versioned"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const waitTimeout = 10 * time.Minute
const pollingInterval = 10 * time.Second

const multiclusterNamespace = "verrazzano-mc"
const verrazzanoSystemNamespace = "verrazzano-system"

var managedClusterName = os.Getenv("MANAGED_CLUSTER_NAME")

var _ = Describe("Multi Cluster Verify Register", func() {
	Context("Admin Cluster", func() {
		BeforeEach(func() {
			os.Setenv("TEST_KUBECONFIG", os.Getenv("ADMIN_KUBECONFIG"))
		})

		It("admin cluster create VerrazzanoProject", func() {
			// create a project
			Eventually(func() error {
				return pkg.CreateOrUpdateResourceFromFile(fmt.Sprintf("testdata/multicluster/verrazzanoproject-%s.yaml", managedClusterName))
			}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())

			Eventually(func() (bool, error) {
				return findVerrazzanoProject(fmt.Sprintf("project-%s", managedClusterName))
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find VerrazzanoProject")
		})

		It("admin cluster has the expected VerrazzanoManagedCluster", func() {
			var client *vmcClient.Clientset
			Eventually(func() (*vmcClient.Clientset, error) {
				var err error
				client, err = pkg.GetVerrazzanoManagedClusterClientset()
				return client, err
			}, waitTimeout, pollingInterval).ShouldNot(BeNil())
			Eventually(func() bool {
				vmc, err := client.ClustersV1alpha1().VerrazzanoManagedClusters(multiclusterNamespace).Get(context.TODO(), managedClusterName, metav1.GetOptions{})
				return err == nil &&
					vmcStatusReady(vmc) &&
					vmc.Status.LastAgentConnectTime != nil &&
					vmc.Status.LastAgentConnectTime.After(time.Now().Add(-30*time.Minute))
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find VerrazzanoManagedCluster")
		})

		It("admin cluster has the expected ServiceAccounts and ClusterRoleBindings", func() {
			pkg.Concurrently(
				func() {
					Eventually(func() (bool, error) {
						return pkg.DoesServiceAccountExist(multiclusterNamespace, fmt.Sprintf("verrazzano-cluster-%s", managedClusterName))
					}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find ServiceAccount")
				},
				func() {
					Eventually(func() (bool, error) {
						return pkg.DoesClusterRoleBindingExist(fmt.Sprintf("verrazzano-cluster-%s", managedClusterName))
					}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find ClusterRoleBinding")
				},
			)
		})

		It("admin cluster has the expected secrets", func() {
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

		It("admin cluster has the expected system logs from admin and managed cluster", func() {
			verrazzanoIndex := "verrazzano-namespace-verrazzano-system"
			systemdIndex := "verrazzano-systemd-journal"
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

		It("admin cluster has the expected metrics from managed cluster", func() {
			Eventually(func() bool {
				return pkg.MetricsExist("up", "managed_cluster", managedClusterName)
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find a metrics from managed cluster")
		})
	})

	Context("Managed Cluster", func() {
		BeforeEach(func() {
			os.Setenv("TEST_KUBECONFIG", os.Getenv("MANAGED_KUBECONFIG"))
		})

		It("managed cluster has the expected secrets", func() {
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
				},
			)
		})

		It("managed cluster has the expected VerrazzanoProject", func() {
			Eventually(func() (bool, error) {
				return findVerrazzanoProject(fmt.Sprintf("project-%s", managedClusterName))
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find VerrazzanoProject")
		})

		It("managed cluster has the expected namespace", func() {
			Eventually(func() bool {
				return findNamespace(fmt.Sprintf("ns-%s", managedClusterName))
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find namespace")
		})

		It("managed cluster has the expected RoleBindings", func() {
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
	})
})

func vmcStatusReady(vmc *vmcv1alpha1.VerrazzanoManagedCluster) bool {
	pkg.Log(pkg.Info, fmt.Sprintf("VMC %s has %d status conditions\n", vmc.Name, len(vmc.Status.Conditions)))
	for _, cond := range vmc.Status.Conditions {
		pkg.Log(pkg.Info, fmt.Sprintf("VMC %s has status condition %s with value %s with message %s\n", vmc.Name, cond.Type, cond.Status, cond.Message))
		if cond.Type == vmcv1alpha1.ConditionReady && cond.Status == v1.ConditionTrue {
			return true
		}
	}
	return false
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
	if labels[constants.LabelVerrazzanoManaged] != constants.LabelVerrazzanoManagedDefault {
		pkg.Log(pkg.Info, fmt.Sprintf("The namespace %q label %q is set to wrong value of %q", namespace, constants.LabelVerrazzanoManaged, labels[constants.LabelVerrazzanoManaged]))
		return false
	}
	if labels[constants.LabelIstioInjection] != constants.LabelIstioInjectionDefault {
		pkg.Log(pkg.Info, fmt.Sprintf("The namespace %q label %q is set to wrong value of %q", namespace, constants.LabelIstioInjection, labels[constants.LabelIstioInjection]))
		return false
	}
	return true
}

func findVerrazzanoProject(projectName string) (bool, error) {
	config, err := clientcmd.BuildConfigFromFlags("", os.Getenv("TEST_KUBECONFIG"))
	if err != nil {
		pkg.Log(pkg.Info, fmt.Sprintf("Failed to build config from %s with error: %v", os.Getenv("TEST_KUBECONFIG"), err))
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
