// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package backup

import (
	"bytes"
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/test/framework/metrics"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	common "github.com/verrazzano/verrazzano/tests/e2e/backup/helpers"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
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
	vmoDeploymentName    = "verrazzano-monitoring-operator"
	osStsName            = "vmi-system-es-master"
	osStsPvcPrefix       = "elasticsearch-master-vmi-system-es-master"
	osDataDepPrefix      = "vmi-system-es-data"
	osIngestDeployment   = "vmi-system-es-ingest"
	osDepPvcPrefix       = "vmi-system-es-data"
	idSearchExactURL     = "verrazzano-system/_search?from=0&size=1"
	idSearchAllURL       = "verrazzano-system/_search?"
)

var _ = t.BeforeSuite(func() {
	start := time.Now()
	common.GatherInfo()
	backupPrerequisites()
	metrics.Emit(t.Metrics.With("deployment_elapsed_time", time.Since(start).Milliseconds()))
})

var _ = t.AfterSuite(func() {
	start := time.Now()
	cleanUpVelero()
	metrics.Emit(t.Metrics.With("undeployment_elapsed_time", time.Since(start).Milliseconds()))
})

var t = framework.NewTestFramework("opensearch-backup")

// CreateVeleroBackupObject creates velero backup object that starts the backup
func CreateVeleroBackupObject() error {
	var b bytes.Buffer
	template, _ := template.New("velero-backup").Parse(common.VeleroBackup)
	data := common.VeleroBackupObject{
		VeleroNamespaceName:              common.VeleroNameSpace,
		VeleroBackupName:                 common.BackupOpensearchName,
		VeleroBackupStorageName:          common.BackupOpensearchStorageName,
		VeleroOpensearchHookResourceName: common.BackupResourceName,
	}
	template.Execute(&b, data)
	err := common.DynamicSSA(context.TODO(), b.String(), t.Logs)
	if err != nil {
		t.Logs.Errorf("Error creating velero backup object", zap.Error(err))
		return err
	}
	return nil
}

// CreateVeleroRestoreObject creates velero restore object that starts restore
func CreateVeleroRestoreObject() error {
	var b bytes.Buffer
	template, _ := template.New("velero-restore").Parse(common.VeleroRestore)
	data := common.VeleroRestoreObject{
		VeleroRestore:                    common.RestoreOpensearchName,
		VeleroNamespaceName:              common.VeleroNameSpace,
		VeleroBackupName:                 common.BackupOpensearchName,
		VeleroOpensearchHookResourceName: common.BackupResourceName,
	}

	template.Execute(&b, data)
	err := common.DynamicSSA(context.TODO(), b.String(), t.Logs)
	if err != nil {
		t.Logs.Errorf("Error creating velero restore object ", zap.Error(err))
		return err
	}
	return nil
}

// GetBackupID fetches an opensearch id before starting the backup
// This will be used to compare the restore process
func GetBackupID() error {
	esURL, err := common.GetEsURL(t.Logs)
	if err != nil {
		t.Logs.Infof("Error getting es url ", zap.Error(err))
		return err
	}
	vzPasswd, err := common.GetVZPasswd(t.Logs)
	if err != nil {
		t.Logs.Errorf("Error getting vz passwd ", zap.Error(err))
		return err
	}

	httpClient := pkg.EventuallyVerrazzanoRetryableHTTPClient()
	searchURL := fmt.Sprintf("%s/%s", esURL, idSearchExactURL)
	creds := fmt.Sprintf("verrazzano:%s", vzPasswd)
	parsedJSON, err := common.HTTPHelper(httpClient, "GET", searchURL, creds, "Basic", http.StatusOK, nil, t.Logs)
	if err != nil {
		t.Logs.Errorf("Error while retrieving http data %v", zap.Error(err))
		return err
	}
	common.BackupID = fmt.Sprintf("%s", parsedJSON.Path("hits.hits.0._id").Data())

	t.Logs.Infof("BackupId ===> = '%s'", common.BackupID)
	return nil
}

