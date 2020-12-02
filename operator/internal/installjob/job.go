// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package installjob

import (
	"github.com/verrazzano/verrazzano/operator/internal"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strconv"
)

// JobConfig Defines the parameters for an install job
type JobConfig struct {
	internal.JobConfigCommon // Extending the base job config

	ConfigMapName string // Name of the install configmap used by the scripts
}

// installMode value for MODE variable for install jobs
const installMode = "INSTALL"

// NewJob returns a job resource for installing Verrazzano
func NewJob(jobConfig *JobConfig) *batchv1.Job {
	var backOffLimit int32 = 0
	mode := installMode
	if jobConfig.DryRun {
		mode = internal.NoOpMode
	}
	annotations := make(map[string]string, 1)
	annotations[internal.DryRunAnnotationName] = strconv.FormatBool(jobConfig.DryRun)
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:        jobConfig.JobName,
			Namespace:   jobConfig.Namespace,
			Labels:      jobConfig.Labels,
			Annotations: annotations,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: &backOffLimit,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:        jobConfig.JobName,
					Namespace:   jobConfig.Namespace,
					Labels:      jobConfig.Labels,
					Annotations: annotations,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:            "install",
						Image:           jobConfig.JobImage,
						ImagePullPolicy: corev1.PullIfNotPresent,
						Env: []corev1.EnvVar{
							{
								Name:  "MODE",
								Value: mode,
							},
							{
								Name:  "INSTALL_CONFIG_FILE",
								Value: "/config/config.json",
							},
							{
								Name:  "VERRAZZANO_KUBECONFIG",
								Value: "/home/verrazzano/kubeconfig",
							},
							{
								// DEBUG property set to value 1 will direct more detailed output to stdout and
								// will thus provide more insight when the installer pod logs are retrieved
								Name:  "DEBUG",
								Value: "1",
							},
						},
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "config-volume",
								MountPath: "/config",
							},
						},
					}},
					RestartPolicy:      corev1.RestartPolicyNever,
					ServiceAccountName: jobConfig.ServiceAccountName,
					Volumes: []corev1.Volume{
						{
							Name: "config-volume",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: jobConfig.ConfigMapName,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	return job
}
