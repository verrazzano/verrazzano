// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package bugreport

import (
	"fmt"
	vzconstants "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	pkghelpers "github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	"io"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"os"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"sync"
)

type ErrorsChannel struct {
	PodName      string `json:"podName"`
	ErrorMessage string `json:"errorMessage"`
}

// The bug-report command captures the following resources from the cluster by default
// - Verrazzano resource
// - Logs from verrazzano-platform-operator, verrazzano-monitoring-operator and verrazzano-application-operator pods
// - Workloads (Deployment and ReplicaSet, StatefulSet, Daemonset), pods, events, ingress and services from verrazzano-system namespace.

// GenerateBugReport creates a bug report by including the resources selectively from the cluster, useful to analyze the issue.
func GenerateBugReport(kubeClient kubernetes.Interface, client clipkg.Client, bugReportFile *os.File, vzHelper pkghelpers.VZHelper) error {

	// Create a temporary directory to place the cluster data
	bugReportDir, err := ioutil.TempDir("", constants.BugReportDir)
	if err != nil {
		return fmt.Errorf("an error occurred while creating the directory to place cluster resources: %s", err.Error())
	}
	defer os.RemoveAll(bugReportDir)

	// Capture list of resources from verrazzano-install and verrazzano-system namespaces
	err = captureVerrazzanoResources(client, kubeClient, bugReportDir, vzHelper.GetOutputStream())
	if err != nil {
		return fmt.Errorf("there is an error with capturing the resources: %s", err.Error())
	}

	// Placeholder to call a method to capture the information from the namespaces/pods/containers
	// explicitly provided by the end user, using flag(s) to be supported by the command

	// Placeholder to call a function to redact sensitive information from all the files in bugReportDir

	// Create the report file
	err = pkghelpers.CreateReportArchive(bugReportDir, bugReportFile)
	if err != nil {
		return fmt.Errorf("there is an error in creating the bug report, %s", err.Error())
	}

	fmt.Fprintf(vzHelper.GetOutputStream(), fmt.Sprintf("Successfully created the bug report: %s\n", bugReportFile.Name()))

	// Display a warning message to review the contents of the report
	fmt.Fprint(vzHelper.GetOutputStream(), "WARNING: Please examine the contents of the bug report for sensitive data.\n")

	return nil
}

// captureVerrazzanoResources captures the resources from verrazzano-install and verrazzano-system namespaces
func captureVerrazzanoResources(client clipkg.Client, kubeClient kubernetes.Interface, bugReportDir string, outputStream io.Writer) error {

	var nameSpaces []string

	// Capture Verrazzano resource as JSON
	if err := pkghelpers.CaptureVZResource(client, bugReportDir, outputStream); err != nil {
		return err
	}

	// Capture logs from pods
	if err := capturePodLogs(client, kubeClient, bugReportDir, outputStream); err != nil {
		return err
	}

	nameSpaces, err := pkghelpers.GetAllUniqueNameSpacesForComponentsNotReady(client)
	if err != nil {
		return err
	}
	nameSpaces = append(nameSpaces, vzconstants.VerrazzanoSystemNamespace)

	// Capture workloads, pods, events, ingress and services in verrazzano-system namespace
	if err := pkghelpers.CaptureK8SResources(kubeClient, nameSpaces, bugReportDir); err != nil {
		return err
	}

	//TODO: Capture ingress from keycloak and cattle-system namespace
	// Capture workloads, events, services, etc from those namespaces for the components which are not ready
	return nil
}

// Captures logs from platform operator, application operator and monitoring operator
func capturePodLogs(client clipkg.Client, kubeClient kubernetes.Interface, bugReportDir string, outStream io.Writer) error {

	// Fixed list of pods for which, capture the log
	vpoPod, _ := pkghelpers.GetPodList(client, constants.AppLabel, constants.VerrazzanoPlatformOperator, vzconstants.VerrazzanoInstallNamespace)
	vaoPod, _ := pkghelpers.GetPodList(client, constants.AppLabel, constants.VerrazzanoApplicationOperator, vzconstants.VerrazzanoSystemNamespace)
	vmoPod, _ := pkghelpers.GetPodList(client, constants.K8SAppLabel, constants.VerrazzanoMonitoringOperator, vzconstants.VerrazzanoSystemNamespace)

	wg := &sync.WaitGroup{}
	wg.Add(3)
	ec := make(chan ErrorsChannel, 1)

	go captureLogsInParallel(wg, ec, kubeClient, vpoPod, vzconstants.VerrazzanoInstallNamespace, bugReportDir, outStream)
	go captureLogsInParallel(wg, ec, kubeClient, vaoPod, vzconstants.VerrazzanoSystemNamespace, bugReportDir, outStream)
	go captureLogsInParallel(wg, ec, kubeClient, vmoPod, vzconstants.VerrazzanoSystemNamespace, bugReportDir, outStream)

	wg.Wait()
	close(ec)

	// Report error, if any
	for err := range ec {
		return fmt.Errorf("an error occurred while capturing the log for pod: %s, error: %s", err.PodName, err.ErrorMessage)
	}
	return nil
}

func captureLogsInParallel(wg *sync.WaitGroup, ec chan ErrorsChannel, kubeClient kubernetes.Interface, pods []corev1.Pod, namespace, bugReportDir string, outStream io.Writer) {
	defer wg.Done()
	if len(pods) == 0 {
		return
	}
	// This won't work when there are more than one pods for the same app label
	fmt.Fprintf(outStream, fmt.Sprintf("Capturing the log from pod %s in the namespace %s ...\n", pods[0].Name, namespace))
	err := pkghelpers.CapturePodLog(kubeClient, pods[0], namespace, bugReportDir)
	if err != nil {
		ec <- ErrorsChannel{PodName: pods[0].Name, ErrorMessage: err.Error()}
	}
}
