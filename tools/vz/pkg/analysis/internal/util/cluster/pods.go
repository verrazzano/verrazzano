// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Package cluster handles cluster analysis
package cluster

import (
	"bytes"
	encjson "encoding/json"
	"fmt"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/files"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/report"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	"go.uber.org/zap"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	"os"
	"regexp"
	"strings"
	"sync"
	"text/template"
)

// podListMap holds podLists which have been read in already
var podListMap = make(map[string]*corev1.PodList)
var podCacheMutex = &sync.Mutex{}

var dockerPullRateLimitRe = regexp.MustCompile(`.*You have reached your pull rate limit.*`)
var dockerNameUnknownRe = regexp.MustCompile(`.*name unknown.*`)
var dockerNotFoundRe = regexp.MustCompile(`.*not found.*`)
var dockerServiceUnavailableRe = regexp.MustCompile(`.*Service Unavailable.*`)

// TODO: "Verrazzano Uninstall Pod Issue":    AnalyzeVerrazzanoUninstallIssue,
var podAnalysisFunctions = map[string]func(log *zap.SugaredLogger, directory string, podFile string, pod corev1.Pod, issueReporter *report.IssueReporter) (err error){
	"Pod Container Related Issues":        podContainerIssues,
	"Pod Status Condition Related Issues": podStatusConditionIssues,
}

// AnalyzePodIssues analyzes pod issues. It starts by scanning for problem pod phases
// in the cluster and drill down from there.
func AnalyzePodIssues(log *zap.SugaredLogger, clusterRoot string) (err error) {
	log.Debugf("PodIssues called for %s", clusterRoot)

	// Do a quick scan to find pods.json which have Pod which are not in a good state
	podFiles, err := findProblematicPodFiles(log, clusterRoot)
	if err != nil {
		return err
	}
	totalFound := 0
	for _, podFile := range podFiles {
		found, err := analyzePods(log, clusterRoot, podFile)
		totalFound += found
		if err != nil {
			log.Errorf("Failed during analyze Pods for cluster: %s, pods: %s ", clusterRoot, podFile, err)
			return err
		}
	}

	// If no issues were reported, but there were problem pods, we need to beef up the detection
	// so report an issue (high confidence, low impact)
	if totalFound == 0 && len(podFiles) > 0 {
		reportProblemPodsNoIssues(log, clusterRoot, podFiles)
	}

	return nil
}

func analyzePods(log *zap.SugaredLogger, clusterRoot string, podFile string) (reported int, err error) {
	log.Debugf("analyzePods called with %s", podFile)
	podList, err := GetPodList(log, podFile)
	if err != nil {
		log.Debugf("Failed to get the PodList for %s", podFile, err)
		return 0, err
	}
	if podList == nil {
		log.Debugf("No PodList was returned, skipping")
		return 0, nil
	}

	var issueReporter = report.IssueReporter{
		PendingIssues: make(map[string]report.Issue),
	}
	for _, pod := range podList.Items {
		if !IsPodProblematic(pod) {
			continue
		}

		// Call the pod analysis functions
		for functionName, function := range podAnalysisFunctions {
			err := function(log, clusterRoot, podFile, pod, &issueReporter)
			if err != nil {
				// Log the error and continue on
				log.Errorf("Error processing analysis function %s", functionName, err)
			}
		}
	}

	reported = len(issueReporter.PendingIssues)
	issueReporter.Contribute(log, clusterRoot)
	return reported, nil
}

// IsPodNotReadyNorCompleted will return true if the Pod has containers that are neither Ready nor Completed
// TODO: Extend for transition time correlation (ie: change from bool to struct)
func IsPodNotReadyNorCompleted(podStatus corev1.PodStatus) bool {
	for _, containerStatus := range podStatus.ContainerStatuses {
		state := containerStatus.State
		if state.Terminated != nil && state.Terminated.Reason != "Completed" {
			return true
		}
		if state.Running != nil && !containerStatus.Ready {
			return true
		}
		if state.Waiting != nil {
			return true
		}
	}
	return false
}

