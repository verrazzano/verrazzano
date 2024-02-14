// Copyright (c) 2022, 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package bugreport

import (
	"context"
	"encoding/json"
	"fmt"
	vzconstants "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	pkghelpers "github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
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

type VzComponentNamespaces struct {
	Name      string
	Namespace string
	Label     string
	PodList   pkghelpers.Pods
}

var vpoPod = VzComponentNamespaces{
	Name:      constants.VerrazzanoPlatformOperator,
	Namespace: vzconstants.VerrazzanoInstallNamespace,
	Label:     constants.AppLabel,
	PodList: pkghelpers.Pods{
		Namespace: vzconstants.VerrazzanoInstallNamespace,
		PodList:   nil,
	},
}

var vaoPod = VzComponentNamespaces{
	Name:      constants.VerrazzanoApplicationOperator,
	Namespace: vzconstants.VerrazzanoSystemNamespace,
	Label:     constants.AppLabel,
	PodList: pkghelpers.Pods{
		Namespace: vzconstants.VerrazzanoSystemNamespace,
		PodList:   nil,
	},
}

var vcoPod = VzComponentNamespaces{
	Name:      constants.VerrazzanoClusterOperator,
	Namespace: vzconstants.VerrazzanoSystemNamespace,
	Label:     constants.AppLabel,
	PodList: pkghelpers.Pods{
		Namespace: vzconstants.VerrazzanoSystemNamespace,
		PodList:   nil,
	},
}

var vmoPod = VzComponentNamespaces{
	Name:      constants.VerrazzanoMonitoringOperator,
	Namespace: vzconstants.VerrazzanoSystemNamespace,
	Label:     constants.K8SAppLabel,
	PodList: pkghelpers.Pods{
		Namespace: vzconstants.VerrazzanoSystemNamespace,
		PodList:   nil,
	},
}

var vpoWebHookPod = VzComponentNamespaces{
	Name:      constants.VerrazzanoPlatformOperatorWebhook,
	Namespace: vzconstants.VerrazzanoInstallNamespace,
	Label:     constants.AppLabel,
	PodList: pkghelpers.Pods{
		Namespace: vzconstants.VerrazzanoInstallNamespace,
		PodList:   nil,
	},
}

var externalDNSPod = VzComponentNamespaces{
	Name:      vzconstants.ExternalDNS,
	Namespace: vzconstants.CertManager,
	Label:     constants.K8sAppLabelExternalDNS,
	PodList: pkghelpers.Pods{
		Namespace: vzconstants.CertManager,
		PodList:   nil,
	},
}

var DefaultPodLog = pkghelpers.PodLogs{
	IsPodLog:   false,
	IsPrevious: false,
	Duration:   0,
}

const istioSidecarStatus = "sidecar.istio.io/status"

// CaptureClusterSnapshot selectively captures the resources from the cluster, useful to analyze the issue.
func CaptureClusterSnapshot(kubeClient kubernetes.Interface, dynamicClient dynamic.Interface, client clipkg.Client, vzHelper pkghelpers.VZHelper, podLogs pkghelpers.PodLogs, clusterSnapshotCtx pkghelpers.ClusterSnapshotCtx) error {

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
	nsList, _, err := collectNamespaces(kubeClient, dynamicClient, clusterSnapshotCtx.MoreNS, vz, vzHelper)
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
	err = captureResources(client, kubeClient, dynamicClient, clusterSnapshotCtx.BugReportDir, vz, vzHelper, nsList)
	if err != nil {
		pkghelpers.LogError(fmt.Sprintf("There is an error with capturing the Verrazzano resources: %s", err.Error()))
	}

	// Capture logs from resources when the --include-logs flag
	captureAdditionalResources(client, kubeClient, dynamicClient, vzHelper, clusterSnapshotCtx.BugReportDir, nsList, podLogs)

	// flag pods that are missing sidecar containers
	err = flagMissingSidecarContainers(client, kubeClient)

	// find problematic pods from captured resources
	podNameNamespaces, err := pkghelpers.FindProblematicPods(clusterSnapshotCtx.BugReportDir)
	if err != nil {
		return err
	}
	err = captureProblematicPodLogs(kubeClient, clusterSnapshotCtx.BugReportDir, vzHelper, podNameNamespaces)
	if err != nil {
		return err
	}

	// Capture Verrazzano Projects and VerrazzanoManagedCluster
	if err = captureMultiClusterResources(dynamicClient, clusterSnapshotCtx.BugReportDir, vzHelper); err != nil {
		return err
	}

	// Capture global CAPI resources
	if err = pkghelpers.CaptureGlobalCapiResources(dynamicClient, clusterSnapshotCtx.BugReportDir, vzHelper); err != nil {
		return err
	}

	if err := pkghelpers.CaptureMetadata(clusterSnapshotCtx.BugReportDir); err != nil {
		return err
	}

	// Capture global Rancher resources
	if err = pkghelpers.CaptureGlobalRancherResources(dynamicClient, clusterSnapshotCtx.BugReportDir, vzHelper); err != nil {
		return err
	}
	return nil
}

func captureResources(client clipkg.Client, kubeClient kubernetes.Interface, dynamicClient dynamic.Interface, bugReportDir string, vz *v1beta1.Verrazzano, vzHelper pkghelpers.VZHelper, namespaces []string) error {
	// List of pods to collect the logs
	podsToCollect := []VzComponentNamespaces{vpoPod, vaoPod, vcoPod, vmoPod, vpoWebHookPod}
	for i, component := range podsToCollect {
		podList, _ := pkghelpers.GetPodList(client, component.Label, component.Name, component.Namespace)
		podsToCollect[i].PodList.PodList = podList
	}
	externalDNSPod.PodList.PodList, _ = pkghelpers.GetPodList(client, externalDNSPod.Label, externalDNSPod.Name, externalDNSPod.Namespace)

	wgCount := 5 + len(namespaces)
	wgCount++ // increment for the verrrazzano resource
	if len(externalDNSPod.PodList.PodList) > 0 {
		wgCount++
	}
	wg := &sync.WaitGroup{}
	wg.Add(wgCount)

	// Define channels to get the errors
	evr := make(chan pkghelpers.ErrorsChannel, 1)
	ecr := make(chan pkghelpers.ErrorsChannel, 1)
	ecl := make(chan pkghelpers.ErrorsChannelLogs, 1)

	go captureVZResource(wg, evr, vz, bugReportDir)

	for _, podList := range podsToCollect {
		go pkghelpers.CaptureLogs(wg, ecl, kubeClient, pkghelpers.Pods{Namespace: podList.Namespace, PodList: podList.PodList.PodList}, bugReportDir, vzHelper, DefaultPodLog)
	}
	if len(externalDNSPod.PodList.PodList) > 0 {
		go pkghelpers.CaptureLogs(wg, ecl, kubeClient, pkghelpers.Pods{Namespace: externalDNSPod.Namespace, PodList: externalDNSPod.PodList.PodList}, bugReportDir, vzHelper, DefaultPodLog)
	}

	for _, ns := range namespaces {
		go captureK8SResources(wg, ecr, client, kubeClient, dynamicClient, ns, bugReportDir, vzHelper)
	}

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
func captureAdditionalLogs(client clipkg.Client, kubeClient kubernetes.Interface, bugReportDir string, vzHelper pkghelpers.VZHelper, namespaces []string, podLogs pkghelpers.PodLogs) error {
	wgCount := len(namespaces)
	wg := &sync.WaitGroup{}
	wg.Add(wgCount)
	// Define channels to get the errors
	evr := make(chan pkghelpers.ErrorsChannel, 1)
	ecr := make(chan pkghelpers.ErrorsChannel, 1)
	ecl := make(chan pkghelpers.ErrorsChannelLogs, 1)
	for _, ns := range namespaces {
		podList, _ := pkghelpers.GetPodListAll(client, ns)
		go captureLogsAllPods(wg, ecl, kubeClient, pkghelpers.Pods{PodList: podList, Namespace: ns}, bugReportDir, vzHelper, podLogs)
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
func captureVZResource(wg *sync.WaitGroup, ec chan pkghelpers.ErrorsChannel, vz *v1beta1.Verrazzano, bugReportDir string) {
	defer wg.Done()
	err := pkghelpers.CaptureVZResource(bugReportDir, vz)
	if err != nil {
		ec <- pkghelpers.ErrorsChannel{ErrorMessage: err.Error()}
	}
}

// captureK8SResources captures Kubernetes workloads, pods, events, ingresses and services from the list of namespaces in parallel
func captureK8SResources(wg *sync.WaitGroup, ec chan pkghelpers.ErrorsChannel, client clipkg.Client, kubeClient kubernetes.Interface, dynamicClient dynamic.Interface, namespace, bugReportDir string, vzHelper pkghelpers.VZHelper) {
	defer wg.Done()
	if err := pkghelpers.CaptureK8SResources(client, kubeClient, dynamicClient, namespace, bugReportDir, vzHelper); err != nil {
		ec <- pkghelpers.ErrorsChannel{ErrorMessage: err.Error()}
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
func captureLogsAllPods(wg *sync.WaitGroup, ec chan pkghelpers.ErrorsChannelLogs, kubeClient kubernetes.Interface, pods pkghelpers.Pods, bugReportDir string, vzHelper pkghelpers.VZHelper, podLogs pkghelpers.PodLogs) {

	defer wg.Done()
	if len(pods.PodList) == 0 {
		return
	}
	for index := range pods.PodList {
		pkghelpers.LogMessage(fmt.Sprintf("log from pod %s in %s namespace ...\n", pods.PodList[index].Name, pods.Namespace))
		err := pkghelpers.CapturePodLog(kubeClient, pods.PodList[index], pods.Namespace, bugReportDir, vzHelper, podLogs.Duration, podLogs.IsPrevious)
		if err != nil {
			ec <- pkghelpers.ErrorsChannelLogs{PodName: pods.PodList[index].Name, ErrorMessage: err.Error()}
		}
	}
}

// captureAdditionalResources will capture additional resources from additional namespaces
func captureAdditionalResources(client clipkg.Client, kubeClient kubernetes.Interface, dynamicClient dynamic.Interface, vzHelper pkghelpers.VZHelper, bugReportDir string, additionalNS []string, podLogs pkghelpers.PodLogs) {
	if err := pkghelpers.CaptureOAMResources(dynamicClient, additionalNS, bugReportDir, vzHelper); err != nil {
		pkghelpers.LogError(fmt.Sprintf("There is an error in capturing the resources : %s", err.Error()))
	}
	if podLogs.IsPodLog {
		if err := captureAdditionalLogs(client, kubeClient, bugReportDir, vzHelper, additionalNS, podLogs); err != nil {
			pkghelpers.LogError(fmt.Sprintf("There is an error with capturing the logs: %s", err.Error()))
		}
	}
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

// flagMissingSidecarContainers identifies pods in namespaces with --label istio-injection=enabled that are missing sidecar containers
func flagMissingSidecarContainers(client clipkg.Client, kubeClient kubernetes.Interface) (error error) {
	labelSelector := metav1.LabelSelector{MatchLabels: map[string]string{vzconstants.LabelIstioInjection: "enabled"}}
	listOptions := metav1.ListOptions{LabelSelector: labels.Set(labelSelector.MatchLabels).String()}
	namespaceList, err := kubeClient.CoreV1().Namespaces().List(context.TODO(), listOptions)
	if err != nil {
		return err
	}
	for _, namespace := range namespaceList.Items {
		podList, err := pkghelpers.GetPodListAll(client, namespace.Name)
		if err != nil {
			return err
		}
		err = findMissingSidecarContainers(podList)
		if err != nil {
			return err
		}
	}
	return nil
}

// findMissingSidecarContainers identifies pods that are missing sidecar containers
func findMissingSidecarContainers(pods []corev1.Pod) (error error) {
	for _, pod := range pods {
		if pod.Annotations[istioSidecarStatus] == "" {
			continue
		}
		sidecarContainers := pod.Annotations[istioSidecarStatus]
		var obj map[string]interface{}
		err := json.Unmarshal([]byte(sidecarContainers), &obj)
		if err != nil {
			return err
		}
		sidecarContainersFromAnnotation := obj["containers"]
		containersFromPod := pod.Spec.Containers

		for _, sidecar := range sidecarContainersFromAnnotation.([]interface{}) {
			for i, container := range containersFromPod {
				if sidecar == container.Name {
					continue
				}
				if i+1 == len(containersFromPod) && sidecar != container.Name {
					pkghelpers.LogError(fmt.Sprintf("Sidecar container: %s, was not found for pod: %s, in namespace %s\n", sidecar, pod.Name, pod.Namespace))
				}
			}
		}
	}
	return nil
}

// captureProblematicPodLogs tries to capture previous logs for any problematic pods
func captureProblematicPodLogs(kubeClient kubernetes.Interface, bugReportDir string, vzHelper pkghelpers.VZHelper, podNameNamespaces map[string][]corev1.Pod) error {
	if len(podNameNamespaces) != 0 {
		for namespace := range podNameNamespaces {
			for _, pod := range podNameNamespaces[namespace] {
				_ = pkghelpers.CapturePodLog(kubeClient, pod, namespace, bugReportDir, vzHelper, 0, true)
			}
		}
	}
	return nil
}
