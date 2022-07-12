// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	oamcore "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	vzoamapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	vzconstants "github.com/verrazzano/verrazzano/pkg/constants"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"io"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"os"
	"path/filepath"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

var errBugReport = "an error occurred while creating the bug report: %s"
var createFileError = "an error occurred while creating the file %s: %s"

var containerStartLog = "==== START logs for container %s of pod %s/%s ====\n"
var containerEndLog = "==== END logs for container %s of pod %s/%s ====\n"

// CreateReportArchive creates the .tar.gz file specified by bugReportFile, from the files in captureDir
func CreateReportArchive(captureDir string, bugRepFile *os.File) error {

	// Create new Writers for gzip and tar
	gzipWriter := gzip.NewWriter(bugRepFile)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	walkFn := func(path string, fileInfo os.FileInfo, err error) error {
		if fileInfo.Mode().IsDir() {
			return nil
		}
		// make cluster-dump as the root directory in the archive, to support existing analysis tool
		filePath := constants.BugReportRoot + path[len(captureDir):]
		fileReader, err := os.Open(path)
		if err != nil {
			return fmt.Errorf(errBugReport, err.Error())
		}
		defer fileReader.Close()

		fih, err := tar.FileInfoHeader(fileInfo, filePath)
		if err != nil {
			return fmt.Errorf(errBugReport, err.Error())
		}

		fih.Name = filePath
		err = tarWriter.WriteHeader(fih)
		if err != nil {
			return fmt.Errorf(errBugReport, err.Error())
		}
		_, err = io.Copy(tarWriter, fileReader)
		if err != nil {
			return fmt.Errorf(errBugReport, err.Error())
		}
		return nil
	}

	if err := filepath.Walk(captureDir, walkFn); err != nil {
		return err
	}
	return nil
}

// CaptureK8SResources collects the Workloads (Deployment and ReplicaSet, StatefulSet, Daemonset), pods, events, ingress
// and services from the specified namespace, as JSON files
func CaptureK8SResources(kubeClient kubernetes.Interface, nsList []string, captureDir string, vzHelper VZHelper) error {
	// Run in parallel
	for _, ns := range nsList {
		if err := captureWorkLoads(kubeClient, ns, captureDir, vzHelper); err != nil {
			return err
		}
		if err := capturePods(kubeClient, ns, captureDir, vzHelper); err != nil {
			return err
		}
		if err := captureEvents(kubeClient, ns, captureDir, vzHelper); err != nil {
			return err
		}
		if err := captureIngress(kubeClient, ns, captureDir, vzHelper); err != nil {
			return err
		}
		if err := captureServices(kubeClient, ns, captureDir, vzHelper); err != nil {
			return err
		}
	}
	return nil
}

// DoesNamespaceExists checks whether the namespace exists in the cluster
func DoesNamespaceExists(kubeClient kubernetes.Interface, namespace string, vzHelper VZHelper) (bool, error) {
	if namespace == "" {
		fmt.Fprintf(vzHelper.GetOutputStream(), "Ignoring empty namespace\n")
		return false, nil
	}
	ns, err := kubeClient.CoreV1().Namespaces().Get(context.TODO(), namespace, metav1.GetOptions{})

	if err != nil && errors.IsNotFound(err) {
		fmt.Fprintf(vzHelper.GetOutputStream(), "Namespace %s not found in the cluster\n", namespace)
		return false, err
	}
	if err != nil {
		fmt.Fprintf(vzHelper.GetOutputStream(), "An error occurred while getting the namespace %s: %s\n", namespace, err.Error())
		return false, err
	}
	return ns != nil && len(ns.Name) > 0, nil
}

