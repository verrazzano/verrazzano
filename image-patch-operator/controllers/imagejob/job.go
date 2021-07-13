// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package imagejob

import (
	"strconv"

	"github.com/verrazzano/verrazzano/image-patch-operator/internal/k8s"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// JobConfig Defines the parameters for an image job
type JobConfig struct {
	k8s.JobConfigCommon // Extending the base job config

	ConfigMapName string // Name of the image configmap used by the image job
}

// NewJob returns a job resource for building an ImageBuildRequest
func NewJob(jobConfig *JobConfig) *batchv1.Job {
	var annotations map[string]string = nil
	var backoffLimit int32 = 0
	if jobConfig.DryRun {
		annotations = make(map[string]string, 1)
		annotations[k8s.DryRunAnnotationName] = strconv.FormatBool(jobConfig.DryRun)
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:        jobConfig.JobName,
			Namespace:   jobConfig.Namespace,
			Labels:      jobConfig.Labels,
			Annotations: annotations,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: &backoffLimit,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:        jobConfig.JobName,
					Namespace:   jobConfig.Namespace,
					Labels:      jobConfig.Labels,
					Annotations: annotations,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:            "image-build-request",
						Image:           jobConfig.JobImage,
						ImagePullPolicy: corev1.PullIfNotPresent,
						Env: []corev1.EnvVar{
							{
								// DEBUG property set to value 1 will direct more detailed output to stdout and
								// will thus provide more insight when the installer pod logs are retrieved
								Name:  "DEBUG",
								Value: "1",
							},
						},
					}},
					RestartPolicy:      corev1.RestartPolicyNever,
					ServiceAccountName: jobConfig.ServiceAccountName,
				},
			},
		},
	}

	return job
}
