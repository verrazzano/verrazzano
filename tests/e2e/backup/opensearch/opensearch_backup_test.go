// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package backup

import (
	"bytes"
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/test/framework/metrics"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/tests/e2e/backup"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strconv"
	"strings"
	"text/template"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

const (
	shortWaitTimeout                    = 10 * time.Minute
	shortPollingInterval                = 10 * time.Second
	waitTimeout                         = 15 * time.Minute
	pollingInterval                     = 30 * time.Second
	vmoDeploymentName                   = "verrazzano-monitoring-operator"
	osStsName                           = "vmi-system-es-master"
	osStsPvcPrefix                      = "elasticsearch-master-vmi-system-es-master"
	osDataDepPrefix                     = "vmi-system-es-data"
	osIngestDeployment                  = "vmi-system-es-ingest"
	osDepPvcPrefix                      = "vmi-system-es-data"
	objectStoreCredsAccessKeyName       = "aws_access_key_id"     //nolint:gosec //#gosec G101 //#gosec G204
	objectStoreCredsSecretAccessKeyName = "aws_secret_access_key" //nolint:gosec //#gosec G101 //#gosec G204
)

var _ = t.BeforeSuite(func() {
	start := time.Now()
	backup.GatherInfo()
	backupPrerequisites()
	metrics.Emit(t.Metrics.With("deployment_elapsed_time", time.Since(start).Milliseconds()))
})

var _ = t.AfterSuite(func() {
	start := time.Now()
	cleanUpVelero()
	metrics.Emit(t.Metrics.With("undeployment_elapsed_time", time.Since(start).Milliseconds()))
})

var t = framework.NewTestFramework("opensearch-backup")

// CreateCredentialsSecretFromFile creates opaque secret from the given map of values
func CreateCredentialsSecretFromFile(namespace string, name string) error {
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		t.Logs.Errorf("Failed to get clientset with error: %v", err)
		return err
	}

	var b bytes.Buffer
	template, _ := template.New("testsecrets").Parse(backup.SecretsData)
	data := backup.AccessData{
		AccessName:             objectStoreCredsAccessKeyName,
		ScrtName:               objectStoreCredsSecretAccessKeyName,
		ObjectStoreAccessValue: backup.OciOsAccessKey,
		ObjectStoreScrt:        backup.OciOsAccessSecretKey,
	}
	template.Execute(&b, data)

	secretData := make(map[string]string)
	secretData["cloud"] = b.String()

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Type:       corev1.SecretTypeOpaque,
		StringData: secretData,
	}

	_, err = clientset.CoreV1().Secrets(namespace).Create(context.TODO(), secret, metav1.CreateOptions{})
	if err != nil {
		t.Logs.Errorf("Error creating secret ", zap.Error(err))
		return err
	}
	return nil
}

func DeleteSecret(namespace string, name string) error {
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		t.Logs.Errorf("Failed to get clientset with error: %v", err)
		return err
	}

	err = clientset.CoreV1().Secrets(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
	if err != nil {
		t.Logs.Errorf("Error deleting secret ", zap.Error(err))
		return err
	}
	return nil
}

