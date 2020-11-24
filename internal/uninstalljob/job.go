// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package uninstalljob

import (
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewJob returns a job resource for uninstalling Verrazzano
// namespace - namespace of verrazzano resource
// name - name of job resource
// labels - labels of verrazzano resource
// serviceAccount - name of service account for job
// image - docker image containing uninstall scripts
func NewJob(namespace string, name string, labels map[string]string, serviceAccount string, image string) *batchv1.Job {
	var backOffLimit int32 = 0
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: &backOffLimit,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
					Labels:    labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:            "uninstall",
						Image:           image,
						ImagePullPolicy: corev1.PullIfNotPresent,
						Env: []corev1.EnvVar{
							{
								Name:  "MODE",
								Value: "UNINSTALL",
							},
							{
								Name:  "CLUSTER_TYPE",
								Value: "OKE",
							},
							{
								Name:  "VERRAZZANO_KUBECONFIG",
								Value: "/home/verrazzano/kubeconfig",
							},
							{
								// DEBUG property set to value 1 will direct more detailed output to stdout and
								// will thus provide more insight when the pod logs are retrieved
								Name:  "DEBUG",
								Value: "1",
							},
						},
					}},
					RestartPolicy:      corev1.RestartPolicyNever,
					ServiceAccountName: serviceAccount,
				},
			},
		},
	}

	return job
}
