// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package argocd_test

import (
	"context"
	b64 "encoding/base64"
	"fmt"
	"github.com/verrazzano/verrazzano/tests/e2e/multicluster/examples"
	dump "github.com/verrazzano/verrazzano/tests/e2e/pkg/test/clusterdump"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8s/resource"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework/metrics"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	waitTimeout          = 15 * time.Minute
	pollingInterval      = 10 * time.Second
	consistentlyDuration = 1 * time.Minute
	testNamespace        = "hello-helidon-argo"
	argoCDNamespace      = "argocd"
	expiresAtTimeStamp   = "verrazzano.io/expires-at-timestamp"
	createdTimeStamp     = "verrazzano.io/create-timestamp"
)

const (
	argoCDHelidonApplicationFile = "tests/e2e/multicluster/verify-argocd/testdata/hello-helidon-argocd-mc.yaml"
)

var expectedPodsHelloHelidon = []string{"helidon-config-deployment"}
var managedClusterName = os.Getenv("MANAGED_CLUSTER_NAME")
var adminKubeconfig = os.Getenv("ADMIN_KUBECONFIG")
var managedKubeconfig = os.Getenv("MANAGED_KUBECONFIG")
var argoCDUsernameForRancher = "vz-argoCD-reg"
var ttl = "240"
var createdTimeStampForNewTokenCreated string
var secretName = fmt.Sprintf("%s-argocd-cluster-secret", managedClusterName)

var t = framework.NewTestFramework("argocd_test")

var beforeSuite = t.BeforeSuiteFunc(func() {
	// Get the Hello Helidon Argo CD application yaml file
	// Deploy the Argo CD application in the admin cluster
	// This should internally deploy the helidon app to the managed cluster
	start := time.Now()
	Eventually(func() error {
		file, err := pkg.FindTestDataFile(argoCDHelidonApplicationFile)
		if err != nil {
			return err
		}
		return resource.CreateOrUpdateResourceFromFileInGeneratedNamespace(file, argoCDNamespace)
	}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred(), "Failed to create Argo CD Application Project file")
	metrics.Emit(t.Metrics.With("deployment_elapsed_time", time.Since(start).Milliseconds()))

	beforeSuitePassed = true
})

var _ = BeforeSuite(beforeSuite)

var failed = false
var beforeSuitePassed = false

var _ = t.AfterEach(func() {
	failed = failed || CurrentSpecReport().Failed()
})

