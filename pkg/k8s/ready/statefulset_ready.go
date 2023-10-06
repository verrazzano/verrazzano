// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ready

import (
	"context"

	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common/opensearch"

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
				expectedReplicas, statefulset.Status.UpdatedReplicas)
			return false
		}
		if statefulset.Status.ReadyReplicas < expectedReplicas {
			log.Progressf("%s is waiting for statefulset %s replicas to be %v. Current ready replicas is %v", prefix, namespacedName,
				expectedReplicas, statefulset.Status.ReadyReplicas)
			return false
		}
		if !PodsReadyStatefulSet(log, client, namespacedName, statefulset.Spec.Selector, expectedReplicas, prefix) {
			return false
		}
		log.Oncef("%s has enough replicas for statefulsets %v", prefix, namespacedName)
	}
	return true
}

// DoStatefulSetsExist checks if all the named statefulsets exist
func DoStatefulSetsExist(log vzlog.VerrazzanoLogger, client client.Client, namespacedNames []types.NamespacedName, _ int32, prefix string) bool {
	for _, namespacedName := range namespacedNames {
		exists, err := DoesStatefulsetExist(client, namespacedName)
		if err != nil {
			logErrorf(log, "%s failed getting statefulset %v: %v", prefix, namespacedName, err)
			return false
		}
		if !exists {
			logProgressf(log, "%s is waiting for statefulset %v to exist", prefix, namespacedName)
			return false
		}
	}
	return true
}

// DoesStatefulsetExist checks if the named statefulset exists
func DoesStatefulsetExist(client client.Client, namespacedName types.NamespacedName) (bool, error) {
	sts := appsv1.StatefulSet{}
	if err := client.Get(context.TODO(), namespacedName, &sts); err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// PodsReadyStatefulSet checks for an expected number of pods to be using the latest controllerRevision resource and are
// running and ready
func PodsReadyStatefulSet(log vzlog.VerrazzanoLogger, client client.Client, namespacedName types.NamespacedName, selector *metav1.LabelSelector, expectedReplicas int32, prefix string) bool {
	// Get a list of pods for a given namespace and labels selector
	pods := GetPodsList(log, client, namespacedName, selector)
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
	podsReady, success := EnsurePodsAreReady(log, savedPods, expectedReplicas, prefix)
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

// AreOpensearchStsReady Check that the OS statefulsets have the minimum number of specified replicas ready and available. It ignores the updated replicas check if updated replicas are zero or cluster is not healthy.
func AreOpensearchStsReady(log vzlog.VerrazzanoLogger, client client.Client, namespacedNames []types.NamespacedName, expectedReplicas int32, prefix string) bool {
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
		if !areOSReplicasUpdated(log, statefulset, expectedReplicas, client, prefix, namespacedName) {
			return false
		}
		if statefulset.Status.ReadyReplicas < expectedReplicas {
			log.Progressf("%s is waiting for statefulset %s replicas to be %v. Current ready replicas is %v", prefix, namespacedName,
				expectedReplicas, statefulset.Status.ReadyReplicas)
			return false
		}
		log.Oncef("%s has enough ready replicas for statefulsets %v", prefix, namespacedName)
	}
	return true
}

// areOSReplicasUpdated check whether all replicas of opensearch are updated or not. In case of yellow cluster status, we skip this check and consider replicas are updated.
func areOSReplicasUpdated(log vzlog.VerrazzanoLogger, statefulset appsv1.StatefulSet, expectedReplicas int32, client client.Client, prefix string, namespacedName types.NamespacedName) bool {
	if statefulset.Status.UpdatedReplicas > 0 && statefulset.Status.UpdateRevision != statefulset.Status.CurrentRevision && statefulset.Status.UpdatedReplicas < expectedReplicas {
		pas, err := opensearch.GetVerrazzanoPassword(client)
		if err != nil {
			log.Errorf("Failed getting OS secret to check OS cluster health: %v", err)
			return false
		}
		osClient := opensearch.NewOSClient(pas)
		healthy, err := osClient.IsClusterHealthy(client)
		if err != nil {
			log.Errorf("Failed getting Opensearch cluster health: %v", err)
			return false
		}
		if !healthy {
			log.Progressf("Opensearch Cluster is not healthy. Please check Opensearch operator for more information")
			return true
		}
		log.Progressf("%s is waiting for statefulset %s replicas to be %v. Current updated replicas is %v", prefix, namespacedName,
			expectedReplicas, statefulset.Status.UpdatedReplicas)
		return false
	}
	return true
}
