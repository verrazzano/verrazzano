// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	"bytes"
	"crypto/sha256"
	"fmt"

	"github.com/verrazzano/verrazzano/pkg/k8sutil"

	"github.com/verrazzano/verrazzano/pkg/k8s/resource"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (s *Syncer) syncCattleClusterAgent() error {
	// Get the manifest secret from the admin cluster
	// Decode and Parse it however
	// If first time, store the values that need to be compared and apply the yaml
	// Else compare the values and if different apply the yaml
	s.Log.Infof("Starting to sync the cattle-cluster-agent")

	manifestSecret := corev1.Secret{}
	err := s.AdminClient.Get(s.Context, client.ObjectKey{
		Namespace: constants.VerrazzanoMultiClusterNamespace,
		Name:      getManifestSecretName(s.ManagedClusterName),
	}, &manifestSecret)

	if err != nil {
		return fmt.Errorf("Failed to fetch manifest secret for %s cluster: %v", s.ManagedClusterName, err)
	}
	s.Log.Infof(fmt.Sprintf("Found manifest secret for %s cluster: %s", s.ManagedClusterName, manifestSecret.Name))

	manifestData := manifestSecret.Data["yaml"]

	yamlSlices := bytes.Split(manifestData, []byte("---\n"))
	cattleAgentSlice := yamlSlices[10]

	cattleAgentHash := createHash(cattleAgentSlice)

	// We have a previous manifest secret to compare to
	if len(s.CattleAgentHash) > 0 {
		// Compare the current hash to previous one
		// If different apply the manifest
		// else do nothing
		if s.CattleAgentHash == cattleAgentHash {
			return nil
		}
	}

	// No previous hash or change in hash
	// Apply the manifest secret and store the hash for next iterations
	s.Log.Infof("No previous cattle has found. Applying manifest and updating hash")
	err = s.applyManifest(yamlSlices, s.Log)
	if err != nil {
		return fmt.Errorf("Failed to apply the updated manifest on %s cluster: %v", s.ManagedClusterName, err)
	}

	s.CattleAgentHash = cattleAgentHash

	return nil
}

func (s *Syncer) applyManifest(data [][]byte, log *zap.SugaredLogger) error {

	config, err := k8sutil.BuildKubeConfig("")
	if err != nil {
		log.Errorf("failed to create incluster config: %v", err)
	}
	s.Log.Infof("Built Incluster config: %s, now applying manifest", config.Host)

	for i, each := range data {
		err = resource.CreateOrUpdateResourceFromBytesUsingConfig(each, config)
		if err != nil {
			log.Errorf("failed to apply resource %d: %v", i, err)
			return err
		}
		log.Infof("Successfully applied resource %d", i)
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
