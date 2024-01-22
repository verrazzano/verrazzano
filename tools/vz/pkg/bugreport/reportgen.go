// Copyright (c) 2022, 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package bugreport

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	vzconstants "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	pkghelpers "github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
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
func CaptureClusterSnapshot(kubeClient kubernetes.Interface, dynamicClient dynamic.Interface, client clipkg.Client, vzHelper pkghelpers.VZHelper, podLogs PodLogs, clusterSnapshotCtx pkghelpers.ClusterSnapshotCtx) error {

	// Create a file to capture the standard out to a file
	stdOutFile, err := os.OpenFile(filepath.Join(clusterSnapshotCtx.BugReportDir, constants.BugReportOut), os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		return fmt.Errorf("an error occurred while creating the file include the summary of the resources captured: %s", err.Error())
	}
	defer stdOutFile.Close()

	// Create a file to capture the standard err to a file
	stdErrFile, err := os.OpenFile(filepath.Join(clusterSnapshotCtx.BugReportDir, constants.BugReportErr), os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		return fmt.Errorf("an error occurred while creating the file include the summary of the resources captured: %s", err.Error())
	}
	defer stdErrFile.Close()

	// Create MultiWriters for standard out and err
	pkghelpers.SetMultiWriterOut(vzHelper.GetOutputStream(), stdOutFile)
	pkghelpers.SetMultiWriterErr(vzHelper.GetErrorStream(), stdErrFile)

	// Find the Verrazzano resource to analyze.
	vz, err := pkghelpers.FindVerrazzanoResource(client)
	if err != nil {
		pkghelpers.LogMessage(fmt.Sprintf("Verrazzano is not installed: %s", err.Error()))
	}

	// Get the list of namespaces based on the failed components and value specified by flag --include-namespaces
	nsList, additionalNS, err := collectNamespaces(kubeClient, dynamicClient, clusterSnapshotCtx.MoreNS, vz, vzHelper)
	if err != nil {
		return err
	}
	var msgPrefix string
	if pkghelpers.GetIsLiveCluster() {
		msgPrefix = constants.AnalysisMsgPrefix
	} else {
		msgPrefix = constants.BugReportMsgPrefix
	}
	if clusterSnapshotCtx.PrintReportToConsole {
		// Print initial message to console output only
		fmt.Fprintf(vzHelper.GetOutputStream(), "\n"+msgPrefix+"resources from the cluster ...\n")
	}
	// Capture list of resources from verrazzano-install and verrazzano-system namespaces
	err = captureResources(client, kubeClient, dynamicClient, clusterSnapshotCtx.BugReportDir, vz, vzHelper, nsList, podLogs)
	if err != nil {
		pkghelpers.LogError(fmt.Sprintf("There is an error with capturing the Verrazzano resources: %s", err.Error()))
	}

	// Capture OAM resources from the namespaces specified using --include-namespaces
	if len(additionalNS) > 0 {
		captureAdditionalResources(client, kubeClient, dynamicClient, vzHelper, clusterSnapshotCtx.BugReportDir, additionalNS, podLogs)
	}

	// Capture Verrazzano Projects and VerrazzanoManagedCluster
	if err = captureMultiClusterResources(dynamicClient, clusterSnapshotCtx.BugReportDir, vzHelper); err != nil {
		return err
	}

	// Capture global CAPI resources
	if err = pkghelpers.CaptureGlobalCapiResources(dynamicClient, clusterSnapshotCtx.BugReportDir, vzHelper); err != nil {
		return err
	}

	// Capture global Rancher resources
	if err = pkghelpers.CaptureGlobalRancherResources(dynamicClient, clusterSnapshotCtx.BugReportDir, vzHelper); err != nil {
		return err
	}
	return nil
}

