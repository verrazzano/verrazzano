// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package secrets

import (
	"fmt"
	"k8s.io/apimachinery/pkg/api/errors"
	"os"

	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

const (
	//PipelineImagePullSecName Image pull secr env var name for pipeline
	PipelineImagePullSecName = "IMAGE_PULL_SECRET"
	//PipelineRegistryKey Docker registry env var name for pipeline
	PipelineRegistryKey = "DOCKER_REGISTRY"
	//PipelineDockerUserKey Docker user env var name for pipeline
	PipelineDockerUserKey = "DOCKER_CREDS_USR"
	//PipelineDockerPswKey Docker credential env var name for pipeline
	PipelineDockerPswKey = "DOCKER_CREDS_PSW"

	//DefaultImagePullSecName Default image pull sec name
	DefaultImagePullSecName = "verrazzano-container-registry"
)

// CreateOrUpdatePipelineImagePullSecret Creates an image pull secret for a Pipeline test run if the variable
// "IMAGE_PULL_SECRET" is defined.
//
// If IMAGE_PULL_SECRET is defined, the secret is created from the following env vars:
// - DOCKER_REGISTRY (defaults to "ghcr.io")
// - DOCKER_CREDS_USR
// - DOCKER_CREDS_PSW
func CreateOrUpdatePipelineImagePullSecret(log vzlog.VerrazzanoLogger, namespace string, kubeconfigPath string) error {
	pullSecretName := os.Getenv(PipelineImagePullSecName)
	if pullSecretName == "" {
		log.Infof("Image pull secret not defined, skipping secret creation")
		return nil
	}
	registryName := os.Getenv(PipelineRegistryKey)
	if registryName == "" {
		registryName = "ghcr.io"
		log.Infof("Image registry not defined, using default %s", registryName)
	}
	registryUser := os.Getenv(PipelineDockerUserKey)
	if registryName == "" {
		return fmt.Errorf("registry user %s not defined", PipelineDockerUserKey)
	}
	registryPwd := os.Getenv(PipelineDockerPswKey)
	if registryName == "" {
		return fmt.Errorf("registry cred %s not defined", PipelineDockerPswKey)
	}
	_, err := pkg.CreateDockerSecretInCluster(namespace, pullSecretName, registryName, registryUser, registryPwd, kubeconfigPath)
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	return nil
}
