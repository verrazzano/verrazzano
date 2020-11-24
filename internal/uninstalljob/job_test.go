// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package uninstalljob

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestNewJob(t *testing.T) {
	namespace := "verrazzano"
	name := "test-job"
	labels := map[string]string{"label1": "test", "label2": "test2"}
	serviceAccount := "job"
	image := "docker-image"

	job := NewJob(namespace, name, labels, serviceAccount, image)

	assert.Equalf(t, namespace, job.Namespace, "Expected namespace did not match")
	assert.Equalf(t, name, job.Name, "Expected job name did not match")
	assert.Equalf(t, labels, job.Labels, "Expected labels did not match")
	assert.Equalf(t, namespace, job.Spec.Template.Namespace, "Expected namespace did not match")
	assert.Equalf(t, name, job.Spec.Template.Name, "Expected job name did not match")
	assert.Equalf(t, labels, job.Spec.Template.Labels, "Expected labels did not match")
	assert.Equalf(t, int32(0), *job.Spec.BackoffLimit, "Expected backoff limit did not match")
	assert.Equalf(t, serviceAccount, job.Spec.Template.Spec.ServiceAccountName, "Expected service account name did not match")
	assert.Equalf(t, corev1.RestartPolicyNever, job.Spec.Template.Spec.RestartPolicy, "Expected restart policy did not match")
	assert.Equalf(t, "uninstall", job.Spec.Template.Spec.Containers[0].Name, "Expected container name did not match")
	assert.Equalf(t, image, job.Spec.Template.Spec.Containers[0].Image, "Expected image did not match")
	assert.Equalf(t, corev1.PullIfNotPresent, job.Spec.Template.Spec.Containers[0].ImagePullPolicy, "Expected image pull policy did not match")
	assert.Equal(t, 4, len(job.Spec.Template.Spec.Containers[0].Env), "Expected length of env list did not match")
	assert.Equal(t, "MODE", job.Spec.Template.Spec.Containers[0].Env[0].Name, "Expected env name did not match")
	assert.Equal(t, "UNINSTALL", job.Spec.Template.Spec.Containers[0].Env[0].Value, "Expected env value did not match")
	assert.Equal(t, "CLUSTER_TYPE", job.Spec.Template.Spec.Containers[0].Env[1].Name, "Expected env name did not match")
	assert.Equal(t, "OKE", job.Spec.Template.Spec.Containers[0].Env[1].Value, "Expected env value did not match")
	assert.Equal(t, "VERRAZZANO_KUBECONFIG", job.Spec.Template.Spec.Containers[0].Env[2].Name, "Expected env name did not match")
	assert.Equal(t, "/home/verrazzano/kubeconfig", job.Spec.Template.Spec.Containers[0].Env[2].Value, "Expected env value did not match")
	assert.Equal(t, "DEBUG", job.Spec.Template.Spec.Containers[0].Env[3].Name, "Expected env name did not match")
	assert.Equal(t, "1", job.Spec.Template.Spec.Containers[0].Env[3].Value, "Expected env value did not match")
}
