// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Package cluster handles cluster analysis
package cluster

import (
	encjson "encoding/json"
	"go.uber.org/zap"
	"io/ioutil"
	appsv1 "k8s.io/api/apps/v1"
	"os"
	"sync"
)

var daemonSetListMap = make(map[string]*appsv1.DaemonSetList)
var daemonSetCacheMutex = &sync.Mutex{}

// GetDaemonSetList gets an daemonSet list
func GetDaemonSetList(log *zap.SugaredLogger, path string) (daemonSetList *appsv1.DaemonSetList, err error) {
	// Check the cache first
	daemonSetList = getDaemonSetListIfPresent(path)
	if daemonSetList != nil {
		log.Debugf("Returning cached daemonSetList for %s", path)
		return daemonSetList, nil
	}

	// Not found in the cache, get it from the file
	file, err := os.Open(path)
	if err != nil {
		log.Debugf("file %s not found", path)
		return nil, err
	}
	defer file.Close()

	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		log.Debugf("Failed reading Json file %s", path)
		return nil, err
	}
	err = encjson.Unmarshal(fileBytes, &daemonSetList)
	if err != nil {
		log.Debugf("Failed to unmarshal daemonSetList at %s", path)
		return nil, err
	}
	putDaemonSetListIfNotPresent(path, daemonSetList)
	return daemonSetList, nil
}

func getDaemonSetListIfPresent(path string) (daemonSetList *appsv1.DaemonSetList) {
	daemonSetCacheMutex.Lock()
	daemonSetListTest := daemonSetListMap[path]
	if daemonSetListTest != nil {
		daemonSetList = daemonSetListTest
	}
	daemonSetCacheMutex.Unlock()
	return daemonSetList
}

func putDaemonSetListIfNotPresent(path string, daemonSetList *appsv1.DaemonSetList) {
	daemonSetCacheMutex.Lock()
	daemonSetListInMap := daemonSetListMap[path]
	if daemonSetListInMap == nil {
		daemonSetListMap[path] = daemonSetList
	}
	daemonSetCacheMutex.Unlock()
}
