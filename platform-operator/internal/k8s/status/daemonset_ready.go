// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package status

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8s/status"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DaemonSetsAreReady Check that the named daemonsets have the minimum number of specified nodes ready and available
func DaemonSetsAreReady(log vzlog.VerrazzanoLogger, client client.Client, namespacedNames []types.NamespacedName, expectedNodes int32, prefix string) bool {
	resticPodLabel := map[string]string{
		"name": constants.ResticDaemonSetName,
	}
	resticPodSelector := &metav1.LabelSelector{
		MatchLabels: resticPodLabel,
	}
	for _, namespacedName := range namespacedNames {
		daemonset := appsv1.DaemonSet{}
		if err := client.Get(context.TODO(), namespacedName, &daemonset); err != nil {
			if errors.IsNotFound(err) {
				log.Progressf("%s is waiting for daemonsets %v to exist", prefix, namespacedName)
				return false
			}
			log.Errorf("Failed getting daemonset %v: %v", namespacedName, err)
			return false
		}
		if daemonset.Status.UpdatedNumberScheduled < expectedNodes {
			log.Progressf("%s is waiting for daemonset %s nodes to be %v. Current updated nodes is %v", prefix, namespacedName,
				expectedNodes, daemonset.Status.NumberAvailable)
			return false
		}

		if daemonset.Status.NumberAvailable < expectedNodes {
			log.Progressf("%s is waiting for daemonset %s nodes to be %v. Current available nodes is %v", prefix, namespacedName,
				expectedNodes, daemonset.Status.NumberAvailable)
			return false
		}

		// Velero install deploys a daemonset and deployment with common labels. The labels need to be adjusted so the pod fetch logic works
		// as expected
		podSelector := daemonset.Spec.Selector
		if namespacedName.Namespace == constants.VeleroNameSpace {
			podSelector = resticPodSelector
		}

		if !podsReadyDaemonSet(log, client, namespacedName, podSelector, expectedNodes, prefix) {
			return false
		}
		log.Oncef("%s has enough nodes for daemonsets %v", prefix, namespacedName)
	}
	return true
}

// podsReadyDaemonSet checks for an expected number of pods to be using the latest controllerRevision resource and are
// running and ready
func podsReadyDaemonSet(log vzlog.VerrazzanoLogger, client client.Client, namespacedName types.NamespacedName, selector *metav1.LabelSelector, expectedNodes int32, prefix string) bool {
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
		crName := fmt.Sprintf("%s-%s", namespacedName.Name, pod.Labels[controllerRevisionHashLabel])
		err := client.Get(context.TODO(), types.NamespacedName{Namespace: namespacedName.Namespace, Name: crName}, &cr)
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
	podsReady, success := status.EnsurePodsAreReady(log, savedPods, expectedNodes, prefix)
	if !success {
		return false
	}

	if podsReady < expectedNodes {
		log.Progressf("%s is waiting for daemonset %s pods to be %v. Current available pods are %v", prefix, namespacedName,
			expectedNodes, podsReady)
		return false
	}

	return true
}
