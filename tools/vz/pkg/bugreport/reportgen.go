// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package bugreport

import (
	"context"
	"fmt"
	vzconstants "github.com/verrazzano/verrazzano/pkg/constants"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	pkghelpers "github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	"io"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"os"
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

// GenerateBugReport creates a bug report by including the resources selectively from the cluster, useful to analyze the issue.
func GenerateBugReport(kubeClient kubernetes.Interface, dynamicClient dynamic.Interface, client clipkg.Client, bugReportFile *os.File, moreNS []string, vzHelper pkghelpers.VZHelper) error {
	// Create a temporary directory to place the cluster data
	bugReportDir, err := ioutil.TempDir("", constants.BugReportDir)
	if err != nil {
		return fmt.Errorf("an error occurred while creating the directory to place cluster resources: %s", err.Error())
	}
	defer os.RemoveAll(bugReportDir)

	// Verrazzano as a list is required for the analysis tool
	vz := vzapi.VerrazzanoList{}
	err = client.List(context.TODO(), &vz)
	if (err != nil && len(vz.Items) == 0) || len(vz.Items) == 0 {
		fmt.Fprintf(vzHelper.GetOutputStream(), "Verrazzano is not installed ...\n")
		return nil
	}

	// Get the list of namespaces based on the failed components and value specified by flag --include-namespaces
	nsList, additionalNS := collectNamespaces(kubeClient, moreNS, vz, vzHelper)

	// Capture list of resources from verrazzano-install and verrazzano-system namespaces
	err = captureResources(client, kubeClient, bugReportDir, vz, vzHelper, nsList)
	if err != nil {
		fmt.Fprintf(vzHelper.GetErrorStream(), "There is an error with capturing the Verrazzano resources: %s", err.Error())
	}

	// Capture OAM resources from the namespaces specified using --include-namespaces
	if len(additionalNS) > 0 {
		if err := pkghelpers.CaptureOAMResources(dynamicClient, additionalNS, bugReportDir, vzHelper); err != nil {
			fmt.Fprintf(vzHelper.GetErrorStream(), "There is an error in capturing the resources : %s", err.Error())
		}
		if err := pkghelpers.CaptureMultiClusterResources(dynamicClient, additionalNS, bugReportDir, vzHelper); err != nil {
			fmt.Fprintf(vzHelper.GetErrorStream(), "There is an error in capturing the multi-cluster resources : %s", err.Error())
		}
	}

	// Return an error when the command fails to collect anything from the cluster
	if isDirEmpty(bugReportDir) {
		return fmt.Errorf("The bug-report command did not collect any file from the cluster. " +
			"Please go through errors (if any), in the standard output.\n")
	}

	// Create the report file
	err = pkghelpers.CreateReportArchive(bugReportDir, bugReportFile)
	if err != nil {
		return fmt.Errorf("there is an error in creating the bug report, %s", err.Error())
	}
	return nil
}

// captureResources captures the resources from various namespaces, resources are collected in parallel as appropriate
func captureResources(client clipkg.Client, kubeClient kubernetes.Interface, bugReportDir string, vz vzapi.VerrazzanoList, vzHelper pkghelpers.VZHelper, namespaces []string) error {

	// List of pods to collect the logs
	vpoPod, _ := pkghelpers.GetPodList(client, constants.AppLabel, constants.VerrazzanoPlatformOperator, vzconstants.VerrazzanoInstallNamespace)
	vaoPod, _ := pkghelpers.GetPodList(client, constants.AppLabel, constants.VerrazzanoApplicationOperator, vzconstants.VerrazzanoSystemNamespace)
	vmoPod, _ := pkghelpers.GetPodList(client, constants.K8SAppLabel, constants.VerrazzanoMonitoringOperator, vzconstants.VerrazzanoSystemNamespace)

	// Define WaitGroups for the parallel executions
	wgCount := 3 + len(namespaces)
	if len(vz.Items) > 0 {
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

	go captureLogs(wg, ecl, kubeClient, vpoPod, vzconstants.VerrazzanoInstallNamespace, bugReportDir, vzHelper)
	go captureLogs(wg, ecl, kubeClient, vaoPod, vzconstants.VerrazzanoSystemNamespace, bugReportDir, vzHelper)
	go captureLogs(wg, ecl, kubeClient, vmoPod, vzconstants.VerrazzanoSystemNamespace, bugReportDir, vzHelper)

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

// captureVZResource collects the Verrazzano resource as a JSON, in parallel
func captureVZResource(wg *sync.WaitGroup, ec chan ErrorsChannel, vz vzapi.VerrazzanoList, bugReportDir string, vzHelper pkghelpers.VZHelper) {
	defer wg.Done()
	err := pkghelpers.CaptureVZResource(bugReportDir, vz, vzHelper)
	if err != nil {
		ec <- ErrorsChannel{ErrorMessage: err.Error()}
	}
}

// captureLogs collects the logs from platform operator, application operator and monitoring operator in parallel
func captureLogs(wg *sync.WaitGroup, ec chan ErrorsChannelLogs, kubeClient kubernetes.Interface, pods []corev1.Pod, namespace, bugReportDir string, vzHelper pkghelpers.VZHelper) {
	defer wg.Done()
	if len(pods) == 0 {
		return
	}
	// This won't work when there are more than one pods for the same app label
	fmt.Fprintf(vzHelper.GetOutputStream(), fmt.Sprintf("Capturing log from pod %s in %s namespace ...\n", pods[0].Name, namespace))
	err := pkghelpers.CapturePodLog(kubeClient, pods[0], namespace, bugReportDir, vzHelper)
	if err != nil {
		ec <- ErrorsChannelLogs{PodName: pods[0].Name, ErrorMessage: err.Error()}
	}
}

// captureK8SResources captures Kubernetes workloads, pods, events, ingresses and services from the list of namespaces in parallel
func captureK8SResources(wg *sync.WaitGroup, ec chan ErrorsChannel, kubeClient kubernetes.Interface, namespace, bugReportDir string, vzHelper pkghelpers.VZHelper) {
	defer wg.Done()
	if err := pkghelpers.CaptureK8SResources(kubeClient, namespace, bugReportDir, vzHelper); err != nil {
		ec <- ErrorsChannel{ErrorMessage: err.Error()}
	}
}

// isDirEmpty returns whether the directory is empty or not
func isDirEmpty(directory string) bool {
	d, err := os.Open(directory)
	if err != nil {
		return false
	}
	defer d.Close()

	_, err = d.Readdirnames(1)
	return err == io.EOF
}

// collectNamespaces gathers list of unique namespaces, to be considered to collect the information
func collectNamespaces(kubeClient kubernetes.Interface, includedNS []string, vz vzapi.VerrazzanoList, vzHelper pkghelpers.VZHelper) ([]string, []string) {

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
