// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Package cluster handles cluster analysis
package cluster

import (
	"fmt"
	"github.com/verrazzano/verrazzano/tools/analysis/internal/util/files"
	"github.com/verrazzano/verrazzano/tools/analysis/internal/util/report"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"regexp"
)

// Compiled Regular expressions
var installNGINXIngressControllerFailed = regexp.MustCompile(`Installing NGINX Ingress Controller.*\[FAILED\]`)

// I'm going with a more general pattern for limit reached as the supporting details should give the precise message
// and the advice can be to refer to the supporting details on the limit that was exceeded. We can change it up
// if we need a more precise match
var ephemeralIPLimitReached = regexp.MustCompile(`.*Limit for non-ephemeral regional public IP per tenant of .* has been already reached`)
var limitReached = regexp.MustCompile(`.*Limit .* has been already reached`)
var reasonFailed = regexp.MustCompile(`.*Failed.*`)

const (
	// Service name
	ingressControllerService = "ingress-controller-ingress-nginx-controller"

	// Function names
	nginxIngressControllerFailed = "nginxIngressControllerFailed"
)

var dispatchMatchMap = map[string]*regexp.Regexp{
	nginxIngressControllerFailed: installNGINXIngressControllerFailed,
}

var dispatchFunctions = map[string]func(log *zap.SugaredLogger, clusterRoot string, podFile string, pod corev1.Pod, issueReporter *report.IssueReporter) (err error){
	nginxIngressControllerFailed: analyzeNGINXIngressController,
}

// AnalyzeVerrazzanoInstallIssue is called when we have reason to believe that the installation has failed
func AnalyzeVerrazzanoInstallIssue(log *zap.SugaredLogger, clusterRoot string, podFile string, pod corev1.Pod, issueReporter *report.IssueReporter) (err error) {
	// Skip if it is not the verrazzano install job pod
	if !IsVerrazzanoInstallJobPod(pod) {
		return nil
	}

	log.Debugf("verrazzanoInstallIssues analysis called for cluster: %s, ns: %s, pod: %s", clusterRoot, pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)
	// TODO: Not correlating time here yet
	if IsContainerNotReady(pod.Status.Conditions) {
		// The install job pod log is currently the only place we can determine where the install process failed at, so we
		// scrape those log messages out.
		logMatches, err := files.SearchFile(log, files.FindPodLogFileName(clusterRoot, pod), `Install.*\[FAILED\]`)
		if err == nil {
			// We likely will only have a single failure message here (we may only want to look at the last one for install failures)
			for _, matched := range logMatches {
				log.Debugf("Install failure message: %s", matched.MatchedText)
				// Loop through the match expressions to see if we have a handler for the message that matches
				for matchKey, matcher := range dispatchMatchMap {
					log.Debugf("Checking matcher: %s", matchKey)
					// If the matcher expression matches the failure message, call the handler function related to that matcher (same key)
					if matcher.MatchString(matched.MatchedText) {
						log.Debugf("Dispatch to handler: %s", matchKey)
						err = dispatchFunctions[matchKey](log, clusterRoot, podFile, pod, issueReporter)
						if err != nil {
							log.Errorf("AnalyzeVerrazzanoInstallIssue failed in %s function", matchKey, err)
						}
					}
				}
			}
		} else {
			log.Errorf("AnalyzeVerrazzanoInstallIssue failed to get log messages to determine install issue", err)
		}
	}

	// TODO: If we got here without determining a specific cause, put out a General Issue that the install has failed with supporting details
	//  Note that we may not have a lot of details to provide here (which is why we are falling back to this general issue)
	if len(issueReporter.PendingIssues) == 0 {
		// TODO: Add more supporting details here
		messages := make(StringSlice, 1)
		messages[0] = fmt.Sprintf("Namespace %s, Pod %s", pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)
		files := make(StringSlice, 1)
		files[0] = podFile
		issueReporter.AddKnownIssueMessagesFiles(report.InstallFailure, clusterRoot, messages, files)
	}
	return nil
}

