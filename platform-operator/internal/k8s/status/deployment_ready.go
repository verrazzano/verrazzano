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

type PodReadyCheck struct {
	NamespacedName types.NamespacedName
	LabelSelector  labels.Selector
}

// DeploymentsReady Check that the named deployments have the minimum number of specified replicas ready and available
func DeploymentsReady(log vzlog.VerrazzanoLogger, client clipkg.Client, deployments []types.NamespacedName, expectedReplicas int32, prefix string) bool {
	for _, namespacedName := range deployments {
		deployment := appsv1.Deployment{}
		if err := client.Get(context.TODO(), namespacedName, &deployment); err != nil {
			if errors.IsNotFound(err) {
				log.Progressf("%s is waiting for deployment %v to exist", prefix, namespacedName)
				return false
			}
			log.Errorf("Failed getting deployment %v: %v", namespacedName, err)
			return false
		}
		if deployment.Status.AvailableReplicas < expectedReplicas {
			log.Progressf("%s is waiting for deployment %s replicas to be %v. Current available replicas is %v", prefix, namespacedName,
				expectedReplicas, deployment.Status.AvailableReplicas)
			return false
		}
		if deployment.Status.UpdatedReplicas < expectedReplicas {
			log.Progressf("%s is waiting for deployment %s replicas to be %v. Current updated replicas is %v", prefix, namespacedName,
				expectedReplicas, deployment.Status.UpdatedReplicas)
			return false
		}
	}
	log.Oncef("%s has enough replicas for deployments %v", prefix, deployments)
	return true
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
			log.Errorf("Failed getting deployment %v: %v", check.NamespacedName, err)
			return false
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

func podsReadyWithLatestRevision(log vzlog.VerrazzanoLogger, client clipkg.Client, check PodReadyCheck, expectedReplicas int32, prefix string) bool {
	// Get a list of pods for a given namespace and labels selector
	var pods corev1.PodList
	err := client.List(context.TODO(), &pods,
		&clipkg.ListOptions{Namespace: check.NamespacedName.Namespace, LabelSelector: check.LabelSelector})
	if err != nil {
		log.Errorf("Failed listing pods in namespace %s: %v", check.NamespacedName.Namespace, err)
		return false
	}

	// If no pods found log a message and return
	if len(pods.Items) == 0 {
		log.Progressf("%s found no pods with matching labels selector %v", prefix, check.LabelSelector)
	}

	// Loop through pods identifying pods that are using the latest replicaset revision
	var savedPods []corev1.Pod
	var savedPodTemplateHash string
	var savedRevision int
	for _, pod := range pods.Items {
		// TODO:error if label not found
		if pod.Labels["pod-template-hash"] == savedPodTemplateHash {
			savedPods = append(savedPods, pod)
			continue
		}
		// Get the replica set for the pod given the pod-template-hash
		var rs appsv1.ReplicaSet
		rsName := fmt.Sprintf("%s-%s", check.NamespacedName.Name, pod.Labels["pod-template-hash"])
		err := client.Get(context.TODO(), types.NamespacedName{Namespace: check.NamespacedName.Namespace, Name: rsName}, &rs)
		if err != nil {
			log.Errorf("Failed replicaset to get %s: %v", check.NamespacedName.Namespace, err)
			return false
		}
		// TODO:error if label not found
		revision, _ := strconv.Atoi(rs.Labels["deployment.kubernetes.io/revision"])
		if revision > savedRevision {
			savedRevision = revision
			savedPodTemplateHash = pod.Labels["pod-template-hash"]
			savedPods = []corev1.Pod{}
			savedPods = append(savedPods, pod)
		}
	}

	var podsReady int32 = 0
	for _, pod := range savedPods {
		// Check that init containers are ready
		for _, initContainerStatus := range pod.Status.InitContainerStatuses {
			if !initContainerStatus.Ready {
				log.Progressf("%s is waiting for %d pod(s) to be ready using latest replicaset", prefix, expectedReplicas)
				return false
			}
		}
		// Check that containers are ready
		for _, containerStatus := range pod.Status.ContainerStatuses {
			if !containerStatus.Ready {
				log.Progressf("%s is waiting for %d pod(s) to be ready using latest replicaset", prefix, expectedReplicas)
				return false
			}
		}
		podsReady++

		// No need to look at other pods if the expected replicas is ready
		if podsReady == expectedReplicas {
			return true
		}
	}
	return true
}
