// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package permissions_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	vmcv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

const waitTimeout = 10 * time.Minute
const pollingInterval = 10 * time.Second

const testNamespace = "multiclustertest"
const anotherTestNamespace = "anothermulticlustertest"

var managedClusterName = os.Getenv("MANAGED_CLUSTER_NAME")

var _ = BeforeSuite(func() {
	// Do set up for multi cluster tests
	deployTestResources()
})

var _ = AfterSuite(func() {
	// Do set up for multi cluster tests
	undeployTestResources()
})

var _ = Describe("Multi Cluster Verify Kubeconfig Permissions", func() {

	// vZ-2336: Be able to read MultiClusterXXX resources in the admin cluster
	//			Be able to update the status of MultiClusterXXX resources in the admin cluster
	Context("Admin Cluster - verify mc resources and their status updates", func() {
		BeforeEach(func() {
			os.Setenv("TEST_KUBECONFIG", os.Getenv("ADMIN_KUBECONFIG"))
		})

		It("admin cluster - verify mc config map", func() {
			Eventually(func() bool {
				return findMultiClusterConfigMap(testNamespace, "mymcconfigmap")
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find mc configmap")

			eventuallyIterations := 0
			Eventually(func() bool {
				// Verify we have the expected status update
				configMap := clustersv1alpha1.MultiClusterConfigMap{}
				err := getMultiClusterResource(testNamespace, "mymcconfigmap", &configMap)
				pkg.Log(pkg.Debug, fmt.Sprintf("Size of clusters array: %d", len(configMap.Status.Clusters)))
				if len(configMap.Status.Clusters) > 0 {
					pkg.Log(pkg.Debug, string("cluster reported status: "+configMap.Status.Clusters[0].State))
					pkg.Log(pkg.Debug, "cluster reported name: "+configMap.Status.Clusters[0].Name)
				}
				eventuallyIterations++
				if eventuallyIterations >= 30 && eventuallyIterations%10 == 0 {
					pkg.Log(pkg.Info, "Dumping Status of config map mymcconfigmap every 10 iterations of Eventually block after we hit 30 iterations")
					pkg.Log(pkg.Info, fmt.Sprintf("Conditions: %v", configMap.Status.Conditions))
					pkg.Log(pkg.Info, fmt.Sprintf("Clusters: %v", configMap.Status.Clusters))
					pkg.Log(pkg.Info, fmt.Sprintf("State: %v", configMap.Status.State))
				}
				return err == nil && configMap.Status.State == clustersv1alpha1.Succeeded &&
					isStatusAsExpected(configMap.Status, clustersv1alpha1.DeployComplete, "created", clustersv1alpha1.Succeeded, managedClusterName)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})

		It("admin cluster - verify mc secret", func() {
			Eventually(func() bool {
				return findMultiClusterSecret(anotherTestNamespace, "mymcsecret")
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find mc secret")

			Eventually(func() bool {
				// Verify we have the expected status update
				secret := clustersv1alpha1.MultiClusterSecret{}
				err := getMultiClusterResource(anotherTestNamespace, "mymcsecret", &secret)
				pkg.Log(pkg.Debug, fmt.Sprintf("Size of clusters array: %d", len(secret.Status.Clusters)))
				if len(secret.Status.Clusters) > 0 {
					pkg.Log(pkg.Debug, string("cluster reported status: "+secret.Status.Clusters[0].State))
					pkg.Log(pkg.Debug, "cluster reported name: "+secret.Status.Clusters[0].Name)
				}
				return err == nil && secret.Status.State == clustersv1alpha1.Succeeded &&
					isStatusAsExpected(secret.Status, clustersv1alpha1.DeployComplete, "created", clustersv1alpha1.Succeeded, managedClusterName)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})

		// VZ-2336: Be able to update the status of a VerrazzanoManagedCluster resource
		It("admin cluster vmc status updates", func() {
			Eventually(func() bool {
				// Verify we have the expected status update
				vmc := vmcv1alpha1.VerrazzanoManagedCluster{}
				err := getMultiClusterResource("verrazzano-mc", managedClusterName, &vmc)
				return err == nil && vmc.Status.LastAgentConnectTime.After(time.Now().Add(-30*time.Minute))
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find VerrazzanoManagedCluster")
		})

	})

	Context("Managed Cluster - check for underlying resources", func() {
		BeforeEach(func() {
			os.Setenv("TEST_KUBECONFIG", os.Getenv("MANAGED_KUBECONFIG"))
		})

		It("managed cluster has the expected mc and underlying configmap", func() {
			pkg.Concurrently(
				func() {
					Eventually(func() bool {
						return findConfigMap(testNamespace, "mymcconfigmap")
					}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find configmap")
				},
				func() {
					Eventually(func() bool {
						return findMultiClusterConfigMap(testNamespace, "mymcconfigmap")
					}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find mc configmap")
				},
			)
		})

		It("managed cluster has the expected mc and underlying secret", func() {
			pkg.Concurrently(
				func() {
					Eventually(func() bool {
						return findSecret(anotherTestNamespace, "mymcsecret")
					}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find secret")
				},
				func() {
					Eventually(func() bool {
						return findMultiClusterSecret(anotherTestNamespace, "mymcsecret")
					}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find mc secret")
				},
			)
		})
	})

	// VZ-2336:  NOT be able to update or delete any MultiClusterXXX resources in the admin cluster
	Context("Managed Cluster - MC object access on admin cluster", func() {
		BeforeEach(func() {
			os.Setenv("TEST_KUBECONFIG", os.Getenv("MANAGED_ACCESS_KUBECONFIG"))
		})

		It("managed cluster can access config map but not modify it", func() {
			Eventually(func() bool {
				return findMultiClusterConfigMap(testNamespace, "mymcconfigmap")
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find mc configmap")
			// try to update
			err := CreateOrUpdateResourceFromFile("testdata/multicluster/multicluster_configmap_update.yaml", &clustersv1alpha1.MultiClusterConfigMap{})
			if err == nil {
				Fail("Update to config map succeeded")
			}
			if !errors.IsForbidden(err) {
				Fail("Wrong error generated - should be forbidden")
			}
			// try to delete
			err = DeleteResourceFromFile("testdata/multicluster/multicluster_configmap.yaml", &clustersv1alpha1.MultiClusterConfigMap{})
			if err == nil {
				Fail("Delete of config map succeeded")
			}
			if !errors.IsForbidden(err) {
				Fail("Wrong error generated - should be forbidden")
			}
		})

		It("managed cluster can access secret but not modify it", func() {
			Eventually(func() bool {
				return findMultiClusterSecret(anotherTestNamespace, "mymcsecret")
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find mc secret")
			// try to update
			err := CreateOrUpdateResourceFromFile("testdata/multicluster/multicluster_secret_update.yaml", &v1.Secret{})
			if err == nil {
				Fail("Update to secret succeeded")
			}
			if !errors.IsForbidden(err) {
				Fail("Wrong error generated - should be forbidden")
			}
			// try to delete
			err = DeleteResourceFromFile("testdata/multicluster/multicluster_secret.yaml", &v1.Secret{})
			if err == nil {
				Fail("Delete of secret succeeded")
			}
			if !errors.IsForbidden(err) {
				Fail("Wrong error generated - should be forbidden")
			}
		})

		// VZ-2336: NOT be able to update or delete any VerrazzanoManagedCluster resources
		It("managed cluster cannot modify vmc on admin", func() {
			cluster := vmcv1alpha1.VerrazzanoManagedCluster{}
			err := getMultiClusterResource("verrazzano-mc", managedClusterName, &cluster)
			if err != nil {
				Fail("could not get vmc")
			}
			// try to update
			cluster.Spec.Description = "new Description"
			err = updateObject(&cluster)
			if err == nil {
				Fail("Update to vmc succeeded")
			}
			if !errors.IsForbidden(err) {
				Fail("Wrong error generated - should be forbidden")
			}
			// try to delete
			err = deleteObject(&cluster)
			if err == nil {
				Fail("Delete of vmc succeeded")
			}
			if !errors.IsForbidden(err) {
				Fail("Wrong error generated - should be forbidden")
			}
		})

		// VZ-2336: NOT be able to read other resources such as secrets, config maps or deployments in the admin cluster
		It("managed cluster cannot access resources in other namespaaces", func() {
			err := listResource("verrazzano-system", &v1.SecretList{})
			if err == nil {
				Fail("read access allowed")
			}
			if !errors.IsForbidden(err) {
				Fail("Wrong error generated - should be forbidden")
			}
			err = listResource("verrazzano-system", &v1.ConfigMapList{})
			if err == nil {
				Fail("read access allowed")
			}
			if !errors.IsForbidden(err) {
				Fail("Wrong error generated - should be forbidden")
			}
			err = listResource("verrazzano-mc", &v1.SecretList{})
			if err == nil {
				Fail("read access allowed")
			}
			if !errors.IsForbidden(err) {
				Fail("Wrong error generated - should be forbidden")
			}
			err = listResource("verrazzano-mc", &v1.ConfigMapList{})
			if err == nil {
				Fail("read access allowed")
			}
			if !errors.IsForbidden(err) {
				Fail("Wrong error generated - should be forbidden")
			}
			err = listResource(testNamespace, &v1.SecretList{})
			if err == nil {
				Fail("read access allowed")
			}
			if !errors.IsForbidden(err) {
				Fail("Wrong error generated - should be forbidden")
			}
			err = listResource(testNamespace, &v1.ConfigMapList{})
			if err == nil {
				Fail("read access allowed")
			}
			if !errors.IsForbidden(err) {
				Fail("Wrong error generated - should be forbidden")
			}
		})
	})
})

// CreateOrUpdateResourceFromFile creates or updates a resource using the provided yaml file
func CreateOrUpdateResourceFromFile(yamlFile string, object runtime.Object) error {
	found, err := pkg.FindTestDataFile(yamlFile)
	if err != nil {
		return fmt.Errorf("failed to find test data file: %w", err)
	}
	bytes, err := ioutil.ReadFile(found)
	if err != nil {
		return fmt.Errorf("failed to read test data file: %w", err)
	}
	err = yaml.Unmarshal(bytes, object)
	if err != nil {
		return err
	}
	return updateObject(object)
}

// updateObject updates a resource using the provided object
func updateObject(object runtime.Object) error {
	clustersClient := getClustersClient()

	err := clustersClient.Create(context.TODO(), object)
	if err != nil && errors.IsAlreadyExists(err) {
		err = clustersClient.Update(context.TODO(), object)
	}

	return err
}

// DeleteResourceFromFile deletes a resource using the provided yaml file and object reference
func DeleteResourceFromFile(yamlFile string, object runtime.Object) error {
	found, err := pkg.FindTestDataFile(yamlFile)
	if err != nil {
		return fmt.Errorf("failed to find test data file: %w", err)
	}
	bytes, err := ioutil.ReadFile(found)
	if err != nil {
		return fmt.Errorf("failed to read test data file: %w", err)
	}
	err = yaml.Unmarshal(bytes, object)
	if err != nil {
		return err
	}
	return deleteObject(object)
}

// deleteObject deletes the given object
func deleteObject(object runtime.Object) error {
	clustersClient := getClustersClient()
	return clustersClient.Delete(context.TODO(), object)
}

// deployTestResources deploys the test associated multi cluster resources
func deployTestResources() {
	pkg.Log(pkg.Info, "Deploying MC Resources")

	os.Setenv("TEST_KUBECONFIG", os.Getenv("ADMIN_KUBECONFIG"))

	// create the test project
	pkg.Log(pkg.Info, "Creating test project")
	err := CreateOrUpdateResourceFromFile("testdata/multicluster/verrazzanoproject-permissiontest.yaml", &clustersv1alpha1.VerrazzanoProject{})
	if err != nil {
		Fail(fmt.Sprintf("Failed to create test namespace: %v", err))
	}

	// Wait for the namespaces to be created
	pkg.Log(pkg.Info, "Wait for the project namespaces to be created")
	Eventually(func() (bool, error) {
		return pkg.DoesNamespaceExist(testNamespace)
	}, waitTimeout, pollingInterval).Should(BeTrue(), fmt.Sprintf("Expected to find namespace %s", testNamespace))
	Eventually(func() (bool, error) {
		return pkg.DoesNamespaceExist(anotherTestNamespace)
	}, waitTimeout, pollingInterval).Should(BeTrue(), fmt.Sprintf("Expected to find namespace %s", anotherTestNamespace))

	// create a config map
	pkg.Log(pkg.Info, "Creating MC config map")
	err = CreateOrUpdateResourceFromFile("testdata/multicluster/multicluster_configmap.yaml", &clustersv1alpha1.MultiClusterConfigMap{})
	if err != nil {
		Fail(fmt.Sprintf("Failed to create multi cluster config map: %v", err))
	}

	// create a secret
	pkg.Log(pkg.Info, "Creating MC secret")
	err = CreateOrUpdateResourceFromFile("testdata/multicluster/multicluster_secret.yaml", &clustersv1alpha1.MultiClusterSecret{})
	if err != nil {
		Fail(fmt.Sprintf("Failed to create multi cluster secret: %v", err))
	}
}

// undeployTestResources undeploys the test associated multi cluster resources
func undeployTestResources() {
	pkg.Log(pkg.Info, "Undeploying MC Resources")

	os.Setenv("TEST_KUBECONFIG", os.Getenv("ADMIN_KUBECONFIG"))

	// delete a config map
	pkg.Log(pkg.Info, "Deleting MC config map")
	err := DeleteResourceFromFile("testdata/multicluster/multicluster_configmap.yaml", &clustersv1alpha1.MultiClusterConfigMap{})
	if err != nil {
		Fail(fmt.Sprintf("Failed to create multi cluster config map: %v", err))
	}

	// delete a secret
	pkg.Log(pkg.Info, "Deleting MC secret")
	err = DeleteResourceFromFile("testdata/multicluster/multicluster_secret.yaml", &clustersv1alpha1.MultiClusterSecret{})
	if err != nil {
		Fail(fmt.Sprintf("Failed to create multi cluster secret: %v", err))
	}

	// delete the test project
	pkg.Log(pkg.Info, "Deleting test project")
	err = DeleteResourceFromFile("testdata/multicluster/verrazzanoproject-permissiontest.yaml", &clustersv1alpha1.VerrazzanoProject{})
	if err != nil {
		Fail(fmt.Sprintf("Failed to create test namespace: %v", err))
	}

}

// findSecret finds the secret based on name and namespace
func findSecret(namespace, name string) bool {
	clustersClient := getClustersClient()
	secretList := v1.SecretList{}
	err := clustersClient.List(context.TODO(), &secretList, &client.ListOptions{Namespace: namespace})
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to list secrets with error: %v", err))
		return false
	}
	for _, item := range secretList.Items {
		if item.Name == name && item.Namespace == namespace {
			return true
		}
	}
	return false
}

// findConfigMap finds the config map based on name and namespace
func findConfigMap(namespace, name string) bool {
	clustersClient := getClustersClient()

	configmapList := v1.ConfigMapList{}
	err := clustersClient.List(context.TODO(), &configmapList, &client.ListOptions{Namespace: namespace})
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to list config maps with error: %v", err))
		return false
	}
	for _, item := range configmapList.Items {
		if item.Name == name && item.Namespace == namespace {
			return true
		}
	}
	return false
}

// listResource returns a list of resources based on the object type and namespace
func listResource(namespace string, object runtime.Object) error {
	clustersClient := getClustersClient()
	return clustersClient.List(context.TODO(), object, &client.ListOptions{Namespace: namespace})
}

// getMultiClusterResource returns a multi cluster resource based the provided multi cluster object's type and namespace
func getMultiClusterResource(namespace, name string, object runtime.Object) error {
	clustersClient := getClustersClient()
	return clustersClient.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, object)
}

// findMultiClusterConfigMap returns true if the config map is found based on name and namespace, false otherwise
func findMultiClusterConfigMap(namespace, name string) bool {
	clustersClient := getClustersClient()

	configmapList := clustersv1alpha1.MultiClusterConfigMapList{}
	err := clustersClient.List(context.TODO(), &configmapList, &client.ListOptions{Namespace: namespace})
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to list multi cluster configmaps with error: %v", err))
		return false
	}
	for _, item := range configmapList.Items {
		if item.Name == name && item.Namespace == namespace {
			return true
		}
	}
	return false
}

// findMultiClusterSecret returns true if the secret is found based on name and namespace, false otherwise
func findMultiClusterSecret(namespace, name string) bool {
	clustersClient := getClustersClient()

	secretList := clustersv1alpha1.MultiClusterSecretList{}
	err := clustersClient.List(context.TODO(), &secretList, &client.ListOptions{Namespace: namespace})
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to list multi cluster secrets with error: %v", err))
		return false
	}
	for _, item := range secretList.Items {
		if item.Name == name && item.Namespace == namespace {
			return true
		}
	}
	return false
}

// getClustersClient returns a k8s client
func getClustersClient() client.Client {
	config, err := clientcmd.BuildConfigFromFlags("", os.Getenv("TEST_KUBECONFIG"))
	if err != nil {
		Fail(fmt.Sprintf("Failed to build config from %s with error: %v", os.Getenv("TEST_KUBECONFIG"), err))
	}

	scheme := runtime.NewScheme()
	_ = clustersv1alpha1.AddToScheme(scheme)
	_ = vmcv1alpha1.AddToScheme(scheme)
	_ = v1alpha1.AddToScheme(scheme)
	_ = v1.AddToScheme(scheme)

	clustersClient, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		Fail(fmt.Sprintf("Failed to get clusters client with error: %v", err))
	}
	return clustersClient
}

// isStatusAsExpected checks whehter the provided inputs align with the provided status
func isStatusAsExpected(status clustersv1alpha1.MultiClusterResourceStatus,
	expectedConditionType clustersv1alpha1.ConditionType,
	conditionMsgContains string,
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
	return matchingConditionCount >= 1 && matchingClusterStatusCount == 1
}
