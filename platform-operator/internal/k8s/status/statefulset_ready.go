// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package status

import (
	"context"

	"github.com/verrazzano/verrazzano/pkg/k8s/status"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// pod label used to identify the controllerRevision resource for daemonsets and statefulsets
const controllerRevisionHashLabel = "controller-revision-hash"

// StatefulSetsAreReady Check that the named statefulsets have the minimum number of specified replicas ready and available
func StatefulSetsAreReady(log vzlog.VerrazzanoLogger, client client.Client, namespacedNames []types.NamespacedName, expectedReplicas int32, prefix string) bool {
	for _, namespacedName := range namespacedNames {
		statefulset := appsv1.StatefulSet{}
		if err := client.Get(context.TODO(), namespacedName, &statefulset); err != nil {
			if errors.IsNotFound(err) {
				log.Progressf("%s is waiting for statefulset %v to exist", prefix, namespacedName)
				// StatefulSet not found
				return false
			}
			log.Errorf("Failed getting statefulset %v: %v", namespacedName, err)
			return false
		}
		if statefulset.Status.UpdatedReplicas < expectedReplicas {
			log.Progressf("%s is waiting for statefulset %s replicas to be %v. Current updated replicas is %v", prefix, namespacedName,
				expectedReplicas, statefulset.Status.ReadyReplicas)
			return false
		}
		if statefulset.Status.ReadyReplicas < expectedReplicas {
			log.Progressf("%s is waiting for statefulset %s replicas to be %v. Current ready replicas is %v", prefix, namespacedName,
				expectedReplicas, statefulset.Status.ReadyReplicas)
			return false
		}
		if !podsReadyStatefulSet(log, client, namespacedName, statefulset.Spec.Selector, expectedReplicas, prefix) {
			return false
		}
		log.Oncef("%s has enough replicas for statefulsets %v", prefix, namespacedName)
	}
	return true
}

// DoStatefulSetsExist checks if the named statefulsets exist
func DoStatefulSetsExist(log vzlog.VerrazzanoLogger, client client.Client, namespacedNames []types.NamespacedName, _ int32, prefix string) bool {
	for _, namespacedName := range namespacedNames {
		statuefulset := appsv1.StatefulSet{}
		if err := client.Get(context.TODO(), namespacedName, &statuefulset); err != nil {
			if errors.IsNotFound(err) {
				log.Progressf("%s is waiting for statuefulset %v to exist", prefix, namespacedName)
				return false
			}
			log.Errorf("%s failed getting statuefulset %v: %v", prefix, namespacedName, err)
			return false
		}
	}
	return true
}

// podsReadyStatefulSet checks for an expected number of pods to be using the latest controllerRevision resource and are
// running and ready
func podsReadyStatefulSet(log vzlog.VerrazzanoLogger, client client.Client, namespacedName types.NamespacedName, selector *metav1.LabelSelector, expectedReplicas int32, prefix string) bool {
	// Get a list of pods for a given namespace and labels selector
	pods := status.GetPodsList(log, client, namespacedName, selector)
	if pods == nil {
		return false
	}

	// If no pods found log a progress message and return
	if len(pods.Items) == 0 {
		log.Progressf("Found no pods with matching labels selector %v for namespace %s", selector, namespacedName.Namespace)
		return false
	}

	// Loop through pods identifying pods that are using the latest controllerRevision resource
	var savedPods []corev1.Pod
	var savedRevision int64
	var savedControllerRevisionHash string
	for _, pod := range pods.Items {
		// Log error and return if the controller-revision-hash label is not found.  This should never happen.
		if _, ok := pod.Labels[controllerRevisionHashLabel]; !ok {
			log.Errorf("Failed to find pod label [controller-revision-hash] for pod %s/%s", pod.Namespace, pod.Name)
			return false
		}

		if pod.Labels[controllerRevisionHashLabel] == savedControllerRevisionHash {
			savedPods = append(savedPods, pod)
			continue
		}

		// Get the controllerRevision resource for the pod given the controller-revision-hash label
		var cr appsv1.ControllerRevision
		err := client.Get(context.TODO(), types.NamespacedName{Namespace: namespacedName.Namespace, Name: pod.Labels[controllerRevisionHashLabel]}, &cr)
		if err != nil {
			log.Errorf("Failed to get controllerRevision %s: %v", namespacedName, err)
			return false
		}

		if cr.Revision > savedRevision {
			savedRevision = cr.Revision
			savedControllerRevisionHash = pod.Labels[controllerRevisionHashLabel]
			savedPods = []corev1.Pod{}
			savedPods = append(savedPods, pod)
		}
	}

	// Make sure pods using the latest controllerRevision resource are ready.
	podsReady, success := status.EnsurePodsAreReady(log, savedPods, expectedReplicas, prefix)
	if !success {
		return false
	}

	if podsReady < expectedReplicas {
		log.Progressf("%s is waiting for statefulset %s pods to be %v. Current available pods are %v", prefix, namespacedName,
			expectedReplicas, podsReady)
		return false
	}

	return true
}
