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
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"io"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
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
func CaptureK8SResources(kubeClient kubernetes.Interface, namespace, captureDir string, vzHelper VZHelper) error {
	if err := captureWorkLoads(kubeClient, namespace, captureDir, vzHelper); err != nil {
		return err
	}
	if err := capturePods(kubeClient, namespace, captureDir, vzHelper); err != nil {
		return err
	}
	if err := captureEvents(kubeClient, namespace, captureDir, vzHelper); err != nil {
		return err
	}
	if err := captureIngress(kubeClient, namespace, captureDir, vzHelper); err != nil {
		return err
	}
	if err := captureServices(kubeClient, namespace, captureDir, vzHelper); err != nil {
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
