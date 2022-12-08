// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Package cluster handles cluster analysis
package cluster

import (
	encjson "encoding/json"
	"fmt"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/files"
	"go.uber.org/zap"
	"io"
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

	fileBytes, err := io.ReadAll(file)
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

// TODO: Need to add an optional time range to the get events, that will allow us to find events related to
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
	log.Debugf("GetEventsRelatedToService called for %s in namespace %s", service.ObjectMeta.Name, service.ObjectMeta.Namespace)
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

// GetEventsRelatedToComponentNamespace gets events related to a component namespace
func GetEventsRelatedToComponentNamespace(log *zap.SugaredLogger, clusterRoot string, componentNamespace string, timeRange *files.TimeRange) (componentEvents []corev1.Event, err error) {
	log.Debugf("GetEventsRelatedToComponentNs called for component in namespace %s", componentNamespace)

	podFile := files.FindFileInNamespace(clusterRoot, componentNamespace, podsJSON)
	podList, err := GetPodList(log, podFile)
	if err != nil {
		log.Debugf("Failed to get the list of pods for the given pod file %s, skipping", podFile)
	}
	if podList == nil {
		log.Debugf("No pod was returned, skipping")
		return nil, nil
	}

	for _, pod := range podList.Items {
		eventList, err := GetEventsRelatedToPod(log, clusterRoot, pod, nil)
		if err != nil {
			log.Debugf("Failed to get events for %s pod in namespace %s", pod.Name, pod.Namespace)
			continue
		}
		componentEvents = append(componentEvents, eventList...)
	}
	return componentEvents, nil
}

// CheckEventsForWarnings goes through the events list for a specific Component/Service/Pod
// and checks if there exists an event with type Warning and returns the corresponding reason and message
// as an additional supporting data
func CheckEventsForWarnings(log *zap.SugaredLogger, events []corev1.Event, timeRange *files.TimeRange) (message []string, err error) {
	log.Debugf("Searching the events list for any additional support data")
	var finalEventList []corev1.Event
	for _, event := range events {
		foundInvolvedObject := false
		for index, previousEvent := range finalEventList {
			// If we already iterated through an event for the same involved object with type Warning
			// And that event occurred before this current event, replace it accordingly
			if IsInvolvedObjectSame(event, previousEvent) && event.LastTimestamp.Time.After(previousEvent.LastTimestamp.Time) {
				foundInvolvedObject = true
				// Remove the previous event from the final list
				finalEventList = append(finalEventList[:index], finalEventList[index+1:]...)
				if event.Type == "Warning" {
					finalEventList = append(finalEventList, event)
				}
				break
			}
		}
		// No previous event for this specific involved object
		// If the type is warning, add to the finalEventList
		if !foundInvolvedObject && event.Type == "Warning" {
			finalEventList = append(finalEventList, event)
		}
	}

	for _, event := range finalEventList {
		message = append(message, fmt.Sprintf("Reosurce: %s %s, Namespace: %s, Reason: %s, Message: %s", event.InvolvedObject.Kind, event.InvolvedObject.Name, event.InvolvedObject.Namespace, event.Reason, event.Message))
	}

	return message, nil
}

func IsInvolvedObjectSame(eventA corev1.Event, eventB corev1.Event) bool {
	return eventA.InvolvedObject.Kind == eventB.InvolvedObject.Kind &&
		(eventA.InvolvedObject.Name == eventB.InvolvedObject.Name) &&
		eventA.InvolvedObject.Namespace == eventB.InvolvedObject.Namespace
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
