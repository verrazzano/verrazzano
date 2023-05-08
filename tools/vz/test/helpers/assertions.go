// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// AssertPrivateRegistryEnvVars asserts that the deployment container has the expected private registry environment variables
func AssertPrivateRegistryEnvVars(t *testing.T, client client.Client, deployment *appsv1.Deployment, expectedImageRegistry, expectedImagePrefix string) {
	envVar := corev1.EnvVar{Name: "REGISTRY", Value: expectedImageRegistry}
	assert.Contains(t, deployment.Spec.Template.Spec.Containers[0].Env, envVar)
	envVar = corev1.EnvVar{Name: "IMAGE_REPO", Value: expectedImagePrefix}
	assert.Contains(t, deployment.Spec.Template.Spec.Containers[0].Env, envVar)
}

// AssertPrivateRegistryImage asserts that the deployment container and init container VPO images have the correct registry and prefix for private registry
func AssertPrivateRegistryImage(t *testing.T, client client.Client, deployment *appsv1.Deployment, expectedImageRegistry, expectedImagePrefix string) {
	vpoRepo := expectedImageRegistry + "/" + expectedImagePrefix + "/verrazzano/" + constants.VerrazzanoPlatformOperator
	assert.True(t, strings.HasPrefix(deployment.Spec.Template.Spec.InitContainers[0].Image, vpoRepo),
		"Expected container image %s to start with %s", deployment.Spec.Template.Spec.InitContainers[0].Image, vpoRepo)
	assert.True(t, strings.HasPrefix(deployment.Spec.Template.Spec.Containers[0].Image, vpoRepo),
		"Expected init container image %s to start with %s", deployment.Spec.Template.Spec.InitContainers[0].Image, vpoRepo)
}
