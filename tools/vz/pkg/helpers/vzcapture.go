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

var errBugReport = "an error occurred while creating the bug report"

// CreateReportArchive creates the .tar.gz file specified by bugReportFile, from the files in captureDir
func CreateReportArchive(captureDir, bugReportFile string) error {

	// Create the bug report file
	bugRepFile, err := os.Create(bugReportFile)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("parent directory for the bug report file doesn't exit")
		}
		return fmt.Errorf("%s: %s", errBugReport, err.Error())
	}
	defer bugRepFile.Close()

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
			return fmt.Errorf("%s: %s", errBugReport, err.Error())
		}
		defer fileReader.Close()

		fih, err := tar.FileInfoHeader(fileInfo, filePath)
		if err != nil {
			return fmt.Errorf("%s: %s", errBugReport, err.Error())
		}

		fih.Name = filePath
		err = tarWriter.WriteHeader(fih)
		if err != nil {
			return fmt.Errorf("%s: %s", errBugReport, err.Error())
		}
		_, err = io.Copy(tarWriter, fileReader)
		if err != nil {
			return fmt.Errorf("%s: %s", errBugReport, err.Error())
		}
		return nil
	}

	if err := filepath.Walk(captureDir, walkFn); err != nil {
		return err
	}
	return nil
}

func CaptureK8SResources (kubeClient kubernetes.Interface, namespace, captureDir string) error {
	if err := captureWorkLoads(kubeClient, namespace, captureDir); err != nil {
		return err
	}
	if err := capturePods(kubeClient, namespace, captureDir); err != nil {
		return err
	}
	if err := captureEvents(kubeClient, namespace, captureDir); err != nil {
		return err
	}
	if err := captureIngress(kubeClient, namespace, captureDir); err != nil {
		return err
	}
	if err := captureServices(kubeClient, namespace, captureDir); err != nil {
		return err
	}
	return nil
}

func GetPodName(client clipkg.Client, podPrefix, namespace string) (string, error) {
	appLabel, _ := labels.NewRequirement("app", selection.Equals, []string{podPrefix})
	labelSelector := labels.NewSelector()
	labelSelector = labelSelector.Add(*appLabel)
	podList := corev1.PodList{}
	err := client.List(
		context.TODO(),
		&podList,
		&clipkg.ListOptions{
			Namespace:     namespace,
			LabelSelector: labelSelector,
		})
	if err != nil {
		return "", fmt.Errorf("waiting for %s, failed to list pods: %s", podPrefix, err.Error())
	}
	if len(podList.Items) == 0 {
		return "", fmt.Errorf("failed to find %s in namespace %s", podPrefix, namespace)
	}
	if len(podList.Items) > 1 {
		return "", fmt.Errorf("waiting for %s, more than one %s pod was found in namespace %s", podPrefix, podPrefix, namespace)
	}
	return podList.Items[0].Name, nil
}

// captureVZResource captures Verrazzano resources as a json under captureDir
func CaptureVZResource(vz *vzapi.Verrazzano, captureDir string) error {
	var vzRes = captureDir + string(os.PathSeparator) + constants.VzResource
	f, err := os.OpenFile(vzRes, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("an error occurred while creating the file %s: %s", vzRes, err.Error())
	}
	defer f.Close()

	a, _ := json.MarshalIndent(vz, constants.JSONPrefix, constants.JSONIndent)
	vzJSON := string(a)
	_, err = f.WriteString(vzJSON)
	if err != nil {
		return fmt.Errorf("an error occurred while writing the file %s: %s", vzRes, err.Error())
	}
	return nil
}

func captureEvents(kubeClient kubernetes.Interface, namespace, captureDir string) error {
	events, err := kubeClient.CoreV1().Events(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("an error occurred while getting the Events in namespace %s: %s", namespace, err.Error())
	}
	if err = createFile(namespace, constants.EventsJSON, captureDir, events); err != nil {
		return err
	}
	return nil
}

