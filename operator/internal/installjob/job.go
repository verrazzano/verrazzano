// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package installjob

import (
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewJob returns a job resource for installing Verrazzano
// namespace - namespace of verrazzano resource
// jobNname - name of verrazzano resource
// labels - labels of verrazzano resource
// saName - service account name
// image - docker image
func NewJob(namespace string, jobName string, labels map[string]string, configMapName string, saName string, image string) *batchv1.Job {
	var backOffLimit int32 = 0
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: &backOffLimit,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      jobName,
					Namespace: namespace,
					Labels:    labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:            "install",
						Image:           image,
						ImagePullPolicy: corev1.PullIfNotPresent,
						Env: []corev1.EnvVar{
							{
								Name:  "MODE",
								Value: "INSTALL",
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
					ServiceAccountName: saName,
					Volumes: []corev1.Volume{
						{
							Name: "config-volume",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: configMapName,
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
