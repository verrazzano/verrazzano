// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"bytes"
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
	"text/template"
	"time"
)

// VeleroObjectDelete utility to clean up velero objects in the cluster
func VeleroObjectDelete(objectType, objectname, nameSpaceName string, log *zap.SugaredLogger) error {
	var cmdArgs []string
	cmdArgs = append(cmdArgs, "kubectl")
	cmdArgs = append(cmdArgs, "-n")
	cmdArgs = append(cmdArgs, nameSpaceName)
	cmdArgs = append(cmdArgs, "delete")

	switch objectType {
	case "backup":
		cmdArgs = append(cmdArgs, "backup.velero.io")
	case "restore":
		cmdArgs = append(cmdArgs, "restore.velero.io")
	case "storage":
		cmdArgs = append(cmdArgs, "backupstoragelocation.velero.io")
	case "podvolumerestores":
		cmdArgs = append(cmdArgs, "podvolumerestores.velero.io")
	case "podvolumebackups":
		cmdArgs = append(cmdArgs, "podvolumebackups.velero.io")
	}

	if objectType == "podvolumerestores" || objectType == "podvolumebackups" {
		cmdArgs = append(cmdArgs, "--all")
	} else {
		cmdArgs = append(cmdArgs, objectname)
	}
	cmdArgs = append(cmdArgs, "--ignore-not-found")

	var kcmd BashCommand
	kcmd.Timeout = 1 * time.Minute
	kcmd.CommandArgs = cmdArgs
	bashResponse := Runner(&kcmd, log)
	if bashResponse.CommandError != nil {
		return bashResponse.CommandError
	}
	return nil
}

// RancherObjectDelete utility to clean up rancher backup/restore objects in the cluster
func RancherObjectDelete(objectType, objectname string, log *zap.SugaredLogger) error {
	var cmdArgs []string
	cmdArgs = append(cmdArgs, "kubectl")
	cmdArgs = append(cmdArgs, "delete")

	switch objectType {
	case "backup":
		cmdArgs = append(cmdArgs, "backup.resources.cattle.io")
	case "restore":
		cmdArgs = append(cmdArgs, "restore.resources.cattle.io")
	}
	cmdArgs = append(cmdArgs, objectname)
	cmdArgs = append(cmdArgs, "--ignore-not-found")

	var kcmd BashCommand
	kcmd.Timeout = 1 * time.Minute
	kcmd.CommandArgs = cmdArgs
	bashResponse := Runner(&kcmd, log)
	if bashResponse.CommandError != nil {
		return bashResponse.CommandError
	}
	return nil
}

// DisplayHookLogs is used to display the logs from the pod where the backup hook was run
// It execs into the pod and fetches the log file contents
func DisplayHookLogs(log *zap.SugaredLogger) error {
	log.Infof("Retrieving verrazzano hook logs ...")
	var cmdArgs []string
	logFileCmd := "kubectl exec -it -n verrazzano-system  vmi-system-es-master-0 -- ls -alt --time=ctime /tmp/ | grep verrazzano | cut -d ' ' -f9 | head -1"
	cmdArgs = append(cmdArgs, "/bin/sh")
	cmdArgs = append(cmdArgs, "-c")
	cmdArgs = append(cmdArgs, logFileCmd)

	var kcmd BashCommand
	kcmd.Timeout = 1 * time.Minute
	kcmd.CommandArgs = cmdArgs
	bashResponse := Runner(&kcmd, log)
	logFileName := strings.TrimSpace(strings.Trim(bashResponse.StandardOut.String(), "\n"))

	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		log.Errorf("Failed to get clientset with error: %v", err)
		return err
	}

	config, err := k8sutil.GetKubeConfig()
	if err != nil {
		log.Errorf("Failed to get config with error: %v", err)
		return err
	}

	podSpec, err := clientset.CoreV1().Pods(constants.VerrazzanoSystemNamespace).Get(context.TODO(), "vmi-system-es-master-0", metav1.GetOptions{})
	if err != nil {
		return err
	}

	var execCmd []string
	execCmd = append(execCmd, "cat")
	execCmd = append(execCmd, fmt.Sprintf("/tmp/%s", logFileName))
	stdout, _, err := k8sutil.ExecPod(clientset, config, podSpec, "es-master", execCmd)
	if err != nil {
		return err
	}
	log.Infof(stdout)
	return nil
}

// GetRancherBackupFileName gets the filename backed up to object store
func GetRancherBackupFileName(backupName string, log *zap.SugaredLogger) (string, error) {
	var cmdArgs []string
	cmdArgs = append(cmdArgs, "kubectl")
	cmdArgs = append(cmdArgs, "get")
	cmdArgs = append(cmdArgs, "backup.resources.cattle.io")
	cmdArgs = append(cmdArgs, backupName)
	cmdArgs = append(cmdArgs, "-o")
	cmdArgs = append(cmdArgs, "jsonpath={.status.filename}")

	var kcmd BashCommand
	kcmd.Timeout = 2 * time.Minute
	kcmd.CommandArgs = cmdArgs

	fileNameResponse := Runner(&kcmd, log)
	if fileNameResponse.CommandError != nil {
		return "", fileNameResponse.CommandError
	}
	return strings.TrimSpace(strings.Trim(fileNameResponse.StandardOut.String(), "\n")), nil
}

