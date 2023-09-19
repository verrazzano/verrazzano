// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	"bytes"
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	common "github.com/verrazzano/verrazzano/tests/e2e/backup/helpers"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework/metrics"
	"go.uber.org/zap"
	k8serror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"net/http"
	"text/template"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
)

const (
	shortWaitTimeout     = 10 * time.Minute
	shortPollingInterval = 10 * time.Second
	waitTimeout          = 20 * time.Minute
	pollingInterval      = 30 * time.Second
	opsterDeploymentName = "opensearch-operator-controller-manager"
	osMasterStsName      = "opensearch-es-master"
	osDataStsName        = "opensearch-es-data"
	osIngestStsName      = "opensearch-es-ingest"
	idSearchExactURL     = "verrazzano-system/_search?from=0&size=1"
	idSearchAllURL       = "verrazzano-system/_search?"
)

var esPods = []string{"opensearch-es-master", "opensearch-es-ingest", "opensearch-es-data"}
var esPodsUp = []string{"opensearch-es-master", "opensearch-es-ingest", "opensearch-es-data", opsterDeploymentName, "opensearch-dashboards"}

var beforeSuite = t.BeforeSuiteFunc(func() {
	start := time.Now()
	common.GatherInfo()
	backupPrerequisites()
	metrics.Emit(t.Metrics.With("deployment_elapsed_time", time.Since(start).Milliseconds()))
})

var _ = BeforeSuite(beforeSuite)

var afterSuite = t.AfterSuiteFunc(func() {
	start := time.Now()
	cleanUpVelero()
	metrics.Emit(t.Metrics.With("undeployment_elapsed_time", time.Since(start).Milliseconds()))
})

var _ = AfterSuite(afterSuite)

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

	var b bytes.Buffer
	template, _ := template.New("id-query").Parse(common.EsQueryBody)
	data := common.EsQueryObject{
		BackupIDBeforeBackup: common.BackupID,
	}
	template.Execute(&b, data)

	httpClient := pkg.EventuallyVerrazzanoRetryableHTTPClient()
	fetchURL := fmt.Sprintf("%s/%s", esURL, idSearchAllURL)
	creds := fmt.Sprintf("verrazzano:%s", vzPasswd)
	parsedJSON, err := common.HTTPHelper(httpClient, "GET", fetchURL, creds, "Basic", http.StatusOK, b.Bytes(), t.Logs)
	if err != nil {
		t.Logs.Errorf("Error while retrieving http data %v", zap.Error(err))
		return ""
	}

	backupID := fmt.Sprintf("%s", parsedJSON.Search("hits", "hits", "0", "_id").Data())
	t.Logs.Infof("Opensearch id before backup = '%v'", common.BackupID)
	t.Logs.Infof("Opensearch id fetched after restore = '%v'", backupID)
	return backupID
}

// NukeOpensearch is used to destroy the opensearch cluster including data
// This is only done after a successful backup was taken
func NukeOpensearch() error {
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		t.Logs.Errorf("Failed to get clientset with error: %v", err)
		return err
	}

	t.Logs.Infof("Scaling down Opster Operator")
	getScale, err := clientset.AppsV1().Deployments(constants.VerrazzanoLoggingNamespace).GetScale(context.TODO(), opsterDeploymentName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	newScale := *getScale
	newScale.Spec.Replicas = 0

	_, err = clientset.AppsV1().Deployments(constants.VerrazzanoLoggingNamespace).UpdateScale(context.TODO(), opsterDeploymentName, &newScale, metav1.UpdateOptions{})
	if err != nil {
		t.Logs.Infof("Error = %v", zap.Error(err))
		return err
	}

	t.Logs.Infof("Deleting opensearch master sts")
	err = clientset.AppsV1().StatefulSets(constants.VerrazzanoLoggingNamespace).Delete(context.TODO(), osMasterStsName, metav1.DeleteOptions{})
	if err != nil {
		if !k8serror.IsNotFound(err) {
			t.Logs.Errorf("Unable to delete opensearch master sts due to '%v'", zap.Error(err))
			return err
		}
	}

	t.Logs.Infof("Deleting opensearch data sts")
	err = clientset.AppsV1().StatefulSets(constants.VerrazzanoLoggingNamespace).Delete(context.TODO(), osDataStsName, metav1.DeleteOptions{})
	if err != nil {
		if !k8serror.IsNotFound(err) {
			t.Logs.Errorf("Unable to delete opensearch data sts due to '%v'", zap.Error(err))
			return err
		}
	}

	t.Logs.Infof("Deleting opensearch ingest sts")
	err = clientset.AppsV1().StatefulSets(constants.VerrazzanoLoggingNamespace).Delete(context.TODO(), osIngestStsName, metav1.DeleteOptions{})
	if err != nil {
		if !k8serror.IsNotFound(err) {
			t.Logs.Errorf("Unable to delete opensearch ingest sts due to '%v'", zap.Error(err))
			return err
		}
	}

	t.Logs.Infof("Deleting opensearch master pvc if still present")
	for i := 0; i < 3; i++ {
		err = clientset.CoreV1().PersistentVolumeClaims(constants.VerrazzanoLoggingNamespace).Delete(context.TODO(), fmt.Sprintf("data-%s-%v", osMasterStsName, i), metav1.DeleteOptions{})
		if err != nil {
			if !k8serror.IsNotFound(err) {
				t.Logs.Errorf("Unable to delete opensearch master pvc due to '%v'", zap.Error(err))
				return err
			}
		}
	}

	t.Logs.Infof("Deleting opensearch data pvc")
	for i := 0; i < 3; i++ {
		err = clientset.CoreV1().PersistentVolumeClaims(constants.VerrazzanoLoggingNamespace).Delete(context.TODO(), fmt.Sprintf("data-%s-%v", osDataStsName, i), metav1.DeleteOptions{})
		if err != nil {
			if !k8serror.IsNotFound(err) {
				t.Logs.Errorf("Unable to delete opensearch data pvc due to '%v'", zap.Error(err))
				return err
			}
		}
	}

	if err = resetClusterStatus(); err != nil {
		t.Logs.Errorf("Failed to reset OpensearchCluster CR staus: %v", zap.Error(err))
		return err
	}

	return nil
}

