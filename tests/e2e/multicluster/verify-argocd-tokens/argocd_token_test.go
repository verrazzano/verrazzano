// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package argocd_token_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	waitTimeout                     = 15 * time.Minute
	pollingInterval                 = 10 * time.Second
	argoCDNamespace                 = "argocd"
	argocdClusterTokenTTLEnvVarName = "ARGOCD_CLUSTER_TOKEN_TTL" //nolint:gosec
)

var t = framework.NewTestFramework("argocd_token_sync_test")
var managedClusterName = os.Getenv("MANAGED_CLUSTER_NAME")
var adminKubeconfig = os.Getenv("ADMIN_KUBECONFIG")

var argoCDUsernameForRancher = "vz-argoCD-reg"
var ttl = os.Getenv(argocdClusterTokenTTLEnvVarName)
var createdTimeStampForNewTokenCreated = ""

var _ = t.Describe("ArgoCD Token Sync Testing", Label("f:platform-lcm.install"), func() {
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

func findSecret(namespace, name string) (bool, error) {
	s, err := pkg.GetSecret(namespace, name)
	if err != nil {
		return false, err
	}
	return s != nil, nil
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
	decodedArgoCDPasswordForSecret, err := base64.StdEncoding.DecodeString(string(encodedArgoCDPasswordForSecret))
	if err != nil {
		pkg.Log(pkg.Error, "Error occurred decoding the password string")
		return "", err
	}
	return string(decodedArgoCDPasswordForSecret), nil
}