// CaptureOAMResources captures OAM resources in the given list of namespaces
func CaptureOAMResources(dynamicClient dynamic.Interface, nsList []string, captureDir string, vzHelper VZHelper) error {
	for _, ns := range nsList {
		if err := captureAppConfigurations(dynamicClient, ns, captureDir, vzHelper); err != nil {
			return err
		}
		if err := captureComponents(dynamicClient, ns, captureDir, vzHelper); err != nil {
			return err
		}
		if err := captureIngressTraits(dynamicClient, ns, captureDir, vzHelper); err != nil {
			return err
		}
		if err := captureMetricsTraits(dynamicClient, ns, captureDir, vzHelper); err != nil {
			return err
		}

		// for multi-cluster resources
		if err := captureMCAppConfigurations(dynamicClient, ns, captureDir, vzHelper); err != nil {
			return err
		}
	}

	// The following resources need to captured only on admin cluster. For now, attempt to get them on all the clusters
	// Capture Verrazzano projects in verrazzano-mc namespace
	if err := captureVerrazzanoProjects(dynamicClient, captureDir, vzHelper); err != nil {
		return err
	}

	// Capture Verrazzano projects in verrazzano-mc namespace
	if err := captureVerrazzanoManagedCluster(dynamicClient, captureDir, vzHelper); err != nil {
		return err
	}

	return nil
}

// GetPodList returns list of pods matching the label in the given namespace
func GetPodList(client clipkg.Client, appLabel, appName, namespace string) ([]corev1.Pod, error) {
	aLabel, _ := labels.NewRequirement(appLabel, selection.Equals, []string{appName})
	labelSelector := labels.NewSelector()
	labelSelector = labelSelector.Add(*aLabel)
	podList := corev1.PodList{}
	err := client.List(
		context.TODO(),
		&podList,
		&clipkg.ListOptions{
			Namespace:     namespace,
			LabelSelector: labelSelector,
		})
	if err != nil {
		return nil, fmt.Errorf("an error while listing pods: %s", err.Error())
	}
	return podList.Items, nil
}

// captureVZResource captures Verrazzano resources as a JSON file
func CaptureVZResource(captureDir string, vz vzapi.VerrazzanoList, vzHelper VZHelper) error {
	var vzRes = captureDir + string(os.PathSeparator) + constants.VzResource
	f, err := os.OpenFile(vzRes, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf(createFileError, vzRes, err.Error())
	}
	defer f.Close()

	fmt.Fprintf(vzHelper.GetOutputStream(), "Capturing Verrazzano resource ...\n")
	vzJSON, err := json.MarshalIndent(vz, constants.JSONPrefix, constants.JSONIndent)
	if err != nil {
		fmt.Fprintf(vzHelper.GetOutputStream(), "An error occurred while creating JSON encoding of %s: %s\n", vzRes, err.Error())
		return nil
	}
	_, err = f.WriteString(SanitizeString(string(vzJSON)))
	if err != nil {
		fmt.Fprintf(vzHelper.GetOutputStream(), "An error occurred while writing the file %s: %s\n", vzRes, err.Error())
	}
	return nil
}

// captureEvents captures the events in the given namespace, as a JSON file
func captureEvents(kubeClient kubernetes.Interface, namespace, captureDir string, vzHelper VZHelper) error {
	events, err := kubeClient.CoreV1().Events(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Fprintf(vzHelper.GetOutputStream(), "An error occurred while getting the Events in namespace %s: %s\n", namespace, err.Error())
	}
	if len(events.Items) > 0 {
		fmt.Fprintf(vzHelper.GetOutputStream(), "Capturing Events in namespace: %s ...\n", namespace)
		if err = createFile(events, namespace, constants.EventsJSON, captureDir, vzHelper); err != nil {
			return err
		}
	}
	return nil
}

// capturePods captures the pods in the given namespace, as a JSON file
func capturePods(kubeClient kubernetes.Interface, namespace, captureDir string, vzHelper VZHelper) error {
	pods, err := kubeClient.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Fprintf(vzHelper.GetOutputStream(), "An error occurred while getting the Pods in namespace %s: %s\n", namespace, err.Error())
	}
	if len(pods.Items) > 0 {
		fmt.Fprintf(vzHelper.GetOutputStream(), "Capturing Pods in namespace: %s ...\n", namespace)
		if err = createFile(pods, namespace, constants.PodsJSON, captureDir, vzHelper); err != nil {
			return err
		}
	}
	return nil
}