// This is evolving as we add more cases in podContainerIssues
//
//   One thing that switching to this drill-down model makes harder to do is track overarching
//   issues that are related. I have an IssueReporter that is being passed along and will
//   consolidate the same KnownIssue types to help with similar issues.
//
//   Note that this is not showing it here as the current analysis only is using the IssueReporter
//   but analysis code is free to use the NewKnown* helpers or form fully custom issues and Contribute
//   those directly to the report.Contribute* helpers
func podContainerIssues(log *zap.SugaredLogger, clusterRoot string, podFile string, pod corev1.Pod, issueReporter *report.IssueReporter) (err error) {
	log.Debugf("podContainerIssues analysis called for cluster: %s, ns: %s, pod: %s", clusterRoot, pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)
	podEvents, err := GetEventsRelatedToPod(log, clusterRoot, pod, nil)
	if err != nil {
		log.Debugf("Failed to get events related to ns: %s, pod: %s", pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)
	}
	// TODO: We can get duplicated event drilldown messages if the initcontainers and containers are both impacted similarly
	//       Since we contribute it to the IssueReporter, thinking maybe can handle de-duplication under the covers to allow
	//       discrete analysis to be handled various ways, though could rethink the approach here as well to reduce the need too.
	if len(pod.Status.InitContainerStatuses) > 0 {
		for _, initContainerStatus := range pod.Status.InitContainerStatuses {
			if initContainerStatus.State.Waiting != nil {
				if initContainerStatus.State.Waiting.Reason == "ImagePullBackOff" {
					handleImagePullBackOff(log, clusterRoot, podFile, pod, podEvents, initContainerStatus.Image,
						fmt.Sprintf("Namespace %s, Pod %s, InitContainer %s, Message %s",
							pod.ObjectMeta.Namespace, pod.ObjectMeta.Name, initContainerStatus.Name, initContainerStatus.State.Waiting.Message),
						issueReporter)
				}
			}
		}
	}
	if len(pod.Status.ContainerStatuses) > 0 {
		for _, containerStatus := range pod.Status.ContainerStatuses {
			if containerStatus.State.Waiting != nil {
				if containerStatus.State.Waiting.Reason == "ImagePullBackOff" {
					handleImagePullBackOff(log, clusterRoot, podFile, pod, podEvents, containerStatus.Image,
						fmt.Sprintf("Namespace %s, Pod %s, Container %s, Message %s",
							pod.ObjectMeta.Namespace, pod.ObjectMeta.Name, containerStatus.Name, containerStatus.State.Waiting.Message),
						issueReporter)
				}
			}
		}
	}
	return nil
}

func handleImagePullBackOff(log *zap.SugaredLogger, clusterRoot string, podFile string, pod corev1.Pod, podEvents []corev1.Event,
	image string, initialMessage string, issueReporter *report.IssueReporter) {
	messages := make(StringSlice, 1)
	messages[0] = initialMessage
	messages.addMessages(drillIntoEventsForImagePullIssue(log, pod, image, podEvents))

	var files []string
	if helpers.GetIsLiveCluster() {
		files = []string{report.GetRelatedPodMessage(pod.ObjectMeta.Name, pod.ObjectMeta.Namespace)}
	} else {
		files = []string{podFile}
	}

	reported := 0
	for _, message := range messages {
		if dockerPullRateLimitRe.MatchString(message) {
			issueReporter.AddKnownIssueMessagesFiles(
				report.ImagePullRateLimit,
				clusterRoot,
				messages,
				files,
			)
			reported++
		} else if dockerServiceUnavailableRe.MatchString(message) {
			issueReporter.AddKnownIssueMessagesFiles(
				report.ImagePullService,
				clusterRoot,
				messages,
				files,
			)
			reported++
		} else if dockerNameUnknownRe.MatchString(message) || dockerNotFoundRe.MatchString(message) {
			issueReporter.AddKnownIssueMessagesFiles(
				report.ImagePullNotFound,
				clusterRoot,
				messages,
				files,
			)
			reported++
		}
	}

	// If we didn't detect more specific issues here, fall back to the general ImagePullBackOff
	if reported == 0 {
		issueReporter.AddKnownIssueMessagesFiles(
			report.ImagePullBackOff,
			clusterRoot,
			messages,
			files,
		)
	}
}

func podStatusConditionIssues(log *zap.SugaredLogger, clusterRoot string, podFile string, pod corev1.Pod, issueReporter *report.IssueReporter) (err error) {
	log.Debugf("MemoryIssues called for %s", clusterRoot)

	if len(pod.Status.Conditions) > 0 {
		messages := make([]string, 0)
		for _, condition := range pod.Status.Conditions {
			if strings.Contains(condition.Message, "Insufficient memory") {
				messages = append(messages, fmt.Sprintf("Namespace %s, Pod %s, Status %s, Reason %s, Message %s",
					pod.ObjectMeta.Namespace, pod.ObjectMeta.Name, condition.Status, condition.Reason, condition.Message))
			}
		}
		if len(messages) > 0 {
			var files []string
			if helpers.GetIsLiveCluster() {
				files = []string{report.GetRelatedPodMessage(pod.ObjectMeta.Name, pod.ObjectMeta.Namespace)}
			} else {
				files = []string{podFile}
			}
			issueReporter.AddKnownIssueMessagesFiles(report.InsufficientMemory, clusterRoot, messages, files)
		}
	}
	return nil
}

