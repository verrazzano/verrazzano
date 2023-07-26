// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package argocd_test

import (
	"context"
	b64 "encoding/base64"
	"fmt"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/k8s/resource"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/multicluster/examples"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	dump "github.com/verrazzano/verrazzano/tests/e2e/pkg/test/clusterdump"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework/metrics"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	waitTimeout                     = 15 * time.Minute
	pollingInterval                 = 10 * time.Second
	consistentlyDuration            = 1 * time.Minute
	testNamespace                   = "hello-helidon-argo"
	argoCDNamespace                 = "argocd"
	argocdClusterTokenTTLEnvVarName = "ARGOCD_CLUSTER_TOKEN_TTL" //nolint:gosec
)

const (
	argoCDHelidonApplicationFile = "tests/e2e/multicluster/verify-argocd/testdata/hello-helidon-argocd-mc.yaml"
)

var expectedPodsHelloHelidon = []string{"helidon-config-deployment"}
var managedClusterName = os.Getenv("MANAGED_CLUSTER_NAME")
var adminKubeconfig = os.Getenv("ADMIN_KUBECONFIG")
var managedKubeconfig = os.Getenv("MANAGED_KUBECONFIG")
var argoCDUsernameForRancher = "vz-argoCD-reg"
var ttl = os.Getenv(argocdClusterTokenTTLEnvVarName)
var createdTimeStampForNewTokenCreated = ""

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
		t.It("The expected secret currently contains both the createdAt and ExpiredAt annotations", func() {
			secretName := fmt.Sprintf("%s-argocd-cluster-secret", managedClusterName)
			Eventually(func() error {
				result, err := verifyCreatedAtAndExpiresAtTimestampsExist(argoCDNamespace, secretName)
				if result != true {
					pkg.Log(pkg.Error, fmt.Sprintf("Failed to get an ExpiredAt or Created Annotation in secret %s in namespace %s with error: %v", secretName, argoCDNamespace, err))
					return err
				}
				return err
			}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred(), "Expected to find both Created and Expired At Annotations "+secretName)
		})
		t.It("A new ArgoCD token is able to be created through the Rancher API", func() {
			Eventually(func() error {
				argoCDPasswordForRancher, err := retrieveArgoCDPassword("verrazzano-mc", "verrazzano-argocd-secret")
				if err != nil {
					return err
				}
				rancherConfigForArgoCD, err := pkg.CreateNewRancherConfigForUser(t.Logs, adminKubeconfig, argoCDUsernameForRancher, argoCDPasswordForRancher)
				if err != nil {
					pkg.Log(pkg.Error, "Error occurred when created a Rancher Config for ArgoCD")
					return err
				}
				client, err := pkg.GetClusterOperatorClientset(adminKubeconfig)
				if err != nil {
					pkg.Log(pkg.Error, "Error creating the client set used by the cluster operator")
					return err
				}
				managedCluster, err := client.ClustersV1alpha1().VerrazzanoManagedClusters(constants.VerrazzanoMultiClusterNamespace).Get(context.TODO(), managedClusterName, metav1.GetOptions{})
				if err != nil {
					pkg.Log(pkg.Error, "Error getting the current managed cluster resource")
					return err
				}
				clusterID := managedCluster.Status.RancherRegistration.ClusterID
				if clusterID == "" {
					pkg.Log(pkg.Error, "The managed cluster does not have a clusterID value")
					err := fmt.Errorf("ClusterID value is not yet populated for the managed cluster")
					return err
				}
				httpClientForRancher, err := pkg.GetVerrazzanoHTTPClient(adminKubeconfig)
				if err != nil {
					pkg.Log(pkg.Error, "Error getting the Verrazzano http client")
					return err
				}
				createdTimeStampForNewTokenCreated, err = pkg.AddAccessTokenToRancherForLoggedInUser(httpClientForRancher, adminKubeconfig, clusterID, ttl, rancherConfigForArgoCD.APIAccessToken, *t.Logs)
				if err != nil {
					pkg.Log(pkg.Error, "Error creating New Token")
					return err
				}
				return err

			}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred(), "Expected to Be Able To Create a Token through the API")
		})
		t.It("The secret can be successfully altered to trigger an update locally and sent through the client", func() {
			secretName := fmt.Sprintf("%s-argocd-cluster-secret", managedClusterName)
			Eventually(func() error {
				secretToTriggerUpdate, err := pkg.GetSecret(argoCDNamespace, secretName)
				if err != nil {
					pkg.Log(pkg.Error, "Unable to find the secret in the cluster after token creation has occurred")
					return err
				}
				delete(secretToTriggerUpdate.Annotations, "verrazzano.io/expires-at-timestamp")
				clientSetForCluster, err := pkg.GetKubernetesClientsetForCluster(adminKubeconfig)
				if err != nil {
					pkg.Log(pkg.Error, "Unable to get admin client set")
				}
				editedSecret, err := clientSetForCluster.CoreV1().Secrets(argoCDNamespace).Update(context.TODO(), secretToTriggerUpdate, metav1.UpdateOptions{})
				if err != nil {
					pkg.Log(pkg.Error, "Error editing the secret through the client")
					return err
				}
				_, ok := editedSecret.Annotations["verrazzano.io/expires-at-timestamp"]
				if ok {
					pkg.Log(pkg.Error, "The client was successful, but the secret was not successful edited")
					return fmt.Errorf("The client was successful, but the secret was not successful edited")
				}
				return err
			}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred(), "Expected to be able to edit the secret locally and send the update request through the cluster")
		})
		t.It("The secret has gone through an update and eventually has an expires at annotation and the created Timestamp of the most recently created token", func() {
			secretName := fmt.Sprintf("%s-argocd-cluster-secret", managedClusterName)
			Eventually(func() error {
				updatedSecret, err := pkg.GetSecret(argoCDNamespace, secretName)
				if err != nil {
					pkg.Log(pkg.Error, "Failed to query the edited secret")
					return err
				}
				_, ok := updatedSecret.Annotations["verrazzano.io/expires-at-timestamp"]
				if !ok {
					pkg.Log(pkg.Error, "Failed to add an expires-at-timestamp to the secret based on a new token that is created")
					return fmt.Errorf("The secret was not successfully edited, as it does not have an expired timestamp")
				}
				createdTimestampCurrentlyOnUpdatedSecret, ok := updatedSecret.Annotations["verrazzano.io/create-timestamp"]
				if createdTimestampCurrentlyOnUpdatedSecret != createdTimeStampForNewTokenCreated || !ok {
					pkg.Log(pkg.Error, "Failed to successfully update the secret with the created timestamp of the most recent token created")
					return fmt.Errorf("The created-at timestamp of the secret was not correctly updated with the created timestamp of the most recently created token")
				}
				return err
			}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred(), "Expected to have the secret reflect the created timestamp of the new token that was created")
		})

	})

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
func verifyCreatedAtAndExpiresAtTimestampsExist(namespace, name string) (bool, error) {
	s, err := pkg.GetSecret(namespace, name)
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to get secret %s in namespace %s with error: %v", name, namespace, err))
		return false, err
	}
	annotationMap := s.GetAnnotations()
	createdValue, ok := annotationMap["verrazzano.io/create-timestamp"]
	if !ok || createdValue == "" {
		return false, fmt.Errorf("Created Annotation Value Not Found")
	}
	expiresAtValue, ok := annotationMap["verrazzano.io/expires-at-timestamp"]
	if !ok || expiresAtValue == "" {
		return false, fmt.Errorf("Expiration Value is Not Found")
	}
	return true, nil
}

func retrieveArgoCDPassword(namespace, name string) (string, error) {
	s, err := pkg.GetSecret(namespace, name)
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to get secret %s in namespace %s with error: %v", name, namespace, err))
		return "", err
	}
	encodedArgoCDPasswordForSecret, ok := s.Data["password"]
	if !ok {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to find password value in ArgoCD secret %s in namespace %s", name, namespace))
		return "", fmt.Errorf("Failed to find password value in ArgoCD secret %s in namespace %s", name, namespace)
	}
	decodedArgoCDPasswordForSecret, err := b64.StdEncoding.DecodeString(string(encodedArgoCDPasswordForSecret))
	if err != nil {
		pkg.Log(pkg.Error, "Error occurred decoding the password string")
		return "", err
	}
	return string(decodedArgoCDPasswordForSecret), nil
}