// IsRestoreSuccessful fetches the same backup id and returns the result
func IsRestoreSuccessful() string {
	esURL, err := common.GetEsURL(t.Logs)
	if err != nil {
		t.Logs.Infof("Error getting es url ", zap.Error(err))
		return ""
	}

	vzPasswd, err := common.GetVZPasswd(t.Logs)
	if err != nil {
		t.Logs.Infof("Error getting vz passwd ", zap.Error(err))
		return ""
	}

	httpClient := pkg.EventuallyVerrazzanoRetryableHTTPClient()
	fetchURL := fmt.Sprintf("%s/%s", esURL, idSearchAllURL)
	creds := fmt.Sprintf("verrazzano:%s", vzPasswd)
	parsedJSON, err := common.HTTPHelper(httpClient, "GET", fetchURL, creds, "Basic", http.StatusOK, nil, t.Logs)
	if err != nil {
		t.Logs.Errorf("Error while retrieving http data %v", zap.Error(err))
		return ""
	}
	return fmt.Sprintf("%s", parsedJSON.Search("hits", "hits", "0", "_id").Data())
}

// NukeOpensearch is used to destroy the opensearch cluster including data
// This is only done after a successful backup was taken
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

	err = common.CheckPodsTerminated("verrazzano-component=opensearch", constants.VerrazzanoSystemNamespace, t.Logs)
	if err != nil {
		return err
	}

	return common.CheckPvcsTerminated("verrazzano-component=opensearch", constants.VerrazzanoSystemNamespace, t.Logs)
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

// Run as part of BeforeSuite
func backupPrerequisites() {
	t.Logs.Info("Setup backup pre-requisites")
	t.Logs.Info("Create backup secret for velero backup objects")
	Eventually(func() error {
		return common.CreateCredentialsSecretFromFile(common.VeleroNameSpace, common.VeleroOpenSearchSecretName, t.Logs)
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

	t.Logs.Info("Create backup storage location for velero backup objects")
	Eventually(func() error {
		return common.CreateVeleroBackupLocationObject(common.BackupOpensearchStorageName, common.VeleroOpenSearchSecretName, t.Logs)
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

	t.Logs.Info("Get backup id before starting the backup process")
	Eventually(func() error {
		return GetBackupID()
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

}

// Run as part of AfterSuite
func cleanUpVelero() {
	t.Logs.Info("Cleanup backup and restore objects")

	t.Logs.Info("Cleanup restore object")
	Eventually(func() error {
		return common.CrdPruner("velero.io", "v1", "restores", common.RestoreOpensearchName, common.VeleroNameSpace, t.Logs)
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

	t.Logs.Info("Cleanup backup object")
	Eventually(func() error {
		return common.CrdPruner("velero.io", "v1", "backups", common.BackupOpensearchName, common.VeleroNameSpace, t.Logs)
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

	t.Logs.Info("Cleanup backup storage object")
	Eventually(func() error {
		return common.CrdPruner("velero.io", "v1", "backupstoragelocations", common.BackupOpensearchStorageName, common.VeleroNameSpace, t.Logs)
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

	t.Logs.Info("Cleanup velero secrets")
	Eventually(func() error {
		return common.DeleteSecret(common.VeleroNameSpace, common.VeleroOpenSearchSecretName, t.Logs)
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
				return common.TrackOperationProgress("velero", "backups", common.BackupOpensearchName, common.VeleroNameSpace, t.Logs)
			}, waitTimeout, pollingInterval).Should(BeNil())
		})
	})

	t.Context("Fetch logs after backup is complete", func() {
		WhenVeleroInstalledIt("Fetch logs after backup is complete", func() {
			Eventually(func() error {
				return common.DisplayHookLogs(t.Logs)
			}, waitTimeout, pollingInterval).Should(BeNil())
		})
	})

	/*

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
					return common.TrackOperationProgress("velero", "restores", common.RestoreOpensearchName, common.VeleroNameSpace, t.Logs)
				}, waitTimeout, pollingInterval).Should(BeNil())
			})
		})

		t.Context("Fetch logs after restore is complete", func() {
			WhenVeleroInstalledIt("Fetch logs after restore is complete", func() {
				Eventually(func() error {
					return common.DisplayHookLogs(t.Logs)
				}, waitTimeout, pollingInterval).Should(BeNil())
			})
		})

		t.Context("Is Restore good? Verify restore", func() {
			WhenVeleroInstalledIt("Is Restore good? Verify restore", func() {
				Eventually(func() string {
					return IsRestoreSuccessful()
				}, waitTimeout, pollingInterval).Should(Equal(common.BackupID))
			})

		})
	*/

})
