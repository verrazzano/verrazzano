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

	// Verrazzano as a list is required for the analysis tool
	vz := vzapi.VerrazzanoList{}
	err = client.List(context.TODO(), &vz)
	if (err != nil && len(vz.Items) == 0) || len(vz.Items) == 0 {
		fmt.Fprintf(vzHelper.GetOutputStream(), "Verrazzano is not installed ...\n")
	}

	// Capture list of resources from verrazzano-install and verrazzano-system namespaces
	err = captureVerrazzanoResources(client, kubeClient, bugReportDir, vz, vzHelper)
	if err != nil {
		fmt.Fprintf(vzHelper.GetOutputStream(), "There is an error with capturing the resources: %s", err.Error())
	}

	// Placeholder to call a method to capture the information from the namespaces/pods/containers
	// explicitly provided by the end user, using flag(s) to be supported by the command

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

// captureVerrazzanoResources captures the resources from verrazzano-install and verrazzano-system namespaces
func captureVerrazzanoResources(client clipkg.Client, kubeClient kubernetes.Interface, bugReportDir string, vz vzapi.VerrazzanoList, vzHelper pkghelpers.VZHelper) error {

	var nameSpaces []string

	// Capture Verrazzano resource as JSON
	if len(vz.Items) > 0 {
		if err := pkghelpers.CaptureVZResource(bugReportDir, vz, vzHelper); err != nil {
			return err
		}
	}

	// Capture logs from pods
	if err := capturePodLogs(client, kubeClient, bugReportDir, vzHelper); err != nil {
		return err
	}

	nameSpaces, err := pkghelpers.GetNamespacesForNotReadyComponents(client)
	if err != nil {
		return err
	}
	nameSpaces = append(nameSpaces, vzconstants.VerrazzanoSystemNamespace)

	// Capture workloads, pods, events, ingress and services in verrazzano-system namespace
	if err := pkghelpers.CaptureK8SResources(kubeClient, nameSpaces, bugReportDir, vzHelper); err != nil {
		return err
	}

	//TODO: Capture ingress from keycloak and cattle-system namespace
	// Capture workloads, events, services, etc from those namespaces for the components which are not ready
	return nil
}

// capturePodLogs captures logs from platform operator, application operator and monitoring operator
func capturePodLogs(client clipkg.Client, kubeClient kubernetes.Interface, bugReportDir string, vzHelper pkghelpers.VZHelper) error {

	// Fixed list of pods for which, capture the log
	vpoPod, _ := pkghelpers.GetPodList(client, constants.AppLabel, constants.VerrazzanoPlatformOperator, vzconstants.VerrazzanoInstallNamespace)
	vuoPod, _ := pkghelpers.GetPodList(client, constants.JobNameLabel, constants.VerrazzanoUninstallExampleVerrazzano, vzconstants.VerrazzanoInstallNamespace)
	vaoPod, _ := pkghelpers.GetPodList(client, constants.AppLabel, constants.VerrazzanoApplicationOperator, vzconstants.VerrazzanoSystemNamespace)
	vmoPod, _ := pkghelpers.GetPodList(client, constants.K8SAppLabel, constants.VerrazzanoMonitoringOperator, vzconstants.VerrazzanoSystemNamespace)

	wg := &sync.WaitGroup{}
	wg.Add(4)
	ec := make(chan ErrorsChannel, 1)

	go captureLogsInParallel(wg, ec, kubeClient, vpoPod, vzconstants.VerrazzanoInstallNamespace, bugReportDir, vzHelper)
	go captureLogsInParallel(wg, ec, kubeClient, vuoPod, vzconstants.VerrazzanoInstallNamespace, bugReportDir, vzHelper)
	go captureLogsInParallel(wg, ec, kubeClient, vaoPod, vzconstants.VerrazzanoSystemNamespace, bugReportDir, vzHelper)
	go captureLogsInParallel(wg, ec, kubeClient, vmoPod, vzconstants.VerrazzanoSystemNamespace, bugReportDir, vzHelper)

	wg.Wait()
	close(ec)

	// Report error, if any
	for err := range ec {
		return fmt.Errorf("an error occurred while capturing the log for pod: %s, error: %s", err.PodName, err.ErrorMessage)
	}
	return nil
}

// captureLogsInParallel collects the log from pods in parallel
func captureLogsInParallel(wg *sync.WaitGroup, ec chan ErrorsChannel, kubeClient kubernetes.Interface, pods []corev1.Pod, namespace, bugReportDir string, vzHelper pkghelpers.VZHelper) {
	defer wg.Done()
	if len(pods) == 0 {
		return
	}
	// This won't work when there are more than one pods for the same app label
	fmt.Fprintf(vzHelper.GetOutputStream(), fmt.Sprintf("Capturing log from pod %s in %s namespace ...\n", pods[0].Name, namespace))
	err := pkghelpers.CapturePodLog(kubeClient, pods[0], namespace, bugReportDir, vzHelper)
	if err != nil {
		ec <- ErrorsChannel{PodName: pods[0].Name, ErrorMessage: err.Error()}
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
