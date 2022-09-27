// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package imagejob

import (
	"os"
	"strconv"

	"github.com/verrazzano/verrazzano/image-patch-operator/constants"
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
	var annotations map[string]string
	var backoffLimit int32
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
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								"cpu":    jobConfig.CPULimit,
								"memory": jobConfig.MemoryLimit,
							},
							Requests: corev1.ResourceList{
								"cpu":    jobConfig.CPURequest,
								"memory": jobConfig.MemoryRequest,
							},
						},
						Name:            "image-build-request",
						Image:           jobConfig.JobImage,
						ImagePullPolicy: corev1.PullIfNotPresent,
						// WARNING a Privileged SecurityContext is being used !!
						SecurityContext: newPrivilegedSecurityContext(),
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "registry-creds",
								MountPath: "/registry-creds",
								ReadOnly:  true,
							},
							{
								Name:      "installers-storage",
								MountPath: "/installers",
								ReadOnly:  true,
							},
						},
						Env: []corev1.EnvVar{
							{
								Name:  "IMAGE_NAME",
								Value: jobConfig.IBR.Spec.Image.Name,
							},
							{
								Name:  "IMAGE_TAG",
								Value: jobConfig.IBR.Spec.Image.Tag,
							},
							{
								Name:  "IMAGE_REGISTRY",
								Value: jobConfig.IBR.Spec.Image.Registry,
							},
							{
								Name:  "IMAGE_REPOSITORY",
								Value: jobConfig.IBR.Spec.Image.Repository,
							},
							{
								Name:  "BASE_IMAGE",
								Value: jobConfig.IBR.Spec.BaseImage,
							},
							{
								Name:  "JDK_INSTALLER_BINARY",
								Value: jobConfig.IBR.Spec.JDKInstaller,
							},
							{
								Name:  "JDK_INSTALLER_VERSION",
								Value: jobConfig.IBR.Spec.JdkInstallerVersion,
							},
							{
								Name:  "WEBLOGIC_INSTALLER_BINARY",
								Value: jobConfig.IBR.Spec.WebLogicInstaller,
							},
							{
								Name:  "WEBLOGIC_INSTALLER_VERSION",
								Value: jobConfig.IBR.Spec.WebLogicInstallerVersion,
							},
							{
								Name:  "WDT_INSTALLER_BINARY",
								Value: os.Getenv("WDT_INSTALLER_BINARY"),
							},
							{
								Name:  "WDT_INSTALLER_VERSION",
								Value: os.Getenv("WDT_INSTALLER_VERSION"),
							},
							{
								Name:  "LATEST_PSU",
								Value: strconv.FormatBool(jobConfig.IBR.Spec.LatestPSU),
							},
							{
								Name:  "RECOMMENDED_PATCHES",
								Value: strconv.FormatBool(jobConfig.IBR.Spec.RecommendedPatches),
							},
							{
								Name:  "IBR_DRY_RUN",
								Value: os.Getenv("IBR_DRY_RUN"),
							},
						},
					}},
					RestartPolicy:      corev1.RestartPolicyNever,
					ServiceAccountName: jobConfig.ServiceAccountName,
					Volumes: []corev1.Volume{
						{
							Name: "registry-creds",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: constants.ImageJobSecretName,
								},
							},
						},
						{
							Name: "installers-storage",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "installers-storage-claim"},
							},
						},
					},
				},
			},
		},
	}

	return job
}

// newPrivilegedSecurityContext returns a new Security Context with the Privileged flag set to true
func newPrivilegedSecurityContext() *corev1.SecurityContext {
	trueFlag := true
	securityContext := corev1.SecurityContext{Privileged: &trueFlag}
	return &securityContext
}
