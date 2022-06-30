// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package bugreport

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	vzconstants "github.com/verrazzano/verrazzano/pkg/constants"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	cmdhelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	"io"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"os"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var errBugReport = "an error occurred while creating the bug report"

// CaptureInstallIssues captures the cluster resources under bugReportDir, when there is an install
func CaptureInstallIssues(client client.Client, kubeClient kubernetes.Interface, bugReportDir string) error {
	vz, err := helpers.FindVerrazzanoResource(client)
	if err != nil {
		return fmt.Errorf("verrazzano is not installed: %s", err.Error())
	}

	// Find out whether the installation is successful from the Verrazzano resource
	// If install is not successful, get the list of components which are not in ready state
	// Capture the platform operator log, some logs from the component pods

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

	if len(compsNotReady) > 0 {
		// Capture Verrazzano resource
		captureVerrazzanoResource(vz, bugReportDir)

		// Capture VPO log
		vpoPod, _ := cmdhelpers.GetVPOPodName(client)
		err = capturePodLogs(kubeClient, vpoPod, constants.VerrazzanoPlatformOperator, vzconstants.VerrazzanoInstallNamespace, bugReportDir)
		if err != nil {
			return fmt.Errorf("an error in capturing the log from platform operator: %s", err.Error())
		}

		// TODO: capture the log from the pod, and events in the namespace of the component
	}
	return nil
}

// CreateReportArchive creates the .tar.gz file specified by bugReportFile, from the files in bugReportDir
func CreateReportArchive(bugReportDir, bugReportFile string) error {
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
		filePath := constants.BugReportRoot + path[len(bugReportDir):]
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

	if err := filepath.Walk(bugReportDir, walkFn); err != nil {
		return err
	}
	return nil
}

// captureVerrazzanoResource captures Verrazzano resources as a json under bugReportDir
func captureVerrazzanoResource(vz *vzapi.Verrazzano, bugReportDir string) error {
	var vzRes = bugReportDir + string(os.PathSeparator) + constants.VzResource
	f, err := os.OpenFile(vzRes, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("an error occurred while creating the file %s: %s", vzRes, err.Error())
	}
	defer f.Close()

	a, _ := json.MarshalIndent(vz, constants.JsonPrefix, constants.JsonIndent)
	vzJson := string(a)
	_, err = f.WriteString(vzJson)
	if err != nil {
		return fmt.Errorf("an error occurred while writing the file %s: %s", vzRes, err.Error())
	}
	return nil
}

// capturePodLogs captures the log from the pod in the bugReportDir
func capturePodLogs(kubeClient kubernetes.Interface, podName, container, namespace, bugReportDir string) error {
	// TODO: Investigate an efficient way to read the pod log from the beginning
	return capturePodLogsTimeRange(kubeClient, podName, container, namespace, bugReportDir, constants.SinceSeconds)
}

// capturePodLogs captures the log from the pod in the bugReportDir, generated in lastSeconds
func capturePodLogsTimeRange(kubeClient kubernetes.Interface, podName, container, namespace, bugReportDir string, lastSeconds int) error {
	sinceSec := int64(lastSeconds)
	podLog, err := kubeClient.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{
		Container:                    container,
		InsecureSkipTLSVerifyBackend: true,
		SinceSeconds:                 &sinceSec,
	}).Stream(context.TODO())
	if err != nil {
		return fmt.Errorf("an error occurred while reading the logs from pod %s: %s", podName, err.Error())
	}
	defer podLog.Close()

	// Create directory for the namespace and the pod, under the root level directory containing the bug report
	var folderPath = bugReportDir + string(os.PathSeparator) + namespace + string(os.PathSeparator) + podName
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
