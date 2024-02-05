// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	encjson "encoding/json"
	"github.com/verrazzano/verrazzano/pkg/files"
	"go.uber.org/zap"
	"io"
	corev1 "k8s.io/api/core/v1"
	"os"
	"regexp"
	"sync"
)

// podListMap holds podLists which have been read in already
var podListMap = make(map[string]*corev1.PodList)
var podCacheMutex = &sync.Mutex{}

// FindProblematicPodFiles looks at the pods.json files in the cluster and will give a list of files
// if any have pods that are not Running or Succeeded.
func FindProblematicPodFiles(log *zap.SugaredLogger, clusterRoot string) (namespaces []string, podFiles []string, err error) {
	allPodFiles, err := files.GetMatchingFiles(clusterRoot, regexp.MustCompile("pods.json"))
	if err != nil {
		return nil, podFiles, err
	}

	if len(allPodFiles) == 0 {
		return nil, podFiles, nil
	}
	podFiles = make([]string, 0, len(allPodFiles))
	for _, podFile := range allPodFiles {
		podList, err := PodAnylsisGetPodList(log, podFile)
		if err != nil {
			log.Debugf("Failed to get the PodList for %s, skipping", podFile, err)
			continue
		}
		if podList == nil {
			log.Debugf("No PodList was returned, skipping")
			continue
		}

		// If we find any we flag the file as having problematic pods and move to the next file
		// this is just a quick scan to identify which files to drill into
		for _, pod := range podList.Items {
			if !IsPodProblematic(pod) {
				continue
			}
			log.Debugf("Problematic pods detected: %s", podFile)
			namespaces = append(namespaces, pod.Namespace)
			podFiles = append(podFiles, podFile)
			break
		}
	}
	return namespaces, podFiles, nil
}

// PodAnylsisGetPodList gets a pod list
func PodAnylsisGetPodList(log *zap.SugaredLogger, path string) (podList *corev1.PodList, err error) {
	// Check the cache first
	podList = getPodListIfPresent(path)
	if podList != nil {
		log.Debugf("Returning cached podList for %s", path)
		return podList, nil
	}

	// Not found in the cache, get it from the file
	file, err := os.Open(path)
	if err != nil {
		log.Debugf("file %s not found", path)
		return nil, err
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		log.Debugf("Failed reading Json file %s", path)
		return nil, err
	}
	err = encjson.Unmarshal(fileBytes, &podList)
	if err != nil {
		log.Debugf("Failed to unmarshal podList at %s", path)
		return nil, err
	}
	putPodListIfNotPresent(path, podList)
	return podList, nil
}

func getPodListIfPresent(path string) (podList *corev1.PodList) {
	podCacheMutex.Lock()
	podListTest := podListMap[path]
	if podListTest != nil {
		podList = podListTest
	}
	podCacheMutex.Unlock()
	return podList
}

// IsPodProblematic returns a boolean indicating whether a pod is deemed problematic or not
func IsPodProblematic(pod corev1.Pod) bool {
	// If it isn't Running or Succeeded it is problematic
	if pod.Status.Phase == corev1.PodRunning ||
		pod.Status.Phase == corev1.PodSucceeded {
		// The Pod indicates it is Running/Succeeded, check if there are containers that are not ready
		return !IsPodReadyOrCompleted(pod.Status)
	}
	return true
}

// IsPodReadyOrCompleted will return true if the Pod has containers that are neither Ready nor Completed
func IsPodReadyOrCompleted(podStatus corev1.PodStatus) bool {
	for _, containerStatus := range podStatus.ContainerStatuses {
		state := containerStatus.State
		if state.Terminated != nil && state.Terminated.Reason != "Completed" {
			return false
		}
		if state.Running != nil && !containerStatus.Ready {
			return false
		}
		if state.Waiting != nil {
			return false
		}
	}
	return true
}

func putPodListIfNotPresent(path string, podList *corev1.PodList) {
	podCacheMutex.Lock()
	podListInMap := podListMap[path]
	if podListInMap == nil {
		podListMap[path] = podList
	}
	podCacheMutex.Unlock()
}
