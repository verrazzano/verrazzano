// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Package cluster handles cluster analysis
package cluster

import (
	encjson "encoding/json"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/files"
	"go.uber.org/zap"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	"os"
	"sync"
)

var eventListMap = make(map[string]*corev1.EventList)
var eventCacheMutex = &sync.Mutex{}

// GetEventList gets an event list
func GetEventList(log *zap.SugaredLogger, path string) (eventList *corev1.EventList, err error) {
	// Check the cache first
	eventList = getEventListIfPresent(path)
	if eventList != nil {
		log.Debugf("Returning cached eventList for %s", path)
		return eventList, nil
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
	err = encjson.Unmarshal(fileBytes, &eventList)
	if err != nil {
		log.Debugf("Failed to unmarshal eventList at %s", path)
		return nil, err
	}
	putEventListIfNotPresent(path, eventList)
	return eventList, nil
}

// Need to add an optional time range to the get events, that will allow us to find events related to
// pods/services/etc that happened only within a given time range

// GetEventsRelatedToPod gets events related to a pod
func GetEventsRelatedToPod(log *zap.SugaredLogger, clusterRoot string, pod corev1.Pod, timeRange *files.TimeRange) (podEvents []corev1.Event, err error) {
	allEvents, err := GetEventList(log, files.FindFileInNamespace(clusterRoot, pod.ObjectMeta.Namespace, "events.json"))
	if err != nil {
		return nil, err
	}
	if allEvents == nil || len(allEvents.Items) == 0 {
		return nil, nil
	}
	podEvents = make([]corev1.Event, 0, 1)
	for _, event := range allEvents.Items {
		if event.InvolvedObject.Kind == "Pod" &&
			event.InvolvedObject.Namespace == pod.ObjectMeta.Namespace &&
			event.InvolvedObject.Name == pod.ObjectMeta.Name {
			podEvents = append(podEvents, event)
		}
	}
	return podEvents, nil
}

// GetEventsRelatedToService gets events related to a service
func GetEventsRelatedToService(log *zap.SugaredLogger, clusterRoot string, service corev1.Service, timeRange *files.TimeRange) (serviceEvents []corev1.Event, err error) {
	log.Debugf("GetEventsRelatedToService called for %s in namespace ", service.ObjectMeta.Name, service.ObjectMeta.Namespace)
	allEvents, err := GetEventList(log, files.FindFileInNamespace(clusterRoot, service.ObjectMeta.Namespace, "events.json"))
	if err != nil {
		return nil, err
	}
	if allEvents == nil || len(allEvents.Items) == 0 {
		log.Debugf("GetEventsRelatedToService: No events found")
		return nil, nil
	}
	serviceEvents = make([]corev1.Event, 0, 1)
	for _, event := range allEvents.Items {
		log.Debugf("Checking event involved object kind: %s, namespace: %s, name: %s", event.InvolvedObject.Kind, event.InvolvedObject.Namespace, event.InvolvedObject.Name)
		if event.InvolvedObject.Kind == "Service" &&
			event.InvolvedObject.Namespace == service.ObjectMeta.Namespace &&
			event.InvolvedObject.Name == service.ObjectMeta.Name {
			log.Debugf("event matched service")
			serviceEvents = append(serviceEvents, event)
		}
	}
	return serviceEvents, nil
}

func getEventListIfPresent(path string) (eventList *corev1.EventList) {
	eventCacheMutex.Lock()
	eventListTest := eventListMap[path]
	if eventListTest != nil {
		eventList = eventListTest
	}
	eventCacheMutex.Unlock()
	return eventList
}

func putEventListIfNotPresent(path string, eventList *corev1.EventList) {
	eventCacheMutex.Lock()
	eventListInMap := eventListMap[path]
	if eventListInMap == nil {
		eventListMap[path] = eventList
	}
	eventCacheMutex.Unlock()
}
