// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"context"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func DoesLoggingSidecarExist(kubeconfigPath string, name types.NamespacedName, containerName string) (bool, error) {
	clientset, err := GetKubernetesClientsetForCluster(kubeconfigPath)
	if err != nil {
		Log(Error, fmt.Sprintf("Could not get the clientset from the kubeconfig: %v", err))
		return false, err
	}
	appPod, err := clientset.CoreV1().Pods(name.Namespace).Get(context.TODO(), name.Name, metav1.GetOptions{})
	if err != nil {
		Log(Error, fmt.Sprintf("Could not get the application pod from the given name and namespace: %v", err))
		return false, err
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
