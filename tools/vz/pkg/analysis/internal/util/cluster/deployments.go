// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Package cluster handles cluster analysis
package cluster

import (
	encjson "encoding/json"
	"go.uber.org/zap"
	"io"
	appsv1 "k8s.io/api/apps/v1"
	"os"
	"sync"
)

var deploymentListMap = make(map[string]*appsv1.DeploymentList)
var deploymentCacheMutex = &sync.Mutex{}

// GetDeploymentList gets an deployment list
func GetDeploymentList(log *zap.SugaredLogger, path string) (deploymentList *appsv1.DeploymentList, err error) {
	// Check the cache first
	deploymentList = getDeploymentListIfPresent(path)
	if deploymentList != nil {
		log.Debugf("Returning cached deploymentList for %s", path)
		return deploymentList, nil
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
	err = encjson.Unmarshal(fileBytes, &deploymentList)
	if err != nil {
		log.Debugf("Failed to unmarshal deploymentList at %s", path)
		return nil, err
	}
	putDeploymentListIfNotPresent(path, deploymentList)
	return deploymentList, nil
}

// IsDeploymentProblematic returns a boolean indicating whether a deployment is deemed problematic or not
func IsDeploymentProblematic(deployment *appsv1.Deployment) bool {
	// If we can't determine the status conditions, we skip it
	if len(deployment.Status.Conditions) == 0 {
		return false
	}
	// If any conditions aren't in Available, flag this deployment as problematic
	// Note that it could be progressing normally, but flagging it for now as it could be stuck
	for _, condition := range deployment.Status.Conditions {
		if condition.Type == appsv1.DeploymentAvailable {
			continue
		}
		return true
	}
	return false
}

// FindProblematicDeployments will find and return all deployments deemed problematic in the deploymentList
func FindProblematicDeployments(deploymentList *appsv1.DeploymentList) (deployments []appsv1.Deployment) {
	for i, deployment := range deploymentList.Items {
		if IsDeploymentProblematic(&deploymentList.Items[i]) {
			deployments = append(deployments, deployment)
		}
	}
	return deployments
}

func getDeploymentListIfPresent(path string) (deploymentList *appsv1.DeploymentList) {
	deploymentCacheMutex.Lock()
	deploymentListTest := deploymentListMap[path]
	if deploymentListTest != nil {
		deploymentList = deploymentListTest
	}
	deploymentCacheMutex.Unlock()
	return deploymentList
}

func putDeploymentListIfNotPresent(path string, deploymentList *appsv1.DeploymentList) {
	deploymentCacheMutex.Lock()
	deploymentListInMap := deploymentListMap[path]
	if deploymentListInMap == nil {
		deploymentListMap[path] = deploymentList
	}
	deploymentCacheMutex.Unlock()
}
