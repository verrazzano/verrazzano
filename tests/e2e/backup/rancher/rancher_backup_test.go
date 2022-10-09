// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"bytes"
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/test/framework/metrics"
	common "github.com/verrazzano/verrazzano/tests/e2e/backup/helpers"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
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
	shortWaitTimeout     = 10 * time.Minute
	shortPollingInterval = 10 * time.Second
	waitTimeout          = 15 * time.Minute
	pollingInterval      = 30 * time.Second
	rancherPassword      = "rancher@newstack"
	rancherUserPrefix    = "thor"
)

var rancherPods = []string{"rancher"}

var _ = t.BeforeSuite(func() {
	start := time.Now()
	common.GatherInfo()
	backupPrerequisites()
	metrics.Emit(t.Metrics.With("deployment_elapsed_time", time.Since(start).Milliseconds()))
})

var _ = t.AfterSuite(func() {
	start := time.Now()
	cleanUpRancher()
	metrics.Emit(t.Metrics.With("undeployment_elapsed_time", time.Since(start).Milliseconds()))
})

var t = framework.NewTestFramework("rancher-backup-operator")

// CreateSecretFromMap creates opaque rancher secret required for backup/restore
func CreateSecretFromMap(namespace string, name string) error {
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		t.Logs.Errorf("Failed to get clientset with error: %v", err)
		return err
	}

	secretData := make(map[string]string)
	secretData["accessKey"] = common.OciOsAccessKey
	secretData["secretKey"] = common.OciOsAccessSecretKey

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

// CreateRancherBackupObject creates rancher backup object to start the backup process
func CreateRancherBackupObject() error {
	var b bytes.Buffer
	template, _ := template.New("rancher-backup").Parse(common.RancherBackup)
	data := common.RancherBackupData{
		RancherBackupName: common.BackupRancherName,
		RancherSecretData: common.RancherObjectStoreData{
			RancherSecretName:                 common.RancherSecretName,
			RancherSecretNamespaceName:        common.VeleroNameSpace,
			RancherObjectStoreBucketName:      common.OciBucketName,
			RancherBackupRegion:               common.BackupRegion,
			RancherObjectStorageNamespaceName: common.OciNamespaceName,
		},
	}
	template.Execute(&b, data)
	err := common.DynamicSSA(context.TODO(), b.String(), t.Logs)
	if err != nil {
		t.Logs.Errorf("Error creating rancher backup object", zap.Error(err))
		return err
	}
	return nil
}

// CreateRancherRestoreObject creates rancher restore object to start the restore process
func CreateRancherRestoreObject() error {

	rancherFileName, err := common.GetRancherBackupFileName(common.BackupRancherName, t.Logs)
	if err != nil {
		return err
	}

	common.RancherBackupFileName = rancherFileName

	var b bytes.Buffer
	template, _ := template.New("rancher-backup").Parse(common.RancherRestore)
	data := common.RancherRestoreData{
		RancherRestoreName: common.RestoreRancherName,
		BackupFileName:     common.RancherBackupFileName,
		RancherSecretData: common.RancherObjectStoreData{
			RancherSecretName:                 common.RancherSecretName,
			RancherSecretNamespaceName:        common.VeleroNameSpace,
			RancherObjectStoreBucketName:      common.OciBucketName,
			RancherBackupRegion:               common.BackupRegion,
			RancherObjectStorageNamespaceName: common.OciNamespaceName,
		},
	}
	template.Execute(&b, data)
	err = common.DynamicSSA(context.TODO(), b.String(), t.Logs)
	if err != nil {
		t.Logs.Errorf("Error creating rancher backup object", zap.Error(err))
		return err
	}
	t.Logs.Infof("Rancher backup filename = %s", common.RancherBackupFileName)
	return nil
}

