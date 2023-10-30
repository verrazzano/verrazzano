// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"time"

	"github.com/Jeffail/gabs/v2"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/k8s/resource"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const cattleAgent = "cattle-cluster-agent"
const clusterreposName = "rancher-charts"

var cattleClusterReposGVR = schema.GroupVersionResource{
	Group:    "catalog.cattle.io",
	Version:  "v1",
	Resource: "clusterrepos",
}

// syncCattleClusterAgent syncs the Rancher cattle-cluster-agent deployment
// and the cattle-credentials secret from the admin cluster to the managed cluster
// if they have changed in the registration-manifest
func (s *Syncer) syncCattleClusterAgent(currentCattleAgentHash string, kubeconfigPath string) (string, error) {
	manifestSecret := corev1.Secret{}
	err := s.AdminClient.Get(s.Context, client.ObjectKey{
		Namespace: constants.VerrazzanoMultiClusterNamespace,
		Name:      getManifestSecretName(s.ManagedClusterName),
	}, &manifestSecret)

	if err != nil {
		return currentCattleAgentHash, fmt.Errorf("failed to fetch manifest secret for %s cluster: %v", s.ManagedClusterName, err)
	}
	s.Log.Debugf(fmt.Sprintf("Found manifest secret for %s cluster: %s", s.ManagedClusterName, manifestSecret.Name))

	manifestData := manifestSecret.Data["yaml"]
	yamlSections := bytes.Split(manifestData, []byte("---\n"))

	cattleAgentResource, cattleCredentialResource := checkForCattleResources(yamlSections)
	if cattleAgentResource == nil || cattleCredentialResource == nil {
		s.Log.Debugf("The registration manifest doesn't contain the required resources. Will try to update the cattle-cluster-agent in the next iteration")
		return currentCattleAgentHash, nil
	}

	newCattleAgentHash := createHash(cattleAgentResource)

	// We have a previous hash to compare to
	if len(currentCattleAgentHash) > 0 {
		// If they are the same, do nothing
		if currentCattleAgentHash == newCattleAgentHash {
			return currentCattleAgentHash, nil
		}
	}

	// No previous hash or the hash has changed
	// Sync the cattle-agent and update the hash for next iterations
	s.Log.Info("No previous cattle hash found or cattle hash has changed. Updating the cattle-cluster-agent")
	err = updateCattleResources(cattleAgentResource, cattleCredentialResource, s.Log, kubeconfigPath)
	if err != nil {
		return currentCattleAgentHash, fmt.Errorf("failed to update the cattle-cluster-agent on %s cluster: %v", s.ManagedClusterName, err)
	}
	s.Log.Infof("Successfully synched cattle-cluster-agent")

	return newCattleAgentHash, nil
}

// checkForCattleResources iterates through the list of resources in the manifest yaml
// and returns the cattle-cluster-agent deployment and cattle-credentials secret if found
func checkForCattleResources(yamlData [][]byte) (*gabs.Container, *gabs.Container) {
	var cattleAgentResource, cattleCredentialResource *gabs.Container
	for _, eachResource := range yamlData {
		json, _ := yaml.ToJSON(eachResource)
		container, _ := gabs.ParseJSON(json)

		name := strings.Trim(container.Path("metadata.name").String(), "\"")
		namespace := strings.Trim(container.Path("metadata.namespace").String(), "\"")
		kind := strings.Trim(container.Path("kind").String(), "\"")

		if name == cattleAgent && namespace == constants.RancherSystemNamespace && kind == "Deployment" {
			cattleAgentResource = container
		} else if strings.Contains(name, "cattle-credentials-") && namespace == constants.RancherSystemNamespace && kind == "Secret" {
			cattleCredentialResource = container
		}
	}

	return cattleAgentResource, cattleCredentialResource
}

