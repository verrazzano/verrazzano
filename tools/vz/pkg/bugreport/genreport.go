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
	"strings"
	"sync"
)

type ErrorsChannel struct {
	PodName      string `json:"podName"`
	ErrorMessage string `json:"errorMessage"`
}

// The bug-report command captures the following resources from the cluster by default
// - Verrazzano resource
// - Logs from verrazzano-platform-operator, verrazzano-monitoring-operator and verrazzano-application-operator pods
// - Workloads (Deployment and ReplicaSet, StatefulSet, Daemonset), pods, events, ingress and services from namespace verrazzano-system,
//   namespaces specified by flag --include-namespaces and the namespaces for each of the components which are not in Ready state
// - OAM resources like ApplicationConfiguration, Component, IngressTrait, MetricsTrait from namespaces specified by flag --include-namespaces
// - VerrazzanoManagedCluster, VerrazzanoProject and MultiClusterApplicationConfiguration in a multi-clustered environment

// GenerateBugReport creates a bug report by including the resources selectively from the cluster, useful to analyze the issue.
func GenerateBugReport(kubeClient kubernetes.Interface, dynamicClient dynamic.Interface, client clipkg.Client, bugReportFile *os.File, moreNS string, vzHelper pkghelpers.VZHelper) error {
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

	// Get the list of namespaces, to capture information
	var additionalNS []string
	var nsList = []string{vzconstants.VerrazzanoSystemNamespace}

	// Get the list of components not in Ready state, if Verrazzano resource is not in Ready state
	if len(vz.Items) != 0 && vz.Items[0].Status.State != vzapi.VzStateReady {
		failedCompNS := pkghelpers.GetNamespacesForNotReadyComponents(vz.Items[0])
		nsList = append(nsList, failedCompNS...)
	}

	if moreNS != "" {
		additionalNS = getNamespaces(kubeClient, moreNS, vzHelper)
		nsList = append(nsList, additionalNS...)
	}

	// Remove the duplicates from nsList
	nsList = pkghelpers.RemoveDuplicate(nsList)

	// Capture list of resources from verrazzano-install and verrazzano-system namespaces
	err = captureVerrazzanoResources(client, kubeClient, bugReportDir, vz, vzHelper, nsList)
	if err != nil {
		fmt.Fprintf(vzHelper.GetOutputStream(), "There is an error with capturing the Verrazzano resources: %s", err.Error())
	}

	// Capture OAM resources from the namespaces specified using --include-namespaces
	if len(additionalNS) > 0 {
		if err := pkghelpers.CaptureOAMResources(dynamicClient, additionalNS, bugReportDir, vzHelper); err != nil {
			fmt.Fprintf(vzHelper.GetOutputStream(), "There is an error in capturing the resources : %s", err.Error())
		}
		if err := pkghelpers.CaptureMultiClusterResources(dynamicClient, additionalNS, bugReportDir, vzHelper); err != nil {
			fmt.Fprintf(vzHelper.GetOutputStream(), "There is an error in capturing the multi-cluster resources : %s", err.Error())
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

// captureVerrazzanoResources captures the resources from verrazzano-install and verrazzano-system namespaces
func captureVerrazzanoResources(client clipkg.Client, kubeClient kubernetes.Interface, bugReportDir string, vz vzapi.VerrazzanoList, vzHelper pkghelpers.VZHelper, namespaces []string) error {
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

	// Capture workloads, pods, events, ingress and services in verrazzano-system namespace
	if err := captureK8SResources(kubeClient, bugReportDir, namespaces, vzHelper); err != nil {
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
	vaoPod, _ := pkghelpers.GetPodList(client, constants.AppLabel, constants.VerrazzanoApplicationOperator, vzconstants.VerrazzanoSystemNamespace)
	vmoPod, _ := pkghelpers.GetPodList(client, constants.K8SAppLabel, constants.VerrazzanoMonitoringOperator, vzconstants.VerrazzanoSystemNamespace)

	wg := &sync.WaitGroup{}
	wg.Add(3)
	ec := make(chan ErrorsChannel, 1)

	go captureLogsInParallel(wg, ec, kubeClient, vpoPod, vzconstants.VerrazzanoInstallNamespace, bugReportDir, vzHelper)
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

// captureK8SResources captures Kubernetes workloads, pods, events, ingresses and services from the list of namespaces
func captureK8SResources(kubeClient kubernetes.Interface, bugReportDir string, namespaces []string, vzHelper pkghelpers.VZHelper) error {
	// it will never be the case, just to make the method resilient, adding this condition
	if len(namespaces) == 0 {
		return nil
	}
	wg := &sync.WaitGroup{}
	wg.Add(len(namespaces))
	ec := make(chan ErrorsChannel, 1)

	for _, ns := range namespaces {
		go captureK8SResourcesInParallel(wg, ec, kubeClient, ns, bugReportDir, vzHelper)
	}
	wg.Wait()
	close(ec)

	// Report error, if any
	for err := range ec {
		return fmt.Errorf("an error occurred while capturing the resource, error: %s", err.ErrorMessage)
	}
	return nil
}

// captureK8SResources captures Kubernetes workloads, pods, events, ingresses and services from the list of namespaces in parallel
func captureK8SResourcesInParallel(wg *sync.WaitGroup, ec chan ErrorsChannel, kubeClient kubernetes.Interface, namespace, bugReportDir string, vzHelper pkghelpers.VZHelper) {
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

// getNamespaces parses comma separated namespaces and removes the duplicates. It also excludes the namespace which
// do not exist, with a message.
func getNamespaces(kubeClient kubernetes.Interface, includes string, vzHelper pkghelpers.VZHelper) []string {
	var nsList []string
	var includedNS []string

	if includes != "" {
		includes := strings.ReplaceAll(includes, " ", "")
		nsList = strings.Split(includes, ",")
	}

	if len(nsList) > 0 {
		nsList = pkghelpers.RemoveDuplicate(nsList)
		for _, ns := range nsList {
			nsExists, _ := pkghelpers.DoesNamespaceExists(kubeClient, ns, vzHelper)
			if nsExists {
				includedNS = append(includedNS, ns)
			}
		}
	}
	return includedNS
}
