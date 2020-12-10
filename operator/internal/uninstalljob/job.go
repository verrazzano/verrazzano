// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package uninstalljob

import (
	"github.com/verrazzano/verrazzano/operator/internal"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strconv"
)

// JobConfig Defines the parameters for an uninstall job
type JobConfig struct {
	internal.JobConfigCommon
}

// uninstallMode value for MODE variable for uninstall jobs
const uninstallMode = "UNINSTALL"

// NewJob returns a job resource for uninstalling Verrazzano
func NewJob(jobConfig *JobConfig) *batchv1.Job {
	var backOffLimit int32 = 0
	mode := uninstallMode
	if jobConfig.DryRun {
		mode = internal.NoOpMode
	}
	var annotations map[string]string = nil
	if jobConfig.DryRun {
		annotations = make(map[string]string, 1)
		annotations[internal.DryRunAnnotationName] = strconv.FormatBool(jobConfig.DryRun)
	}
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
						Name:            "uninstall",
						Image:           jobConfig.JobImage,
						ImagePullPolicy: corev1.PullIfNotPresent,
						Env: []corev1.EnvVar{
							{
								Name:  "MODE",
								Value: mode,
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
					ServiceAccountName: jobConfig.ServiceAccountName,
				},
			},
		},
	}
	return job
}