// CreateVeleroBackupLocationObject creates opaque secret from the given map of values
func CreateVeleroBackupLocationObject() error {
	var b bytes.Buffer
	template, _ := template.New("velero-backup-location").Parse(backup.VeleroBackupLocation)

	data := backup.VeleroBackupLocationObjectData{
		VeleroBackupStorageName:          backup.BackupStorageName,
		VeleroNamespaceName:              backup.VeleroNameSpace,
		VeleroObjectStoreBucketName:      backup.OciBucketName,
		VeleroSecretName:                 backup.VeleroSecretName,
		VeleroObjectStorageNamespaceName: backup.OciNamespaceName,
		VeleroBackupRegion:               backup.BackupRegion,
	}
	template.Execute(&b, data)
	err := pkg.CreateOrUpdateResourceFromBytes(b.Bytes())
	if err != nil {
		t.Logs.Errorf("Error creating velero backup loaction ", zap.Error(err))
		//return err
	}

	var cmdArgs []string
	cmdArgs = append(cmdArgs, "kubectl")
	cmdArgs = append(cmdArgs, "get")
	cmdArgs = append(cmdArgs, "backupstoragelocation.velero.io")
	cmdArgs = append(cmdArgs, "-n")
	cmdArgs = append(cmdArgs, backup.VeleroNameSpace)
	cmdArgs = append(cmdArgs, backup.BackupStorageName)
	cmdArgs = append(cmdArgs, "-o")
	cmdArgs = append(cmdArgs, "custom-columns=:metadata.name")
	cmdArgs = append(cmdArgs, "--no-headers")
	cmdArgs = append(cmdArgs, "--ignore-not-found")

	var kcmd backup.BashCommand
	kcmd.Timeout = 1 * time.Minute
	kcmd.CommandArgs = cmdArgs
	cmdResponse := backup.Runner(&kcmd, t.Logs)
	if cmdResponse.CommandError != nil {
		return cmdResponse.CommandError
	}
	storageNameRetrieved := strings.TrimSpace(strings.Trim(cmdResponse.StandardOut.String(), "\n"))
	if storageNameRetrieved == backup.BackupStorageName {
		t.Logs.Errorf("backup storage location '%s' already created", backup.BackupStorageName)
	}
	return nil
}

// CreateVeleroBackupObject creates opaque secret from the given map of values
func CreateVeleroBackupObject() error {
	var b bytes.Buffer
	template, _ := template.New("velero-backup").Parse(backup.VeleroBackup)
	data := backup.VeleroBackupObject{
		VeleroNamespaceName:              backup.VeleroNameSpace,
		VeleroBackupName:                 backup.BackupOpensearchName,
		VeleroBackupStorageName:          backup.BackupStorageName,
		VeleroOpensearchHookResourceName: backup.BackupResourceName,
	}
	template.Execute(&b, data)
	err := backup.DynamicSSA(context.TODO(), b.String(), t.Logs)
	if err != nil {
		t.Logs.Errorf("Error creating velero backup object", zap.Error(err))
		return err
	}
	return nil
}

func CreateVeleroRestoreObject() error {
	var b bytes.Buffer
	template, _ := template.New("velero-restore").Parse(backup.VeleroRestore)
	data := backup.VeleroRestoreObject{
		VeleroRestore:                    backup.RestoreOpensearchName,
		VeleroNamespaceName:              backup.VeleroNameSpace,
		VeleroBackupName:                 backup.BackupOpensearchName,
		VeleroOpensearchHookResourceName: backup.BackupResourceName,
	}

	template.Execute(&b, data)
	err := backup.DynamicSSA(context.TODO(), b.String(), t.Logs)
	if err != nil {
		t.Logs.Errorf("Error creating velero restore object ", zap.Error(err))
		return err
	}
	return nil
}

func GetBackupID() error {
	esURL, err := backup.GetEsURL(t.Logs)
	if err != nil {
		t.Logs.Infof("Error getting es url ", zap.Error(err))
		return err
	}

	vzPasswd, err := backup.GetVZPasswd(t.Logs)
	if err != nil {
		t.Logs.Errorf("Error getting vz passwd ", zap.Error(err))
		return err
	}
	var cmdArgs []string
	url := strconv.Quote(fmt.Sprintf("%s/verrazzano-system/_search?from=0&size=1", esURL))
	creds := fmt.Sprintf("verrazzano:%s", vzPasswd)
	jqIDFetch := "| jq -r '.hits.hits[0]._id'"
	curlCmd := fmt.Sprintf("curl -ks %s -u %s %s", url, creds, jqIDFetch)
	cmdArgs = append(cmdArgs, "/bin/sh")
	cmdArgs = append(cmdArgs, "-c")
	cmdArgs = append(cmdArgs, curlCmd)

	var kcmd backup.BashCommand
	kcmd.Timeout = 2 * time.Minute
	kcmd.CommandArgs = cmdArgs

	curlResponse := backup.Runner(&kcmd, t.Logs)
	if curlResponse.CommandError != nil {
		return curlResponse.CommandError
	}
	backup.BackupID = strings.TrimSpace(strings.Trim(curlResponse.StandardOut.String(), "\n"))
	t.Logs.Infof("BackupId ===> = '%s", backup.BackupID)
	if backup.BackupID != "" {
		t.Logs.Errorf("BackupId has already been retrieved = '%s", backup.BackupID)
		//return fmt.Errorf("backupId has already been retrieved = '%s", BackupID)
	}
	return nil
}