// PopulateRancherUsers is used to populate test users on Rancher
func PopulateRancherUsers(rancherURL string, n int) error {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		t.Logs.Errorf("Unable to fetch kubeconfig url due to %v", zap.Error(err))
		return err
	}

	httpClient, err := pkg.GetVerrazzanoHTTPClient(kubeconfigPath)
	if err != nil {
		t.Logs.Errorf("Unable to fetch httpClient due to %v", zap.Error(err))
		return err
	}

	apiPath := "v3/users"
	rancherUserCreateURL := fmt.Sprintf("%s/%s", rancherURL, apiPath)
	token := common.GetRancherLoginToken(t.Logs)
	if token == "" {
		t.Logs.Errorf("rancher login token is empty")
		return fmt.Errorf("rancher login token is empty")
	}

	for i := 0; i < n; i++ {
		id := uuid.New().String()
		uniqueID := strings.Split(id, "-")[len(strings.Split(id, "-"))-1]
		fullName := fmt.Sprintf("john-smith-%v", i+1)
		userName := fmt.Sprintf("%s-%v", rancherUserPrefix, uniqueID)

		var b bytes.Buffer
		template, templateErr := template.New("rancher-user").Parse(common.RancherUserTemplate)
		if templateErr != nil {
			t.Logs.Errorf("Unable to convert template '%v'", templateErr)
			return templateErr
		}
		data := common.RancherUser{
			FullName: strconv.Quote(fullName),
			Username: strconv.Quote(userName),
			Password: strconv.Quote(rancherPassword),
		}
		template.Execute(&b, data)

		_, err = common.HTTPHelper(httpClient, "POST", rancherUserCreateURL, token, "Bearer", http.StatusCreated, b.Bytes(), t.Logs)
		if err != nil {
			t.Logs.Errorf("Error while retrieving http data %v", zap.Error(err))
			return err
		}
		common.RancherUserNameList = append(common.RancherUserNameList, userName)
		t.Logs.Infof("Successfully created rancher user %v", userName)
	}

	return nil
}

// DeleteRancherUsers deletes rancher users
func DeleteRancherUsers(rancherURL string) bool {
	token := common.GetRancherLoginToken(t.Logs)
	if token == "" {
		t.Logs.Errorf("rancher login token is empty")
		return false
	}
	httpClient := pkg.EventuallyVerrazzanoRetryableHTTPClient()
	for i := 0; i < len(common.RancherUserNameList); i++ {
		rancherUserDeleteURL := fmt.Sprintf("%s/v3/users/%s", rancherURL, common.RancherUserIDList[i])
		_, err := common.HTTPHelper(httpClient, "DELETE", rancherUserDeleteURL, token, "Bearer", http.StatusOK, nil, t.Logs)
		if err != nil {
			t.Logs.Errorf("Error while retrieving http data %v", zap.Error(err))
			return false
		}
		t.Logs.Infof("Successfully deleted rancher user '%v' with id '%v' ", common.RancherUserNameList[i], common.RancherUserIDList[i])
	}
	return true
}

// VerifyRancherUsers gets an existing rancher user
func VerifyRancherUsers(rancherURL string) bool {
	token := common.GetRancherLoginToken(t.Logs)
	if token == "" {
		t.Logs.Errorf("rancher login token is empty")
		return false
	}
	httpClient := pkg.EventuallyVerrazzanoRetryableHTTPClient()
	for i := 0; i < len(common.RancherUserNameList); i++ {
		rancherGetURL := fmt.Sprintf("%s/v3/users?username=%s", rancherURL, common.RancherUserNameList[i])
		parsedJSON, err := common.HTTPHelper(httpClient, "GET", rancherGetURL, token, "Bearer", http.StatusOK, nil, t.Logs)
		if err != nil {
			t.Logs.Errorf("Error while retrieving http data %v", zap.Error(err))
			return false
		}
		if common.RancherUserNameList[i] != fmt.Sprintf("%s", parsedJSON.Path("data.0.username").Data()) {
			t.Logs.Errorf("Fetched Name = '%s', Expected Name = '%s'", common.RancherUserNameList[i], fmt.Sprintf("%s", parsedJSON.Path("data.0.username").Data()))
			return false
		}
		t.Logs.Infof("'%s' found in rancher after restore", common.RancherUserNameList[i])
	}
	return true
}