// captureIngress captures the ingresses in the given namespace, as a JSON file
func captureIngress(kubeClient kubernetes.Interface, namespace, captureDir string, vzHelper VZHelper) error {
	ingressList, err := kubeClient.NetworkingV1().Ingresses(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Fprintf(vzHelper.GetOutputStream(), "An error occurred while getting the Ingress in namespace %s: %s\n", namespace, err.Error())
	}
	if len(ingressList.Items) > 0 {
		fmt.Fprintf(vzHelper.GetOutputStream(), "Capturing Ingresses in namespace: %s ...\n", namespace)
		if err = createFile(ingressList, namespace, constants.IngressJSON, captureDir, vzHelper); err != nil {
			return err
		}
	}
	return nil
}

// captureServices captures the services in the given namespace, as a JSON file
func captureServices(kubeClient kubernetes.Interface, namespace, captureDir string, vzHelper VZHelper) error {
	serviceList, err := kubeClient.CoreV1().Services(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Fprintf(vzHelper.GetOutputStream(), "An error occurred while getting the Services in namespace %s: %s\n", namespace, err.Error())
	}
	if len(serviceList.Items) > 0 {
		fmt.Fprintf(vzHelper.GetOutputStream(), "Capturing Services in namespace: %s ...\n", namespace)
		if err = createFile(serviceList, namespace, constants.ServicesJSON, captureDir, vzHelper); err != nil {
			return err
		}
	}
	return nil
}

// captureWorkLoads captures the Deployment and ReplicaSet, StatefulSet, Daemonset in the given namespace
func captureWorkLoads(kubeClient kubernetes.Interface, namespace, captureDir string, vzHelper VZHelper) error {
	deployments, err := kubeClient.AppsV1().Deployments(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Fprintf(vzHelper.GetOutputStream(), "An error occurred while getting the Deployments in namespace %s: %s\n", namespace, err.Error())
	}
	if len(deployments.Items) > 0 {
		fmt.Fprintf(vzHelper.GetOutputStream(), "Capturing Deployments in namespace: %s ...\n", namespace)
		if err = createFile(deployments, namespace, constants.DeploymentsJSON, captureDir, vzHelper); err != nil {
			return err
		}
	}

	replicaSets, err := kubeClient.AppsV1().ReplicaSets(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Fprintf(vzHelper.GetOutputStream(), "An error occurred while getting the ReplicaSets in namespace %s: %s\n", namespace, err.Error())
	}
	if len(replicaSets.Items) > 0 {
		fmt.Fprintf(vzHelper.GetOutputStream(), "Capturing Replicasets in namespace: %s ...\n", namespace)
		if err = createFile(replicaSets, namespace, constants.ReplicaSetsJSON, captureDir, vzHelper); err != nil {
			return err
		}
	}

	daemonSets, err := kubeClient.AppsV1().DaemonSets(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Fprintf(vzHelper.GetOutputStream(), "An error occurred while getting the DaemonSets in namespace %s: %s\n", namespace, err.Error())
	}
	if len(daemonSets.Items) > 0 {
		fmt.Fprintf(vzHelper.GetOutputStream(), "Capturing DaemonSets in namespace: %s ...\n", namespace)
		if err = createFile(daemonSets, namespace, constants.DaemonSetsJSON, captureDir, vzHelper); err != nil {
			return err
		}
	}

	statefulSets, err := kubeClient.AppsV1().StatefulSets(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Fprintf(vzHelper.GetOutputStream(), "An error occurred while getting the StatefulSets in namespace %s: %s\n", namespace, err.Error())
	}
	if len(statefulSets.Items) > 0 {
		fmt.Fprintf(vzHelper.GetOutputStream(), "Capturing StatefulSets in namespace: %s ...\n", namespace)
		if err = createFile(statefulSets, namespace, constants.StatefulSetsJSON, captureDir, vzHelper); err != nil {
			return err
		}
	}
	return nil
}