func IsRestoreSuccessful() bool {
	esURL, err := backup.GetEsURL(t.Logs)
	if err != nil {
		t.Logs.Infof("Error getting es url ", zap.Error(err))
		return false
	}

	vzPasswd, err := backup.GetVZPasswd(t.Logs)
	if err != nil {
		t.Logs.Infof("Error getting vz passwd ", zap.Error(err))
		return false
	}
	var b bytes.Buffer
	template, _ := template.New("velero-restore-verify").Parse(backup.EsQueryBody)
	data := backup.EsQueryObject{
		BackupIDBeforeBackup: backup.BackupID,
	}
	template.Execute(&b, data)

	var cmdArgs []string
	header := "Content-Type: application/json"
	url := strconv.Quote(fmt.Sprintf("%s/verrazzano-system/_search?", esURL))
	creds := fmt.Sprintf("verrazzano:%s", vzPasswd)
	jqIDFetch := "| jq -r '.hits.hits[0]._id'"
	curlCmd := fmt.Sprintf("curl -ks -H %s %s -u %s -d '%s' %s", strconv.Quote(header), url, creds, b.String(), jqIDFetch)
	cmdArgs = append(cmdArgs, "/bin/sh")
	cmdArgs = append(cmdArgs, "-c")
	cmdArgs = append(cmdArgs, curlCmd)

	var kcmd backup.BashCommand
	kcmd.Timeout = 2 * time.Minute
	kcmd.CommandArgs = cmdArgs

	curlResponse := backup.Runner(&kcmd, t.Logs)
	if curlResponse.CommandError != nil {
		return false
	}
	backupIDFetched := strings.TrimSpace(strings.Trim(curlResponse.StandardOut.String(), "\n"))
	return backupIDFetched == backup.BackupID
}

/*
func CheckBackupProgress() error {
	var cmdArgs []string
	cmdArgs = append(cmdArgs, "kubectl")
	cmdArgs = append(cmdArgs, "get")
	cmdArgs = append(cmdArgs, "backup.velero.io")
	cmdArgs = append(cmdArgs, "-n")
	cmdArgs = append(cmdArgs, VeleroNameSpace)
	cmdArgs = append(cmdArgs, BackupOpensearchName)
	cmdArgs = append(cmdArgs, "-o")
	cmdArgs = append(cmdArgs, "jsonpath={.status.phase}")
	cmdArgs = append(cmdArgs, "--ignore-not-found")

	var kcmd BashCommand
	kcmd.Timeout = 1 * time.Minute
	kcmd.CommandArgs = cmdArgs
	return RetryAndCheckShellCommandResponse(100, &kcmd, "backup", BackupOpensearchName, t.Logs)
}

func CheckRestoreProgress() error {
	var cmdArgs []string
	cmdArgs = append(cmdArgs, "kubectl")
	cmdArgs = append(cmdArgs, "get")
	cmdArgs = append(cmdArgs, "restore.velero.io")
	cmdArgs = append(cmdArgs, "-n")
	cmdArgs = append(cmdArgs, VeleroNameSpace)
	cmdArgs = append(cmdArgs, RestoreOpensearchName)
	cmdArgs = append(cmdArgs, "-o")
	cmdArgs = append(cmdArgs, "jsonpath={.status.phase}")
	cmdArgs = append(cmdArgs, "--ignore-not-found")

	var kcmd BashCommand
	kcmd.Timeout = 1 * time.Minute
	kcmd.CommandArgs = cmdArgs
	return RetryAndCheckShellCommandResponse(100, &kcmd, "restore", BackupOpensearchName, t.Logs)
}
*/

