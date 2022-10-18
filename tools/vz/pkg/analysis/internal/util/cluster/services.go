// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Package cluster handles cluster analysis
package cluster

import (
	encjson "encoding/json"
	"go.uber.org/zap"
	"io"
	corev1 "k8s.io/api/core/v1"
	"os"
	"sync"
)

// serviceListMap holds serviceLists which have been read in already
var serviceListMap = make(map[string]*corev1.ServiceList)
var serviceCacheMutex = &sync.Mutex{}

// GetServiceList gets a service list
func GetServiceList(log *zap.SugaredLogger, path string) (serviceList *corev1.ServiceList, err error) {
	// Check the cache first
	serviceList = getServiceListIfPresent(path)
	if serviceList != nil {
		log.Debugf("Returning cached serviceList for %s", path)
		return serviceList, nil
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
	err = encjson.Unmarshal(fileBytes, &serviceList)
	if err != nil {
		log.Debugf("Failed to unmarshal serviceList at %s", path)
		return nil, err
	}
	putServiceListIfNotPresent(path, serviceList)
	return serviceList, nil
}

func getServiceListIfPresent(path string) (serviceList *corev1.ServiceList) {
	serviceCacheMutex.Lock()
	serviceListTest := serviceListMap[path]
	if serviceListTest != nil {
		serviceList = serviceListTest
	}
	serviceCacheMutex.Unlock()
	return serviceList
}

func putServiceListIfNotPresent(path string, serviceList *corev1.ServiceList) {
	serviceCacheMutex.Lock()
	serviceListInMap := serviceListMap[path]
	if serviceListInMap == nil {
		serviceListMap[path] = serviceList
	}
	serviceCacheMutex.Unlock()
}
