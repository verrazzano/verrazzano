// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package k8sutil

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

// pod label used to identify the controllerRevision resource for daemonsets and statefulsets
const controllerRevisionHashLabel = "controller-revision-hash"

// pod label used to identify the replicaset resource for deployments
const podTemplateHashLabel = "pod-template-hash"

// annotation used to identify the revision of a replicaset
const deploymentRevisionAnnotation = "deployment.kubernetes.io/revision"

// getPodsList retrieves a list of pods for a given namespace and labels selector
func getPodsList(client clipkg.Client, namespacedName types.NamespacedName, selector *metav1.LabelSelector) (*corev1.PodList, error) {
	labelSelector, err := metav1.LabelSelectorAsSelector(selector)
	if err != nil {
		return nil, fmt.Errorf("Failed to convert LabelSelector %v for %v: %v", selector, namespacedName, err)
	}
	var pods corev1.PodList
	err = client.List(context.TODO(), &pods,
		&clipkg.ListOptions{Namespace: namespacedName.Namespace, LabelSelector: labelSelector})
	if err != nil {
		return nil, fmt.Errorf("Failed listing pods in namespace %s: %v", namespacedName.Namespace, err)
	}

	return &pods, nil
}

// ensurePodsAreReady makes sure pods using the latest workload revision are ready.
// A list of pods using the latest revision are passed to this function.
func ensurePodsAreReady(podsToCheck []corev1.Pod, expectedPods int32, prefix string) (int32, bool, error) {
	var podsReady int32 = 0
	for _, pod := range podsToCheck {
		// Check that init containers are ready
		for _, initContainerStatus := range pod.Status.InitContainerStatuses {
			if !initContainerStatus.Ready {
				return 0, false, fmt.Errorf("%s is waiting for init container of pod %s to be ready", prefix, pod.Name)
			}
		}
		// Check that containers are ready
		for _, containerStatus := range pod.Status.ContainerStatuses {
			if !containerStatus.Ready {
				return 0, false, fmt.Errorf("%s is waiting for container of pod %s to be ready", prefix, pod.Name)
			}
		}

		podsReady++

		// No need to look at other pods if the expected pods are ready
		if podsReady == expectedPods {
			break
		}
	}
	return podsReady, true, nil
}