// StringSlice is a string slice
type StringSlice []string

func (messages *StringSlice) addMessages(newMessages []string, err error) (errOut error) {
	if err != nil {
		errOut = err
		return errOut
	}
	if len(newMessages) > 0 {
		*messages = append(*messages, newMessages...)
	}
	return nil
}

// This is WIP, initially it will report more specific cause info (like auth, timeout, etc...), but we may want
// to have different known issue types rather than reporting in messages as the runbooks to look at may be different
// so this is evolving...
func drillIntoEventsForImagePullIssue(log *zap.SugaredLogger, pod corev1.Pod, imageName string, eventList []corev1.Event) (messages []string, err error) {
	// This is handed the events that are associated with the Pod that has containers/initContainers that had image pull issues
	// So it will look at what events are found, these may glean more info on the specific cause to help narrow
	// it further
	for _, event := range eventList {
		// TODO: Discern whether the event is relevant or not, we likely will need more info supplied in to correlate
		// whether the event really is related to the issue being drilled into or not, but starting off just dumping out
		// what is there first. Hoping that this can be a more general drilldown here rather than just specific to ImagePulls
		// Ie: drill into events related to Pod/container issue.
		// We may want to add in a "Reason" concept here as well. Ie: the issue is Image pull, and we were able to
		// discern more about the reason that happened, so we can return back up a Reason that can be useful in the
		// action handling to focus on the correct steps, instead of having an entirely separate issue type to handle that.
		// (or just have a more specific issue type, but need to think about it as we are setting the basic patterns
		// here in general). Need to think about it though as it will affect how we handle runbooks as well.
		log.Debugf("Drilldown event reason: %s, message: %s\n", event.Reason, event.Message)
		if event.Reason == "Failed" && strings.Contains(event.Message, imageName) {
			// TBD: We need a better mechanism at this level than just the messages. It can add other types of supporting
			// data to contribute as well here (related files, etc...)
			messages = append(messages, event.Message)
		}

	}
	return messages, nil
}

