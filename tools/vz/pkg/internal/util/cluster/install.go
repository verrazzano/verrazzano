// Copyright (c) 2021, 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Package cluster handles cluster analysis
package cluster

import (
	encjson "encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/mysql"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/internal/util/files"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/internal/util/report"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
)

// Compiled Regular expressions
var installNGINXIngressControllerFailedRe = regexp.MustCompile(`Installing NGINX Ingress Controller.*\[FAILED\]`)
var noIPForIngressControllerRegExp = regexp.MustCompile(`Failed getting DNS suffix: No IP found for service ingress-controller-ingress-nginx-controller with type LoadBalancer`)
var errorSettingRancherTokenRegExp = regexp.MustCompile(`Failed setting Rancher access token.*no such host`)
var noIPForIstioIngressReqExp = regexp.MustCompile(`Ingress external IP pending for component istio: No IP found for service istio-ingressgateway with type *`)
var dbLoadJobFailedRe = regexp.MustCompile(`.*DB load job has failed.*`)
var dbLoadJobCompletedRe = regexp.MustCompile(`.*Keycloak DB successfully migrated.*`)

// I'm going with a more general pattern for limit reached as the supporting details should give the precise message
// and the advice can be to refer to the supporting details on the limit that was exceeded. We can change it up
// if we need a more precise match
var blockStorageLimitExceeded = regexp.MustCompile(`.*failed to provision volume with StorageClass .*New volume creation failed Error returned by Blockstorage Service.*`)
var ephemeralIPLimitReachedRe = regexp.MustCompile(`.*Limit for non-ephemeral regional public IP per tenant of .* has been already reached`)
var lbServiceLimitReachedRe = regexp.MustCompile(`.*The following service limits were exceeded: lb-.*`)
var failedToEnsureLoadBalancer = regexp.MustCompile(`.*failed to ensure load balancer: awaiting load balancer.*`)
var invalidLoadBalancerParameter = regexp.MustCompile(`.*Service error:InvalidParameter. Limits-Service returned 400.*Invalid service/quota load-balancer.*`)
var loadBalancerCreationIssue = regexp.MustCompile(`.*Private subnet.* is not allowed in a public loadbalancer.*`)
var vpoErrorMessages []string

const logLevelError = "error"
const logLevelInfo = "info"
const verrazzanoResource = "verrazzano-resources.json"
const eventsJSON = "events.json"
const servicesJSON = "services.json"
const podsJSON = "pods.json"
const istioSystem = "istio-system"

const installErrorNotFound = "Component specific error(s) not found in the Verrazzano install log for - "
const installErrorMessage = "One or more components listed below did not reach Ready state:"
const installUnavailableMessage = "One or more components listed below is in Ready state but Unavailable:"

const (
	// Service name
	ingressController = "ingress-controller-ingress-nginx-controller"
	istioIngress      = "istio-ingressgateway"

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
	compsNotReady, compsReadyNotAvailable, err := getComponentsNotReady(log, clusterRoot)
	notReady := false
	if err != nil {
		return err
	}

	if len(compsNotReady) > 0 {
		notReady = true
		reportInstallIssue(log, clusterRoot, compsNotReady, issueReporter, notReady)
		analyzeVerrazzanoInstallIssue(log, clusterRoot, issueReporter)
	}

	// When one or more components are in Ready state, but are unavailable, report the issue
	if len(compsReadyNotAvailable) > 0 {
		reportInstallIssue(log, clusterRoot, compsReadyNotAvailable, issueReporter, notReady)
		analyzeVerrazzanoInstallIssue(log, clusterRoot, issueReporter)

	}
	// Handle uninstall issue here, before returning
	return nil
}

func checkIfLoadJobFailed(vpoLog string, errorLogs []files.LogMessage, infoLogs []files.LogMessage, clusterRoot string, issueReporter *report.IssueReporter) {
	dbLoadJobFailure := false
	dbLoadJobSuccess := false

	messages := make(StringSlice, 1)

	for _, errLog := range errorLogs {
		if dbLoadJobFailedRe.MatchString(errLog.Message) {
			dbLoadJobFailure = true
			messages[0] = errLog.Message
		}
	}

	for _, infoLog := range infoLogs {
		if dbLoadJobCompletedRe.MatchString(infoLog.Message) {
			dbLoadJobSuccess = true
		}
	}

	if dbLoadJobFailure && !dbLoadJobSuccess {
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

		issueReporter.AddKnownIssueMessagesFiles(report.KeycloakDataMigrationFailure, clusterRoot, messages, files)
	}
}