// captureAppConfigurations captures the OAM application configurations in the given namespace, as a JSON file
func captureAppConfigurations(dynamicClient dynamic.Interface, namespace, captureDir string, vzHelper VZHelper) error {
	appConfigs, err := dynamicClient.Resource(getAppConfigScheme()).Namespace(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil && errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		fmt.Fprintf(vzHelper.GetOutputStream(), "An error occurred while getting the ApplicationConfigurations in namespace %s: %s\n", namespace, err.Error())
		return nil
	}
	if len(appConfigs.Items) > 0 {
		fmt.Fprintf(vzHelper.GetOutputStream(), "Capturing ApplicationConfigurations in namespace: %s ...\n", namespace)
		if err = createFile(appConfigs, namespace, constants.AppConfigJSON, captureDir, vzHelper); err != nil {
			return err
		}
	}
	return nil
}

// captureComponents captures the OAM components in the given namespace, as a JSON file
func captureComponents(dynamicClient dynamic.Interface, namespace, captureDir string, vzHelper VZHelper) error {
	comps, err := dynamicClient.Resource(getComponentConfigScheme()).Namespace(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil && errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		fmt.Fprintf(vzHelper.GetOutputStream(), "An error occurred while getting the Components in namespace %s: %s\n", namespace, err.Error())
		return nil
	}
	if len(comps.Items) > 0 {
		fmt.Fprintf(vzHelper.GetOutputStream(), "Capturing Components in namespace: %s ...\n", namespace)
		if err = createFile(comps, namespace, constants.ComponentJSON, captureDir, vzHelper); err != nil {
			return err
		}
	}
	return nil
}

// captureIngressTraits captures the ingress traits in the given namespace, as a JSON file
func captureIngressTraits(dynamicClient dynamic.Interface, namespace, captureDir string, vzHelper VZHelper) error {
	ingTraits, err := dynamicClient.Resource(getIngressTraitConfigScheme()).Namespace(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil && errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		fmt.Fprintf(vzHelper.GetOutputStream(), "An error occurred while getting the IngressTraits in namespace %s: %s\n", namespace, err.Error())
		return nil
	}
	if len(ingTraits.Items) > 0 {
		fmt.Fprintf(vzHelper.GetOutputStream(), "Capturing IngressTraits in namespace: %s ...\n", namespace)
		if err = createFile(ingTraits, namespace, constants.IngressTraitJSON, captureDir, vzHelper); err != nil {
			return err
		}
	}
	return nil
}

// captureMetricsTraits captures the metrics traits in the given namespace, as a JSON file
func captureMetricsTraits(dynamicClient dynamic.Interface, namespace, captureDir string, vzHelper VZHelper) error {
	metricsTraits, err := dynamicClient.Resource(getMetricsTraitConfigScheme()).Namespace(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil && errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		fmt.Fprintf(vzHelper.GetOutputStream(), "An error occurred while getting the MetricsTraits in namespace %s: %s\n", namespace, err.Error())
		return nil
	}
	if len(metricsTraits.Items) > 0 {
		fmt.Fprintf(vzHelper.GetOutputStream(), "Capturing MetricsTraits in namespace: %s ...\n", namespace)
		if err = createFile(metricsTraits, namespace, constants.MetricsTraitJSON, captureDir, vzHelper); err != nil {
			return err
		}
	}
	return nil
}

// captureAppConfigurations captures the OAM application configurations in the given namespace, as a JSON file
func captureMCAppConfigurations(dynamicClient dynamic.Interface, namespace, captureDir string, vzHelper VZHelper) error {
	mcAppConfigs, err := dynamicClient.Resource(getMCAppConfigScheme()).Namespace(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil && errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		fmt.Fprintf(vzHelper.GetOutputStream(), "An error occurred while getting the MultiClusterApplicationConfiguration in namespace %s: %s\n", namespace, err.Error())
		return nil
	}
	if len(mcAppConfigs.Items) > 0 {
		fmt.Fprintf(vzHelper.GetOutputStream(), "Capturing MultiClusterApplicationConfiguration in namespace: %s ...\n", namespace)
		if err = createFile(mcAppConfigs, namespace, constants.McAppConfigJSON, captureDir, vzHelper); err != nil {
			return err
		}
	}
	return nil
}