// updateCattleResources patches the cattle-cluster-agent and creates the cattle-credentials secret
func updateCattleResources(cattleAgentResource *gabs.Container, cattleCredentialResource *gabs.Container, log *zap.SugaredLogger, kubeconfigPath string) error {

	config, err := k8sutil.BuildKubeConfig(kubeconfigPath)
	if err != nil {
		log.Errorf("failed to create incluster config: %v", err)
		return err
	}
	log.Debugf("Built kubeconfig: %s, now updating resources", config.Host)

	// Scale the cattle-cluster-agent deployment to 0
	prevReplicas, err := scaleRancherAgentDeployment(config, log, 0)
	if err != nil {
		log.Errorf("failed to scale %s deployment: %v", cattleAgent, err)
		return err
	}
	if err = deleteClusterRepos(config); err != nil {
		log.Error(err)
		return err
	}

	patch := cattleAgentResource.Bytes()
	gvr := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
	err = resource.PatchResourceFromBytes(gvr, types.StrategicMergePatchType, constants.RancherSystemNamespace, cattleAgent, patch, config)
	if err != nil {
		log.Errorf("failed to patch cattle-cluster-agent: %v", err)
		return err
	}

	// Restore replicas count
	_, err = scaleRancherAgentDeployment(config, log, prevReplicas)
	if err != nil {
		log.Errorf("failed to scale %s deployment: %v", cattleAgent, err)
		return err
	}

	err = resource.CreateOrUpdateResourceFromBytesUsingConfig(cattleCredentialResource.Bytes(), config)
	if err != nil {
		log.Errorf("failed to create new cattle-credential: %v", err)
		return err
	}
	log.Debugf("Successfully patched cattle-cluster-agent and created a new cattle-credential secret")

	return nil
}

// createHash returns a hash of the cattle-cluster-agent deployment
func createHash(cattleAgent *gabs.Container) string {
	data := cattleAgent.Path("spec.template.spec.containers.0").Bytes()
	sha := sha256.New()
	sha.Write(data)

	return string(sha.Sum(nil))
}

// getManifestSecretName returns the manifest secret name for a managed cluster on the admin cluster
func getManifestSecretName(clusterName string) string {
	manifestSecretSuffix := "-manifest"
	return generateManagedResourceName(clusterName) + manifestSecretSuffix
}

// scaleRancherAgentDeployment scales the Rancher Agent deployment
func scaleRancherAgentDeployment(config *rest.Config, log *zap.SugaredLogger, replicas int32) (int32, error) {
	var prevReplicas int32 = 1

	c, err := kubernetes.NewForConfig(config)
	if err != nil {
		return 0, err
	}

	// Get the deployment object
	deployment := &appsv1.Deployment{}
	namespacedName := types.NamespacedName{Name: cattleAgent, Namespace: common.CattleSystem}
	deployment, err = c.AppsV1().Deployments(common.CattleSystem).Get(context.TODO(), cattleAgent, metav1.GetOptions{})
	if err != nil {
		return prevReplicas, client.IgnoreNotFound(err)
	}
	if deployment.Spec.Replicas != nil {
		prevReplicas = *deployment.Spec.Replicas
	}

	if deployment.Status.AvailableReplicas == 0 {
		// deployment is scaled down, we're done
		return prevReplicas, nil
	}

	if deployment.Spec.Replicas == nil || *deployment.Spec.Replicas > 0 {
		log.Infof("Scaling Rancher deployment %s to %d replicas %v", namespacedName, replicas)
		deployment.Spec.Replicas = &replicas
		deployment, err = c.AppsV1().Deployments(common.CattleSystem).Update(context.TODO(), deployment, metav1.UpdateOptions{})
		if err != nil {
			err2 := fmt.Errorf("Failed to scale Rancher deployment %v to % replicas: %v", namespacedName, replicas, err)
			log.Error(err2)
			return prevReplicas, err2
		}
	}

	// Wait for replica count to be reached
	for tries := 0; tries < retryCount; tries++ {
		deployment, err = c.AppsV1().Deployments(common.CattleSystem).Get(context.TODO(), cattleAgent, metav1.GetOptions{})
		if err != nil {
			return prevReplicas, err
		}
		if deployment.Status.AvailableReplicas == replicas {
			break
		}
		time.Sleep(retryDelay)
	}
	return prevReplicas, nil
}

// deleteClusterRepos - delete the clusterrepos object that contains the cached charts
func deleteClusterRepos(config *rest.Config) error {
	c, err := dynamic.NewForConfig(config)
	if err != nil {
		return err
	}
	err = c.Resource(cattleClusterReposGVR).Delete(context.TODO(), clusterreposName, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("Failed to delete clusterrrepos %s: %v", clusterreposName, err)
	}
	return nil
}
