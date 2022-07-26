// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package backup

import (
	"bytes"
	"context"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/verrazzano/verrazzano/pkg/test/framework/metrics"
	"github.com/verrazzano/verrazzano/tests/e2e/backup"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"text/template"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

const (
	shortWaitTimeout     = 10 * time.Minute
	shortPollingInterval = 10 * time.Second
	waitTimeout          = 15 * time.Minute
	pollingInterval      = 30 * time.Second
)

var _ = t.BeforeSuite(func() {
	start := time.Now()
	backup.GatherInfo()
	backupPrerequisites()
	metrics.Emit(t.Metrics.With("deployment_elapsed_time", time.Since(start).Milliseconds()))
})

var _ = t.AfterSuite(func() {
	start := time.Now()
	cleanUpRancher()
	metrics.Emit(t.Metrics.With("undeployment_elapsed_time", time.Since(start).Milliseconds()))
})

var t = framework.NewTestFramework("opensearch-backup")

func CreateSecretFromMap(namespace string, name string) error {
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		t.Logs.Errorf("Failed to get clientset with error: %v", err)
		return err
	}

	secretData := make(map[string]string)
	secretData["accessKey"] = backup.OciOsAccessKey
	secretData["secretKey"] = backup.OciOsAccessSecretKey

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

// CreateRancherBackupObject creates opaque secret from the given map of values
func CreateRancherBackupObject() error {
	var b bytes.Buffer
	template, _ := template.New("rancher-backup").Parse(backup.RancherBackup)
	data := backup.RancherBackupData{
		RancherBackupName: backup.BackupRancherName,
		RancherSecretData: backup.RancherObjectStoreData{
			RancherSecretName:                 backup.RancherSecretName,
			RancherSecretNamespaceName:        backup.VeleroNameSpace,
			RancherObjectStoreBucketName:      backup.OciBucketName,
			RancherBackupRegion:               backup.BackupRegion,
			RancherObjectStorageNamespaceName: backup.OciNamespaceName,
		},
	}
	spew.Dump(data)
	template.Execute(&b, data)
	spew.Dump(b)
	err := backup.DynamicSSA(context.TODO(), b.String(), t.Logs)
	if err != nil {
		t.Logs.Errorf("Error creating rancher backup object", zap.Error(err))
		return err
	}
	return nil
}

// CreateRancherBackupObject creates opaque secret from the given map of values
func CreateRancherRestoreObject() error {

	rancherFileName, err := backup.GetRancherBackupFileName(backup.BackupRancherName, t.Logs)
	if err != nil {
		return err
	}

	backup.RancherBackupFileName = rancherFileName

	var b bytes.Buffer
	template, _ := template.New("rancher-backup").Parse(backup.RancherRestore)
	data := backup.RancherRestoreData{
		RancherRestoreName: backup.RestoreRancherName,
		BackupFileName:     backup.RancherBackupFileName,
		RancherSecretData: backup.RancherObjectStoreData{
			RancherSecretName:                 backup.RancherSecretName,
			RancherSecretNamespaceName:        backup.VeleroNameSpace,
			RancherObjectStoreBucketName:      backup.OciBucketName,
			RancherBackupRegion:               backup.BackupRegion,
			RancherObjectStorageNamespaceName: backup.OciNamespaceName,
		},
	}

	spew.Dump(data)
	template.Execute(&b, data)
	spew.Dump(b)
	err = backup.DynamicSSA(context.TODO(), b.String(), t.Logs)
	if err != nil {
		t.Logs.Errorf("Error creating rancher backup object", zap.Error(err))
		return err
	}
	t.Logs.Infof("Rancher backup filename = %s", backup.RancherBackupFileName)
	return nil
}

// 'It' Wrapper to only run spec if the Velero is supported on the current Verrazzano version
func WhenRancherBackupInstalledIt(description string, f func()) {
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

	t.Logs.Info("Create backup secret for rancher backup objects")
	Eventually(func() error {
		return CreateSecretFromMap(backup.VeleroNameSpace, backup.RancherSecretName)
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

}

func cleanUpRancher() {
	t.Logs.Info("Cleanup backup and restore objects")

	t.Logs.Info("Cleanup rancher secrets")
	Eventually(func() error {
		return DeleteSecret(backup.VeleroNameSpace, backup.RancherSecretName)
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

}

var _ = t.Describe("Backup Flow,", Label("f:platform-verrazzano.rancher-backup"), Serial, func() {

	// Rancher backup section

	t.Context("Start rancher backup", func() {
		WhenRancherBackupInstalledIt("Start rancher backup", func() {
			Eventually(func() error {
				return CreateRancherBackupObject()
			}, waitTimeout, pollingInterval).Should(BeNil())
		})
	})

	t.Context("Check backup progress after rancher backup object was created", func() {
		WhenRancherBackupInstalledIt("Check backup progress after rancher backup object was created", func() {
			Eventually(func() error {
				//return CheckBackupProgress()
				return backup.CheckOperatorOperationProgress("rancher", "backup", backup.VeleroNameSpace, backup.BackupRancherName, t.Logs)
			}, waitTimeout, pollingInterval).Should(BeNil())
		})
	})

	t.Context("Start restore after rancher backup is completed", func() {
		WhenRancherBackupInstalledIt("Start restore after rancher backup is completed", func() {
			Eventually(func() error {
				return CreateRancherRestoreObject()
			}, waitTimeout, pollingInterval).Should(BeNil())
		})
	})

	t.Context("Check rancher restore progress", func() {
		WhenRancherBackupInstalledIt("Check rancher restore progress", func() {
			Eventually(func() error {
				//return CheckRestoreProgress()
				return backup.CheckOperatorOperationProgress("rancher", "restore", backup.VeleroNameSpace, backup.RestoreRancherName, t.Logs)
			}, waitTimeout, pollingInterval).Should(BeNil())
		})
	})

})
