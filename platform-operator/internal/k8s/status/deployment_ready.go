// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package status

import (
	"context"
	"fmt"
	pkgConstants "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
)

var veleroPodSelector = &metav1.LabelSelector{
	MatchLabels: map[string]string{
		"name": pkgConstants.Velero,
	},
}

func DeploymentsReadyBySelectors(log vzlog.VerrazzanoLogger, client clipkg.Client, expectedReplicas int32, prefix string, opts ...clipkg.ListOption) bool {
	deploymentList := &appsv1.DeploymentList{}
	if err := client.List(context.TODO(), deploymentList, opts...); err != nil {
		log.Errorf("%s failed listing deployments for selectors %v: %v", prefix, opts, err)
		return false
	}
	if deploymentList.Items == nil || len(deploymentList.Items) < 1 {
		log.Errorf("%s is waiting for deployments matching selector %s to exist", prefix, opts)
		return false
	}
	for idx := range deploymentList.Items {
		deployment := &deploymentList.Items[idx]
		namespacedName := types.NamespacedName{
			Namespace: deployment.Namespace,
			Name:      deployment.Name,
		}
		if !deploymentFullyReady(log, client, deployment, namespacedName, expectedReplicas, prefix) {
			return false
		}
	}
	return true
}

// DeploymentsAreReady check that the named deployments have the minimum number of specified replicas ready and available
func DeploymentsAreReady(log vzlog.VerrazzanoLogger, client clipkg.Client, namespacedNames []types.NamespacedName, expectedReplicas int32, prefix string) bool {
	for _, namespacedName := range namespacedNames {
		deployment := appsv1.Deployment{}
		if err := client.Get(context.TODO(), namespacedName, &deployment); err != nil {
			if errors.IsNotFound(err) {
				log.Progressf("%s is waiting for deployment %v to exist", prefix, namespacedName)
				return false
			}
			log.Errorf("%s failed getting deployment %v: %v", prefix, namespacedName, err)
			return false
		}
		if !deploymentFullyReady(log, client, &deployment, namespacedName, expectedReplicas, prefix) {
			return false
		}
	}
	return true
}

// DoDeploymentsExist checks if the named deployments exist
func DoDeploymentsExist(log vzlog.VerrazzanoLogger, client clipkg.Client, namespacedNames []types.NamespacedName, _ int32, prefix string) bool {
	for _, namespacedName := range namespacedNames {
		deployment := appsv1.Deployment{}
		if err := client.Get(context.TODO(), namespacedName, &deployment); err != nil {
			if errors.IsNotFound(err) {
				log.Progressf("%s is waiting for deployment %v to exist", prefix, namespacedName)
				return false
			}
			log.Errorf("%s failed getting deployment %v: %v", prefix, namespacedName, err)
			return false
		}
	}
	return true
}

func deploymentFullyReady(log vzlog.VerrazzanoLogger, client clipkg.Client, deployment *appsv1.Deployment, namespacedName types.NamespacedName, expectedReplicas int32, prefix string) bool {
	if deployment.Status.UpdatedReplicas < expectedReplicas {
		log.Progressf("%s is waiting for deployment %s replicas to be %v. Current updated replicas is %v", prefix, namespacedName,
			expectedReplicas, deployment.Status.UpdatedReplicas)
		return false
	}
	if deployment.Status.AvailableReplicas < expectedReplicas {
		log.Progressf("%s is waiting for deployment %s replicas to be %v. Current available replicas is %v", prefix, namespacedName,
			expectedReplicas, deployment.Status.AvailableReplicas)
		return false
	}

	// Velero install deploys a daemonset and deployment with common labels. The labels need to be adjusted so the pod fetch logic works
	// as expected
	podSelector := deployment.Spec.Selector
	if namespacedName.Namespace == constants.VeleroNameSpace && namespacedName.Name == pkgConstants.Velero {
		podSelector = veleroPodSelector
	}

	if !podsReadyDeployment(log, client, namespacedName, podSelector, expectedReplicas, prefix) {
		return false
	}
	log.Oncef("%s has enough replicas for deployment %v", prefix, namespacedName)
	return true
}

// podsReadyDeployment checks for an expected number of pods to be using the latest replicaset revision and are
// running and ready
func podsReadyDeployment(log vzlog.VerrazzanoLogger, client clipkg.Client, namespacedName types.NamespacedName, selector *metav1.LabelSelector, expectedReplicas int32, prefix string) bool {
	// Get a list of pods for a given namespace and labels selector
	pods := getPodsList(log, client, namespacedName, selector)
	if pods == nil {
		return false
	}

	// If no pods found log a progress message and return
	if len(pods.Items) == 0 {
		log.Progressf("Found no pods with matching labels selector %v for namespace %s", selector, namespacedName.Namespace)
		return false
	}

	// Loop through pods identifying pods that are using the latest replicaset revision
	var savedPods []corev1.Pod
	var savedPodTemplateHash string
	var savedRevision int
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

		// Get the replica set for the pod given the pod-template-hash label
		var rs appsv1.ReplicaSet
		rsName := fmt.Sprintf("%s-%s", namespacedName.Name, pod.Labels[podTemplateHashLabel])
		err := client.Get(context.TODO(), types.NamespacedName{Namespace: namespacedName.Namespace, Name: rsName}, &rs)
		if err != nil {
			log.Errorf("Failed to get replicaset %s: %v", namespacedName, err)
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
	podsReady, success := ensurePodsAreReady(log, savedPods, expectedReplicas, prefix)
	if !success {
		return false
	}

	if podsReady < expectedReplicas {
		log.Progressf("%s is waiting for deployment %s pods to be %v. Current available pods are %v", prefix, namespacedName,
			expectedReplicas, podsReady)
		return false
	}

	return true
}