// resetClusterStatus set the .status.initialized to false in OpenSearchCluster CR
// So that cluster can be bootstrapped again
func resetClusterStatus() error {
	dynamicClient, err := pkg.GetDynamicClient()
	if err != nil {
		return err
	}
	gvr := schema.GroupVersionResource{
		Group:    "opensearch.opster.io",
		Version:  "v1",
		Resource: "opensearchclusters",
	}
	opensearchFetched, err := dynamicClient.Resource(gvr).Namespace(constants.VerrazzanoLoggingNamespace).Get(context.Background(), "opensearch", metav1.GetOptions{})
	if err != nil {
		return err
	}

	t.Logs.Infof("Setting cluster initialized status to false")
	opensearchFetched.Object["status"].(map[string]interface{})["initialized"] = false

	_, updateErr := dynamicClient.Resource(gvr).Namespace(constants.VerrazzanoLoggingNamespace).UpdateStatus(context.TODO(), opensearchFetched, metav1.UpdateOptions{})
	return updateErr
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

// checkPodsRunning checks whether the pods are ready in a given namespace
func checkPodsRunning(namespace string, expectedPods []string) bool {
	result, err := pkg.PodsRunning(namespace, expectedPods)
	if err != nil {
		AbortSuite(fmt.Sprintf("One or more pods are not running in the namespace: %v, error: %v", namespace, err))
	}
	return result
}

// checkPodsNotRunning checks whether the pods are not ready in a given namespace
func checkPodsNotRunning(namespace string, expectedPods []string) bool {
	result, err := pkg.PodsNotRunning(namespace, expectedPods)
	if err != nil {
		AbortSuite(fmt.Sprintf("One or more pods are running in the namespace: %v, error: %v", namespace, err))
	}
	return result
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

var _ = t.Describe("OpenSearch Backup and Restore,", Label("f:platform-verrazzano.opensearch-backup"), Serial, func() {

	t.Context("OpenSearch backup", func() {
		WhenVeleroInstalledIt("Start opensearch backup after velero backup storage location created", func() {
			Eventually(func() error {
				return CreateVeleroBackupObject()
			}, waitTimeout, pollingInterval).Should(BeNil())
		})

		WhenVeleroInstalledIt("Check backup progress after velero backup object was created", func() {
			Eventually(func() error {
				return common.TrackOperationProgress("velero", common.BackupResource, common.BackupOpensearchName, common.VeleroNameSpace, t.Logs)
			}, waitTimeout, pollingInterval).Should(BeNil())
		})

		WhenVeleroInstalledIt("Fetch logs after backup is complete", func() {
			Eventually(func() error {
				return common.DisplayHookLogs(t.Logs)
			}, waitTimeout, pollingInterval).Should(BeNil())
		})

	})

	t.Context("Disaster simulation", func() {
		WhenVeleroInstalledIt("Cleanup opensearch once backup is done", func() {
			Eventually(func() error {
				return NukeOpensearch()
			}, waitTimeout, pollingInterval).Should(BeNil())
		})

		WhenVeleroInstalledIt("Ensure the pods are not running before starting a restore", func() {
			Eventually(func() bool {
				return checkPodsNotRunning(constants.VerrazzanoLoggingNamespace, esPods)
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Check if pods are down")
		})

		WhenVeleroInstalledIt("After pods are down check if pvcs are deleted before starting a restore", func() {
			Eventually(func() error {
				return common.CheckPvcsTerminated("opster.io/opensearch-cluster=opensearch", constants.VerrazzanoLoggingNamespace, t.Logs)
			}, waitTimeout, pollingInterval).Should(BeNil(), "Check if pvcs are removed")
		})

	})

	t.Context("OpenSearch restore", func() {
		WhenVeleroInstalledIt("Start restore after velero backup is completed", func() {
			Eventually(func() error {
				return CreateVeleroRestoreObject()
			}, waitTimeout, pollingInterval).Should(BeNil())
		})
		WhenVeleroInstalledIt("Check velero restore progress", func() {
			Eventually(func() error {
				return common.TrackOperationProgress("velero", common.RestoreResource, common.RestoreOpensearchName, common.VeleroNameSpace, t.Logs)
			}, waitTimeout, pollingInterval).Should(BeNil())
		})
		WhenVeleroInstalledIt("Fetch logs after restore is complete", func() {
			Eventually(func() error {
				return common.DisplayHookLogs(t.Logs)
			}, waitTimeout, pollingInterval).Should(BeNil())
		})
	})

	t.Context("OpenSearch Data and Infra verification", func() {
		WhenVeleroInstalledIt("Wait for all pods to come up in verrazzano-logging", func() {
			Eventually(func() bool {
				return checkPodsRunning(constants.VerrazzanoLoggingNamespace, esPodsUp)
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Check if pods are up")
		})
		WhenVeleroInstalledIt("Is Restore good? Verify restore", func() {
			Eventually(func() string {
				return IsRestoreSuccessful()
			}, waitTimeout, pollingInterval).Should(Equal(common.BackupID))
		})

	})

})