// captureAppConfigurations captures the Verrazzano projects in the verrazzano-mc namespace, as a JSON file
func captureVerrazzanoProjects(dynamicClient dynamic.Interface, captureDir string, vzHelper VZHelper) error {
	vzProjectConfigs, err := dynamicClient.Resource(getVzProjectsConfigScheme()).Namespace(vzconstants.VerrazzanoMultiClusterNamespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil && errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		fmt.Fprintf(vzHelper.GetOutputStream(), "An error occurred while getting the VerrazzanoProjects in namespace %s: %s\n", vzconstants.VerrazzanoMultiClusterNamespace, err.Error())
		return nil
	}
	if len(vzProjectConfigs.Items) > 0 {
		fmt.Fprintf(vzHelper.GetOutputStream(), "Capturing VerrazzanoProjects in namespace: %s ...\n", vzconstants.VerrazzanoMultiClusterNamespace)
		if err = createFile(vzProjectConfigs, vzconstants.VerrazzanoMultiClusterNamespace, constants.VzProjectsJSON, captureDir, vzHelper); err != nil {
			return err
		}
	}
	return nil
}

// captureVerrazzanoManagedCluster captures VerrazzanoManagedCluster in verrazzano-mc namespace, as a JSON file
func captureVerrazzanoManagedCluster(dynamicClient dynamic.Interface, captureDir string, vzHelper VZHelper) error {
	vzProjectConfigs, err := dynamicClient.Resource(getManagedClusterConfigScheme()).Namespace(vzconstants.VerrazzanoMultiClusterNamespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil && errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		fmt.Fprintf(vzHelper.GetOutputStream(), "An error occurred while getting the VerrazzanoManagedClusters in namespace %s: %s\n", vzconstants.VerrazzanoMultiClusterNamespace, err.Error())
		return nil
	}
	if len(vzProjectConfigs.Items) > 0 {
		fmt.Fprintf(vzHelper.GetOutputStream(), "Capturing VerrazzanoManagedClusters in namespace: %s ...\n", vzconstants.VerrazzanoMultiClusterNamespace)
		if err = createFile(vzProjectConfigs, vzconstants.VerrazzanoMultiClusterNamespace, constants.VzProjectsJSON, captureDir, vzHelper); err != nil {
			return err
		}
	}
	return nil
}

// captureLog captures the log from the pod in the captureDir
func CapturePodLog(kubeClient kubernetes.Interface, pod corev1.Pod, namespace, captureDir string, vzHelper VZHelper) error {
	podName := pod.Name
	if len(podName) == 0 {
		return nil
	}

	// Create directory for the namespace and the pod, under the root level directory containing the bug report
	var folderPath = captureDir + string(os.PathSeparator) + namespace + string(os.PathSeparator) + podName
	err := os.MkdirAll(folderPath, os.ModePerm)
	if err != nil {
		return fmt.Errorf("an error occurred while creating the directory %s: %s", folderPath, err.Error())
	}

	// Create logs.txt under the directory for the namespace
	var logPath = folderPath + string(os.PathSeparator) + constants.LogFile
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf(createFileError, logPath, err.Error())
	}
	defer f.Close()

	// Capture logs for both init containers and containers
	var cs []corev1.Container
	cs = append(cs, pod.Spec.InitContainers...)
	cs = append(cs, pod.Spec.Containers...)

	// Write the log from all the containers to a single file, with lines differentiating the logs from each of the containers
	for _, c := range cs {
		writeToFile := func(contName string) error {
			podLog, err := kubeClient.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{
				Container:                    contName,
				InsecureSkipTLSVerifyBackend: true,
			}).Stream(context.TODO())
			if err != nil {
				fmt.Fprintf(vzHelper.GetOutputStream(), "An error occurred while reading the logs from pod %s: %s\n", podName, err.Error())
				return nil
			}
			defer podLog.Close()

			reader := bufio.NewScanner(podLog)
			f.WriteString(fmt.Sprintf(containerStartLog, contName, namespace, podName))
			for reader.Scan() {
				f.WriteString(SanitizeString(reader.Text() + "\n"))
			}
			f.WriteString(fmt.Sprintf(containerEndLog, contName, namespace, podName))
			return nil
		}
		writeToFile(c.Name)
	}
	return nil
}

