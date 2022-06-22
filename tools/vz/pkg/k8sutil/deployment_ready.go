// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package k8sutil

import (
	"context"
	"fmt"
	"strconv"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

// DeploymentsAreReady check that the named deployments have the minimum number of specified replicas ready and available
func DeploymentsAreReady(client clipkg.Client, namespacedNames []types.NamespacedName, expectedReplicas int32) (bool, error) {
	for _, namespacedName := range namespacedNames {
		deployment := appsv1.Deployment{}
		if err := client.Get(context.TODO(), namespacedName, &deployment); err != nil {
			if errors.IsNotFound(err) {
				return false, fmt.Errorf("waiting for deployment %v to exist", namespacedName)
			}
			return false, fmt.Errorf("failed getting deployment %v: %v", namespacedName, err)
		}
		if deployment.Status.UpdatedReplicas < expectedReplicas {
			return false, fmt.Errorf("waiting for deployment %s replicas to be %v, current updated replicas is %v", namespacedName,
				expectedReplicas, deployment.Status.UpdatedReplicas)
		}
		if deployment.Status.AvailableReplicas < expectedReplicas {
			return false, fmt.Errorf("waiting for deployment %s replicas to be %v, current available replicas is %v", namespacedName,
				expectedReplicas, deployment.Status.AvailableReplicas)
		}
		ready, err := podsReadyDeployment(client, namespacedName, deployment.Spec.Selector, expectedReplicas)
		if !ready {
			return false, err
		}
	}
	return true, nil
}

// podsReadyDeployment checks for an expected number of pods to be using the latest replicaset revision and are
// running and ready
func podsReadyDeployment(client clipkg.Client, namespacedName types.NamespacedName, selector *metav1.LabelSelector, expectedReplicas int32) (bool, error) {
	// Get a list of pods for a given namespace and labels selector
	pods, err := getPodsList(client, namespacedName, selector)
	if err != nil {
		return false, err
	}
	if pods == nil {
		return false, nil
	}

	// If no pods found log a progress message and return
	if len(pods.Items) == 0 {
		return true, fmt.Errorf("no pods found with matching labels selector %v for namespace %s", selector, namespacedName.Namespace)
	}

	// Loop through pods identifying pods that are using the latest replicaset revision
	var savedPods []corev1.Pod
	var savedPodTemplateHash string
	var savedRevision int
	for _, pod := range pods.Items {
		// Log error and return if the pod-template-hash label is not found.  This should never happen.
		if _, ok := pod.Labels[podTemplateHashLabel]; !ok {
			return false, fmt.Errorf("failed to find label [pod-template-hash] for pod %s/%s", pod.Namespace, pod.Name)
		}

		if pod.Labels[podTemplateHashLabel] == savedPodTemplateHash {
			savedPods = append(savedPods, pod)
			continue
		}

		// Get the replica set for the pod given the pod-template-hash label
		var rs appsv1.ReplicaSet
		rsName := fmt.Sprintf("%s-%s", namespacedName.Name, pod.Labels[podTemplateHashLabel])
		err := client.Get(context.TODO(), types.NamespacedName{Namespace: namespacedName.Namespace, Name: rsName}, &rs)
		if err != nil {
			return false, fmt.Errorf("failed to get replicaset %s", namespacedName)
		}

		// Log error and return if the deployment.kubernetes.io/revision annotation is not found.  This should never happen.
		if _, ok := rs.Annotations[deploymentRevisionAnnotation]; !ok {
			return false, fmt.Errorf("failed to find annotation [deployment.kubernetes.io/revision] for pod %s/%s", pod.Namespace, pod.Name)
		}

		revision, _ := strconv.Atoi(rs.Annotations[deploymentRevisionAnnotation])
		if revision > savedRevision {
			savedRevision = revision
			savedPodTemplateHash = pod.Labels[podTemplateHashLabel]
			savedPods = []corev1.Pod{}
			savedPods = append(savedPods, pod)
		}
	}

	// Make sure pods using the latest replicaset revision are ready.
	podsReady, success, err := ensurePodsAreReady(savedPods, expectedReplicas)
	if !success {
		return false, err
	}

	if podsReady < expectedReplicas {
		return false, fmt.Errorf("waiting for deployment %s pods to be %v, current available pods are %v", namespacedName,
			expectedReplicas, podsReady)
	}

	return true, nil
}