func capturePods(kubeClient kubernetes.Interface, namespace, captureDir string) error {
	pods, err := kubeClient.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("an error occurred while getting the Pods in namespace %s: %s", namespace, err.Error())
	}
	if err = createFile(namespace, constants.PodsJSON, captureDir, pods); err != nil {
		return err
	}
	return nil
}

func captureIngress(kubeClient kubernetes.Interface, namespace, captureDir string) error {
	ingressList, err := kubeClient.NetworkingV1().Ingresses(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("an error occurred while getting the Ingress in namespace %s: %s", namespace, err.Error())
	}
	if err = createFile(namespace, constants.IngressJSON, captureDir, ingressList); err != nil {
		return err
	}
	return nil
}

func captureServices(kubeClient kubernetes.Interface, namespace, captureDir string) error {
	serviceList, err := kubeClient.CoreV1().Services(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("an error occurred while getting the Services in namespace %s: %s", namespace, err.Error())
	}
	if err = createFile(namespace, constants.ServicesJSON, captureDir, serviceList); err != nil {
		return err
	}
	return nil
}

func captureWorkLoads(kubeClient kubernetes.Interface, namespace, captureDir string) error {
	deployments, err := kubeClient.AppsV1().Deployments(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("an error occurred while getting the Deployments in namespace %s: %s", namespace, err.Error())
	}

	if err = createFile(namespace, constants.DeploymentsJSON, captureDir, deployments); err != nil {
		return err
	}

	replicasets, err := kubeClient.AppsV1().ReplicaSets(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("an error occurred while getting the ReplicaSets in namespace %s: %s", namespace, err.Error())
	}
	if err = createFile(namespace, constants.ReplicaSetsJSON, captureDir, replicasets); err != nil {
		return err
	}

	daemonsets, err := kubeClient.AppsV1().DaemonSets(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("an error occurred while getting the DaemonSets in namespace %s: %s", namespace, err.Error())
	}
	if err = createFile(namespace, constants.DaemonSetsJSON, captureDir, daemonsets); err != nil {
		return err
	}

	statefulsets, err := kubeClient.AppsV1().StatefulSets(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("an error occurred while getting the StatefulSets in namespace %s: %s", namespace, err.Error())
	}
	if err = createFile(namespace, constants.StatefulSetsJSON, captureDir, statefulsets); err != nil {
		return err
	}
	return nil
}

// captureLog captures the log from the pod in the captureDir
func CaptureLog(kubeClient kubernetes.Interface, podName, container, namespace, captureDir string) error {
	podLog, err := kubeClient.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{
		Container:                    container,
		InsecureSkipTLSVerifyBackend: true,
	}).Stream(context.TODO())
	if err != nil {
		return fmt.Errorf("an error occurred while reading the logs from pod %s: %s", podName, err.Error())
	}
	defer podLog.Close()

	// Create directory for the namespace and the pod, under the root level directory containing the bug report
	var folderPath = captureDir + string(os.PathSeparator) + namespace + string(os.PathSeparator) + podName
	err = os.MkdirAll(folderPath, os.ModePerm)
	if err != nil {
		return fmt.Errorf("an error occurred while creating the directory %s: %s", folderPath, err.Error())
	}

	// Create logs.txt under the directory for the namespace
	var logPath = folderPath + string(os.PathSeparator) + constants.LogFile
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("an error occurred while creating the file %s: %s", logPath, err.Error())
	}
	defer f.Close()

	// Write the log from the pod to the file
	reader := bufio.NewScanner(podLog)
	var line string
	for reader.Scan() {
		line = reader.Text()
		f.WriteString(line + "\n")
	}
	return nil
}

func createFile (namespace, resourceFile, captureDir string, v interface{}) error {
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
		return fmt.Errorf("an error occurred while creating the file %s: %s", res, err.Error())
	}
	defer f.Close()

	resJSON, _ := json.MarshalIndent(v, constants.JSONPrefix, constants.JSONIndent)
	_, err = f.WriteString(string(resJSON))
	if err != nil {
		return fmt.Errorf("an error occurred while writing the file %s: %s", res, err.Error())
	}
	return nil
}