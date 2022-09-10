// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Package cluster handles cluster analysis
package cluster

import (
	encjson "encoding/json"
	"fmt"
	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/files"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/report"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	"go.uber.org/zap"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	"os"
	"regexp"
	"strings"
)

// Compiled Regular expressions
var installNGINXIngressControllerFailedRe = regexp.MustCompile(`Installing NGINX Ingress Controller.*\[FAILED\]`)
var noIPForIngressControllerRegExp = regexp.MustCompile(`Failed getting DNS suffix: No IP found for service ingress-controller-ingress-nginx-controller with type LoadBalancer`)
var errorSettingRancherTokenRegExp = regexp.MustCompile(`Failed setting Rancher access token.*no such host`)

// I'm going with a more general pattern for limit reached as the supporting details should give the precise message
// and the advice can be to refer to the supporting details on the limit that was exceeded. We can change it up
// if we need a more precise match
var ephemeralIPLimitReachedRe = regexp.MustCompile(`.*Limit for non-ephemeral regional public IP per tenant of .* has been already reached`)
var lbServiceLimitReachedRe = regexp.MustCompile(`.*The following service limits were exceeded: lb-.*`)
var failedToEnsureLoadBalancer = regexp.MustCompile(`.*failed to ensure load balancer: awaiting load balancer.*`)
var invalidLoadBalancerParameter = regexp.MustCompile(`.*Service error:InvalidParameter. Limits-Service returned 400.*Invalid service/quota load-balancer.*`)

var vpoErrorMessages []string

const logLevelError = "error"
const verrazzanoResource = "verrazzano-resources.json"
const eventsJSON = "events.json"
const servicesJSON = "services.json"
const podsJSON = "pods.json"
const ingressNginx = "ingress-nginx"

const installErrorNotFound = "Component specific error(s) not found in the Verrazzano install log for - "
const installErrorMessage = "One or more components listed below did not reach Ready state:"

const (
	// Service name
	ingressController = "ingress-controller-ingress-nginx-controller"

	// Function names
	nginxIngressControllerFailed = "nginxIngressControllerFailed"
	noIPForIngressController     = "noIPForIngressController"
	errorSettingRancherToken     = "errorSettingRancherToken"
)

var dispatchMatchMap = map[string]*regexp.Regexp{
	nginxIngressControllerFailed: installNGINXIngressControllerFailedRe,
	noIPForIngressController:     noIPForIngressControllerRegExp,
	errorSettingRancherToken:     errorSettingRancherTokenRegExp,
}

var dispatchFunctions = map[string]func(log *zap.SugaredLogger, clusterRoot string, podFile string, pod corev1.Pod, issueReporter *report.IssueReporter) (err error){
	nginxIngressControllerFailed: analyzeNGINXIngressController,
	noIPForIngressController:     analyzeNGINXIngressController,
	errorSettingRancherToken:     analyzeNGINXIngressController,
}

func AnalyzeVerrazzanoResource(log *zap.SugaredLogger, clusterRoot string, issueReporter *report.IssueReporter) (err error) {
	compsNotReady, err := getComponentsNotReady(log, clusterRoot)
	if err != nil {
		return err
	}

	if len(compsNotReady) > 0 {
		reportInstallIssue(log, clusterRoot, compsNotReady, issueReporter)
	}

	// When one or more components are not in Ready state, get the events from the pods based on the list of known failures and report
	if len(compsNotReady) > 0 {
		analyzeVerrazzanoInstallIssue(log, clusterRoot, issueReporter)
	}
	// Handle uninstall issue here, before returning
	return nil
}

