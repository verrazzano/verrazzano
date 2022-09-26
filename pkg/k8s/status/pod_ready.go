// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package status

import (
	"context"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

// pod label used to identify the replicaset resource for deployments
const podTemplateHashLabel = "pod-template-hash"

// annotation used to identify the revision of a replicaset
const deploymentRevisionAnnotation = "deployment.kubernetes.io/revision"

// GetPodsList retrieves a list of pods for a given namespace and labels selector
func GetPodsList(log vzlog.VerrazzanoLogger, client clipkg.Client, namespacedName types.NamespacedName, selector *metav1.LabelSelector) *corev1.PodList {
	labelSelector, err := metav1.LabelSelectorAsSelector(selector)
	if err != nil {
		logErrorf(log, "Failed to convert LabelSelector %v for %v: %v", selector, namespacedName, err)
		return nil
	}
	var pods corev1.PodList
	err = client.List(context.TODO(), &pods,
		&clipkg.ListOptions{Namespace: namespacedName.Namespace, LabelSelector: labelSelector})
	if err != nil {
		logErrorf(log, "Failed listing pods in namespace %s: %v", namespacedName.Namespace, err)
		return nil
	}

	return &pods
}

// EnsurePodsAreReady makes sure pods using the latest workload revision are ready.
// A list of pods using the latest revision are passed to this function.
func EnsurePodsAreReady(log vzlog.VerrazzanoLogger, podsToCheck []corev1.Pod, expectedPods int32, prefix string) (int32, bool) {
	var podsReady int32
	for _, pod := range podsToCheck {
		// Check that init containers are ready
		for _, initContainerStatus := range pod.Status.InitContainerStatuses {
			if !initContainerStatus.Ready {
				logProgressf(log, "%s is waiting for init container of pod %s to be ready", prefix, pod.Name)
				return 0, false
			}
		}
		// Check that containers are ready
		for _, containerStatus := range pod.Status.ContainerStatuses {
			if !containerStatus.Ready {
				logProgressf(log, "%s is waiting for container of pod %s to be ready", prefix, pod.Name)
				return 0, false
			}
		}

		podsReady++

		// No need to look at other pods if the expected pods are ready
		if podsReady == expectedPods {
			break
		}
	}
	return podsReady, true
}

func logOncef(log vzlog.VerrazzanoLogger, template string, args ...interface{}) {
	if log != nil {
		log.Oncef(template, args...)
	}
}

func logErrorf(log vzlog.VerrazzanoLogger, template string, args ...interface{}) {
	if log != nil {
		log.Errorf(template, args...)
	}
}

func logProgressf(log vzlog.VerrazzanoLogger, template string, args ...interface{}) {
	if log != nil {
		log.Progressf(template, args...)
	}
}