// createFile creates file from a workload, as a JSON file
func createFile(v interface{}, namespace, resourceFile, captureDir string, vzHelper VZHelper) error {
	var folderPath = captureDir + string(os.PathSeparator) + namespace

	if _, err := os.Stat(folderPath); os.IsNotExist(err) {
		err := os.MkdirAll(folderPath, os.ModePerm)
		if err != nil {
			return fmt.Errorf("an error occurred while creating the directory %s: %s", folderPath, err.Error())
		}
	}

	var res = folderPath + string(os.PathSeparator) + resourceFile
	f, err := os.OpenFile(res, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf(createFileError, res, err.Error())
	}
	defer f.Close()

	resJSON, _ := json.MarshalIndent(v, constants.JSONPrefix, constants.JSONIndent)
	_, err = f.WriteString(SanitizeString(string(resJSON)))
	if err != nil {
		fmt.Fprintf(vzHelper.GetOutputStream(), "An error occurred while writing the file %s: %s", res, err.Error())
	}
	return nil
}

// RemoveDuplicate removes duplicates from origSlice
func RemoveDuplicate(origSlice []string) []string {
	allKeys := make(map[string]bool)
	returnSlice := []string{}
	for _, item := range origSlice {
		if _, value := allKeys[item]; !value {
			allKeys[item] = true
			returnSlice = append(returnSlice, item)
		}
	}
	return returnSlice
}

// getAppConfigScheme returns GroupVersionResource for ApplicationConfiguration
func getAppConfigScheme() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    oamcore.Group,
		Version:  oamcore.Version,
		Resource: constants.OAMAppConfigurations,
	}
}

// getComponentConfigScheme returns GroupVersionResource for Component
func getComponentConfigScheme() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    oamcore.Group,
		Version:  oamcore.Version,
		Resource: constants.OAMComponents,
	}
}

// getMetricsTraitConfigScheme returns GroupVersionResource for MetricsTrait
func getMetricsTraitConfigScheme() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    vzoamapi.SchemeGroupVersion.Group,
		Version:  vzoamapi.SchemeGroupVersion.Version,
		Resource: constants.OAMMetricsTraits,
	}
}

// getIngressTraitConfigScheme returns GroupVersionResource for IngressTrait
func getIngressTraitConfigScheme() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    vzoamapi.SchemeGroupVersion.Group,
		Version:  vzoamapi.SchemeGroupVersion.Version,
		Resource: constants.OAMIngressTraits,
	}
}

// getMCAppConfigScheme returns GroupVersionResource for MulticlusterApplicationConfiguration
func getMCAppConfigScheme() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    clustersv1alpha1.SchemeGroupVersion.Group,
		Version:  clustersv1alpha1.SchemeGroupVersion.Version,
		Resource: constants.OAMMCAppConfigurations,
	}
}

// getVzProjectsConfigScheme returns GroupVersionResource for VerrazzanoProject
func getVzProjectsConfigScheme() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    clustersv1alpha1.SchemeGroupVersion.Group,
		Version:  clustersv1alpha1.SchemeGroupVersion.Version,
		Resource: constants.OAMProjects,
	}
}

// getManagedClusterConfigScheme returns GroupVersionResource for VerrazzanoManagedCluster
func getManagedClusterConfigScheme() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    clustersv1alpha1.SchemeGroupVersion.Group,
		Version:  clustersv1alpha1.SchemeGroupVersion.Version,
		Resource: constants.OAMManagedClusters,
	}
}