// BuildRancherUserIDList gets an existing rancher user
func BuildRancherUserIDList(rancherURL string) bool {
	token := common.GetRancherLoginToken(t.Logs)
	if token == "" {
		t.Logs.Errorf("rancher login token is empty")
		return false
	}
	httpClient := pkg.EventuallyVerrazzanoRetryableHTTPClient()
	for i := 0; i < len(common.RancherUserNameList); i++ {
		rancherGetURL := fmt.Sprintf("%s/v3/users?username=%s", rancherURL, common.RancherUserNameList[i])
		parsedJSON, err := common.HTTPHelper(httpClient, "GET", rancherGetURL, token, "Bearer", http.StatusOK, nil, t.Logs)
		if err != nil {
			t.Logs.Errorf("Error while retrieving http data %v", zap.Error(err))
			return false
		}
		common.RancherUserIDList = append(common.RancherUserIDList, fmt.Sprintf("%s", parsedJSON.Path("data.0.id").Data()))
		t.Logs.Infof("'%s' found in rancher", common.RancherUserNameList[i])
	}
	return true
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

// checkPodsRunning checks whether the pods are ready in a given namespace
func checkPodsRunning(namespace string, expectedPods []string) bool {
	result, err := pkg.SpecificPodsRunning(namespace, "app=rancher")
	if err != nil {
		AbortSuite(fmt.Sprintf("One or more pods are not running in the namespace: %v, error: %v", namespace, err))
	}
	return result
}

// Run as part of BeforeSuite
func backupPrerequisites() {
	t.Logs.Info("Setup backup pre-requisites")

	var err error

	t.Logs.Info("Create backup secret for rancher backup objects")
	Eventually(func() error {
		return CreateSecretFromMap(common.VeleroNameSpace, common.RancherSecretName)
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

	t.Logs.Info("Get rancher URL")
	Eventually(func() (string, error) {
		common.RancherURL, err = common.GetRancherURL(t.Logs)
		return common.RancherURL, err
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

	t.Logs.Info("Creating multiple Rancher users")
	Eventually(func() error {
		return PopulateRancherUsers(common.RancherURL, common.RancherUserCount)
	}, waitTimeout, pollingInterval).Should(BeNil())

	t.Logs.Info("Build user id list for rancher users")
	Eventually(func() bool {
		return BuildRancherUserIDList(common.RancherURL)
	}, waitTimeout, pollingInterval).Should(BeTrue())

	time.Sleep(60 * time.Second)
}

// Run as part of AfterSuite
func cleanUpRancher() {
	t.Logs.Info("Cleanup backup and restore objects")

	t.Logs.Info("Cleanup restore object")
	Eventually(func() error {
		return common.CrdPruner("resources.cattle.io", "v1", common.RestoreResource, common.RestoreRancherName, "", t.Logs)
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

	t.Logs.Info("Cleanup backup object")
	Eventually(func() error {
		return common.CrdPruner("resources.cattle.io", "v1", common.BackupResource, common.BackupRancherName, "", t.Logs)
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

	t.Logs.Info("Cleanup rancher secrets")
	Eventually(func() error {
		return common.DeleteSecret(common.VeleroNameSpace, common.RancherSecretName, t.Logs)
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

	t.Logs.Info("Cleanup rancher users")
	Eventually(func() bool {
		return DeleteRancherUsers(common.RancherURL)
	}, waitTimeout, pollingInterval).Should(BeTrue())

}

var _ = t.Describe("Rancher Backup and Restore,", Label("f:platform-verrazzano.rancher-backup"), Serial, func() {

	t.Context("Rancher backup", func() {
		WhenRancherBackupInstalledIt("Start rancher backup", func() {
			Eventually(func() error {
				return CreateRancherBackupObject()
			}, waitTimeout, pollingInterval).Should(BeNil(), "Create rancher backup CRD")
		})

		WhenRancherBackupInstalledIt("Check backup progress after rancher backup object was created", func() {
			Eventually(func() error {
				return common.TrackOperationProgress("rancher", common.BackupResource, common.BackupRancherName, common.VeleroNameSpace, t.Logs)
			}, waitTimeout, pollingInterval).Should(BeNil(), "Check if rancher backup operation completed successfully")
		})

	})

	t.Context("Disaster simulation", func() {
		WhenRancherBackupInstalledIt("Delete all users that were created as part of pre-suite", func() {
			Eventually(func() bool {
				return DeleteRancherUsers(common.RancherURL)
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Delete rancher user")
		})
	})

	t.Context("Rancher restore", func() {
		WhenRancherBackupInstalledIt("Start restore after rancher backup is completed", func() {
			Eventually(func() error {
				return CreateRancherRestoreObject()
			}, waitTimeout, pollingInterval).Should(BeNil(), "Create rancher restore CRD")
		})
		WhenRancherBackupInstalledIt("Check rancher restore progress", func() {
			Eventually(func() error {
				return common.TrackOperationProgress("rancher", common.RestoreResource, common.RestoreRancherName, common.VeleroNameSpace, t.Logs)
			}, waitTimeout, pollingInterval).Should(BeNil(), "Check if rancher restore operation completed successfully")
		})
	})

	t.Context("Rancher Data and Infra verification", func() {
		WhenRancherBackupInstalledIt("After restore is complete wait for rancher pods to come up", func() {
			Eventually(func() bool {
				return checkPodsRunning(constants.RancherSystemNamespace, rancherPods)
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Check if rancher infra is up")
		})
		WhenRancherBackupInstalledIt("Verify users are present rancher restore is complete", func() {
			Eventually(func() bool {
				return VerifyRancherUsers(common.RancherURL)
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Check if rancher user has been restored successfully")
		})
	})

})
