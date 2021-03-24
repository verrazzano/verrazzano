// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package permissions_test

import (
	"context"
	"fmt"
	"k8s.io/api/apps/v1beta1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"os"
	"strings"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	vmcv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const waitTimeout = 10 * time.Minute
const pollingInterval = 10 * time.Second

const testNamespace = "multiclustertest"

var managedClusterName = os.Getenv("MANAGED_CLUSTER_NAME")

var _ = ginkgo.Describe("Multi Cluster Verify Kubeconfig Permissions", func() {

	ginkgo.Context("Admin Cluster - create mc resources and update statuses", func() {
		ginkgo.BeforeEach(func() {
			os.Setenv("TEST_KUBECONFIG", os.Getenv("ADMIN_KUBECONFIG"))
		})

		ginkgo.It("admin cluster create mc config map", func() {
			// create a config map
			err := pkg.CreateOrUpdateResourceFromFile("testdata/multicluster/multicluster_configmap.yaml")
			if err != nil {
				ginkgo.Fail(fmt.Sprintf("Failed to create multi cluster config map: %v", err))
			}

			gomega.Eventually(func() bool {
				return findMultiClusterConfigMap(testNamespace, "mymcconfigmap")
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue(), "Expected to find mc configmap")

			gomega.Eventually(func() bool {
				// Verify we have the expected status update
				configMap := clustersv1alpha1.MultiClusterConfigMap{}
				err := getMultiClusterResource(testNamespace, "mymcconfigmap", &configMap)
				return err == nil && configMap.Status.State == clustersv1alpha1.Succeeded &&
					isStatusAsExpected(configMap.Status, clustersv1alpha1.DeployComplete, "created", clustersv1alpha1.Succeeded, managedClusterName)
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue())
		})

		ginkgo.It("admin cluster create mc logging scope", func() {
			// create a logging scope
			err := pkg.CreateOrUpdateResourceFromFile("testdata/multicluster/multicluster_loggingscope.yaml")
			if err != nil {
				ginkgo.Fail(fmt.Sprintf("Failed to create multi cluster logging scope: %v", err))
			}

			gomega.Eventually(func() bool {
				return findMultiClusterLoggingScope(testNamespace, "mymcloggingscope")
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue(), "Expected to find mc logging scope")

			gomega.Eventually(func() bool {
				// Verify we have the expected status update
				loggingScope := clustersv1alpha1.MultiClusterLoggingScope{}
				err := getMultiClusterResource(testNamespace, "mymcloggingscope", &loggingScope)
				return err == nil && loggingScope.Status.State == clustersv1alpha1.Succeeded &&
					isStatusAsExpected(loggingScope.Status, clustersv1alpha1.DeployComplete, "created", clustersv1alpha1.Succeeded, managedClusterName)
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue())
		})

		ginkgo.It("admin cluster create mc secret", func() {
			// create a config map
			err := pkg.CreateOrUpdateResourceFromFile("testdata/multicluster/multicluster_secret.yaml")
			if err != nil {
				ginkgo.Fail(fmt.Sprintf("Failed to create multi cluster secret: %v", err))
			}

			gomega.Eventually(func() bool {
				return findMultiClusterSecret(testNamespace, "mymcsecret")
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue(), "Expected to find mc secret")

			gomega.Eventually(func() bool {
				// Verify we have the expected status update
				secret := clustersv1alpha1.MultiClusterSecret{}
				err := getMultiClusterResource(testNamespace, "mymcsecret", &secret)
				return err == nil && secret.Status.State == clustersv1alpha1.Succeeded &&
					isStatusAsExpected(secret.Status, clustersv1alpha1.DeployComplete, "created", clustersv1alpha1.Succeeded, managedClusterName)
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue())
		})

		ginkgo.It("admin cluster vmc status updates", func() {
			gomega.Eventually(func() bool {
				// Verify we have the expected status update
				vmc := vmcv1alpha1.VerrazzanoManagedCluster{}
				err := getMultiClusterResource("verrazzano-mc", managedClusterName, &vmc)
				return err == nil && vmc.Status.LastAgentConnectTime.After(time.Now().Add(-30*time.Minute))
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue(), "Expected to find VerrazzanoManagedCluster")
		})

	})

	ginkgo.Context("Managed Cluster - check for underlying resources", func() {
		ginkgo.BeforeEach(func() {
			os.Setenv("TEST_KUBECONFIG", os.Getenv("MANAGED_KUBECONFIG"))
		})

		ginkgo.It("managed cluster has the expected mc and underlying configmap", func() {
			pkg.Concurrently(
				func() {
					gomega.Eventually(findConfigMap(testNamespace, "mymcconfigmap"),
						waitTimeout, pollingInterval).Should(gomega.BeTrue(), "Expected to find configmap")
				},
				func() {
					gomega.Eventually(findMultiClusterConfigMap(testNamespace, "myconfigmap"),
						waitTimeout, pollingInterval).Should(gomega.BeTrue(), "Expected to find mc configmap")
				},
			)
		})

		ginkgo.It("managed cluster has the expected mc and underlying logging scope", func() {
			pkg.Concurrently(
				func() {
					gomega.Eventually(findLoggingScope(testNamespace, "mymcloggingscope"),
						waitTimeout, pollingInterval).Should(gomega.BeTrue(), "Expected to find logging scope")
				},
				func() {
					gomega.Eventually(findMultiClusterLoggingScope(testNamespace, "mymcloggingscope"),
						waitTimeout, pollingInterval).Should(gomega.BeTrue(), "Expected to find mc loggging scope")
				},
			)
		})

		ginkgo.It("managed cluster has the expected mc and underlying secret", func() {
			pkg.Concurrently(
				func() {
					gomega.Eventually(findSecret(testNamespace, "mymcsecret"),
						waitTimeout, pollingInterval).Should(gomega.BeTrue(), "Expected to find secret")
				},
				func() {
					gomega.Eventually(findMultiClusterSecret(testNamespace, "mymcsecret"),
						waitTimeout, pollingInterval).Should(gomega.BeTrue(), "Expected to find mc secret")
				},
			)
		})
	})

	ginkgo.Context("Managed Cluster - MC object access on admin cluster", func() {
		ginkgo.BeforeEach(func() {
			os.Setenv("TEST_KUBECONFIG", os.Getenv("MANAGED_ACCESS_KUBECONFIG"))
		})

		ginkgo.It("managed cluster can access config map but not modify it", func() {
			gomega.Eventually(findMultiClusterConfigMap(testNamespace, "myconfigmap"),
				waitTimeout, pollingInterval).Should(gomega.BeTrue(), "Expected to find mc configmap")
			// try to update
			err := pkg.CreateOrUpdateResourceFromFile("testdata/multicluster/multicluster_configmap_update.yaml")
			if err == nil {
				ginkgo.Fail("Update to config map succeeded")
			}
			if !errors.IsForbidden(err) {
				ginkgo.Fail("Wrong error generated - should be forbidden")
			}
			// try to delete
			err = pkg.DeleteResourceFromFile("testdata/multicluster/multicluster_configmap.yaml")
			if err == nil {
				ginkgo.Fail("Delete of config map succeeded")
			}
			if !errors.IsForbidden(err) {
				ginkgo.Fail("Wrong error generated - should be forbidden")
			}
		})

		ginkgo.It("managed cluster can access logging scope but not modify it", func() {
			gomega.Eventually(findMultiClusterLoggingScope(testNamespace, "mymcloggingscope"),
				waitTimeout, pollingInterval).Should(gomega.BeTrue(), "Expected to find mc loggging scope")
			// try to update
			err := pkg.CreateOrUpdateResourceFromFile("testdata/multicluster/multicluster_loggingscope_update.yaml")
			if err == nil {
				ginkgo.Fail("Update to logging scope succeeded")
			}
			if !errors.IsForbidden(err) {
				ginkgo.Fail("Wrong error generated - should be forbidden")
			}
			// try to delete
			err = pkg.DeleteResourceFromFile("testdata/multicluster/multicluster_loggingscope.yaml")
			if err == nil {
				ginkgo.Fail("Delete of logging scope succeeded")
			}
			if !errors.IsForbidden(err) {
				ginkgo.Fail("Wrong error generated - should be forbidden")
			}
		})

		ginkgo.It("managed cluster can access secret but not modify it", func() {
			gomega.Eventually(findMultiClusterSecret(testNamespace, "mymcsecret"),
				waitTimeout, pollingInterval).Should(gomega.BeTrue(), "Expected to find mc secret")
			// try to update
			err := pkg.CreateOrUpdateResourceFromFile("testdata/multicluster/multicluster_secret_update.yaml")
			if err == nil {
				ginkgo.Fail("Update to secret succeeded")
			}
			if !errors.IsForbidden(err) {
				ginkgo.Fail("Wrong error generated - should be forbidden")
			}
			// try to delete
			err = pkg.DeleteResourceFromFile("testdata/multicluster/multicluster_secret.yaml")
			if err == nil {
				ginkgo.Fail("Delete of secret succeeded")
			}
			if !errors.IsForbidden(err) {
				ginkgo.Fail("Wrong error generated - should be forbidden")
			}
		})

		ginkgo.It("managed cluster cannot access resources in other namespaaces", func() {
			err := listResource("verrazzano-system", &v1.SecretList{})
			if err == nil {
				ginkgo.Fail("read access allowed")
			}
			if !errors.IsForbidden(err) {
				ginkgo.Fail("Wrong error generated - should be forbidden")
			}
			err = listResource("verrazzano-system", &v1.ConfigMapList{})
			if err == nil {
				ginkgo.Fail("read access allowed")
			}
			if !errors.IsForbidden(err) {
				ginkgo.Fail("Wrong error generated - should be forbidden")
			}
			err = listResource("verrazzano-mc", &v1.SecretList{})
			if err == nil {
				ginkgo.Fail("read access allowed")
			}
			if !errors.IsForbidden(err) {
				ginkgo.Fail("Wrong error generated - should be forbidden")
			}
			err = listResource("verrazzano-mc", &v1.ConfigMapList{})
			if err == nil {
				ginkgo.Fail("read access allowed")
			}
			if !errors.IsForbidden(err) {
				ginkgo.Fail("Wrong error generated - should be forbidden")
			}
			err = listResource(testNamespace, &v1.SecretList{})
			if err == nil {
				ginkgo.Fail("read access allowed")
			}
			if !errors.IsForbidden(err) {
				ginkgo.Fail("Wrong error generated - should be forbidden")
			}
			err = listResource(testNamespace, &v1.ConfigMapList{})
			if err == nil {
				ginkgo.Fail("read access allowed")
			}
			if !errors.IsForbidden(err) {
				ginkgo.Fail("Wrong error generated - should be forbidden")
			}
		})
	})
})

func findSecret(namespace, name string) bool {
	s, err := pkg.GetSecret(namespace, name)
	return s != nil && err == nil
}

func findConfigMap(namespace, name string) bool {
	s := pkg.GetConfigMap(name, namespace)
	return s != nil
}

func listResource(namespace string, object runtime.Object) error {
	err, clustersClient := getClustersClient()
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to obtain client with error: %v", err))
	}

	return clustersClient.List(context.TODO(), object, &client.ListOptions{Namespace: namespace})
}

func findLoggingScope(namespace, name string) bool {
	err, clustersClient := getClustersClient()
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to obtain client with error: %v", err))
	}

	loggingscopeList := v1alpha1.LoggingScopeList{}
	err = clustersClient.List(context.TODO(), &loggingscopeList, &client.ListOptions{Namespace: namespace})
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to list logging scopes with error: %v", err))
	}
	for _, item := range loggingscopeList.Items {
		if item.Name == name && item.Namespace == namespace {
			return true
		}
	}
	return false
}

func getMultiClusterResource(namespace, name string, object runtime.Object) error {
	err, clustersClient := getClustersClient()
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to obtain client with error: %v", err))
	}

	err = clustersClient.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, object)

	return err
}

func findMultiClusterConfigMap(namespace, name string) bool {
	err, clustersClient := getClustersClient()
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to obtain client with error: %v", err))
	}

	configmapList := clustersv1alpha1.MultiClusterConfigMapList{}
	err = clustersClient.List(context.TODO(), &configmapList, &client.ListOptions{Namespace: namespace})
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to list multi cluster configmaps with error: %v", err))
	}
	for _, item := range configmapList.Items {
		if item.Name == name && item.Namespace == namespace {
			return true
		}
	}
	return false
}

func findMultiClusterLoggingScope(namespace, name string) bool {
	err, clustersClient := getClustersClient()
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to obtain client with error: %v", err))
	}

	loggingscopeList := clustersv1alpha1.MultiClusterLoggingScopeList{}
	err = clustersClient.List(context.TODO(), &loggingscopeList, &client.ListOptions{Namespace: namespace})
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to list multi cluster logging scopes with error: %v", err))
	}
	for _, item := range loggingscopeList.Items {
		if item.Name == name && item.Namespace == namespace {
			return true
		}
	}
	return false
}

