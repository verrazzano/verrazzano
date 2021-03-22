// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Package cluster handles cluster analysis
package cluster

import (
	"github.com/verrazzano/verrazzano/tools/analysis/internal/util/files"
	"github.com/verrazzano/verrazzano/tools/analysis/internal/util/report"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"regexp"
)

var installNGINXIngressControllerFailed = regexp.MustCompile("Installing NGINX Ingress Controller.*[FAILED]")

// I'm going with a more general pattern for limit reached as the supporting details should give the precise message
// and the advice can be to refer to the supporting details on the limit that was exceeded. We can change it up
// if we need a more precise match
var limitReached = regexp.MustCompile("Limit .* has been already reached")
var reasonFailed = regexp.MustCompile(".*Failed.*")

const (
	// Service name
	ingressControllerService = "ingress-controller-ingress-nginx-controller"

	// function map names
	nginxIngressController = ""
)

var dispatchMatchMap = map[string]*Regexp {
	nginxIngressController: installNGINXIngressControllerFailed,
}

var dispatchFunctions = map[string]func(log *zap.SugaredLogger, clusterRoot string, podFile string, pod corev1.Pod, issueReporter *report.IssueReporter) (err error){
	nginxIngressController: analyzeNGINXIngressController,
}

// AnalyzeVerrazzanoInstallIssue is called when we have reason to believe that the installation has failed
func AnalyzeVerrazzanoInstallIssue(log *zap.SugaredLogger, clusterRoot string, podFile string, pod corev1.Pod, issueReporter *report.IssueReporter) (err error) {
	// Skip if it is not the verrazzano install job pod
	if !IsVerrazzanoInstallJobPod(pod) {
		return nil
	}
	log.Debugf("verrazzanoInstallIssues analysis called for cluster: %s, ns: %s, pod: %s", clusterRoot, pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)
	if IsContainerNotReady(pod.Status.Conditions) {

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
	services, err := GetServiceList(log, files.FindFileInNamespace(clusterRoot, "ingress-nginx", "services.json" ))
	if err != nil {
		return err
	}
	var controllerService *corev1.Service
	for _, service := range services.Items {
		if service.ObjectMeta.Name == ingressControllerService {
			controllerService = service
		}
	}
	if controllerService != nil {
		// TODO: Need to handle time range correlation (only events within a time range)
		events, err := GetEventsRelatedToService(log, clusterRoot, controllerService)
		if err != nil {
			return err
		}
		// Check if the event matches failure
		for event := range events {
			if !reasonFailed.MatchString(event.Reason) {
				continue
			}
			if limitReached.MatchString(event.Message) {
				// TODO: Add and report a known issue for limit here, we
			}
		}
	}

	return nil
}