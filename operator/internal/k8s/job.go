// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package k8s

import (
	"context"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

// JobConfigCommon Common configuration for install/uninstall jobs
type JobConfigCommon struct {
	JobName            string            // Name of the job
	Namespace          string            // Namespace for the job
	Labels             map[string]string // Container labels for the job
	ServiceAccountName string            // Service account name to execute the job as
	JobImage           string            // Image name/tag for the job
	DryRun             bool              // Perform the job as a dry-run/no-op, for testing purposes
}

// NoOpMode value for MODE variable for no-op (test) jobs
const NoOpMode = "NOOP"

// DryRunAnnotationName annotation used on jobs to indicate dry-run state [true|false]
const DryRunAnnotationName = "dry-run"

// DeleteJob the job if it exists
func DeleteJob(client clipkg.Client, jobName string, namespace string) error {
	job := &batchv1.Job{}
	err := client.Get(context.TODO(), types.NamespacedName{Name: jobName, Namespace: namespace}, job)
	if err == nil {
		propagationPolicy := metav1.DeletePropagationBackground
		deleteOptions := &clipkg.DeleteOptions{PropagationPolicy: &propagationPolicy}
		err = client.Delete(context.TODO(), job, deleteOptions)
		if err != nil {
			return err
		}
	}
	return nil
}
