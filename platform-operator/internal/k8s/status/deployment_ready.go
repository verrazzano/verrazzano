// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package status

import (
	"context"
	"fmt"
	"strconv"

	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	deploymentRevisionAnnotation = "deployment.kubernetes.io/revision"
	podTemplateHashLabel         = "pod-template-hash"
)

type PodReadyCheck struct {
	NamespacedName types.NamespacedName
	LabelSelector  labels.Selector
}

// DeploymentsAreReady check that the named deployments have the minimum number of specified replicas ready and available
func DeploymentsAreReady(log vzlog.VerrazzanoLogger, client clipkg.Client, checks []PodReadyCheck, expectedReplicas int32, prefix string) bool {
	for _, check := range checks {
		deployment := appsv1.Deployment{}
		if err := client.Get(context.TODO(), check.NamespacedName, &deployment); err != nil {
			if errors.IsNotFound(err) {
				log.Progressf("%s is waiting for deployment %v to exist", prefix, check.NamespacedName)
				return false
			}
			log.Errorf("%s failed getting deployment %v: %v", prefix, check.NamespacedName, err)
			return false
		}
		if deployment.Status.Replicas == 0 {
			log.Oncef("%s deployment is defined with zero replicas. No ready checking is performed.", prefix)
			return true
		}
		if deployment.Status.UpdatedReplicas < expectedReplicas {
			log.Progressf("%s is waiting for deployment %s replicas to be %v. Current updated replicas is %v", prefix, check.NamespacedName,
				expectedReplicas, deployment.Status.UpdatedReplicas)
			return false
		}
		if deployment.Status.AvailableReplicas < expectedReplicas {
			log.Progressf("%s is waiting for deployment %s replicas to be %v. Current available replicas is %v", prefix, check.NamespacedName,
				expectedReplicas, deployment.Status.AvailableReplicas)
			return false
		}
		if !podsReadyWithLatestRevision(log, client, check, expectedReplicas, prefix) {
			return false
		}
		log.Oncef("%s has enough replicas for deployment %v", prefix, check.NamespacedName)
	}
	return true
}

// podsReadyWithLatestRevision checks for an expected number of pods to be using the latest replicaset revision and are
// running and ready
func podsReadyWithLatestRevision(log vzlog.VerrazzanoLogger, client clipkg.Client, check PodReadyCheck, expectedReplicas int32, prefix string) bool {
	// Get a list of pods for a given namespace and labels selector
	var pods corev1.PodList
	err := client.List(context.TODO(), &pods,
		&clipkg.ListOptions{Namespace: check.NamespacedName.Namespace, LabelSelector: check.LabelSelector})
	if err != nil {
		log.Errorf("Failed listing pods in namespace %s: %v", check.NamespacedName.Namespace, err)
		return false
	}

	// If no pods found log a progress message and return
	if len(pods.Items) == 0 {
		log.Progressf("Found no pods with matching labels selector %v for namespace %s", check.LabelSelector, check.NamespacedName.Namespace)
		return true
	}

	// Loop through pods identifying pods that are using the latest replicaset revision
	var savedPods []corev1.Pod
	var savedPodTemplateHash string
	var savedRevision int
	var rsName string
	for _, pod := range pods.Items {
		// Log error and return if the pod-template-hash label is not found.  This should never happen.
		if _, ok := pod.Labels[podTemplateHashLabel]; !ok {
			log.Errorf("Failed to find pod label [pod-template-hash] for pod %s/%s", pod.Namespace, pod.Name)
			return false
		}

		if pod.Labels[podTemplateHashLabel] == savedPodTemplateHash {
			savedPods = append(savedPods, pod)
			continue
		}

		// Get the replica set for the pod given the pod-template-hash
		var rs appsv1.ReplicaSet
		rsName = fmt.Sprintf("%s-%s", check.NamespacedName.Name, pod.Labels[podTemplateHashLabel])
		err := client.Get(context.TODO(), types.NamespacedName{Namespace: check.NamespacedName.Namespace, Name: rsName}, &rs)
		if err != nil {
			log.Errorf("Failed to get replicaset %s: %v", check.NamespacedName.Namespace, err)
			return false
		}

		// Log error and return if the deployment.kubernetes.io/revision annotation is not found.  This should never happen.
		if _, ok := rs.Annotations[deploymentRevisionAnnotation]; !ok {
			log.Errorf("Failed to find pod annotation [deployment.kubernetes.io/revision] for pod %s/%s", pod.Namespace, pod.Name)
			return false
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
	var podsReady int32 = 0
	for _, pod := range savedPods {
		// Check that init containers are ready
		for _, initContainerStatus := range pod.Status.InitContainerStatuses {
			if !initContainerStatus.Ready {
				log.Progressf("%s is waiting for %d pod(s) to be ready using latest revision", prefix, expectedReplicas)
				return false
			}
		}
		// Check that containers are ready
		for _, containerStatus := range pod.Status.ContainerStatuses {
			if !containerStatus.Ready {
				log.Progressf("%s is waiting for %d pod(s) to be ready using latest revision", prefix, expectedReplicas)
				return false
			}
		}

		podsReady++

		// No need to look at other pods if the expected replicas is ready
		if podsReady == expectedReplicas {
			break
		}
	}

	if podsReady < expectedReplicas {
		log.Progressf("Found %d pods with matching labels selector %v for namespace %s when at least %d pods were expected", len(pods.Items), check.LabelSelector, check.NamespacedName.Namespace, expectedReplicas)
		return false
	}

	return true
}
