// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	"bytes"
	"crypto/sha256"
	"fmt"

	"github.com/verrazzano/verrazzano/pkg/k8sutil"

	"k8s.io/apimachinery/pkg/util/yaml"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/verrazzano/verrazzano/pkg/k8s/resource"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (s *Syncer) syncCattleClusterAgent() error {
	// Get the manifest secret from the admin cluster
	// Parse and separate out the cattle-agent and the cattle-credential
	// If first time, create a hash and apply the yaml for the resources
	// Else compare the hash and if different apply the yaml
	s.Log.Infof("Starting to sync the cattle-cluster-agent")

	manifestSecret := corev1.Secret{}
	err := s.AdminClient.Get(s.Context, client.ObjectKey{
		Namespace: constants.VerrazzanoMultiClusterNamespace,
		Name:      getManifestSecretName(s.ManagedClusterName),
	}, &manifestSecret)

	if err != nil {
		return fmt.Errorf("failed to fetch manifest secret for %s cluster: %v", s.ManagedClusterName, err)
	}
	s.Log.Infof(fmt.Sprintf("Found manifest secret for %s cluster: %s", s.ManagedClusterName, manifestSecret.Name))

	manifestData := manifestSecret.Data["yaml"]

	yamlSlices := bytes.Split(manifestData, []byte("---\n"))
	cattleAgentSlice := yamlSlices[10]

	cattleAgentHash := createHash(cattleAgentSlice)

	// We have a previous hash to compare to
	if len(s.CattleAgentHash) > 0 {
		// If they are the same, do nothing
		if s.CattleAgentHash == cattleAgentHash {
			s.Log.Infof("Cattle Hash hasn't changed. Nothing to update")
			return nil
		}
	}

	// No previous hash or change in hash
	// Apply the manifest secret and store the hash for next iterations
	s.Log.Infof("No previous cattle hash found or cattle hash has changed. Updating the cattle-cluster-agent")
	err = updateCattleAgent(yamlSlices, s.Log)
	if err != nil {
		return fmt.Errorf("failed to update the cattle-cluster-agent on %s cluster: %v", s.ManagedClusterName, err)
	}

	s.Log.Infof("Updating cattle hash")
	s.CattleAgentHash = cattleAgentHash

	return nil
}

func updateCattleAgent(data [][]byte, log *zap.SugaredLogger) error {

	config, err := k8sutil.BuildKubeConfig("")
	if err != nil {
		log.Errorf("failed to create incluster config: %v", err)
	}
	log.Infof("Built Incluster config: %s, now applying manifest", config.Host)

	// Data[8] contains the yaml for the cattle-credential used by the cattle-cluster-agent
	err = resource.CreateOrUpdateResourceFromBytesUsingConfig(data[8], config)
	if err != nil {
		log.Errorf("failed to apply resource: %v", err)
		return err
	}
	log.Infof("Successfully created new cattle-credential")

	// Data[10] contains the yaml for the cattle-cluster-agent
	//err := resource.CreateOrUpdateResourceFromBytes(data[10], log)
	patch, err := yaml.ToJSON(data[10])
	if err != nil {
		log.Errorf("failed to convert cattle-agent yaml to json: %v", err)
	}

	gvr := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
	err = resource.PatchResourceFromBytes(gvr, "cattle-system", "cattle-cluster-agent", patch, config)
	if err != nil {
		log.Errorf("failed to apply resource: %v", err)
		return err
	}

	return nil
}

func createHash(data []byte) string {
	split := bytes.SplitAfter(data, []byte("secretName: "))

	sha := sha256.New()
	sha.Write(split[0])

	return fmt.Sprintf("%v", sha.Sum(nil))
}

func getManifestSecretName(clusterName string) string {
	manifestSecretSuffix := "-manifest"
	return generateManagedResourceName(clusterName) + manifestSecretSuffix
}