func captureResources(client clipkg.Client, kubeClient kubernetes.Interface, dynamicClient dynamic.Interface, bugReportDir string, vz *v1beta1.Verrazzano, vzHelper pkghelpers.VZHelper, namespaces []string, podLogs PodLogs) error {
	// List of pods to collect the logs
	vpoPod, _ := pkghelpers.GetPodList(client, constants.AppLabel, constants.VerrazzanoPlatformOperator, vzconstants.VerrazzanoInstallNamespace)
	vaoPod, _ := pkghelpers.GetPodList(client, constants.AppLabel, constants.VerrazzanoApplicationOperator, vzconstants.VerrazzanoSystemNamespace)
	vcoPod, _ := pkghelpers.GetPodList(client, constants.AppLabel, constants.VerrazzanoClusterOperator, vzconstants.VerrazzanoSystemNamespace)
	vmoPod, _ := pkghelpers.GetPodList(client, constants.K8SAppLabel, constants.VerrazzanoMonitoringOperator, vzconstants.VerrazzanoSystemNamespace)
	vpoWebHookPod, _ := pkghelpers.GetPodList(client, constants.AppLabel, constants.VerrazzanoPlatformOperatorWebhook, vzconstants.VerrazzanoInstallNamespace)
	externalDNSPod, _ := pkghelpers.GetPodList(client, constants.K8sAppLabelExternalDNS, vzconstants.ExternalDNS, vzconstants.CertManager)
	wgCount := 5 + len(namespaces)
	wgCount++ // increment for the verrrazzano resource
	if len(externalDNSPod) > 0 {
		wgCount++
	}
	wg := &sync.WaitGroup{}
	wg.Add(wgCount)

	// Define channels to get the errors
	evr := make(chan ErrorsChannel, 1)
	ecr := make(chan ErrorsChannel, 1)
	ecl := make(chan ErrorsChannelLogs, 1)

	go captureVZResource(wg, evr, vz, bugReportDir)

	go captureLogs(wg, ecl, kubeClient, Pods{PodList: vpoPod, Namespace: vzconstants.VerrazzanoInstallNamespace}, bugReportDir, vzHelper, 0)
	go captureLogs(wg, ecl, kubeClient, Pods{PodList: vpoWebHookPod, Namespace: vzconstants.VerrazzanoInstallNamespace}, bugReportDir, vzHelper, 0)
	go captureLogs(wg, ecl, kubeClient, Pods{PodList: vmoPod, Namespace: vzconstants.VerrazzanoSystemNamespace}, bugReportDir, vzHelper, 0)
	go captureLogs(wg, ecl, kubeClient, Pods{PodList: vaoPod, Namespace: vzconstants.VerrazzanoSystemNamespace}, bugReportDir, vzHelper, 0)
	go captureLogs(wg, ecl, kubeClient, Pods{PodList: vcoPod, Namespace: vzconstants.VerrazzanoSystemNamespace}, bugReportDir, vzHelper, 0)

	if len(externalDNSPod) > 0 {
		go captureLogs(wg, ecl, kubeClient, Pods{PodList: externalDNSPod, Namespace: vzconstants.CertManager}, bugReportDir, vzHelper, 0)
	}
	for _, ns := range namespaces {
		go captureK8SResources(wg, ecr, client, kubeClient, dynamicClient, ns, bugReportDir, vzHelper)
	}
	// captures pod logs of resources in namespaces if --include-logs flag is enabled
	capturePodLogs(client, kubeClient, vzHelper, bugReportDir, namespaces, podLogs)

	wg.Wait()
	close(ecl)
	close(ecr)
	close(evr)
	// Report errors (if any), in capturing the verrazzano resource
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

// captureAdditionalLogs will be used for capture logs from additional namespace.
func captureAdditionalLogs(client clipkg.Client, kubeClient kubernetes.Interface, bugReportDir string, vzHelper pkghelpers.VZHelper, namespaces []string, duration int64) error {
	wgCount := len(namespaces)
	wg := &sync.WaitGroup{}
	wg.Add(wgCount)
	// Define channels to get the errors
	evr := make(chan ErrorsChannel, 1)
	ecr := make(chan ErrorsChannel, 1)
	ecl := make(chan ErrorsChannelLogs, 1)
	for _, ns := range namespaces {
		podList, _ := pkghelpers.GetPodListAll(client, ns)
		go captureLogsAllPods(wg, ecl, kubeClient, Pods{PodList: podList, Namespace: ns}, bugReportDir, vzHelper, duration)
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
func captureVZResource(wg *sync.WaitGroup, ec chan ErrorsChannel, vz *v1beta1.Verrazzano, bugReportDir string) {
	defer wg.Done()
	err := pkghelpers.CaptureVZResource(bugReportDir, vz)
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
func captureK8SResources(wg *sync.WaitGroup, ec chan ErrorsChannel, client clipkg.Client, kubeClient kubernetes.Interface, dynamicClient dynamic.Interface, namespace, bugReportDir string, vzHelper pkghelpers.VZHelper) {
	defer wg.Done()
	if err := pkghelpers.CaptureK8SResources(client, kubeClient, dynamicClient, namespace, bugReportDir, vzHelper); err != nil {
		ec <- ErrorsChannel{ErrorMessage: err.Error()}
	}
}

// collectNamespaces gathers list of unique namespaces, to be considered to collect the information
func collectNamespaces(kubeClient kubernetes.Interface, dynamicClient dynamic.Interface, includedNS []string, vz *v1beta1.Verrazzano, vzHelper pkghelpers.VZHelper) ([]string, []string, error) {

	var nsList []string

	// Include namespaces for all the vz components
	allCompNS := pkghelpers.GetNamespacesForAllComponents(vz)
	nsList = append(nsList, allCompNS...)

	// Verify and Include verrazzano-install namespace
	if pkghelpers.VerifyVzInstallNamespaceExists(kubeClient) {
		nsList = append(nsList, vzconstants.VerrazzanoInstallNamespace)
	}

	// Add any namespaces that have CAPI clusters
	capiNSList, err := getCAPIClusterNamespaces(kubeClient, dynamicClient)
	if err != nil {
		return nil, nil, err
	}
	nsList = append(nsList, capiNSList...)

	// Add Rancher namespaces
	rancherNSList, err := getRancherNamespaces(kubeClient, dynamicClient)
	if err != nil {
		return nil, nil, err
	}
	nsList = append(nsList, rancherNSList...)

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
	return nsList, additionalNS, nil
}

// This function returns a list of namespaces that have a CAPI cluster resource.
// We want to always capture these resources.
func getCAPIClusterNamespaces(kubeClient kubernetes.Interface, dynamicClient dynamic.Interface) ([]string, error) {
	namespaces, err := kubeClient.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	nsList := []string{}
	gvr := schema.GroupVersionResource{Group: "cluster.x-k8s.io", Version: "v1beta1", Resource: "clusters"}
	for _, namespace := range namespaces.Items {
		list, err := dynamicClient.Resource(gvr).Namespace(namespace.Name).List(context.TODO(), metav1.ListOptions{})
		// Resource type does not exist, return here since there will be no "cluster" resources.
		// This will be the case if the cluster-api component is not installed.
		if errors.IsNotFound(err) {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		if len(list.Items) > 0 {
			nsList = append(nsList, namespace.Name)
		}
	}
	return nsList, nil
}

// This function returns a list of namespaces that have a Rancher annotation.
// We want to always capture these resources.
func getRancherNamespaces(kubeClient kubernetes.Interface, dynamicClient dynamic.Interface) ([]string, error) {
	namespaces, err := kubeClient.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	nsList := []string{}
	for _, namespace := range namespaces.Items {
		if namespace.Annotations["lifecycle.cattle.io/create.namespace-auth"] == "true" {
			nsList = append(nsList, namespace.Name)
		}
	}
	return nsList, nil
}

// captureLogsAllPods captures logs from all pods without filtering in given namespace.
func captureLogsAllPods(wg *sync.WaitGroup, ec chan ErrorsChannelLogs, kubeClient kubernetes.Interface, pods Pods, bugReportDir string, vzHelper pkghelpers.VZHelper, duration int64) {

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

// captureAdditionalResources will capture additional resources from additional namespaces
func captureAdditionalResources(client clipkg.Client, kubeClient kubernetes.Interface, dynamicClient dynamic.Interface, vzHelper pkghelpers.VZHelper, bugReportDir string, additionalNS []string, podLogs PodLogs) {
	if err := pkghelpers.CaptureOAMResources(dynamicClient, additionalNS, bugReportDir, vzHelper); err != nil {
		pkghelpers.LogError(fmt.Sprintf("There is an error in capturing the resources : %s", err.Error()))
	}
	// capturePodLogs gets pod logs if the --include-logs  flag is enabled
	capturePodLogs(client, kubeClient, vzHelper, bugReportDir, additionalNS, podLogs)
	if err := pkghelpers.CaptureMultiClusterOAMResources(dynamicClient, additionalNS, bugReportDir, vzHelper); err != nil {
		pkghelpers.LogError(fmt.Sprintf("There is an error in capturing the multi-cluster resources : %s", err.Error()))
	}
}

// captureMultiClusterResources captures Projects and VerrazzanoManagedCluster resource
func captureMultiClusterResources(dynamicClient dynamic.Interface, captureDir string, vzHelper pkghelpers.VZHelper) error {
	// Return nil when dynamicClient is nil, useful to get clean unit tests
	if dynamicClient == nil {
		return nil
	}

	// Capture Verrazzano projects in verrazzano-mc namespace
	if err := pkghelpers.CaptureVerrazzanoProjects(dynamicClient, captureDir, vzHelper); err != nil {
		return err
	}

	// Capture Verrazzano projects in verrazzano-mc namespace
	if err := pkghelpers.CaptureVerrazzanoManagedCluster(dynamicClient, captureDir, vzHelper); err != nil {
		return err
	}
	return nil
}

// capturePodLogs gets pod logs if the --include-logs flag is enabled
func capturePodLogs(client clipkg.Client, kubeClient kubernetes.Interface, vzHelper pkghelpers.VZHelper, bugReportDir string, additionalNS []string, podLogs PodLogs) {
	if podLogs.IsPodLog {
		if err := captureAdditionalLogs(client, kubeClient, bugReportDir, vzHelper, additionalNS, podLogs.Duration); err != nil {
			pkghelpers.LogError(fmt.Sprintf("There is an error with capturing the logs: %s", err.Error()))
		}
	}
}
