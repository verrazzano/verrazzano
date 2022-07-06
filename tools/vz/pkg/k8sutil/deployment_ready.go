// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package k8sutil

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

// DeploymentsAreReady check that the named deployments have the minimum number of specified replicas ready and available
func DeploymentsAreReady(client clipkg.Client, namespacedNames []types.NamespacedName, expectedReplicas int32, lastTransitionTime metav1.Time) (bool, error) {
	readyCount := 0
	for _, namespacedName := range namespacedNames {
		deployment := appsv1.Deployment{}
		if err := client.Get(context.TODO(), namespacedName, &deployment); err != nil {
			if errors.IsNotFound(err) {
				return false, fmt.Errorf("waiting for deployment %v to exist", namespacedName)
			}
			return false, fmt.Errorf("failed getting deployment %v: %v", namespacedName, err)
		}
		// Check the deployment condition status
		for _, deploymentCondition := range deployment.Status.Conditions {
			if lastTransitionTime.After(deploymentCondition.LastTransitionTime.Time) {
				return false, fmt.Errorf("waiting for deployment %s condition to be %s", namespacedName, appsv1.DeploymentAvailable)
			}
			if deploymentCondition.Type == appsv1.DeploymentAvailable {
				if deploymentCondition.Status == corev1.ConditionTrue {
					readyCount++
					break
				}
			}
			return false, fmt.Errorf("waiting for deployment %s condition to be %s", namespacedName, appsv1.DeploymentAvailable)
		}
	}
	if readyCount != len(namespacedNames) {
		return false, fmt.Errorf("waiting for deployments to be ready")
	}
	return true, nil
}
