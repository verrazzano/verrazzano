// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package bugreport

import (
	"context"
	"fmt"
	vzconstants "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/vzchecks"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	pkghelpers "github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"os"
	"path/filepath"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"sync"
)

// The bug-report command captures the following resources from the cluster by default
// - Verrazzano resource
// - Logs from verrazzano-platform-operator, verrazzano-monitoring-operator and verrazzano-application-operator pods
// - Workloads (Deployment and ReplicaSet, StatefulSet, Daemonset), pods, events, ingress and services from the namespaces of
//   installed verrazzano components and namespaces specified by flag --include-namespaces
// - OAM resources like ApplicationConfiguration, Component, IngressTrait, MetricsTrait from namespaces specified by flag --include-namespaces
// - VerrazzanoManagedCluster, VerrazzanoProject and MultiClusterApplicationConfiguration in a multi-clustered environment

type ErrorsChannelLogs struct {
	PodName      string `json:"podName"`
	ErrorMessage string `json:"errorMessage"`
}

type ErrorsChannel struct {
	ErrorMessage string `json:"errorMessage"`
}

type PodLogs struct {
	IsPodLog bool
	Duration int64
}
type Pods struct {
	Namespace string
	PodList   []corev1.Pod
}

// CaptureClusterSnapshot selectively captures the resources from the cluster, useful to analyze the issue.
// func CaptureClusterSnapshot(kubeClient kubernetes.Interface, dynamicClient dynamic.Interface, client clipkg.Client, bugReportDir string, moreNS []string, vzHelper pkghelpers.VZHelper, isPodLog bool, duration int64) error {
func CaptureClusterSnapshot(kubeClient kubernetes.Interface, dynamicClient dynamic.Interface, client clipkg.Client, bugReportDir string, moreNS []string, vzHelper pkghelpers.VZHelper, podLogs PodLogs) error {

	// Create a file to capture the standard out to a file
	stdOutFile, err := os.OpenFile(filepath.Join(bugReportDir, constants.BugReportOut), os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		return fmt.Errorf("an error occurred while creating the file include the summary of the resources captured: %s", err.Error())
	}
	defer stdOutFile.Close()

	// Create a file to capture the standard err to a file
	stdErrFile, err := os.OpenFile(filepath.Join(bugReportDir, constants.BugReportErr), os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		return fmt.Errorf("an error occurred while creating the file include the summary of the resources captured: %s", err.Error())
	}
	defer stdErrFile.Close()

	// Create MultiWriters for standard out and err
	pkghelpers.SetMultiWriterOut(vzHelper.GetOutputStream(), stdOutFile)
	pkghelpers.SetMultiWriterErr(vzHelper.GetErrorStream(), stdErrFile)

	// Verrazzano as a list is required for the analysis tool
	vz := v1beta1.VerrazzanoList{}
	err = client.List(context.TODO(), &vz)
	if err != nil && !meta.IsNoMatchError(err) {
		return err
	}

	// Loop through the existing v1alpha1 Verrazzano and convert them to v1beta1
	// Add them to the vz list so that the bug report is not skipped
	vzA1 := v1alpha1.VerrazzanoList{}
	err = client.List(context.TODO(), &vzA1)
	if err != nil && !meta.IsNoMatchError(err) {
		return err
	}
	if len(vzA1.Items) != 0 {
		for _, vzA1Item := range vzA1.Items {
			convertedVZ := v1beta1.Verrazzano{}
			err = vzA1Item.ConvertTo(&convertedVZ)
			if err != nil {
				return err
			}
			vz.Items = append(vz.Items, convertedVZ)
		}
	}

	if len(vz.Items) == 0 {
		return fmt.Errorf("skip analyzing the cluster as Verrazzano is not installed")
	}

	// Get the list of namespaces based on the failed components and value specified by flag --include-namespaces
	nsList, additionalNS := collectNamespaces(kubeClient, moreNS, vz, vzHelper)
	var msgPrefix string
	if pkghelpers.GetIsLiveCluster() {
		msgPrefix = constants.AnalysisMsgPrefix
	} else {
		msgPrefix = constants.BugReportMsgPrefix
	}
	// Print initial message to console output only
	fmt.Fprintf(vzHelper.GetOutputStream(), msgPrefix+"resources from the cluster ...\n")
	// Capture list of resources from verrazzano-install and verrazzano-system namespaces
	err = captureResources(client, kubeClient, bugReportDir, vz, vzHelper, nsList, podLogs.Duration)
	if err != nil {
		pkghelpers.LogError(fmt.Sprintf("There is an error with capturing the Verrazzano resources: %s", err.Error()))
	}

	for _, e := range vzchecks.PrerequisiteCheck(client, vzchecks.ProfileType(vz.Items[0].Spec.Profile)) {
		fmt.Fprintf(vzHelper.GetOutputStream(), "Warning: "+e.Error()+"\n")
	}

	// Capture OAM resources from the namespaces specified using --include-namespaces
	if len(additionalNS) > 0 {
		checkadditionalResources(client, kubeClient, dynamicClient, vzHelper, bugReportDir, additionalNS, podLogs)
	}
	return nil
}

