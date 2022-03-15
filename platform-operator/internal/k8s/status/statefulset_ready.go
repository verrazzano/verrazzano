// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package status

import (
	"context"

	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

// StatefulSetsAreReady Check that the named statefulsets have the minimum number of specified replicas ready and available
func StatefulSetsAreReady(log vzlog.VerrazzanoLogger, client client.Client, checks []PodReadyCheck, expectedReplicas int32, prefix string) bool {
	for _, check := range checks {
		statefulset := appsv1.StatefulSet{}
		if err := client.Get(context.TODO(), check.NamespacedName, &statefulset); err != nil {
			if errors.IsNotFound(err) {
				log.Progressf("%s is waiting for statefulset %v to exist", prefix, check.NamespacedName)
				// StatefulSet not found
				return false
			}
			log.Errorf("Failed getting statefulset %v: %v", check.NamespacedName, err)
			return false
		}
		if statefulset.Status.UpdatedReplicas < expectedReplicas {
			log.Progressf("%s is waiting for statefulset %s replicas to be %v. Current updated replicas is %v", prefix, check.NamespacedName,
				expectedReplicas, statefulset.Status.ReadyReplicas)
			return false
		}
		if statefulset.Status.ReadyReplicas < expectedReplicas {
			log.Progressf("%s is waiting for statefulset %s replicas to be %v. Current ready replicas is %v", prefix, check.NamespacedName,
				expectedReplicas, statefulset.Status.ReadyReplicas)
			return false
		}
		if !podsReadyStatefulSet(log, client, check, expectedReplicas, prefix) {
			return false
		}
		log.Oncef("%s has enough replicas for statefulsets %v", prefix, check.NamespacedName)
	}
	return true
}

// podsReadyStatefulSet checks for an expected number of pods to be using the latest controllerRevision resource and are
// running and ready
func podsReadyStatefulSet(log vzlog.VerrazzanoLogger, client clipkg.Client, check PodReadyCheck, expectedReplicas int32, prefix string) bool {
	// Get a list of pods for a given namespace and labels selector
	pods := getPodsList(log, client, check)
	if pods == nil {
		return false
	}

	// If no pods found log a progress message and return
	if len(pods.Items) == 0 {
		log.Progressf("Found no pods with matching labels selector %v for namespace %s", check.LabelSelector, check.NamespacedName.Namespace)
		return true
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
		err := client.Get(context.TODO(), types.NamespacedName{Namespace: check.NamespacedName.Namespace, Name: pod.Labels[controllerRevisionHashLabel]}, &cr)
		if err != nil {
			log.Errorf("Failed to get controllerRevision %s: %v", check.NamespacedName, err)
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
	podsReady, success := ensurePodsAreReady(log, savedPods, expectedReplicas, prefix)
	if !success {
		return false
	}

	if podsReady < expectedReplicas {
		log.Progressf("%s is waiting for statefulset %s pods to be %v. Current available pods are %v", prefix, check.NamespacedName,
			expectedReplicas, podsReady)
		return false
	}

	return true
}
