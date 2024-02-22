// Copyright (c) 2021, 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Package cluster handles cluster analysis
package cluster

import (
	"bytes"
	encjson "encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"
	"text/template"

	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/internal/util/files"
	utillog "github.com/verrazzano/verrazzano/tools/vz/pkg/internal/util/log"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/internal/util/report"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
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
	"Pod Deletion Issues":                 podDeletionIssues,
	"Pod Readiness Gates Issues":          podReadinessGateIssues,
}

// AnalyzePodIssues analyzes pod issues. It starts by scanning for problem pod phases
// in the cluster and drill down from there.
func AnalyzePodIssues(log *zap.SugaredLogger, clusterRoot string) (err error) {
	utillog.DebugfIfNotNil(log, "PodIssues called for %s", clusterRoot)

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
			utillog.ErrorfIfNotNil(log, "Failed during analyze Pods for cluster: %s, pods: %s ", clusterRoot, podFile, err)
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
	utillog.DebugfIfNotNil(log, "analyzePods called with %s", podFile)
	podList, err := GetPodList(log, podFile)
	if err != nil {
		utillog.DebugfIfNotNil(log, "Failed to get the PodList for %s", podFile, err)
		return 0, err
	}
	if podList == nil {
		utillog.DebugfIfNotNil(log, "No PodList was returned, skipping")
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
				utillog.ErrorfIfNotNil(log, "Error processing analysis function %s", functionName, err)
			}
		}
	}

	reported = len(issueReporter.PendingIssues)
	issueReporter.Contribute(log, clusterRoot)
	return reported, nil
}

// IsPodReadyOrCompleted will return true if the Pod has containers that are neither Ready nor Completed
// TODO: Extend for transition time correlation (ie: change from bool to struct)
func IsPodReadyOrCompleted(pod corev1.Pod) bool {
	podStatus := pod.Status
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
	if pod.DeletionTimestamp != nil {
		return false
	}
	return arePodReadinessGatesReady(pod)
}

func arePodReadinessGatesReady(pod corev1.Pod) bool {
	conditions := pod.Status.Conditions
	if len(conditions) == 0 {
		return false
	}
	readyCount := 0
	for _, condition := range conditions {
		for _, gate := range pod.Spec.ReadinessGates {
			if condition.Type == gate.ConditionType && condition.Status == corev1.ConditionTrue {
				readyCount++
				continue
			}
		}
	}
	// All readiness gates must be true
	return len(pod.Spec.ReadinessGates) == readyCount
}