func captureResources(client clipkg.Client, kubeClient kubernetes.Interface, bugReportDir string, vz v1beta1.VerrazzanoList, vzHelper pkghelpers.VZHelper, namespaces []string, duration int64) error {
	// List of pods to collect the logs
	vpoPod, _ := pkghelpers.GetPodList(client, constants.AppLabel, constants.VerrazzanoPlatformOperator, vzconstants.VerrazzanoInstallNamespace)
	vaoPod, _ := pkghelpers.GetPodList(client, constants.AppLabel, constants.VerrazzanoApplicationOperator, vzconstants.VerrazzanoSystemNamespace)
	vmoPod, _ := pkghelpers.GetPodList(client, constants.K8SAppLabel, constants.VerrazzanoMonitoringOperator, vzconstants.VerrazzanoSystemNamespace)
	externaldnsPod, _ := pkghelpers.GetPodList(client, constants.K8sAppLabelExternalDNS, vzconstants.ExternalDNS, vzconstants.CertManager)
	wgCount := 3 + len(namespaces)
	if len(vz.Items) > 0 {
		wgCount++
	}
	if len(externaldnsPod) > 0 {
		wgCount++
	}
	wg := &sync.WaitGroup{}
	wg.Add(wgCount)

	// Define channels to get the errors
	evr := make(chan ErrorsChannel, 1)
	ecr := make(chan ErrorsChannel, 1)
	ecl := make(chan ErrorsChannelLogs, 1)

	if len(vz.Items) > 0 {
		go captureVZResource(wg, evr, vz, bugReportDir, vzHelper)
	}

	go captureLogs(wg, ecl, kubeClient, Pods{PodList: vpoPod, Namespace: vzconstants.VerrazzanoInstallNamespace}, bugReportDir, vzHelper, duration)
	go captureLogs(wg, ecl, kubeClient, Pods{PodList: vmoPod, Namespace: vzconstants.VerrazzanoSystemNamespace}, bugReportDir, vzHelper, duration)
	go captureLogs(wg, ecl, kubeClient, Pods{PodList: vaoPod, Namespace: vzconstants.VerrazzanoSystemNamespace}, bugReportDir, vzHelper, duration)

	if len(externaldnsPod) > 0 {
		go captureLogs(wg, ecl, kubeClient, Pods{PodList: externaldnsPod, Namespace: vzconstants.CertManager}, bugReportDir, vzHelper, duration)
	}
	for _, ns := range namespaces {
		go captureK8SResources(wg, ecr, kubeClient, ns, bugReportDir, vzHelper)
	}

	wg.Wait()
	close(ecl)
	close(ecr)
	close(evr)
	// Report errors (if any), in collecting the logs from various pods
	for err := range evr {
		return fmt.Errorf("an error occurred while capturing the Verrazzano resource, error: %s", err.ErrorMessage)
	}

	// Report errors (if any), in collecting the logs from various pods
	for err := range ecl {
		return fmt.Errorf("an error occurred while capturing the log for pod: %s, error: %s", err.PodName, err.ErrorMessage)
	}

	// Report errors (if any), in collecting resources from various namespaces
	for err := range ecr {
		return fmt.Errorf("an error occurred while capturing the resource, error: %s", err.ErrorMessage)
	}
	return nil
}

func captureResourcesCustom(client clipkg.Client, kubeClient kubernetes.Interface, bugReportDir string, vzHelper pkghelpers.VZHelper, namespaces []string, duration int64) error {

	wgCount := len(namespaces)
	wg := &sync.WaitGroup{}
	wg.Add(wgCount)
	// Define channels to get the errors
	evr := make(chan ErrorsChannel, 1)
	ecr := make(chan ErrorsChannel, 1)
	ecl := make(chan ErrorsChannelLogs, 1)
	for _, ns := range namespaces {
		podList, _ := pkghelpers.GetPodListAll(client, ns)
		go captureLogsAll(wg, ecl, kubeClient, Pods{PodList: podList, Namespace: ns}, bugReportDir, vzHelper, duration)
	}

	wg.Wait()
	close(ecl)
	close(ecr)
	close(evr)
	// Report errors (if any), in collecting the logs from various pods
	for err := range evr {
		return fmt.Errorf("an error occurred while capturing the Verrazzano resource, error: %s", err.ErrorMessage)
	}

	// Report errors (if any), in collecting the logs from various pods
	for err := range ecl {
		return fmt.Errorf("an error occurred while capturing the log for pod: %s, error: %s", err.PodName, err.ErrorMessage)
	}

	// Report errors (if any), in collecting resources from various namespaces
	for err := range ecr {
		return fmt.Errorf("an error occurred while capturing the resource, error: %s", err.ErrorMessage)
	}
	return nil
}

