// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	"bytes"
	"crypto/sha256"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"

	"github.com/verrazzano/verrazzano/pkg/k8s/resource"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

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

	if len(yamlSlices) < 12 {
		s.Log.Debugf("The registration manifest doesn't have all the resources. Will try to update the cattle-cluster-agent in the next iteration")
		return nil
	}

	cattleAgentSlice := yamlSlices[10]
	cattleAgentHash := createHash(cattleAgentSlice)

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
	err = updateCattleAgent(yamlSlices, s.Log, kubeconfigPath)
	if err != nil {
		return fmt.Errorf("failed to update the cattle-cluster-agent on %s cluster: %v", s.ManagedClusterName, err)
	}

	s.CattleAgentHash = cattleAgentHash
	s.Log.Infof("Successfully synched cattle-cluster-agent")

	return nil
}

func updateCattleAgent(data [][]byte, log *zap.SugaredLogger, kubeconfigPath string) error {

	config, err := k8sutil.BuildKubeConfig(kubeconfigPath)
	if err != nil {
		log.Errorf("failed to create incluster config: %v", err)
		return err
	}
	log.Debugf("Built kubeconfig: %s, now applying manifest", config.Host)

	// Data[10] contains the yaml for the cattle-cluster-agent
	//err := resource.CreateOrUpdateResourceFromBytes(data[10], log)
	patch, err := yaml.ToJSON(data[10])
	if err != nil {
		log.Errorf("failed to convert cattle-agent yaml to json: %v", err)
		return err
	}

	gvr := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
	err = resource.PatchResourceFromBytes(gvr, types.StrategicMergePatchType, "cattle-system", "cattle-cluster-agent", patch, config)
	if err != nil {
		log.Errorf("failed to patch cattle-cluster-agent: %v", err)
		return err
	}

	// Data[8] contains the yaml for the cattle-credential used by the cattle-cluster-agent
	err = resource.CreateOrUpdateResourceFromBytesUsingConfig(data[8], config)
	if err != nil {
		log.Errorf("failed to create new cattle-credential: %v", err)
		return err
	}
	log.Debugf("Successfully patched cattle-cluster-agent and created a new cattle-credential secret")

	return nil
}

func createHash(data []byte) string {
	// The cattle-credential-xxxxx contains a short-lived token
	// It changes every few pulls of the rancher manifest even if there is no change in the cattle-agent
	// So remove the secret name before creating the hash
	split := bytes.SplitAfter(data, []byte("secretName: "))

	sha := sha256.New()
	sha.Write(split[0])

	return fmt.Sprintf("%v", sha.Sum(nil))
}

func getManifestSecretName(clusterName string) string {
	manifestSecretSuffix := "-manifest"
	return generateManagedResourceName(clusterName) + manifestSecretSuffix
}