func NukeOpensearch() error {
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		t.Logs.Errorf("Failed to get clientset with error: %v", err)
		return err
	}

	t.Logs.Infof("Scaling down VMO")
	getScale, err := clientset.AppsV1().Deployments(constants.VerrazzanoSystemNamespace).GetScale(context.TODO(), vmoDeploymentName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	newScale := *getScale
	newScale.Spec.Replicas = 0

	_, err = clientset.AppsV1().Deployments(constants.VerrazzanoSystemNamespace).UpdateScale(context.TODO(), vmoDeploymentName, &newScale, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	t.Logs.Infof("Deleting opensearch master sts")
	err = clientset.AppsV1().StatefulSets(constants.VerrazzanoSystemNamespace).Delete(context.TODO(), osStsName, metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	t.Logs.Infof("Deleting opensearch data deployments")
	for i := 0; i < 3; i++ {
		err = clientset.AppsV1().Deployments(constants.VerrazzanoSystemNamespace).Delete(context.TODO(), fmt.Sprintf("%s-%v", osDataDepPrefix, i), metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	t.Logs.Infof("Deleting opensearch ingest deployment")
	err = clientset.AppsV1().Deployments(constants.VerrazzanoSystemNamespace).Delete(context.TODO(), osIngestDeployment, metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	t.Logs.Infof("Deleting opensearch master pvc if still present")
	for i := 0; i < 3; i++ {
		err = clientset.CoreV1().PersistentVolumeClaims(constants.VerrazzanoSystemNamespace).Delete(context.TODO(), fmt.Sprintf("%s-%v", osStsPvcPrefix, i), metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	t.Logs.Infof("Deleting opensearch data pvc")
	for i := 0; i < 3; i++ {
		if i == 0 {
			err = clientset.CoreV1().PersistentVolumeClaims(constants.VerrazzanoSystemNamespace).Delete(context.TODO(), osDepPvcPrefix, metav1.DeleteOptions{})
		} else {
			err = clientset.CoreV1().PersistentVolumeClaims(constants.VerrazzanoSystemNamespace).Delete(context.TODO(), fmt.Sprintf("%s-%v", osDepPvcPrefix, i), metav1.DeleteOptions{})
		}
		if err != nil {
			return err
		}
	}

	err = backup.CheckPodsTerminated("verrazzano-component=opensearch", constants.VerrazzanoSystemNamespace, t.Logs)
	if err != nil {
		return err
	}

	return backup.CheckPvcsTerminated("verrazzano-component=opensearch", constants.VerrazzanoSystemNamespace, t.Logs)
}

// 'It' Wrapper to only run spec if the Velero is supported on the current Verrazzano version
func WhenVeleroInstalledIt(description string, f func()) {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		t.It(description, func() {
			Fail(fmt.Sprintf("Failed to get default kubeconfig path: %s", err.Error()))
		})
	}
	supported, err := pkg.IsVerrazzanoMinVersion("1.4.0", kubeconfigPath)
	if err != nil {
		t.It(description, func() {
			Fail(fmt.Sprintf("Failed to check Verrazzano version 1.4.0: %s", err.Error()))
		})
	}
	if supported {
		t.It(description, f)
	} else {
		t.Logs.Infof("Skipping check '%v', the Velero is not supported", description)
	}
}

func backupPrerequisites() {
	t.Logs.Info("Setup backup pre-requisites")
	t.Logs.Info("Create backup secret for velero backup objects")
	Eventually(func() error {
		return CreateCredentialsSecretFromFile(backup.VeleroNameSpace, backup.VeleroSecretName)
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

	t.Logs.Info("Create backup storage location for velero backup objects")
	Eventually(func() error {
		return CreateVeleroBackupLocationObject()
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

	t.Logs.Info("Get backup id before starting the backup process")
	Eventually(func() error {
		return GetBackupID()
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

}

func cleanUpVelero() {
	t.Logs.Info("Cleanup backup and restore objects")

	t.Logs.Info("Cleanup restore object")
	Eventually(func() error {
		return backup.VeleroObjectDelete("restore", backup.RestoreOpensearchName, backup.VeleroNameSpace, t.Logs)
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

	t.Logs.Info("Cleanup backup object")
	Eventually(func() error {
		return backup.VeleroObjectDelete("backup", backup.BackupOpensearchName, backup.VeleroNameSpace, t.Logs)
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

	t.Logs.Info("Cleanup backup storage object")
	Eventually(func() error {
		return backup.VeleroObjectDelete("storage", backup.BackupStorageName, backup.VeleroNameSpace, t.Logs)
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

	t.Logs.Info("Cleanup velero secrets")
	Eventually(func() error {
		return DeleteSecret(backup.VeleroNameSpace, backup.VeleroSecretName)
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

}

var _ = t.Describe("Backup Flow,", Label("f:platform-verrazzano.backup"), Serial, func() {

	t.Context("Start backup after velero backup storage location created", func() {
		WhenVeleroInstalledIt("Start backup after velero backup storage location created", func() {
			Eventually(func() error {
				return CreateVeleroBackupObject()
			}, waitTimeout, pollingInterval).Should(BeNil())
		})
	})

	t.Context("Check backup progress after velero backup object was created", func() {
		WhenVeleroInstalledIt("Check backup progress after velero backup object was created", func() {
			Eventually(func() error {
				return backup.CheckOperatorOperationProgress("velero", "backup", backup.VeleroNameSpace, backup.BackupOpensearchName, t.Logs)
			}, waitTimeout, pollingInterval).Should(BeNil())
		})
	})

	t.Context("Fetch logs after backup is complete", func() {
		WhenVeleroInstalledIt("Fetch logs after backup is complete", func() {
			Eventually(func() error {
				return backup.DisplayHookLogs(t.Logs)
			}, waitTimeout, pollingInterval).Should(BeNil())
		})
	})

	t.Context("Cleanup opensearch once backup is done", func() {
		WhenVeleroInstalledIt("Cleanup opensearch once backup is done", func() {
			Eventually(func() error {
				return NukeOpensearch()
			}, waitTimeout, pollingInterval).Should(BeNil())
		})

	})

	t.Context("Start restore after velero backup is completed", func() {
		WhenVeleroInstalledIt("Start restore after velero backup is completed", func() {
			Eventually(func() error {
				return CreateVeleroRestoreObject()
			}, waitTimeout, pollingInterval).Should(BeNil())
		})
	})

	t.Context("Check velero restore progress", func() {
		WhenVeleroInstalledIt("Check velero restore progress", func() {
			Eventually(func() error {
				return backup.CheckOperatorOperationProgress("velero", "restore", backup.VeleroNameSpace, backup.RestoreOpensearchName, t.Logs)
			}, waitTimeout, pollingInterval).Should(BeNil())
		})
	})

	t.Context("Is Restore good? Verify restore", func() {
		WhenVeleroInstalledIt("Is Restore good? Verify restore", func() {
			Eventually(func() bool {
				return IsRestoreSuccessful()
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})

	})

	t.Context("Fetch logs after restore is complete", func() {
		WhenVeleroInstalledIt("Fetch logs after restore is complete", func() {
			Eventually(func() error {
				return backup.DisplayHookLogs(t.Logs)
			}, waitTimeout, pollingInterval).Should(BeNil())
		})
	})

})
