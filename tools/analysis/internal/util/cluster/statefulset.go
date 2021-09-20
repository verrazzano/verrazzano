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

var statefulSetListMap = make(map[string]*appsv1.StatefulSetList)
var statefulSetCacheMutex = &sync.Mutex{}

// GetStatefulSetList gets an statefulSet list
func GetStatefulSetList(log *zap.SugaredLogger, path string) (statefulSetList *appsv1.StatefulSetList, err error) {
	// Check the cache first
	statefulSetList = getStatefulSetListIfPresent(path)
	if statefulSetList != nil {
		log.Debugf("Returning cached statefulSetList for %s", path)
		return statefulSetList, nil
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
	err = encjson.Unmarshal(fileBytes, &statefulSetList)
	if err != nil {
		log.Debugf("Failed to unmarshal statefulSetList at %s", path)
		return nil, err
	}
	putStatefulSetListIfNotPresent(path, statefulSetList)
	return statefulSetList, nil
}

func getStatefulSetListIfPresent(path string) (statefulSetList *appsv1.StatefulSetList) {
	statefulSetCacheMutex.Lock()
	statefulSetListTest := statefulSetListMap[path]
	if statefulSetListTest != nil {
		statefulSetList = statefulSetListTest
	}
	statefulSetCacheMutex.Unlock()
	return statefulSetList
}

func putStatefulSetListIfNotPresent(path string, statefulSetList *appsv1.StatefulSetList) {
	statefulSetCacheMutex.Lock()
	statefulSetListInMap := statefulSetListMap[path]
	if statefulSetListInMap == nil {
		statefulSetListMap[path] = statefulSetList
	}
	statefulSetCacheMutex.Unlock()
}
