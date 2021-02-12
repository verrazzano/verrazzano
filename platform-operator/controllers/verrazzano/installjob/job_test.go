// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package installjob

import (
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewJob tests the creation of a job with xip.io DNS specified
// GIVEN a request to create a job
// WHEN xip.io DNS has been specified
// THEN a job is created with the appropriate items to support an xip.io DNS based installation in INSTALL mode
func TestNewJob(t *testing.T) {
	namespace := "verrazzano"
	name := "test-job"
	labels := map[string]string{"label1": "test", "label2": "test2"}
	configMapName := "test-config"
	serviceAccount := "job"
	image := "docker-image"

	job := NewJob(&JobConfig{
		JobConfigCommon: k8s.JobConfigCommon{
			JobName:            name,
			Namespace:          namespace,
			Labels:             labels,
			ServiceAccountName: serviceAccount,
			JobImage:           image,
			DryRun:             false,
		},
		ConfigMapName: configMapName,
	})

	assert.Equalf(t, namespace, job.Namespace, "Expected namespace did not match")
	assert.Equalf(t, name, job.Name, "Expected job name did not match")
	assert.Equalf(t, labels, job.Labels, "Expected labels did not match")
	assert.Equalf(t, configMapName, job.Spec.Template.Spec.Volumes[0].ConfigMap.Name, "Expected configmap name did not match")
	assert.Equalf(t, serviceAccount, job.Spec.Template.Spec.ServiceAccountName, "Expected service account name did not match")
	assert.Equalf(t, image, job.Spec.Template.Spec.Containers[0].Image, "Expected service account name did not match")

	assert.Equal(t, "MODE", job.Spec.Template.Spec.Containers[0].Env[0].Name)
	assert.Equal(t, "INSTALL", job.Spec.Template.Spec.Containers[0].Env[0].Value)
	_, ok := job.Annotations["dry-run"]
	assert.False(t, ok)
}

// TestNewJobDryRun tests the creation of a job with dryRun=true, that the MODE env var is NOOP
// GIVEN a request to create a job
// WHEN dryRun==true
// THEN a job is created with the env var MODE=NOOP
func TestNewJobDryRun(t *testing.T) {
	namespace := "verrazzano"
	name := "test-job"
	labels := map[string]string{"label1": "test", "label2": "test2"}
	configMapName := "test-config"
	serviceAccount := "job"
	image := "docker-image"

	job := NewJob(&JobConfig{
		JobConfigCommon: k8s.JobConfigCommon{
			JobName:            name,
			Namespace:          namespace,
			Labels:             labels,
			ServiceAccountName: serviceAccount,
			JobImage:           image,
			DryRun:             true,
		},
		ConfigMapName: configMapName,
	})

	assert.Equal(t, "MODE", job.Spec.Template.Spec.Containers[0].Env[0].Name)
	assert.Equal(t, "NOOP", job.Spec.Template.Spec.Containers[0].Env[0].Value)

	dryRun, ok := job.Annotations["dry-run"]
	assert.True(t, ok)
	assert.Equal(t, "true", dryRun)
}
