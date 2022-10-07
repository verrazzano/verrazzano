// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package permissions_test

import (
	"context"
	goerrors "errors"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8s/resource"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/pkg/test/framework/metrics"
	"os"
	"strings"
	"time"

	oamv1alpha2 "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	vmcv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const waitTimeout = 10 * time.Minute
const pollingInterval = 10 * time.Second

const testNamespace = "multiclustertest"
const permissionTest1Namespace = "permissions-test1-ns"
const permissionTest2Namespace = "permissions-test2-ns"

var managedClusterName = os.Getenv("MANAGED_CLUSTER_NAME")
var adminKubeconfig = os.Getenv("ADMIN_KUBECONFIG")
var managedKubeconfig = os.Getenv("MANAGED_KUBECONFIG")

const vpTest1 = "permissions-test1"
const vpTest2 = "permissions-test2"

var t = framework.NewTestFramework("permissions_test")

var _ = t.BeforeSuite(func() {
	// Do set up for multi cluster tests
	deployTestResources()
})

var _ = t.AfterSuite(func() {
	// Do set up for multi cluster tests
	undeployTestResources()
})

var _ = t.AfterEach(func() {})

var _ = t.Describe("Multi Cluster Verify Kubeconfig Permissions", Label("f:multicluster.register"), func() {

	// vZ-2336: Be able to read MultiClusterXXX resources in the admin cluster
	//			Be able to update the status of MultiClusterXXX resources in the admin cluster
	t.Context("In Admin Cluster, verify mc resources and their status updates.", func() {
		t.BeforeEach(func() {
			_ = os.Setenv(k8sutil.EnvVarTestKubeConfig, os.Getenv("ADMIN_KUBECONFIG"))
		})

		t.It("Verify mc config map", func() {
			Eventually(func() (bool, error) {
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

		t.It("Verify mc secret", func() {
			Eventually(func() (bool, error) {
				return findMultiClusterSecret(permissionTest1Namespace, "mymcsecret")
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find mc secret")

			Eventually(func() bool {
				// Verify we have the expected status update
				secret := clustersv1alpha1.MultiClusterSecret{}
				err := getMultiClusterResource(permissionTest1Namespace, "mymcsecret", &secret)
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
		t.It("vmc status updates", func() {
			Eventually(func() bool {
				// Verify we have the expected status update
				vmc := vmcv1alpha1.VerrazzanoManagedCluster{}
				err := getMultiClusterResource("verrazzano-mc", managedClusterName, &vmc)
				return err == nil && vmc.Status.LastAgentConnectTime.After(time.Now().Add(-30*time.Minute))
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find VerrazzanoManagedCluster")
		})

	})

	t.Context("In the Managed Cluster, check for ", func() {
		t.BeforeEach(func() {
			_ = os.Setenv(k8sutil.EnvVarTestKubeConfig, os.Getenv("MANAGED_KUBECONFIG"))
		})

		t.It("the expected mc and underlying configmap", func() {
			pkg.Concurrently(
				func() {
					Eventually(func() (bool, error) {
						return findConfigMap(testNamespace, "mymcconfigmap")
					}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find configmap")
				},
				func() {
					Eventually(func() (bool, error) {
						return findMultiClusterConfigMap(testNamespace, "mymcconfigmap")
					}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find mc configmap")
				},
			)
		})

		t.It("the expected mc and underlying secret", func() {
			pkg.Concurrently(
				func() {
					Eventually(func() (bool, error) {
						return findSecret(permissionTest1Namespace, "mymcsecret")
					}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find secret")
				},
				func() {
					Eventually(func() (bool, error) {
						return findMultiClusterSecret(permissionTest1Namespace, "mymcsecret")
					}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find mc secret")
				},
			)
		})
	})

	// VZ-2336:  NOT be able to update or delete any MultiClusterXXX resources in the admin cluster
	t.Context("Managed Cluster", func() {
		t.BeforeEach(func() {
			_ = os.Setenv(k8sutil.EnvVarTestKubeConfig, os.Getenv("MANAGED_ACCESS_KUBECONFIG"))
		})

		t.It("can access MultiClusterConfigMap but not modify it on admin", func() {
			Eventually(func() (bool, error) {
				return findMultiClusterConfigMap(testNamespace, "mymcconfigmap")
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find mc configmap")
			// try to update
			Eventually(func() (bool, error) {
				err := resource.CreateOrUpdateResourceFromFile("testdata/multicluster/multicluster_configmap_update.yaml")
				// if we didn't get an error, fail immediately
				if err == nil {
					return false, goerrors.New("expected error from CreateOrUpdateResourceFromFile")
				}
				return errors.IsForbidden(err), nil
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to get a forbidden error")
			// try to delete
			Eventually(func() (bool, error) {
				err := resource.DeleteResourceFromFile("testdata/multicluster/multicluster_configmap.yaml")
				// if we didn't get an error, fail immediately
				if err == nil {
					return false, goerrors.New("expected error from DeleteResourceFromFile")
				}
				return errors.IsForbidden(err), nil
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to get a forbidden error")
		})

		t.It("can access MultiClusterSecret but not modify it on admin", func() {
			Eventually(func() (bool, error) {
				return findMultiClusterSecret(permissionTest1Namespace, "mymcsecret")
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find mc secret")
			// try to update
			Eventually(func() (bool, error) {
				err := resource.CreateOrUpdateResourceFromFile("testdata/multicluster/permissiontest1-multicluster-secret-update.yaml")
				// if we didn't get an error, fail immediately
				if err == nil {
					return false, goerrors.New("expected error from CreateOrUpdateResourceFromFile")
				}
				return errors.IsForbidden(err), nil
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to get a forbidden error")
			// try to delete
			Eventually(func() (bool, error) {
				err := resource.DeleteResourceFromFile("testdata/multicluster/multicluster_secret_permissiontest1.yaml")
				// if we didn't get an error, fail immediately
				if err == nil {
					return false, goerrors.New("expected error from DeleteResourceFromFile")
				}
				return errors.IsForbidden(err), nil
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to get a forbidden error")
		})

		t.It("can access OAM Component but not modify it on admin", func() {
			Eventually(func() (bool, error) {
				return findOAMComponent(permissionTest1Namespace, "oam-component")
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find OAM Component")
			// try to update
			Eventually(func() (bool, error) {
				err := resource.CreateOrUpdateResourceFromFile("testdata/multicluster/permissiontest1-oam-component.yaml")
				// if we didn't get an error, fail immediately
				if err == nil {
					return false, goerrors.New("expected error from CreateOrUpdateResourceFromFile")
				}
				return errors.IsForbidden(err), nil
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to get a forbidden error")
			// try to delete
			Eventually(func() (bool, error) {
				err := resource.DeleteResourceFromFile("testdata/multicluster/permissiontest1-oam-component.yaml")
				// if we didn't get an error, fail immediately
				if err == nil {
					return false, goerrors.New("expected error from DeleteResourceFromFile")
				}
				return errors.IsForbidden(err), nil
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to get a forbidden error")
		})

		t.It("can access secrets on admin from a namespace placed by a VerrazzanoProject", func() {
			Eventually(func() (bool, error) {
				return findSecret(permissionTest1Namespace, "mysecret")
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find Secret")
			// try to update
			Eventually(func() (bool, error) {
				err := resource.CreateOrUpdateResourceFromFile("testdata/multicluster/permissiontest1-secret.yaml")
				// if we didn't get an error, fail immediately
				if err == nil {
					return false, goerrors.New("expected error from CreateOrUpdateResourceFromFile")
				}
				return errors.IsForbidden(err), nil
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to get a forbidden error")
			// try to delete
			Eventually(func() (bool, error) {
				err := resource.DeleteResourceFromFile("testdata/multicluster/permissiontest1-secret.yaml")
				// if we didn't get an error, fail immediately
				if err == nil {
					return false, goerrors.New("expected error from DeleteResourceFromFile")
				}
				return errors.IsForbidden(err), nil
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to get a forbidden error")
		})

		t.It("cannot access secrets on admin for namespaces not placed by a VerrazzanoProject", func() {

			// Expect success while namespace is placed on the managed cluster
			Eventually(func() (bool, error) {
				return findSecret(permissionTest2Namespace, "mysecret")
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find Secret")
			// Change the placement to be on the admin cluster
			pkg.Log(pkg.Info, fmt.Sprintf("Change the placement of the namespace %s to be on the admin cluster", permissionTest2Namespace))
			Eventually(func() error {
				return resource.CreateOrUpdateResourceFromFileInCluster("testdata/multicluster/permissiontest2-verrazzanoproject-new-placement.yaml", adminKubeconfig)
			}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
			// Wait for the project resource to be deleted from the managed cluster
			pkg.Log(pkg.Info, "Wait for the VerrazzanoProject to be removed from the managed cluster")
			Eventually(func() (bool, error) {
				return pkg.DoesVerrazzanoProjectExistInCluster(vpTest2, managedKubeconfig)
			}, waitTimeout, pollingInterval).Should(BeFalse(), fmt.Sprintf("Expected VerrazzanoProject %s to be removed from managed cluster", vpTest2))
			Eventually(func() (bool, error) {
				return findSecret(permissionTest2Namespace, "mysecret")
			}, waitTimeout, pollingInterval).Should(BeFalse(), "Expected to get a forbidden error")
		})

		// VZ-2336: NOT be able to update or delete any VerrazzanoManagedCluster resources
		t.It("cannot modify vmc on admin", func() {
			cluster := vmcv1alpha1.VerrazzanoManagedCluster{}
			Eventually(func() error {
				return getMultiClusterResource("verrazzano-mc", managedClusterName, &cluster)
			}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
			// try to update
			Eventually(func() (bool, error) {
				cluster.Spec.Description = "new Description"
				err := updateObject(&cluster)
				// if we didn't get an error, fail immediately
				if err == nil {
					return false, goerrors.New("expected error from updateObject")
				}
				return errors.IsForbidden(err), nil
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to get a forbidden error")
			// try to delete
			Eventually(func() (bool, error) {
				err := deleteObject(&cluster)
				// if we didn't get an error, fail immediately
				if err == nil {
					return false, goerrors.New("expected error from deleteObject")
				}
				return errors.IsForbidden(err), nil
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to get a forbidden error")
		})

		// VZ-2336: NOT be able to read other resources such as config maps in the admin cluster
		t.It("cannot access resources in other namespaaces", func() {
			Eventually(func() (bool, error) {
				err := listResource("verrazzano-system", &v1.ConfigMapList{})
				// if we didn't get an error, return false to retry
				if err == nil {
					return false, goerrors.New("expected error from listResource")
				}
				return errors.IsForbidden(err), nil
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to get a forbidden error")
			Eventually(func() (bool, error) {
				err := listResource("verrazzano-mc", &v1.ConfigMapList{})
				// if we didn't get an error, return false to retry
				if err == nil {
					return false, goerrors.New("expected error from listResource")
				}
				return errors.IsForbidden(err), nil
			}, waitTimeout, pollingInterval).Should(BeTrue(), "expected to get a forbidden error")
			Eventually(func() (bool, error) {
				err := listResource(testNamespace, &v1.ConfigMapList{})
				// if we didn't get an error, fail immediately
				if err == nil {
					return false, goerrors.New("expected error from listResource")
				}
				return errors.IsForbidden(err), nil
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to get a forbidden error")
		})
	})
})

// updateObject updates a resource using the provided object
func updateObject(object client.Object) error {
	clustersClient, err := getClustersClient()
	if err != nil {
		return err
	}
	err = clustersClient.Create(context.TODO(), object)
	if err != nil && errors.IsAlreadyExists(err) {
		err = clustersClient.Update(context.TODO(), object)
	}

	return err
}

// deleteObject deletes the given object
func deleteObject(object client.Object) error {
	clustersClient, err := getClustersClient()
	if err != nil {
		return err
	}
	return clustersClient.Delete(context.TODO(), object)
}

// deployTestResources deploys the test associated multi cluster resources
func deployTestResources() {
	pkg.Log(pkg.Info, "Deploying MC Resources")

	_ = os.Setenv(k8sutil.EnvVarTestKubeConfig, os.Getenv("ADMIN_KUBECONFIG"))
	start := time.Now()
	// create the test projects
	pkg.Log(pkg.Info, "Creating test projects")
	Eventually(func() error {
		return resource.CreateOrUpdateResourceFromFile("testdata/multicluster/permissiontest1-verrazzanoproject.yaml")
	}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
	Eventually(func() error {
		return resource.CreateOrUpdateResourceFromFile("testdata/multicluster/permissiontest2-verrazzanoproject.yaml")
	}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())

	// Wait for the namespaces to be created
	pkg.Log(pkg.Info, "Wait for the project namespaces to be created")
	Eventually(func() (bool, error) {
		return pkg.DoesNamespaceExist(testNamespace)
	}, waitTimeout, pollingInterval).Should(BeTrue(), fmt.Sprintf("Expected to find namespace %s", testNamespace))
	Eventually(func() (bool, error) {
		return pkg.DoesNamespaceExist(permissionTest1Namespace)
	}, waitTimeout, pollingInterval).Should(BeTrue(), fmt.Sprintf("Expected to find namespace %s", permissionTest1Namespace))
	Eventually(func() (bool, error) {
		return pkg.DoesNamespaceExist(permissionTest2Namespace)
	}, waitTimeout, pollingInterval).Should(BeTrue(), fmt.Sprintf("Expected to find namespace %s", permissionTest2Namespace))

	// create a MC config map
	pkg.Log(pkg.Info, "Creating MC config map")
	Eventually(func() error {
		return resource.CreateOrUpdateResourceFromFile("testdata/multicluster/multicluster_configmap.yaml")
	}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())

	// create a MC secret
	pkg.Log(pkg.Info, "Creating MC secret")
	Eventually(func() error {
		return resource.CreateOrUpdateResourceFromFile("testdata/multicluster/multicluster_secret_permissiontest1.yaml")
	}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())

	// create a OAM Component
	pkg.Log(pkg.Info, "Creating OAM Component")
	Eventually(func() error {
		return resource.CreateOrUpdateResourceFromFile("testdata/multicluster/permissiontest1-oam-component.yaml")
	}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())

	// create a k8s secret
	pkg.Log(pkg.Info, "Creating k8s secrets")
	Eventually(func() error {
		return resource.CreateOrUpdateResourceFromFile("testdata/multicluster/permissiontest1-secret.yaml")
	}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
	Eventually(func() error {
		return resource.CreateOrUpdateResourceFromFile("testdata/multicluster/permissiontest2-secret.yaml")
	}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
	metrics.Emit(t.Metrics.With("undeployment_elapsed_time", time.Since(start).Milliseconds()))
}

// undeployTestResources undeploys the test associated multi cluster resources
func undeployTestResources() {
	pkg.Log(pkg.Info, "Undeploying MC Resources")

	_ = os.Setenv(k8sutil.EnvVarTestKubeConfig, os.Getenv("ADMIN_KUBECONFIG"))

	// delete a MC config map
	pkg.Log(pkg.Info, "Deleting MC config map")
	start := time.Now()
	Eventually(func() error {
		return resource.DeleteResourceFromFile("testdata/multicluster/multicluster_configmap.yaml")
	}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())

	// delete a MC secret
	pkg.Log(pkg.Info, "Deleting MC secret")
	Eventually(func() error {
		return resource.DeleteResourceFromFile("testdata/multicluster/multicluster_secret_permissiontest1.yaml")
	}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())

	// delete a OAM Component
	pkg.Log(pkg.Info, "Deleting OAM Component")
	Eventually(func() error {
		return resource.DeleteResourceFromFile("testdata/multicluster/permissiontest1-oam-component.yaml")
	}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())

	// delete k8s secrets
	pkg.Log(pkg.Info, "Deleting k8s secrets")
	Eventually(func() error {
		return resource.DeleteResourceFromFile("testdata/multicluster/permissiontest1-secret.yaml")
	}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
	Eventually(func() error {
		return resource.DeleteResourceFromFile("testdata/multicluster/permissiontest2-secret.yaml")
	}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())

	// delete the test projects
	pkg.Log(pkg.Info, "Deleting test projects")
	Eventually(func() error {
		return resource.DeleteResourceFromFile("testdata/multicluster/permissiontest1-verrazzanoproject.yaml")
	}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
	Eventually(func() error {
		return resource.DeleteResourceFromFile("testdata/multicluster/permissiontest2-verrazzanoproject.yaml")
	}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())

	// Wait for the project resources to be deleted from the managed cluster
	pkg.Log(pkg.Info, "Wait for the VerrazzanoProject resources to be removed from the managed cluster")
	Eventually(func() (bool, error) {
		return pkg.DoesVerrazzanoProjectExistInCluster(vpTest1, managedKubeconfig)
	}, waitTimeout, pollingInterval).Should(BeFalse(), fmt.Sprintf("Expected VerrazzanoProject %s to be removed from managed cluster", vpTest1))
	Eventually(func() (bool, error) {
		return pkg.DoesVerrazzanoProjectExistInCluster(vpTest2, managedKubeconfig)
	}, waitTimeout, pollingInterval).Should(BeFalse(), fmt.Sprintf("Expected VerrazzanoProject %s to be removed from managed cluster", vpTest2))

	// delete the test namespaces
	pkg.Log(pkg.Info, fmt.Sprintf("Deleting namespace %s on admin cluster", permissionTest1Namespace))
	Eventually(func() error {
		return pkg.DeleteNamespaceInCluster(permissionTest1Namespace, adminKubeconfig)
	}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
	pkg.Log(pkg.Info, fmt.Sprintf("Deleting namespace %s on managed cluster", permissionTest1Namespace))
	Eventually(func() error {
		return pkg.DeleteNamespaceInCluster(permissionTest1Namespace, managedKubeconfig)
	}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())

	pkg.Log(pkg.Info, fmt.Sprintf("Deleting namespace %s on admin cluster", permissionTest2Namespace))
	Eventually(func() error {
		return pkg.DeleteNamespaceInCluster(permissionTest2Namespace, adminKubeconfig)
	}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
	pkg.Log(pkg.Info, fmt.Sprintf("Deleting namespace %s on managed cluster", permissionTest2Namespace))
	Eventually(func() error {
		return pkg.DeleteNamespaceInCluster(permissionTest2Namespace, managedKubeconfig)
	}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())

	pkg.Log(pkg.Info, fmt.Sprintf("Deleting namespace %s on admin cluster", testNamespace))
	Eventually(func() error {
		return pkg.DeleteNamespaceInCluster(testNamespace, adminKubeconfig)
	}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
	pkg.Log(pkg.Info, fmt.Sprintf("Deleting namespace %s on managed cluster", testNamespace))
	Eventually(func() error {
		return pkg.DeleteNamespaceInCluster(testNamespace, managedKubeconfig)
	}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
	metrics.Emit(t.Metrics.With("undeployment_elapsed_time", time.Since(start).Milliseconds()))
}

// findSecret finds the secret based on name and namespace
func findSecret(namespace, name string) (bool, error) {
	clustersClient, err := getClustersClient()
	if err != nil {
		return false, err
	}
	secretList := v1.SecretList{}
	err = clustersClient.List(context.TODO(), &secretList, &client.ListOptions{Namespace: namespace})
	// Handle the case of forbidden as secret not found
	if err != nil && errors.IsForbidden(err) {
		return false, nil
	}
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to list secrets with error: %v", err))
		return false, err
	}
	for _, item := range secretList.Items {
		if item.Name == name && item.Namespace == namespace {
			return true, nil
		}
	}
	return false, nil
}

// findConfigMap finds the config map based on name and namespace
func findConfigMap(namespace, name string) (bool, error) {
	clustersClient, err := getClustersClient()
	if err != nil {
		return false, err
	}
	configmapList := v1.ConfigMapList{}
	err = clustersClient.List(context.TODO(), &configmapList, &client.ListOptions{Namespace: namespace})
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to list config maps with error: %v", err))
		return false, err
	}
	for _, item := range configmapList.Items {
		if item.Name == name && item.Namespace == namespace {
			return true, nil
		}
	}
	return false, nil
}

// listResource returns a list of resources based on the object type and namespace
func listResource(namespace string, objectList client.ObjectList) error {
	clustersClient, err := getClustersClient()
	if err != nil {
		return err
	}
	return clustersClient.List(context.TODO(), objectList, &client.ListOptions{Namespace: namespace})
}

// getMultiClusterResource returns a multi cluster resource based the provided multi cluster object's type and namespace
func getMultiClusterResource(namespace, name string, object client.Object) error {
	clustersClient, err := getClustersClient()
	if err != nil {
		return err
	}
	return clustersClient.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, object)
}

// findMultiClusterConfigMap returns true if the config map is found based on name and namespace, false otherwise
func findMultiClusterConfigMap(namespace, name string) (bool, error) {
	clustersClient, err := getClustersClient()
	if err != nil {
		return false, err
	}
	configmapList := clustersv1alpha1.MultiClusterConfigMapList{}
	err = clustersClient.List(context.TODO(), &configmapList, &client.ListOptions{Namespace: namespace})
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to list multi cluster configmaps with error: %v", err))
		return false, err
	}
	for _, item := range configmapList.Items {
		if item.Name == name && item.Namespace == namespace {
			return true, nil
		}
	}
	return false, nil
}

// findMultiClusterSecret returns true if the secret is found based on name and namespace, false otherwise
func findMultiClusterSecret(namespace, name string) (bool, error) {
	clustersClient, err := getClustersClient()
	if err != nil {
		return false, err
	}
	secretList := clustersv1alpha1.MultiClusterSecretList{}
	err = clustersClient.List(context.TODO(), &secretList, &client.ListOptions{Namespace: namespace})
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to list multi cluster secrets with error: %v", err))
		return false, err
	}
	for _, item := range secretList.Items {
		if item.Name == name && item.Namespace == namespace {
			return true, nil
		}
	}
	return false, nil
}

// findComponent returns true if the OAM component is found based on name and namespace, false otherwise
func findOAMComponent(namespace, name string) (bool, error) {
	clustersClient, err := getClustersClient()
	if err != nil {
		return false, err
	}
	component := &oamv1alpha2.Component{}
	err = clustersClient.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, component)
	if err != nil {
		return false, err
	}
	return true, nil
}

// getClustersClient returns a k8s client
func getClustersClient() (client.Client, error) {
	config, err := clientcmd.BuildConfigFromFlags("", os.Getenv(k8sutil.EnvVarTestKubeConfig))
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("failed to build config from %s with error: %v", os.Getenv(k8sutil.EnvVarTestKubeConfig), err))
		return nil, err
	}

	scheme := runtime.NewScheme()
	_ = clustersv1alpha1.AddToScheme(scheme)
	_ = vmcv1alpha1.AddToScheme(scheme)
	_ = v1alpha1.AddToScheme(scheme)
	_ = v1.AddToScheme(scheme)
	_ = oamv1alpha2.SchemeBuilder.AddToScheme(scheme)

	clustersClient, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to get clusters client with error: %v", err))
		return nil, err
	}
	return clustersClient, nil
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
