// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"context"
	"fmt"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"strings"
)

func DoesLoggingSidecarExist(kubeconfigPath string, name types.NamespacedName, containerName string) (bool, error) {
	clientset, err := GetKubernetesClientsetForCluster(kubeconfigPath)
	if err != nil {
		Log(Error, fmt.Sprintf("Could not get the clientset from the kubeconfig: %v", err))
		return false, err
	}
	podList, err := clientset.CoreV1().Pods(name.Namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		Log(Error, fmt.Sprintf("Could not List the application pod from the given namespace: %v", err))
		return false, err
	}
	var appPod v1.Pod
	for _, pod := range podList.Items {
		if strings.HasPrefix(pod.Name, name.Name) {
			appPod = pod
		}
	}
	if appPod.Name == "" {
		Log(Error, fmt.Sprintf("Could not find the pod with the given name and namespace: %v", err))
		return false, nil
	}
	for _, container := range appPod.Spec.Containers {
		if container.Name == containerName {
			Log(Info, fmt.Sprintf("Container %v found", containerName))
			return true, nil
		}
	}
	Log(Info, fmt.Sprintf("Container was NOT found for the pod %v", name.Name))
	return false, nil
}
