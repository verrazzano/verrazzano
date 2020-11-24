// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package installjob

import (
	installv1alpha1 "github.com/verrazzano/verrazzano-platform-operator/api/v1alpha1"
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
func NewJob(namespace string, jobName string, labels map[string]string, configMapName string, saName string, image string, dns installv1alpha1.DNS) *batchv1.Job {
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
	if dns.OCI != (installv1alpha1.OCI{}) {
		mountPrivateKeyFile(job, dns)
	}

	return job
}

func mountPrivateKeyFile(job *batchv1.Job, dns installv1alpha1.DNS) {
	// need to attach the secret as a volume
	job.Spec.Template.Spec.Volumes = append(job.Spec.Template.Spec.Volumes, corev1.Volume{
		Name: "oci-private-key",
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: dns.OCI.PrivateKeyFileSecretName,
			},
		},
	})
	job.Spec.Template.Spec.Containers[0].VolumeMounts = append(
		job.Spec.Template.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
			Name:      "oci-private-key",
			ReadOnly:  true,
			MountPath: installv1alpha1.OciPrivateKeyFilePath,
		})
}