// WaitForPodsShell utility to wait for all pods to be ready in a given namespace
func WaitForPodsShell(namespace string, log *zap.SugaredLogger) error {
	var cmdArgs []string
	cmdArgs = append(cmdArgs, "kubectl")
	cmdArgs = append(cmdArgs, "wait")
	cmdArgs = append(cmdArgs, "-n")
	cmdArgs = append(cmdArgs, namespace)
	cmdArgs = append(cmdArgs, "--for=condition=ready")
	cmdArgs = append(cmdArgs, "pod")
	cmdArgs = append(cmdArgs, "--all")
	cmdArgs = append(cmdArgs, "--timeout=10m")

	var kcmd BashCommand
	kcmd.Timeout = 2 * time.Minute
	kcmd.CommandArgs = cmdArgs

	waitCmd := Runner(&kcmd, log)
	if waitCmd.CommandError != nil {
		return waitCmd.CommandError
	}
	time.Sleep(1 * time.Minute)
	return nil
}

// CheckOperatorOperationProgress is a common utility to check the progress of backup and restore
// operation for both velero and rancher.
func CheckOperatorOperationProgress(operator, operation, namespace, objectName string, log *zap.SugaredLogger) error {
	var cmdArgs []string
	var kind, jsonPath string

	cmdArgs = append(cmdArgs, "kubectl")
	cmdArgs = append(cmdArgs, "get")

	switch operator {
	case "velero":
		cmdArgs = append(cmdArgs, "-n")
		cmdArgs = append(cmdArgs, namespace)
		kind = "velero.io"
		jsonPath = "{.status.phase}"
	case "rancher":
		kind = "resources.cattle.io"
		jsonPath = "{.status.conditions[].message}"
	}

	cmdArgs = append(cmdArgs, fmt.Sprintf("%s.%s", operation, kind))
	cmdArgs = append(cmdArgs, objectName)
	cmdArgs = append(cmdArgs, "-o")
	cmdArgs = append(cmdArgs, fmt.Sprintf("jsonpath=%s", jsonPath))
	cmdArgs = append(cmdArgs, "--ignore-not-found")

	var kcmd BashCommand
	kcmd.Timeout = 1 * time.Minute
	kcmd.CommandArgs = cmdArgs
	return RetryAndCheckShellCommandResponse(30, &kcmd, operation, objectName, log)
}

// CreateVeleroBackupLocationObject creates velero backup object location
func CreateVeleroBackupLocationObject(backupStorageName, backupSecretName string, log *zap.SugaredLogger) error {
	var b bytes.Buffer
	template, _ := template.New("velero-backup-location").Parse(VeleroBackupLocation)

	data := VeleroBackupLocationObjectData{
		VeleroBackupStorageName:          backupStorageName,
		VeleroNamespaceName:              VeleroNameSpace,
		VeleroObjectStoreBucketName:      OciBucketName,
		VeleroSecretName:                 backupSecretName,
		VeleroObjectStorageNamespaceName: OciNamespaceName,
		VeleroBackupRegion:               BackupRegion,
	}
	template.Execute(&b, data)
	err := pkg.CreateOrUpdateResourceFromBytes(b.Bytes())
	if err != nil {
		log.Errorf("Error creating velero backup location ", zap.Error(err))
		//return err
	}

	var cmdArgs []string
	cmdArgs = append(cmdArgs, "kubectl")
	cmdArgs = append(cmdArgs, "get")
	cmdArgs = append(cmdArgs, "backupstoragelocation.velero.io")
	cmdArgs = append(cmdArgs, "-n")
	cmdArgs = append(cmdArgs, VeleroNameSpace)
	cmdArgs = append(cmdArgs, backupStorageName)
	cmdArgs = append(cmdArgs, "-o")
	cmdArgs = append(cmdArgs, "custom-columns=:metadata.name")
	cmdArgs = append(cmdArgs, "--no-headers")
	cmdArgs = append(cmdArgs, "--ignore-not-found")

	var kcmd BashCommand
	kcmd.Timeout = 1 * time.Minute
	kcmd.CommandArgs = cmdArgs
	cmdResponse := Runner(&kcmd, log)
	if cmdResponse.CommandError != nil {
		return cmdResponse.CommandError
	}
	storageNameRetrieved := strings.TrimSpace(strings.Trim(cmdResponse.StandardOut.String(), "\n"))
	if storageNameRetrieved == backupStorageName {
		log.Errorf("backup storage location '%s' already created", backupStorageName)
	}
	return nil
}