// captureVZResource collects the Verrazzano resource as a JSON, in parallel
func captureVZResource(wg *sync.WaitGroup, ec chan ErrorsChannel, vz v1beta1.VerrazzanoList, bugReportDir string, vzHelper pkghelpers.VZHelper) {
	defer wg.Done()
	err := pkghelpers.CaptureVZResource(bugReportDir, vz, vzHelper)
	if err != nil {
		ec <- ErrorsChannel{ErrorMessage: err.Error()}
	}
}

// captureLogs collects the logs from platform operator, application operator and monitoring operator in parallel
func captureLogs(wg *sync.WaitGroup, ec chan ErrorsChannelLogs, kubeClient kubernetes.Interface, pod Pods, bugReportDir string, vzHelper pkghelpers.VZHelper, duration int64) {
	defer wg.Done()
	if len(pod.PodList) == 0 {
		return
	}
	// This won't work when there are more than one pods for the same app label
	pkghelpers.LogMessage(fmt.Sprintf("log from pod %s in %s namespace ...\n", pod.PodList[0].Name, pod.Namespace))
	err := pkghelpers.CapturePodLog(kubeClient, pod.PodList[0], pod.Namespace, bugReportDir, vzHelper, duration)
	if err != nil {
		ec <- ErrorsChannelLogs{PodName: pod.PodList[0].Name, ErrorMessage: err.Error()}
	}

}

// captureK8SResources captures Kubernetes workloads, pods, events, ingresses and services from the list of namespaces in parallel
func captureK8SResources(wg *sync.WaitGroup, ec chan ErrorsChannel, kubeClient kubernetes.Interface, namespace, bugReportDir string, vzHelper pkghelpers.VZHelper) {
	defer wg.Done()
	if err := pkghelpers.CaptureK8SResources(kubeClient, namespace, bugReportDir, vzHelper); err != nil {
		ec <- ErrorsChannel{ErrorMessage: err.Error()}
	}
}

// collectNamespaces gathers list of unique namespaces, to be considered to collect the information
func collectNamespaces(kubeClient kubernetes.Interface, includedNS []string, vz v1beta1.VerrazzanoList, vzHelper pkghelpers.VZHelper) ([]string, []string) {

	var nsList []string

	// Include namespaces for all the components
	if len(vz.Items) != 0 {
		allCompNS := pkghelpers.GetNamespacesForAllComponents(vz.Items[0])
		nsList = append(nsList, allCompNS...)
	}

	// Include the namespaces specified by flag --include-namespaces
	var additionalNS []string
	if len(includedNS) > 0 {
		includedList := pkghelpers.RemoveDuplicate(includedNS)
		for _, ns := range includedList {
			nsExists, _ := pkghelpers.DoesNamespaceExist(kubeClient, ns, vzHelper)
			if nsExists {
				additionalNS = append(additionalNS, ns)
			}
		}
		nsList = append(nsList, additionalNS...)
	}

	// Remove the duplicates from nsList
	nsList = pkghelpers.RemoveDuplicate(nsList)
	return nsList, additionalNS
}

// capture customer namespace pods logs without filtering
func captureLogsAll(wg *sync.WaitGroup, ec chan ErrorsChannelLogs, kubeClient kubernetes.Interface, pods Pods, bugReportDir string, vzHelper pkghelpers.VZHelper, duration int64) {
	{
		defer wg.Done()
		if len(pods.PodList) == 0 {
			return
		}
		for index := range pods.PodList {
			pkghelpers.LogMessage(fmt.Sprintf("log from pod %s in %s namespace ...\n", pods.PodList[index].Name, pods.Namespace))
			err := pkghelpers.CapturePodLog(kubeClient, pods.PodList[index], pods.Namespace, bugReportDir, vzHelper, duration)
			if err != nil {
				ec <- ErrorsChannelLogs{PodName: pods.PodList[index].Name, ErrorMessage: err.Error()}
			}
		}
	}
}

// checkadditionalResources will capture additional resources with logs with duration.
func checkadditionalResources(client clipkg.Client, kubeClient kubernetes.Interface, dynamicClient dynamic.Interface, vzHelper pkghelpers.VZHelper, bugReportDir string, additionalNS []string, podLogs PodLogs) {
	if err := pkghelpers.CaptureOAMResources(dynamicClient, additionalNS, bugReportDir, vzHelper); err != nil {
		pkghelpers.LogError(fmt.Sprintf("There is an error in capturing the resources : %s", err.Error()))
	}
	if podLogs.IsPodLog {
		if err := captureResourcesCustom(client, kubeClient, bugReportDir, vzHelper, additionalNS, podLogs.Duration); err != nil {
			pkghelpers.LogError(fmt.Sprintf("There is an error with capturing the logs: %s", err.Error()))
		}
	}
	if err := pkghelpers.CaptureMultiClusterResources(dynamicClient, additionalNS, bugReportDir, vzHelper); err != nil {
		pkghelpers.LogError(fmt.Sprintf("There is an error in capturing the multi-cluster resources : %s", err.Error()))
	}
}
