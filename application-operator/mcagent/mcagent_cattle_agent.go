// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/k8s/resource"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"

	"github.com/Jeffail/gabs/v2"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const cattleAgent = "cattle-cluster-agent"

// syncCattleClusterAgent syncs the cattle-cluster-agent on the managed cluster
func (s *Syncer) syncCattleClusterAgent(kubeconfigPath string) error {
	manifestSecret := corev1.Secret{}
	err := s.AdminClient.Get(s.Context, client.ObjectKey{
		Namespace: constants.VerrazzanoMultiClusterNamespace,
		Name:      getManifestSecretName(s.ManagedClusterName),
	}, &manifestSecret)

	if err != nil {
		return fmt.Errorf("failed to fetch manifest secret for %s cluster: %v", s.ManagedClusterName, err)
	}
	s.Log.Debugf(fmt.Sprintf("Found manifest secret for %s cluster: %s", s.ManagedClusterName, manifestSecret.Name))

	manifestData := manifestSecret.Data["yaml"]
	yamlSlices := bytes.Split(manifestData, []byte("---\n"))

	cattleAgentResource, cattleCredentialResource := checkForCattleResources(yamlSlices)
	if cattleAgentResource == nil || cattleCredentialResource == nil {
		s.Log.Debugf("The registration manifest doesn't contain the required resources. Will try to update the cattle-cluster-agent in the next iteration")
		return nil
	}

	cattleAgentHash := createHash(cattleAgentResource)

	// We have a previous hash to compare to
	if len(s.CattleAgentHash) > 0 {
		// If they are the same, do nothing
		if s.CattleAgentHash == cattleAgentHash {
			return nil
		}
	}

	// No previous hash or the hash has changed
	// Sync the cattle-agent and update the hash for next iterations
	s.Log.Infof("No previous cattle hash found or cattle hash has changed. Updating the cattle-cluster-agent")
	err = updateCattleResources(cattleAgentResource, cattleCredentialResource, s.Log, kubeconfigPath)
	if err != nil {
		return fmt.Errorf("failed to update the cattle-cluster-agent on %s cluster: %v", s.ManagedClusterName, err)
	}

	s.CattleAgentHash = cattleAgentHash
	s.Log.Infof("Successfully synched cattle-cluster-agent")

	return nil
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

	patch := cattleAgentResource.Bytes()
	gvr := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
	err = resource.PatchResourceFromBytes(gvr, types.StrategicMergePatchType, constants.RancherSystemNamespace, cattleAgent, patch, config)
	if err != nil {
		log.Errorf("failed to patch cattle-cluster-agent: %v", err)
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

	return fmt.Sprintf("%v", sha.Sum(nil))
}

// getManifestSecretName returns the manifest secret name for a managed cluster on the admin cluster
func getManifestSecretName(clusterName string) string {
	manifestSecretSuffix := "-manifest"
	return generateManagedResourceName(clusterName) + manifestSecretSuffix
}