// analyzeVerrazzanoInstallIssue is called when we have reason to believe that the installation has failed
func analyzeVerrazzanoInstallIssue(log *zap.SugaredLogger, clusterRoot string, issueReporter *report.IssueReporter) (err error) {
	// Because NGINX depends on the Istio Component, we should verify its service external IP exists
	// before verifying that the NGINX service is in good standing
	analyzeIstioIngressService(log, clusterRoot, issueReporter)
	analyzeExternalDNS(log, clusterRoot, issueReporter)
	analyzeKeycloakIssue(log, clusterRoot, issueReporter)

	ingressNGINXNamespace, err := checkIngressNGINXNamespace(clusterRoot)
	if err != nil {
		log.Errorf("Unexpected error checking for Ingress NGINX namespace: %v", err)
		return err
	}

	podFile := files.FormFilePathInNamespace(clusterRoot, ingressNGINXNamespace, podsJSON)

	podList, err := GetPodList(log, podFile)
	if err != nil {
		log.Debugf("Failed to get the list of pods for the given pod file %s, skipping", podFile)
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
	ingressNGINXNamespace, err := checkIngressNGINXNamespace(clusterRoot)
	if err != nil {
		log.Errorf("Unexpected error checking for Ingress NGINX namespace: %v", err)
		return err
	}

	services, err := GetServiceList(log, files.FormFilePathInNamespace(clusterRoot, ingressNGINXNamespace, servicesJSON))
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
		loadBalancerCheck := false

		var reportPodIssue string
		var reportEvent string
		if helpers.GetIsLiveCluster() {
			reportPodIssue = report.GetRelatedPodMessage(pod.ObjectMeta.Name, pod.ObjectMeta.Namespace)
			reportEvent = report.GetRelatedEventMessage(pod.ObjectMeta.Namespace)
		} else {
			reportPodIssue = podFile
			eventFile := files.FormFilePathInNamespace(clusterRoot, controllerService.ObjectMeta.Namespace, eventsJSON)
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
			} else if loadBalancerCreationIssue.MatchString(event.Message) && !loadBalancerCheck {
				messages := make(StringSlice, 1)
				messages[0] = event.Message
				issueReporter.AddKnownIssueMessagesFiles(report.NginxIngressPrivateSubnet, clusterRoot, messages, files)
				issueDetected = true
				loadBalancerCheck = true
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

		fileName := files.FormFilePathInClusterRoot(clusterRoot, ingressNGINXNamespace)
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

// analyseIstioIngressService generates an issue report if the Istio Ingress Service lacks an external IP
func analyzeIstioIngressService(log *zap.SugaredLogger, clusterRoot string, issueReporter *report.IssueReporter) {
	istioServicesFile := files.FormFilePathInNamespace(clusterRoot, istioSystem, servicesJSON)
	serviceList, err := GetServiceList(log, istioServicesFile)
	if err != nil || serviceList == nil {
		log.Debugf("Failed to get the list of services for the given service file %s, or no service was returned, skipping", serviceList)
		return
	}
	for _, service := range serviceList.Items {
		if !strings.HasPrefix(service.Name, istioIngress) {
			continue
		}
		log.Debugf("Service found for Istio ingress verification. namespace: %s, name: %s", service.ObjectMeta.Namespace, service.ObjectMeta.Name)
		if len(service.Spec.ExternalIPs) < 0 || len(service.Status.LoadBalancer.Ingress) < 0 {
			log.Debugf("External IP located for service %s/%s, skipping issue report", service.ObjectMeta.Namespace, service.ObjectMeta.Name)
			return
		}
		for _, errorMsg := range vpoErrorMessages {
			if noIPForIstioIngressReqExp.MatchString(errorMsg) {
				// Populate the message from the matched error message
				messages := make(StringSlice, 1)
				messages[0] = noIPForIngressControllerRegExp.String()
				// Create the service message from the object metadata
				servFiles := make(StringSlice, 1)
				servFiles[0] = report.GetRelatedServiceMessage(service.ObjectMeta.Name, service.ObjectMeta.Namespace)
				// Reporting as an Ingress not found issue with the relevant Service data from the istio-system service file
				issueReporter.AddKnownIssueMessagesFiles(report.IstioIngressNoIP, clusterRoot, messages, servFiles)
				issueReporter.Contribute(log, clusterRoot)
			}
		}
	}
	analyzeIstioLoadBalancerIssue(log, clusterRoot, issueReporter)
}

// Read the Verrazzano resource and return the two lists, first of components which did not reach Ready state and the second of components that are Ready but Unavailable
func getComponentsNotReady(log *zap.SugaredLogger, clusterRoot string) ([]string, []string, error) {
	var compsNotReady = make([]string, 0)
	compsReadyNotAvailable := compsNotReady
	vzResourcesPath := files.FormFilePathInClusterRoot(clusterRoot, verrazzanoResource)
	fileInfo, e := os.Stat(vzResourcesPath)
	if e != nil || fileInfo.Size() == 0 {
		log.Infof("Verrazzano resource file %s is either empty or there is an issue in getting the file info about it", vzResourcesPath)
		// The cluster dump taken by the latest script is expected to contain the verrazzano-resources.json.
		// In order to support cluster dumps taken in earlier release, return nil rather than an error.
		return nil, nil, nil
	}

	file, err := os.Open(vzResourcesPath)
	if err != nil {
		log.Infof("file %s not found", vzResourcesPath)
		return compsNotReady, compsReadyNotAvailable, err
	}
	defer file.Close()
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		log.Infof("Failed reading Json file %s", vzResourcesPath)
		return compsNotReady, compsReadyNotAvailable, err
	}

	var vzResourceList installv1alpha1.VerrazzanoList
	err = encjson.Unmarshal(fileBytes, &vzResourceList)
	if err != nil {
		log.Infof("Failed to unmarshal Verrazzano resource at %s", vzResourcesPath)
		return compsNotReady, compsReadyNotAvailable, err
	}

	var vzRes installv1alpha1.Verrazzano
	if len(vzResourceList.Items) > 0 {
		// There should be only one Verrazzano resource, so the first item from the list should be good enough
		vzRes = vzResourceList.Items[0]
	} else {
		// If the items are empty, try unmarshalling directly to Verrazzano type
		err := encjson.Unmarshal(fileBytes, &vzRes)
		if err != nil {
			log.Infof("Failed to unmarshal Verrazzano resource at %s", vzResourcesPath)
			return compsNotReady, compsReadyNotAvailable, err
		}
	}

	// Verrazzano installation is not complete, find out the list of components which are not ready
	for _, compStatusDetail := range vzRes.Status.Components {
		if compStatusDetail.State != installv1alpha1.CompStateReady {
			if compStatusDetail.State == installv1alpha1.CompStateDisabled {
				continue
			}
			log.Debugf("Component %s is not in ready state, state is %s", compStatusDetail.Name, vzRes.Status.State)
			compsNotReady = append(compsNotReady, compStatusDetail.Name)
		} else if compStatusDetail.Available != nil && *compStatusDetail.Available != installv1alpha1.ComponentAvailable {
			log.Debugf("Component %s is in ready state, but is unavailable, availability is %s", compStatusDetail.Name, vzRes.Status.Available)
			compsReadyNotAvailable = append(compsReadyNotAvailable, compStatusDetail.Name)
		}
	}
	return compsNotReady, compsReadyNotAvailable, nil
}

// Read the platform operator log, report the errors found for the list of components which either failed to reach Ready state or are Unavailable
func reportInstallIssue(log *zap.SugaredLogger, clusterRoot string, compsNotReady []string, issueReporter *report.IssueReporter, notReadyComponentsFound bool) error {
	vpologRegExp := regexp.MustCompile(`verrazzano-install/verrazzano-platform-operator-.*/logs.txt`)
	allPodFiles, err := files.GetMatchingFileNames(log, clusterRoot, vpologRegExp)
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
	if !notReadyComponentsFound {
		messages[0] = installUnavailableMessage
	}

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

		if comp == mysql.ComponentName {
			allInfo, err := files.FilterLogsByLevelComponent(logLevelInfo, comp, allMessages)
			if err != nil {
				log.Debugf("Failed to get info logs for %s component", comp)
			}
			checkIfLoadJobFailed(vpoLog, allErrors, allInfo, clusterRoot, issueReporter)
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

	// Create a single message to display the list of components without specific error in the platform operator
	if len(compsNoMessages) > 0 {
		errorMessage := "\t " + installErrorNotFound + strings.Join(compsNoMessages[:], ", ")
		messages = append(messages, errorMessage)
	}

	// Get unique namespaces associated with the components with no error messages
	namespacesCompNoMsg := getUniqueNamespaces(log, compsNoMessages)

	// Check the events related to failed components namespace to provide additional support data
	for _, namespace := range namespacesCompNoMsg {
		eventList, err := GetEventsRelatedToComponentNamespace(log, clusterRoot, namespace, nil)
		if err != nil {
			log.Debugf("Failed to get events related to %s namespace", namespace)
		}
		messages.addMessages(CheckEventsForWarnings(log, eventList, "Warning", nil))
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
	if !notReadyComponentsFound {
		issueReporter.AddKnownIssueMessagesFiles(report.ComponentsUnavailable, clusterRoot, messages, files)
	} else {
		issueReporter.AddKnownIssueMessagesFiles(report.ComponentsNotReady, clusterRoot, messages, files)
	}
	return nil
}

func analyzeIstioLoadBalancerIssue(log *zap.SugaredLogger, clusterRoot string, issueReporter *report.IssueReporter) {
	eventsList, e := GetEventList(log, files.FormFilePathInNamespace(clusterRoot, istioSystem, eventsJSON))
	if e != nil {
		log.Debugf("Failed to get events file %s", eventsJSON)
		return
	}
	log.Debugf("Found %d events in events file", len(eventsList.Items))
	serviceEvents := make([]corev1.Event, 0, 1)
	isIssueAlreadyExists := false
	for _, event := range eventsList.Items {
		if loadBalancerCreationIssue.MatchString(event.Message) && !isIssueAlreadyExists {
			isIssueAlreadyExists = true
			serviceEvents = append(serviceEvents, event)
			messages := make(StringSlice, 1)
			messages[0] = event.Message
			// Create the service message from the object metadata
			servFiles := make(StringSlice, 1)
			servFiles[0] = report.GetRelatedServiceMessage(serviceEvents[0].ObjectMeta.Name, istioSystem)
			issueReporter.AddKnownIssueMessagesFiles(report.IstioIngressPrivateSubnet, clusterRoot, messages, servFiles)
		}
	}
}

// Analyze error in the Keycloak namespace
func analyzeKeycloakIssue(log *zap.SugaredLogger, clusterRoot string, issueReporter *report.IssueReporter) {
	eventsList, e := GetEventList(log, files.FormFilePathInNamespace(clusterRoot, "keycloak", eventsJSON))
	if e != nil {
		log.Debugf("Failed to get events file %s", eventsJSON)
		return
	}
	log.Debugf("Found %d events in events file", len(eventsList.Items))
	serviceEvents := make([]corev1.Event, 0, 1)
	isIssueAlreadyExists := false
	for _, event := range eventsList.Items {
		if blockStorageLimitExceeded.MatchString(event.Message) && !isIssueAlreadyExists {
			isIssueAlreadyExists = true
			serviceEvents = append(serviceEvents, event)
			messages := make(StringSlice, 1)
			messages[0] = event.Message
			// Create the service message from the object metadata
			servFiles := make(StringSlice, 1)
			servFiles[0] = report.GetRelatedServiceMessage(serviceEvents[0].ObjectMeta.Name, "keycloak")
			issueReporter.AddKnownIssueMessagesFiles(report.BlockStorageLimitExceeded, clusterRoot, messages, servFiles)
		}
	}
}

func getUniqueNamespaces(log *zap.SugaredLogger, compsNoMessages []string) []string {
	var uniqueNamespaces []string

	for _, comp := range compsNoMessages {
		found, component := registry.FindComponent(comp)
		if !found {
			log.Debugf("Couldn't find the namespace related to %s component", comp)
			continue
		}
		uniqueNamespaces = append(uniqueNamespaces, component.Namespace())
	}
	uniqueNamespaces = helpers.RemoveDuplicate(uniqueNamespaces)
	return uniqueNamespaces
}

func analyzeExternalDNS(log *zap.SugaredLogger, clusterRoot string, issueReporter *report.IssueReporter) {
	errorMessagesRegex := []string{}
	errorMessagesRegex = append(errorMessagesRegex, `.*level=error.*"getting zones: listing zones in.*Service error:NotAuthorizedOrNotFound.*`)
	// External DNS will be in the cert-manager ns prior to 1.6, afterwards in verrazzano-system, but we must account for both
	pattern := fmt.Sprintf("(%s|%s)", vzconst.ExternalDNSNamespace, vzconst.CertManagerNamespace) + `/external-dns-.*/logs.txt`
	externalDnslogRegExp := regexp.MustCompile(pattern)
	allPodFiles, err := files.GetMatchingFileNames(log, clusterRoot, externalDnslogRegExp)
	if err != nil {
		return
	}

	if len(allPodFiles) < 1 {
		return
	}
	// We should get only one pod file, use the first element rather than going through the slice
	logFile := allPodFiles[0]
	for index := range errorMessagesRegex {
		status, err := files.SearchFile(log, logFile, regexp.MustCompile(errorMessagesRegex[index]), nil)
		if err != nil {
			return
		}
		if len(status) > 0 {
			messages := make(StringSlice, 1)
			messages[0] = status[0].MatchedText
			servFiles := make(StringSlice, 1)
			issueReporter.AddKnownIssueMessagesFiles(report.ExternalDNSConfigureIssue, clusterRoot, messages, servFiles)
		}
	}
}

func checkIngressNGINXNamespace(clusterRoot string) (string, error) {
	_, err := os.Stat(fmt.Sprintf("%s/%s", clusterRoot, constants.IngressNginxNamespace))
	if err == nil {
		return constants.IngressNginxNamespace, nil
	} else if os.IsNotExist(err) {
		return constants.LegacyIngressNginxNamespace, nil
	}
	return "", err
}
