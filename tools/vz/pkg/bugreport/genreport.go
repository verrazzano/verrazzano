// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package bugreport

import (
	"fmt"
	vzconstants "github.com/verrazzano/verrazzano/pkg/constants"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	pkghelpers "github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	"io/ioutil"
	"k8s.io/client-go/kubernetes"
	"os"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

// The bug-report command captures the following resources from the cluster
// - Verrazzano resource
// - Logs from verrazzano-platform-operator, verrazzano-monitoring-operator and verrazzano-application-operator pods
// - Workloads (Deployment and ReplicaSet, StatefulSet, Daemonset), pods, events, ingress and services from verrazzano-system namespace.
//   Also these resources are captured from other namespaces containing components in not ready state

// GenerateBugReport creates a bug report by including the resources selectively from the cluster, useful to analyze the issue.
func GenerateBugReport(kubeClient kubernetes.Interface, client clipkg.Client, bugReportFile string, vzHelper pkghelpers.VZHelper) error {
	tmpDir, err := ioutil.TempDir("", constants.BugReportDir)
	if err != nil {
		return fmt.Errorf("an error occurred while creating the directory to place cluster resources: %s", err.Error())
	}
	defer os.RemoveAll(tmpDir)

	bugReportDir := tmpDir + string(os.PathSeparator) + constants.BugReportDir
	err = os.Mkdir(bugReportDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("an error creating the directory: %s", bugReportDir)
	}

	fmt.Fprintf(vzHelper.GetOutputStream(), "Capturing cluster resource for bug report ...")
	// Capture list of resources from verrazzano-install and verrazzano-system namespaces
	err = captureVerrazzanoResources(client, kubeClient, bugReportDir)
	if err != nil {
		return fmt.Errorf("there is an error with capturing the resources: %s", err.Error())
	}

	// Placeholder to call a method, which will be used to capture resources from specific namespaces, specified by the user
	// using a flag to be supported by the command

	// Placeholder to call a function to redact sensitive information from all the files in bugReportDir

	// Create the report file
	err = pkghelpers.CreateReportArchive(bugReportDir, bugReportFile)
	if err != nil {
		return fmt.Errorf("there is an error in creating the bug report: %s", err.Error())
	}

	fmt.Fprintf(vzHelper.GetOutputStream(), fmt.Sprintf("Successfully generated the bug report: %s\n", bugReportFile))

	// Display a warning message to review the contents of the report
	fmt.Fprint(vzHelper.GetOutputStream(), "WARNING: Please examine the contents of the bug report for any sensitive data.\n")

	return nil
}

// captureVerrazzanoResources captures the resources from verrazzano-install and verrazzano-system namespaces
func captureVerrazzanoResources(client clipkg.Client, kubeClient kubernetes.Interface, bugReportDir string) error {
	vz, err := pkghelpers.FindVerrazzanoResource(client)
	if err != nil {
		return fmt.Errorf("verrazzano is not installed: %s", err.Error())
	}

	// Capture Verrazzano resource
	if err = pkghelpers.CaptureVZResource(vz, bugReportDir); err != nil {
		return err
	}

	// Capture logs from pods
	if err = capturePodLogs(client, kubeClient, bugReportDir); err != nil {
		return err
	}

	// Capture workloads, pods, events, ingress and services in verrazzano-system namespace
	if err = pkghelpers.CaptureK8SResources(kubeClient, vzconstants.VerrazzanoSystemNamespace, bugReportDir); err != nil {
		return err
	}

	//TODO: Capture ingress from keycloak and cattle-system namespace

	// Find out whether the installation is successful from the Verrazzano resource
	// If install is not successful, get the list of components which are not in ready state
	// If all the components are in a ready state, we might need to check the health of the pods, as something might have gone wrong after the installation
	var compsNotReady = make([]string, 0)
	if vz.Status.State != vzapi.VzStateReady {
		for _, compStatus := range vz.Status.Components {
			if compStatus.State != vzapi.CompStateReady {
				if compStatus.State == vzapi.CompStateDisabled {
					continue
				}
				compsNotReady = append(compsNotReady, compStatus.Name)
			}
		}
	}
	// What will you do with compsNotReady ?
	// Find out the list of components in namespaces other than verrazzano-system and get workloads, events, services, etc from those namespaces
	return nil
}

// Captures logs from platform operator, application operator and monitoring operator
func capturePodLogs(client clipkg.Client, kubeClient kubernetes.Interface, bugReportDir string) error {

	if vpoPod, _ := pkghelpers.GetPodName(client, constants.VerrazzanoPlatformOperator, vzconstants.VerrazzanoInstallNamespace); len(vpoPod) != 0 {
		if err := pkghelpers.CaptureLog(kubeClient, vpoPod, constants.VerrazzanoPlatformOperator, vzconstants.VerrazzanoInstallNamespace, bugReportDir); err != nil {
			return err
		}
	}

	if vaoPod, _ := pkghelpers.GetPodName(client, constants.VerrazzanoApplicationOperator, vzconstants.VerrazzanoSystemNamespace); len(vaoPod) != 0 {
		if err := pkghelpers.CaptureLog(kubeClient, vaoPod, constants.VerrazzanoApplicationOperator, vzconstants.VerrazzanoSystemNamespace, bugReportDir); err != nil {
			return err
		}
	}

	if vmoPod, _ := pkghelpers.GetPodName(client, constants.VerrazzanoMonitoringOperator, vzconstants.VerrazzanoSystemNamespace); len(vmoPod) != 0 {
		if err := pkghelpers.CaptureLog(kubeClient, vmoPod, constants.VerrazzanoMonitoringOperator, vzconstants.VerrazzanoSystemNamespace, bugReportDir); err != nil {
			return err
		}
	}
	return nil
}