func analyzeNGINXIngressController(log *zap.SugaredLogger, clusterRoot string, podFile string, pod corev1.Pod, issueReporter *report.IssueReporter) (err error) {
	// TODO: We need to add in time range handling here. The timestamps from structured K8S JSON should already be there, but we will also need to
	//     be able to correlate timestamps which are coming from Pod logs (not in the initial handling but we will almost certainly need that)
	//
	// 1) Find the events related to ingress-controller-ingress-nginx-controller service in the ingress-nginx namespace
	// If we have a start/end time for the install containerStatus, then we can use that to only look at logs which are in that time range

	// Look at the ingress-controller-ingress-nginx-controller, and look at the events related to it
	services, err := GetServiceList(log, files.FindFileInNamespace(clusterRoot, "ingress-nginx", "services.json"))
	if err != nil {
		return err
	}
	var controllerService corev1.Service
	controllerServiceSet := false
	for _, service := range services.Items {
		log.Debugf("Service found. namespace: %s, name: %s", service.ObjectMeta.Namespace, service.ObjectMeta.Name)
		if service.ObjectMeta.Name == ingressControllerService {
			log.Debugf("NGINX Ingress Controller service. namespace: %s, name: %s", service.ObjectMeta.Namespace, service.ObjectMeta.Name)
			controllerService = service
			controllerServiceSet = true
		}
	}
	if controllerServiceSet {
		issueDetected := false

		// TODO: Need to handle time range correlation (only events within a time range)
		events, err := GetEventsRelatedToService(log, clusterRoot, controllerService)
		if err != nil {
			log.Debugf("Failed to get events related to the NGINX ingress controller service", err)
			return err
		}
		// Check if the event matches failure
		log.Debugf("Found %d events", len(events))
		for _, event := range events {
			log.Debugf("analyzeNGINXIngressController event Reason: %s", event.Reason)
			if !reasonFailed.MatchString(event.Reason) {
				continue
			}
			log.Debugf("analyzeNGINXIngressController event Reason: %s", event.Message)
			if ephemeralIPLimitReached.MatchString(event.Message) {
				messages := make(StringSlice, 1)
				messages[0] = event.Message
				eventFile := files.FindFileInNamespace(clusterRoot, controllerService.ObjectMeta.Namespace, "events.json")
				files := make(StringSlice, 2)
				files[0] = podFile
				files[1] = eventFile
				issueReporter.AddKnownIssueMessagesFiles(report.IngressOciIPLimitExceeded, clusterRoot, messages, files)
				issueDetected = true
			}
		}

		// If we detected a more specific issue above, return now. If we didn't we check for cases where
		// we may not be able to narrow it down fully
		if issueDetected {
			return nil
		}

		// We check the LoadBalancer status to see if there is an IP address set. If not, we can at least
		// advise them that the LoadBalancer may not be setup
		if len(controllerService.Status.LoadBalancer.Ingress) == 0 {
			// TODO: Add and report a known issue here (we know the IP is not set, but not more than that)
			messages := make(StringSlice, 1)
			messages[0] = fmt.Sprintf("Namespace %s, Pod %s", pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)
			files := make(StringSlice, 1)
			files[0] = podFile
			issueReporter.AddKnownIssueMessagesFiles(report.IngressNoLoadBalancerIP, clusterRoot, messages, files)
			return nil
		}

		// if we made it this far we know that there is an issue with the ingress controller but
		// we haven't found anything, so give general advise for now.
		messages := make(StringSlice, 1)
		messages[0] = fmt.Sprintf("Namespace %s, Pod %s", pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)
		files := make(StringSlice, 1)
		files[0] = podFile
		issueReporter.AddKnownIssueMessagesFiles(report.IngressInstallFailure, clusterRoot, messages, files)
		return nil
	}

	return nil
}