// GetPodList gets a pod list
func GetPodList(log *zap.SugaredLogger, path string) (podList *corev1.PodList, err error) {
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

	fileBytes, err := ioutil.ReadAll(file)
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

// IsPodProblematic returns a boolean indicating whether a pod is deemed problematic or not
func IsPodProblematic(pod corev1.Pod) bool {
	// If it isn't Running or Succeeded it is problematic
	if pod.Status.Phase == corev1.PodRunning ||
		pod.Status.Phase == corev1.PodSucceeded {
		// The Pod indicates it is Running/Succeeded, check if there are containers that are not ready
		return IsPodNotReadyNorCompleted(pod.Status)
	}
	return true
}

// IsPodPending returns a boolean indicating whether a pod is pending or not
func IsPodPending(pod corev1.Pod) bool {
	return pod.Status.Phase == corev1.PodPending
}

// This looks at the pods.json files in the cluster and will give a list of files
// if any have pods that are not Running or Succeeded.
func findProblematicPodFiles(log *zap.SugaredLogger, clusterRoot string) (podFiles []string, err error) {
	allPodFiles, err := files.GetMatchingFiles(log, clusterRoot, PodFilesMatchRe)
	if err != nil {
		return podFiles, err
	}

	if len(allPodFiles) == 0 {
		return podFiles, nil
	}
	podFiles = make([]string, 0, len(allPodFiles))
	for _, podFile := range allPodFiles {
		log.Debugf("Looking at pod file for problematic pods: %s", podFile)
		podList, err := GetPodList(log, podFile)
		if err != nil {
			log.Debugf("Failed to get the PodList for %s, skipping", podFile, err)
			continue
		}
		if podList == nil {
			log.Debugf("No PodList was returned, skipping")
			continue
		}

		// If we find any we flag the file as havin problematic pods and move to the next file
		// this is just a quick scan to identify which files to drill into
		for _, pod := range podList.Items {
			if !IsPodProblematic(pod) {
				continue
			}
			log.Debugf("Problematic pods detected: %s", podFile)
			podFiles = append(podFiles, podFile)
			break
		}
	}
	return podFiles, nil
}

func reportProblemPodsNoIssues(log *zap.SugaredLogger, clusterRoot string, podFiles []string) {
	messages := make([]string, 0, len(podFiles))
	matches := make([]files.TextMatch, 0, len(podFiles))
	problematicNotPending := 0
	pendingPodsSeen := 0
	for _, podFile := range podFiles {
		podList, err := GetPodList(log, podFile)
		if err != nil {
			log.Debugf("Failed to get the PodList for %s", podFile, err)
			continue
		}
		if podList == nil {
			log.Debugf("No PodList was returned, skipping")
			continue
		}
		for _, pod := range podList.Items {
			if !IsPodProblematic(pod) {
				continue
			}
			if pod.Status.Phase == corev1.PodFailed || pod.Status.Phase == corev1.PodUnknown {
				problematicNotPending++
				if len(pod.Status.Reason) > 0 || len(pod.Status.Message) > 0 {
					messages = append(messages, fmt.Sprintf("Namespace %s, Pod %s, Phase %s, Reason %s, Message %s",
						pod.ObjectMeta.Namespace, pod.ObjectMeta.Name, pod.Status.Phase, pod.Status.Reason, pod.Status.Message))
				}
			} else if pod.Status.Phase == corev1.PodPending {
				pendingPodsSeen++
			}

			for _, condition := range pod.Status.Conditions {
				message, err := podConditionMessage(pod.Name, pod.Namespace, condition)
				if err != nil {
					log.Debugf("Failed to create pod condition message: %v", err)
					continue
				}
				messages = append(messages, message)
			}
			// TODO: Time correlation for search

			fileName := files.FindPodLogFileName(clusterRoot, pod)
			matched, err := files.SearchFile(log, fileName, WideErrorSearchRe, nil)
			if err != nil {
				log.Debugf("Failed to search the logfile %s for the ns/pod %s/%s",
					files.FindPodLogFileName(clusterRoot, pod), pod.ObjectMeta.Namespace, pod.ObjectMeta.Name, err)
			}
			if len(matched) > 0 {
				for _, m := range matched {
					if helpers.GetIsLiveCluster() {
						m.FileName = report.GetRelatedLogFromPodMessage(fileName)
					}
					matches = append(matches, m)
				}
			}
		}
	}
	supportingData := make([]report.SupportData, 1)
	supportingData[0] = report.SupportData{
		Messages:    messages,
		TextMatches: matches,
	}
	// If all of the problematic pods were pending only, just report that, otherwise report them as problematic if some are
	// failing or unknown
	if pendingPodsSeen > 0 && problematicNotPending == 0 {
		report.ContributeIssue(log, report.NewKnownIssueSupportingData(report.PendingPods, clusterRoot, supportingData))
	} else {
		report.ContributeIssue(log, report.NewKnownIssueSupportingData(report.PodProblemsNotReported, clusterRoot, supportingData))
	}
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

func putPodListIfNotPresent(path string, podList *corev1.PodList) {
	podCacheMutex.Lock()
	podListInMap := podListMap[path]
	if podListInMap == nil {
		podListMap[path] = podList
	}
	podCacheMutex.Unlock()
}

const podConditionMessageTmpl = "Namespace {{.namespace}}, Pod {{.name}}{{- if .type }}, ConditionType {{.type}}{{- end}}{{- if .status}}, Status {{.status}}{{- end}}{{- if .reason}}, Reason {{.reason}}{{- end }}{{- if .message}}, Message {{.message}}{{- end }}"

func podConditionMessage(name, namespace string, condition corev1.PodCondition) (string, error) {
	vals := map[string]interface{}{}
	addKV := func(k, v string) {
		if v != "" {
			vals[k] = v
		}
	}
	addKV("message", condition.Message)
	addKV("reason", condition.Reason)
	addKV("status", string(condition.Status))
	addKV("type", string(condition.Type))
	vals["name"] = name
	vals["namespace"] = namespace
	tmpl, err := template.New("podConditionMessage").
		Parse(podConditionMessageTmpl)
	if err != nil {
		return "", err
	}

	buffer := &bytes.Buffer{}
	if err = tmpl.Execute(buffer, vals); err != nil {
		return "", err
	}
	return buffer.String(), nil
}