var _ = t.Describe("Multi Cluster Argo CD Validation", Label("f:platform-lcm.install"), func() {
	t.Context("Admin Cluster", func() {
		t.BeforeEach(func() {
			os.Setenv(k8sutil.EnvVarTestKubeConfig, os.Getenv("ADMIN_KUBECONFIG"))
		})

		t.It("has the expected secrets", func() {
			secretName := fmt.Sprintf("%s-argocd-cluster-secret", managedClusterName)
			Eventually(func() error {
				result, err := findSecret(argoCDNamespace, secretName)
				if result != false {
					pkg.Log(pkg.Error, fmt.Sprintf("Failed to get secret %s in namespace %s with error: %v", secretName, argoCDNamespace, err))
					return err
				}
				return err
			}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred(), "Expected to find secret "+secretName)
		})

		t.It("secret has the data content with the same name as managed cluster", func() {
			secretName := fmt.Sprintf("%s-argocd-cluster-secret", managedClusterName)
			Eventually(func() error {
				result, err := findServerName(argoCDNamespace, secretName)
				if result != false {
					pkg.Log(pkg.Error, fmt.Sprintf("Failed to get servername in secret %s with error: %v", secretName, err))
					return err
				}
				return err
			}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred(), "Expected to find managed cluster name "+managedClusterName)
		})
	})

	t.Context("Managed Cluster", func() {
		t.BeforeEach(func() {
			os.Setenv(k8sutil.EnvVarTestKubeConfig, os.Getenv("MANAGED_KUBECONFIG"))
		})
		// GIVEN an admin cluster and at least one managed cluster
		// WHEN the  example application has been  placed in managed cluster
		// THEN expect that the app is deployed to the managed cluster
		t.It("Has application placed", func() {
			Eventually(func() bool {
				result, err := helloHelidonPodsRunning(managedKubeconfig, testNamespace)
				if err != nil {
					pkg.Log(pkg.Error, fmt.Sprintf("One or more pods are not running in the namespace: %v, error: %v", testNamespace, err))
					return false
				}
				return result
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})
	})
	t.Context("Token Update Tests", func() {
		t.BeforeEach(func() {
			os.Setenv(k8sutil.EnvVarTestKubeConfig, adminKubeconfig)
		})
		// Checks that the secret corresponding to the managed-secret in the cluster has both createdAt and ExpiredAt annotations
		t.It("The expected secret currently contains both the createdAt and ExpiredAt annotations", func() {
			secretName := fmt.Sprintf("%s-argocd-cluster-secret", managedClusterName)
			Eventually(func() (bool, error) {
				return verifyCreatedAtAndExpiresAtTimestampsExist(argoCDNamespace, secretName)
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find both Created and Expired At Annotations "+secretName)
		})
		// Tests that a new ArgoCD token is able to be created
		t.It("A new ArgoCD token is able to be created through the Rancher API", func() {
			Eventually(func() (string, error) {
				createdTimeStampForNewTokenCreatedValue, err := pkg.AddAccessTokenToRancherForLoggedInUser(t.Logs, adminKubeconfig, managedClusterName, argoCDUsernameForRancher, ttl)
				createdTimeStampForNewTokenCreated = createdTimeStampForNewTokenCreatedValue
				if err != nil {
					pkg.Log(pkg.Error, "Error creating New Token")
				}
				return createdTimeStampForNewTokenCreated, err

			}, waitTimeout, pollingInterval).ShouldNot(BeEmpty(), "Expected to Be Able To Create a Token through the API")
		})
		// Pre-set up occurs to trigger the update in the secret
		// Tests that the update completes and that the secret now has the timestamps of the old token
		t.It("The secret has gone through its first update and eventually has an expires at annotation and the created Timestamp of the most recently created token", func() {
			Eventually(func() error {
				err := testIfUpdateSuccessfullyTriggeredForArgoCD(secretName)
				if err != nil {
					pkg.Log(pkg.Error, "Failed to trigger the first update")
				}
				return err
			}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred(), "Expected to be able to trigger the first update without any errors")
			Eventually(func() (string, error) {
				updatedSecret, err := pkg.GetSecret(argoCDNamespace, secretName)
				if err != nil {
					pkg.Log(pkg.Error, "Failed to query the edited secret")
					return "", err
				}
				return updatedSecret.Annotations[createdTimeStamp], nil
			}, waitTimeout, pollingInterval).Should(Equal(createdTimeStampForNewTokenCreated), "Expected to have the secret reflect the created timestamp of the new token that was created")
		})
		// Tests that the name of the tokens that have the same cluster ID as the cluster can be fetched from Rancher and that they can be deleted
		// This checks that if no valid tokens are present when an upgrade happens, a new token is created
		t.It("All of the tokens that belong to this user should be retrieved in the cluster without any errors", func() {
			Eventually(func() error {
				err := pkg.GetAndDeleteTokenNamesForLoggedInUserBasedOnClusterID(t.Logs, adminKubeconfig, managedClusterName, argoCDUsernameForRancher)
				if err != nil {
					pkg.Log(pkg.Error, "Error querying the list of ArgoCD API Access tokens for that existing user")
				}
				return err

			}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred(), "Expected for all the tokens with the same clusterID to be deleted ")
		})
		// This triggers another update in the secret
		t.It("The second update of the secret is triggered and executes without errors after all of the tokens with the corresponding cluster ID have been deleted", func() {
			Eventually(func() error {
				err := testIfUpdateSuccessfullyTriggeredForArgoCD(secretName)
				if err != nil {
					pkg.Log(pkg.Error, "Failed to trigger the second update")
				}
				return err
			}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred(), "Expected to be able to trigger the second update without any errors")
		})
		// This checks that the secret's timestamps were successfully updated with the new token
		Eventually(func() (bool, error) {
			updatedSecret, err := pkg.GetSecret(argoCDNamespace, secretName)
			if err != nil {
				pkg.Log(pkg.Error, "Failed to query the edited secret")
				return false, err
			}
			_, ok := updatedSecret.Annotations[expiresAtTimeStamp]
			if !ok {
				pkg.Log(pkg.Error, "Failed to add an expires-at-timestamp to the secret based on a new token that is created")
				return false, fmt.Errorf("The secret was not successfully edited, as it does not have an expired timestamp")
			}
			createdTimestampCurrentlyOnUpdatedSecret, ok := updatedSecret.Annotations[createdTimeStamp]
			if !ok {
				pkg.Log(pkg.Error, "Failed to successfully update the secret with a created time-stamp at all ")
				return false, fmt.Errorf("The created-at timestamp of the secret was not created at all, based on the new token that was created")
			}
			timeOfPriorTimeStamp, _ := time.Parse(time.RFC3339, createdTimeStampForNewTokenCreated)
			timeOfCurrentTimeStamp, _ := time.Parse(time.RFC3339, createdTimestampCurrentlyOnUpdatedSecret)
			return timeOfCurrentTimeStamp.After(timeOfPriorTimeStamp), nil
		}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to have the secret reflect the created timestamp of the new token that was created")
	})
	//This eventually block deletes the cluster
	t.Context("Delete resources", func() {
		t.BeforeEach(func() {
			os.Setenv(k8sutil.EnvVarTestKubeConfig, os.Getenv("ADMIN_KUBECONFIG"))
		})
		t.It("Delete resources on admin cluster", func() {
			Eventually(func() error {
				return deleteArgoCDApplication(adminKubeconfig)
			}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
		})

		t.It("Verify automatic deletion on managed cluster", func() {
			Eventually(func() bool {
				return examples.VerifyAppDeleted(managedKubeconfig, testNamespace)
			}, consistentlyDuration, pollingInterval).Should(BeTrue())
		})

	})

})

var afterSuite = t.AfterSuiteFunc(func() {
	if failed || !beforeSuitePassed {
		dump.ExecuteBugReport(testNamespace)
	}
})

var _ = AfterSuite(afterSuite)

func deleteArgoCDApplication(kubeconfigPath string) error {
	start := time.Now()
	file, err := pkg.FindTestDataFile(argoCDHelidonApplicationFile)
	if err != nil {
		return err
	}
	if err := resource.DeleteResourceFromFileInCluster(file, kubeconfigPath); err != nil {
		return fmt.Errorf("failed to delete Argo CD hello-helidon application: %v", err)
	}

	metrics.Emit(t.Metrics.With("undeployment_elapsed_time", time.Since(start).Milliseconds()))
	return nil
}

func findSecret(namespace, name string) (bool, error) {
	s, err := pkg.GetSecret(namespace, name)
	if err != nil {
		return false, err
	}
	return s != nil, nil
}

func helloHelidonPodsRunning(kubeconfigPath string, namespace string) (bool, error) {
	result, err := pkg.PodsRunningInCluster(namespace, expectedPodsHelloHelidon, kubeconfigPath)
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("One or more pods are not running in the namespace: %v, error: %v", namespace, err))
		return false, err
	}
	return result, nil
}

func findServerName(namespace, name string) (bool, error) {
	s, err := pkg.GetSecret(namespace, name)
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to get secret %s in namespace %s with error: %v", name, namespace, err))
		return false, err
	}
	servername := string(s.Data["name"])
	decodeServerName, err := b64.StdEncoding.DecodeString(servername)
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to decode secret data %s in secret %s with error: %v", servername, name, err))
		return false, err
	}
	return string(decodeServerName) != managedClusterName, nil
}

// This function checks that a create-timestamp value and an expires-at-timestamp value currently exist on the secret's annotations
func verifyCreatedAtAndExpiresAtTimestampsExist(namespace, name string) (bool, error) {
	s, err := pkg.GetSecret(namespace, name)
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to get secret %s in namespace %s with error: %v", name, namespace, err))
		return false, err
	}
	annotationMap := s.GetAnnotations()
	createdValue, ok := annotationMap[createdTimeStamp]
	if !ok || createdValue == "" {
		return false, fmt.Errorf("Created Annotation Value Not Found")
	}
	expiresAtValue, ok := annotationMap[expiresAtTimeStamp]
	if !ok || expiresAtValue == "" {
		return false, fmt.Errorf("Expiration Value is Not Found")
	}
	return true, nil
}

// This function tests if the ArgoCD secret was successfully updated
func testIfUpdateSuccessfullyTriggeredForArgoCD(secretName string) error {
	secretToTriggerUpdate, err := pkg.GetSecret(argoCDNamespace, secretName)
	if err != nil {
		pkg.Log(pkg.Error, "Unable to find the secret in the cluster after token creation has occurred")
		return err
	}
	delete(secretToTriggerUpdate.Annotations, expiresAtTimeStamp)
	clientSetForCluster, err := pkg.GetKubernetesClientsetForCluster(adminKubeconfig)
	if err != nil {
		pkg.Log(pkg.Error, "Unable to get admin client set")
	}
	editedSecret, err := clientSetForCluster.CoreV1().Secrets(argoCDNamespace).Update(context.TODO(), secretToTriggerUpdate, metav1.UpdateOptions{})
	if err != nil {
		pkg.Log(pkg.Error, "Error editing the secret through the client")
		return err
	}
	_, ok := editedSecret.Annotations[expiresAtTimeStamp]
	if ok {
		pkg.Log(pkg.Error, "The client was successful, but the secret was not successfully edited")
		return fmt.Errorf("The client was successful, but the secret was not successfully edited")
	}
	return err
}