// This is evolving as we add more cases in podContainerIssues
//
//	One thing that switching to this drill-down model makes harder to do is track overarching
//	issues that are related. I have an IssueReporter that is being passed along and will
//	consolidate the same KnownIssue types to help with similar issues.
//
//	Note that this is not showing it here as the current analysis only is using the IssueReporter
//	but analysis code is free to use the NewKnown* helpers or form fully custom issues and Contribute
//	those directly to the report.Contribute* helpers
func podContainerIssues(log *zap.SugaredLogger, clusterRoot string, podFile string, pod corev1.Pod, issueReporter *report.IssueReporter) (err error) {
	utillog.DebugfIfNotNil(log, "podContainerIssues analysis called for cluster: %s, ns: %s, pod: %s", clusterRoot, pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)
	podEvents, err := GetEventsRelatedToPod(log, clusterRoot, pod, nil)
	if err != nil {
		utillog.DebugfIfNotNil(log, "Failed to get events related to ns: %s, pod: %s", pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)
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

func registerIssues(messages map[string][]string, files []string, clusterRoot string, issueReporter *report.IssueReporter) {
	if len(messages[report.InsufficientMemory]) > 0 {
		issueReporter.AddKnownIssueMessagesFiles(report.InsufficientMemory, clusterRoot, messages[report.InsufficientMemory], files)
	}
	if len(messages[report.InsufficientCPU]) > 0 {
		issueReporter.AddKnownIssueMessagesFiles(report.InsufficientCPU, clusterRoot, messages[report.InsufficientCPU], files)
	}
}

func podStatusConditionIssues(log *zap.SugaredLogger, clusterRoot string, podFile string, pod corev1.Pod, issueReporter *report.IssueReporter) (err error) {
	utillog.DebugfIfNotNil(log, "Memory or CPU Issues called for %s", clusterRoot)

	if len(pod.Status.Conditions) > 0 {
		messages := make(map[string][]string)
		for _, condition := range pod.Status.Conditions {
			if strings.Contains(condition.Message, "Insufficient memory") {
				messages[report.InsufficientMemory] = append(messages[report.InsufficientMemory], fmt.Sprintf("Namespace %s, Pod %s, Status %s, Reason %s, Message %s",
					pod.ObjectMeta.Namespace, pod.ObjectMeta.Name, condition.Status, condition.Reason, condition.Message))
			}
			if strings.Contains(condition.Message, "Insufficient cpu") {
				messages[report.InsufficientCPU] = append(messages[report.InsufficientCPU], fmt.Sprintf("Namespace %s, Pod %s, Status %s, Reason %s, Message %s",
					pod.ObjectMeta.Namespace, pod.ObjectMeta.Name, condition.Status, condition.Reason, condition.Message))
			}
		}
		if len(messages) > 0 {
			files := []string{podFile}
			if helpers.GetIsLiveCluster() {
				files = []string{report.GetRelatedPodMessage(pod.ObjectMeta.Name, pod.ObjectMeta.Namespace)}
			}
			registerIssues(messages, files, clusterRoot, issueReporter)
		}
	}
	return nil
}

// podDeletionIssues reports an issue if a pod has been stuck deleting for 10 minutes or longer
func podDeletionIssues(log *zap.SugaredLogger, clusterRoot string, podFile string, pod corev1.Pod, issueReporter *report.IssueReporter) error {
	if pod.DeletionTimestamp == nil {
		return nil
	}
	timeOfCapture, err := files.GetTimeOfCapture(log, clusterRoot)
	if err != nil {
		return err
	}
	if timeOfCapture == nil {
		return nil
	}
	diff := timeOfCapture.Sub(pod.DeletionTimestamp.Time)
	if int(diff.Minutes()) < 10 {
		return nil
	}
	deletionMessage := "The pod " + pod.Name + " has spent " + fmt.Sprint(int(diff.Minutes())) + " minutes and " + fmt.Sprint(int(diff.Seconds())%60) + " seconds deleting"
	issueReporter.AddKnownIssueMessagesFiles(report.PodHangingOnDeletion, clusterRoot, []string{deletionMessage}, []string{podFile})

	return nil
}

func podReadinessGateIssues(log *zap.SugaredLogger, clusterRoot string, podFile string, pod corev1.Pod, issueReporter *report.IssueReporter) error {
	if arePodReadinessGatesReady(pod) {
		return nil
	}
	message := "The pod " + pod.Name + " is currently waiting for its readiness gates"
	issueReporter.AddKnownIssueMessagesFiles(report.PodWaitingOnReadinessGates, clusterRoot, []string{message}, []string{podFile})
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
		utillog.DebugfIfNotNil(log, "Drilldown event reason: %s, message: %s\n", event.Reason, event.Message)
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
		utillog.DebugfIfNotNil(log, "Returning cached podList for %s", path)
		return podList, nil
	}

	// Not found in the cache, get it from the file
	file, err := os.Open(path)
	if err != nil {
		utillog.DebugfIfNotNil(log, "file %s not found", path)
		return nil, err
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		utillog.DebugfIfNotNil(log, "Failed reading Json file %s", path)
		return nil, err
	}
	err = encjson.Unmarshal(fileBytes, &podList)
	if err != nil {
		utillog.DebugfIfNotNil(log, "Failed to unmarshal podList at %s", path)
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
		return !IsPodReadyOrCompleted(pod)
	}
	return true
}

// IsPodPending returns a boolean indicating whether a pod is pending or not
func IsPodPending(pod corev1.Pod) bool {
	return pod.Status.Phase == corev1.PodPending
}

// FindProblematicPods looks at the pods.json files in bugReportDir and returns
// problematicPodNamespaces, which is a map from namespaces to lists of problematic pods per namespace.
func FindProblematicPods(bugReportDir string) (problematicPodNamespaces map[string][]corev1.Pod, err error) {
	problematicPodNamespaces, err = findProblematicPodNamespaceMap(bugReportDir)
	if err != nil {
		return nil, fmt.Errorf("an error occurred while trying to find problematic pods %s", err.Error())
	}
	if len(problematicPodNamespaces) == 0 {
		return nil, nil
	}
	return problematicPodNamespaces, nil
}

// findProblematicPodNamespaceMap looks at the pods.json files in the cluster and
// returns problematicPodNamespaces, which is a map from namespaces to lists of problematic pods per namespace.
func findProblematicPodNamespaceMap(clusterRoot string) (problematicPodNamespaces map[string][]corev1.Pod, err error) {
	podLists, err := getPodListsFromClusterRoot(nil, clusterRoot)
	if err != nil {
		return nil, err
	}

	// Inspect each pod, and, if it is problematic, add it to the list of problematic pods
	// for the pod's namespace.
	problematicPodNamespaces = make(map[string][]corev1.Pod)
	for _, podList := range podLists {
		for _, pod := range podList.Items {
			if !IsPodProblematic(pod) {
				continue
			}
			problematicPodNamespaces[pod.Namespace] = append(problematicPodNamespaces[pod.Namespace], pod)
		}
	}
	return problematicPodNamespaces, nil
}

// This looks at the pods.json files in the cluster and will give a list of files
// if any have pods that are not Running or Succeeded.
func findProblematicPodFiles(log *zap.SugaredLogger, clusterRoot string) (podFiles []string, err error) {
	podLists, err := getPodListsFromClusterRoot(log, clusterRoot)
	if err != nil {
		return nil, err
	}

	podFiles = make([]string, 0, len(podLists))
	for podFile, podList := range podLists {
		// If we find any we flag the file as havin problematic pods and move to the next file
		// this is just a quick scan to identify which files to drill into
		for _, pod := range podList.Items {
			if !IsPodProblematic(pod) {
				continue
			}
			utillog.DebugfIfNotNil(log, "Problematic pods detected: %s", podFile)
			podFiles = append(podFiles, podFile)
			break
		}
	}
	return podFiles, nil
}

// getPodListsFromClusterRoot looks at the pods.json files from the provided clusterRoot.
// Then this function returns podLists, which maps from pod file names to a list of pods found in that file.
func getPodListsFromClusterRoot(log *zap.SugaredLogger, clusterRoot string) (podLists map[string]*corev1.PodList, err error) {
	// Find the relevant pod files
	allPodFiles, err := files.GetMatchingFileNames(log, clusterRoot, PodFilesMatchRe)
	if err != nil {
		return nil, err
	}
	if len(allPodFiles) == 0 {
		return nil, nil
	}

	// Build the map of pod file names to pod lists
	podLists = make(map[string]*corev1.PodList)
	for _, podFile := range allPodFiles {
		utillog.DebugfIfNotNil(log, "Looking at pod file for problematic pods: %s", podFile)
		podList, err := GetPodList(log, podFile)
		if err != nil {
			utillog.DebugfIfNotNil(log, "Failed to get the PodList for %s, skipping due to error: %s", podFile, err.Error())
			continue
		}
		if podList == nil {
			utillog.DebugfIfNotNil(log, "No PodList was returned, skipping")
			continue
		}
		podLists[podFile] = podList
	}
	return podLists, nil
}

func reportProblemPodsNoIssues(log *zap.SugaredLogger, clusterRoot string, podFiles []string) {
	messages := make([]string, 0, len(podFiles))
	matches := make([]files.TextMatch, 0, len(podFiles))
	problematicNotPending := 0
	pendingPodsSeen := 0
	for _, podFile := range podFiles {
		podList, err := GetPodList(log, podFile)
		if err != nil {
			utillog.DebugfIfNotNil(log, "Failed to get the PodList for %s", podFile, err)
			continue
		}
		if podList == nil {
			utillog.DebugfIfNotNil(log, "No PodList was returned, skipping")
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
					utillog.DebugfIfNotNil(log, "Failed to create pod condition message: %v", err)
					continue
				}
				messages = append(messages, message)
			}
			// TODO: Time correlation for search

			fileName := files.FindPodLogFileName(clusterRoot, pod)
			matched, err := files.SearchFile(log, fileName, WideErrorSearchRe, nil)
			if err != nil {
				utillog.DebugfIfNotNil(log, "Failed to search the logfile %s for the ns/pod %s/%s",
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
	messages = checkIfHelmPodsAdded(messages)
	supportingData := make([]report.SupportData, 1)
	supportingData[0] = report.SupportData{
		Messages:    messages,
		TextMatches: matches,
	}
	// If all of the problematic pods were pending only, just report that, otherwise report them as problematic if some are
	// failing or unknown
	if len(messages) > 0 {
		if pendingPodsSeen > 0 && problematicNotPending == 0 {
			report.ContributeIssue(log, report.NewKnownIssueSupportingData(report.PendingPods, clusterRoot, supportingData))
		} else {
			report.ContributeIssue(log, report.NewKnownIssueSupportingData(report.PodProblemsNotReported, clusterRoot, supportingData))
		}
	}
}

// checkIfHelmPodsAdded tries to assess whether there is an issue with helm and rancher pods or not
// in cattle-system namespace
// if the helm pods are not fine and rancher pods are working, then it deletes the issues related to helm operation pods in cattle-system
// it edits the message array to remove the messages related to helm-operation pods and returns the modified message array
func checkIfHelmPodsAdded(messages []string) []string {
	isHelm := false
	isRancher := false
	var indices []int
	for i, data := range messages {
		if strings.Contains(data, "Namespace cattle-system, Pod helm-operation") {
			isHelm = true
			indices = append(indices, i)
		}
		if strings.Contains(data, "Namespace cattle-system, Pod rancher") {
			isRancher = true
		}

	}
	if isHelm && !isRancher {
		for i, index := range indices {
			messages = append(messages[:(index-i)], messages[(index+1-i):]...)

		}
	}
	return messages

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