func findMultiClusterSecret(namespace, name string) bool {
	err, clustersClient := getClustersClient()
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to obtain client with error: %v", err))
	}

	secretList := clustersv1alpha1.MultiClusterSecretList{}
	err = clustersClient.List(context.TODO(), &secretList, &client.ListOptions{Namespace: namespace})
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to list multi cluster secrets with error: %v", err))
	}
	for _, item := range secretList.Items {
		if item.Name == name && item.Namespace == namespace {
			return true
		}
	}
	return false
}

func getClustersClient() (error, client.Client) {
	config, err := clientcmd.BuildConfigFromFlags("", os.Getenv("TEST_KUBECONFIG"))
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to build config from %s with error: %v", os.Getenv("TEST_KUBECONFIG"), err))
	}

	scheme := runtime.NewScheme()
	_ = clustersv1alpha1.AddToScheme(scheme)
	_ = vmcv1alpha1.AddToScheme(scheme)

	clustersClient, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to get clusters client with error: %v", err))
	}
	return err, clustersClient
}

func isStatusAsExpected(status clustersv1alpha1.MultiClusterResourceStatus,
	expectedConditionType clustersv1alpha1.ConditionType, conditionMsgContains string,
	expectedClusterState clustersv1alpha1.StateType,
	expectedClusterName string) bool {
	matchingConditionCount := 0
	matchingClusterStatusCount := 0
	for _, condition := range status.Conditions {
		if condition.Type == expectedConditionType && strings.Contains(condition.Message, conditionMsgContains) {
			matchingConditionCount++
		}
	}
	for _, clusterStatus := range status.Clusters {
		if clusterStatus.State == expectedClusterState &&
			clusterStatus.Name == expectedClusterName &&
			clusterStatus.LastUpdateTime != "" {
			matchingClusterStatusCount++
		}
	}
	return matchingConditionCount == 1 && matchingClusterStatusCount == 1
}