// analyzeVerrazzanoInstallIssue is called when we have reason to believe that the installation has failed
func analyzeVerrazzanoInstallIssue(log *zap.SugaredLogger, clusterRoot string, issueReporter *report.IssueReporter) (err error) {
	podFile := files.FindFileInNamespace(clusterRoot, ingressNginx, podsJSON)

	podList, err := GetPodList(log, podFile)
	if err != nil {
		log.Debugf("Failed to get the list of pods for the given pod file %s, skipping", podFile, err)
		return err
	}
	if podList == nil {
		log.Debugf("No pod was returned, skipping")
		return nil
	}
	for _, pod := range podList.Items {
		if !strings.HasPrefix(pod.Name, ingressController) {
			continue
		}
		for _, errorMsg := range vpoErrorMessages {
			for matchKey, matcher := range dispatchMatchMap {
				if matcher.MatchString(errorMsg) {
					err = dispatchFunctions[matchKey](log, clusterRoot, podFile, pod, issueReporter)
					if err != nil {
						log.Errorf("analyzeVerrazzanoInstallIssue failed in %s function", matchKey, err)
					}
				}
			}
		}
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
	services, err := GetServiceList(log, files.FindFileInNamespace(clusterRoot, ingressNginx, servicesJSON))
	if err != nil {
		return err
	}
	var controllerService corev1.Service
	controllerServiceSet := false
	for _, service := range services.Items {
		log.Debugf("Service found. namespace: %s, name: %s", service.ObjectMeta.Namespace, service.ObjectMeta.Name)
		if service.ObjectMeta.Name == ingressController {
			log.Debugf("NGINX Ingress Controller service. namespace: %s, name: %s", service.ObjectMeta.Namespace, service.ObjectMeta.Name)
			controllerService = service
			controllerServiceSet = true
		}
	}
	if controllerServiceSet {
		issueDetected := false

		// TODO: Need to handle time range correlation (only events within a time range)
		events, err := GetEventsRelatedToService(log, clusterRoot, controllerService, nil)
		if err != nil {
			log.Debugf("Failed to get events related to the NGINX ingress controller service", err)
			return err
		}
		//flags to make sure we're not capturing the same event message repeatedly
		ephemeralIPLimitReachedCheck := false
		lbServiceLimitReachedCheck := false
		errorSyncingLoadBalancerCheck := false
		invalidLBShapeCheck := false

		var reportPodIssue string
		var reportEvent string
		if helpers.GetIsLiveCluster() {
			reportPodIssue = report.GetRelatedPodMessage(pod.ObjectMeta.Name, pod.ObjectMeta.Namespace)
			reportEvent = report.GetRelatedEventMessage(pod.ObjectMeta.Namespace)
		} else {
			reportPodIssue = podFile
			eventFile := files.FindFileInNamespace(clusterRoot, controllerService.ObjectMeta.Namespace, eventsJSON)
			reportEvent = eventFile
		}

		// Check if the event matches failure
		log.Debugf("Found %d events", len(events))

		for _, event := range events {
			log.Debugf("analyzeNGINXIngressController event Reason: %s", event.Reason)
			if !EventReasonFailedRe.MatchString(event.Reason) {
				continue
			}
			log.Debugf("analyzeNGINXIngressController event Reason: %s", event.Message)

			files := make(StringSlice, 2)
			files[0] = reportPodIssue
			files[1] = reportEvent

			if ephemeralIPLimitReachedRe.MatchString(event.Message) && !ephemeralIPLimitReachedCheck {
				messages := make(StringSlice, 1)
				messages[0] = event.Message
				issueReporter.AddKnownIssueMessagesFiles(report.IngressOciIPLimitExceeded, clusterRoot, messages, files)
				issueDetected = true
				ephemeralIPLimitReachedCheck = true
			} else if lbServiceLimitReachedRe.MatchString(event.Message) && !lbServiceLimitReachedCheck {
				messages := make(StringSlice, 1)
				messages[0] = event.Message
				issueReporter.AddKnownIssueMessagesFiles(report.IngressLBLimitExceeded, clusterRoot, messages, files)
				issueDetected = true
				lbServiceLimitReachedCheck = true
			} else if failedToEnsureLoadBalancer.MatchString(event.Message) && !errorSyncingLoadBalancerCheck {
				messages := make(StringSlice, 1)
				messages[0] = event.Message
				issueReporter.AddKnownIssueMessagesFiles(report.IngressNoIPFound, clusterRoot, messages, files)
				issueDetected = true
				errorSyncingLoadBalancerCheck = true
				issueReporter.Contribute(log, clusterRoot)
			} else if invalidLoadBalancerParameter.MatchString(event.Message) && !invalidLBShapeCheck {
				messages := make(StringSlice, 1)
				messages[0] = event.Message
				issueReporter.AddKnownIssueMessagesFiles(report.IngressShapeInvalid, clusterRoot, messages, files)
				issueDetected = true
				invalidLBShapeCheck = true
				issueReporter.Contribute(log, clusterRoot)
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
			files[0] = reportPodIssue
			issueReporter.AddKnownIssueMessagesFiles(report.IngressNoLoadBalancerIP, clusterRoot, messages, files)
			return nil
		}

		// if we made it this far we know that there is an issue with the ingress controller but
		// we haven't found anything, so give general advise for now.
		messages := make(StringSlice, 1)
		messages[0] = fmt.Sprintf("Namespace %s, Pod %s", pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)
		// TODO: Time correlation on error search here

		fileName := files.FindFileInClusterRoot(clusterRoot, ingressNginx)
		nginxPodErrors, err := files.FindFilesAndSearch(log, fileName, LogFilesMatchRe, WideErrorSearchRe, nil)
		if err != nil {
			log.Debugf("Failed searching NGINX Ingress namespace log files for supporting error log data", err)
		}

		if helpers.GetIsLiveCluster() && len(nginxPodErrors) > 0 {
			nginxPodErrors[0].FileName = report.GetRelatedLogFromPodMessage(fileName)
		}

		files := make(StringSlice, 1)
		files[0] = reportPodIssue
		supportingData := make([]report.SupportData, 1)
		supportingData[0] = report.SupportData{
			Messages:     messages,
			TextMatches:  nginxPodErrors,
			RelatedFiles: files,
		}
		issueReporter.AddKnownIssueSupportingData(report.IngressInstallFailure, clusterRoot, supportingData)
		return nil
	}

	return nil
}

// Read the Verrazzano resource and return the list of components which did not reach Ready state
func getComponentsNotReady(log *zap.SugaredLogger, clusterRoot string) ([]string, error) {
	var compsNotReady = make([]string, 0)
	vzResourcesPath := files.FindFileInClusterRoot(clusterRoot, verrazzanoResource)
	fileInfo, e := os.Stat(vzResourcesPath)
	if e != nil || fileInfo.Size() == 0 {
		log.Infof("Verrazzano resource file %s is either empty or there is an issue in getting the file info about it", vzResourcesPath)
		// The cluster dump taken by the latest script is expected to contain the verrazzano-resources.json.
		// In order to support cluster dumps taken in earlier release, return nil rather than an error.
		return nil, nil
	}

	file, err := os.Open(vzResourcesPath)
	if err != nil {
		log.Infof("file %s not found", vzResourcesPath)
		return compsNotReady, err
	}
	defer file.Close()
	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		log.Infof("Failed reading Json file %s", vzResourcesPath)
		return compsNotReady, err
	}

	var vzResourceList installv1alpha1.VerrazzanoList
	err = encjson.Unmarshal(fileBytes, &vzResourceList)
	if err != nil {
		log.Infof("Failed to unmarshal Verrazzano resource at %s", vzResourcesPath)
		return compsNotReady, err
	}

	// There should be only one Verrazzano resource, so the first item from the list should be good enough
	for _, vzRes := range vzResourceList.Items {
		if vzRes.Status.State != installv1alpha1.VzStateReady {
			log.Debugf("Verrazzano installation is not complete, installation state %s", vzRes.Status.State)

			// Verrazzano installation is not complete, find out the list of components which are not ready
			for _, compStatusDetail := range vzRes.Status.Components {
				if compStatusDetail.State != installv1alpha1.CompStateReady {
					if compStatusDetail.State == installv1alpha1.CompStateDisabled {
						continue
					}
					log.Debugf("Component %s is not in ready state, state is %s", compStatusDetail.Name, vzRes.Status.State)
					compsNotReady = append(compsNotReady, compStatusDetail.Name)
				}
			}
			return compsNotReady, nil
		}
	}
	return compsNotReady, nil
}

// Read the platform operator log, report the errors found for the list of components which fail to reach Ready state
func reportInstallIssue(log *zap.SugaredLogger, clusterRoot string, compsNotReady []string, issueReporter *report.IssueReporter) error {
	vpologRegExp := regexp.MustCompile(`verrazzano-install/verrazzano-platform-operator-.*/logs.txt`)
	allPodFiles, err := files.GetMatchingFiles(log, clusterRoot, vpologRegExp)
	if err != nil {
		return err
	}

	if len(allPodFiles) < 1 {
		return fmt.Errorf("failed to find Verrazzano Platform Operator pod")
	}
	// We should get only one pod file, use the first element rather than going through the slice
	vpoLog := allPodFiles[0]
	messages := make(StringSlice, 1)
	messages[0] = installErrorMessage

	// Go through all the components which did not reach Ready state
	allMessages, _ := files.ConvertToLogMessage(vpoLog)

	// Slice to hold the components without specific errors in platform operator log
	var compsNoMessages []string
	for _, comp := range compsNotReady {
		var allErrors []files.LogMessage
		allErrors, err := files.FilterLogsByLevelComponent(logLevelError, comp, allMessages)
		if err != nil {
			log.Infof("There is an error: %s reading install log: %s", err, vpoLog)
		}
		// Display only the last error for the component from the install log.
		// Need a better way to handle distinct errors for a component, however some of the errors during the initial
		// stages of the install might not indicate any real issue always, as reconcile takes care of healing those errors.
		if len(allErrors) > 0 {
			errorMessage := allErrors[len(allErrors)-1].Message
			messages = append(messages, "\t "+comp+": "+errorMessage)
			vpoErrorMessages = append(vpoErrorMessages, errorMessage)
		} else {
			compsNoMessages = append(compsNoMessages, comp)
		}
	}

	// Create a a single message to display the list of components without specific error in the platform operator
	if len(compsNoMessages) > 0 {
		errorMessage := "\t " + installErrorNotFound + strings.Join(compsNoMessages[:], ", ")
		messages = append(messages, errorMessage)
	}

	reportVzResource := ""
	reportVpoLog := ""
	// Construct resource in the analysis report, differently for live analysis
	if helpers.GetIsLiveCluster() {
		reportVzResource = report.GetRelatedVZResourceMessage()
		reportVpoLog = report.GetRelatedLogFromPodMessage(vpoLog)
	} else {
		reportVzResource = clusterRoot + "/" + verrazzanoResource
		reportVpoLog = vpoLog
	}
	files := make(StringSlice, 2)
	files[0] = reportVzResource
	files[1] = reportVpoLog
	issueReporter.AddKnownIssueMessagesFiles(report.ComponentsNotReady, clusterRoot, messages, files)
	return nil
}
