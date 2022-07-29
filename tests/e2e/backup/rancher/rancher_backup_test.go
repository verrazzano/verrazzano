// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package backup

import (
	"bytes"
	"context"
	"fmt"
	"github.com/Jeffail/gabs/v2"
	"github.com/google/uuid"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/verrazzano/verrazzano/pkg/httputil"
	"github.com/verrazzano/verrazzano/pkg/test/framework/metrics"
	"github.com/verrazzano/verrazzano/tests/e2e/backup/common"
	"go.uber.org/zap"
	"io/ioutil"
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
)

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

	t.Logs.Infof("Rancher backup filename = %s", common.RancherBackupFileName)

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

func PopulateRancherUsers(token, rancherURL string, n int) error {
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

	for i := 0; i < n; i++ {
		id := uuid.New().String()
		uniqueID := strings.Split(id, "-")[len(strings.Split(id, "-"))-1]
		fullName := fmt.Sprintf("john-smith-%v", i+1)
		userName := fmt.Sprintf("cowboy-%v", uniqueID)

		var b bytes.Buffer
		template, err := template.New("rancher-user").Parse(common.RancherUserTemplate)
		if err != nil {
			t.Logs.Errorf("Unable to convert template '%v'", err)
			return err
		}
		data := common.RancherUser{
			FullName: strconv.Quote(fullName),
			Username: strconv.Quote(userName),
			Password: strconv.Quote(rancherPassword),
		}
		template.Execute(&b, data)

		request, err := retryablehttp.NewRequest("POST", rancherUserCreateURL, b.Bytes())
		if err != nil {
			t.Logs.Errorf("Unable to create a retryable http client= %v", zap.Error(err))
			return err

		}
		request.Header.Add("Authorization", fmt.Sprintf("Bearer %v", token))
		request.Header.Add("Accept", "application/json")
		response, err := httpClient.Do(request)
		if err != nil {
			t.Logs.Errorf("Unable to create retryable http client due to '%v'", zap.Error(err))
			return err
		}

		if response == nil {
			return fmt.Errorf("invalid response")
		}
		defer response.Body.Close()

		if response.StatusCode == 201 {
			t.Logs.Infof("Sucessfully created rancher user %v", userName)
			common.RancherUserNameList = append(common.RancherUserNameList, userName)
		} else {
			t.Logs.Infof("invalid response status: %d", response.StatusCode)
			return fmt.Errorf("invalid response status: %d", response.StatusCode)
		}
	}

	return nil
}

func HTTPHelper(httpClient *retryablehttp.Client, method, httpURL, token, userName string, responseCode int, rawbody interface{}) (*gabs.Container, error) {
	req, err := retryablehttp.NewRequest(method, httpURL, rawbody)
	if err != nil {
		t.Logs.Error(fmt.Sprintf("error creating rancher api request for %s: %v", httpURL, err))
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
	req.Header.Set("Accept", "application/json")
	response, err := httpClient.Do(req)
	if err != nil {
		t.Logs.Error(fmt.Sprintf("error invoking rancher api request %s: %v", httpURL, err))
		return nil, err
	}
	defer response.Body.Close()

	err = httputil.ValidateResponseCode(response, responseCode)
	if err != nil {
		t.Logs.Errorf("did not get expected response code = %v, Error = %v", responseCode, zap.Error(err))
		return nil, err
	}

	// extract the response body
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		t.Logs.Errorf("Failed to read Rancher token response: %v", zap.Error(err))
		return nil, err
	}

	jsonParsed, err := gabs.ParseJSON(body)
	if err != nil {
		t.Logs.Errorf("Failed to parse json: %v", zap.Error(err))
		return nil, err
	}

	if userName == fmt.Sprintf("%s", jsonParsed.Path("data.0.username").Data()) {
		//t.Logs.Infof("'%s' found in rancher after restore", common.RancherUserNameList[i])
		return jsonParsed, nil
	}
	return nil, fmt.Errorf("User '%s' not found", userName)

}

func VerifyRancherUser(method, httpURL, token, userName string, responseCode int, rawbody interface{}) (*gabs.Container, bool) {
	httpClient := pkg.EventuallyVerrazzanoRetryableHTTPClient()
	jsonParsed, err := HTTPHelper(httpClient, method, httpURL, token, userName, responseCode, rawbody)
	if err != nil {
		return nil, false
	}
	return jsonParsed, true
}

// VerifyRancherUsers gets an existing rancher user
func VerifyRancherUsers(token, rancherURL string) bool {
	for i := 0; i < len(common.RancherUserNameList); i++ {
		rancherGetURL := fmt.Sprintf("%s/v3/users?username=%s", rancherURL, common.RancherUserNameList[i])
		_, ok := VerifyRancherUser("GET", rancherGetURL, token, common.RancherUserNameList[i], http.StatusOK, nil)
		if !ok {
			return false
		}
		t.Logs.Infof("'%s' found in rancher after restore", common.RancherUserNameList[i])
	}
	return true
}

// BuildRancherUserIDList gets an existing rancher user
func BuildRancherUserIDList(token, rancherURL string) bool {
	for i := 0; i < len(common.RancherUserNameList); i++ {
		rancherGetURL := fmt.Sprintf("%s/v3/users?username=%s", rancherURL, common.RancherUserNameList[i])
		jsonParsed, ok := VerifyRancherUser("GET", rancherGetURL, token, common.RancherUserNameList[i], http.StatusOK, nil)
		if !ok {
			return false
		}
		common.RancherUserIDList = append(common.RancherUserIDList, fmt.Sprintf("%s", jsonParsed.Path("data.0.id").Data()))
		t.Logs.Infof("'%s' found in rancher after restore", common.RancherUserNameList[i])
	}
	return true
}

func DeleteRancherUsers(token, rancherURL string) error {
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

	for i := 0; i < len(common.RancherUserNameList); i++ {
		rancherUserDeleteURL := fmt.Sprintf("%s/v3/users/%s", rancherURL, common.RancherUserIDList[i])
		request, err := retryablehttp.NewRequest("DELETE", rancherUserDeleteURL, nil)
		if err != nil {
			t.Logs.Errorf("Unable to create a retryable http client= %v", zap.Error(err))
			return err

		}
		request.Header.Add("Authorization", fmt.Sprintf("Bearer %v", token))
		request.Header.Add("Accept", "application/json")
		response, err := httpClient.Do(request)
		if err != nil {
			t.Logs.Errorf("Unable to create retryable http client due to '%v'", zap.Error(err))
			return err
		}

		if response == nil {
			return fmt.Errorf("invalid response")
		}
		defer response.Body.Close()

		t.Logs.Infof("Status code = %v, Status Response = %v", response.StatusCode, response.Status)
		err = httputil.ValidateResponseCode(response, http.StatusOK)
		if err != nil {
			t.Logs.Errorf("did not get expected response code , Error = %v", zap.Error(err))
			return err
		}
		t.Logs.Infof("Sucessfully deleted rancher user '%v' with id '%v' ", common.RancherUserNameList[i], common.RancherUserIDList[i])
	}

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

	t.Logs.Info("Get rancher admin token")
	Eventually(func() string {
		common.RancherToken = common.GetRancherLoginToken(t.Logs)
		return common.RancherToken
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

	t.Logs.Info("Creating multiple user with the retrieved login token")
	Eventually(func() error {
		return PopulateRancherUsers(common.RancherToken, common.RancherURL, 10)
	}, waitTimeout, pollingInterval).Should(BeNil())

	t.Logs.Info("Build user id list for rancher users")
	Eventually(func() bool {
		return BuildRancherUserIDList(common.RancherToken, common.RancherURL)
	}, waitTimeout, pollingInterval).Should(BeTrue())

}

// Run as part of AfterSuite
func cleanUpRancher() {
	t.Logs.Info("Cleanup backup and restore objects")

	t.Logs.Info("Creating multiple user with the retrieved login token")
	Eventually(func() error {
		return DeleteRancherUsers(common.RancherToken, common.RancherURL)
	}, waitTimeout, pollingInterval).Should(BeNil())

	t.Logs.Info("Cleanup restore object")
	Eventually(func() error {
		return common.RancherObjectDelete("restore", common.RestoreRancherName, t.Logs)
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

	t.Logs.Info("Cleanup backup object")
	Eventually(func() error {
		return common.RancherObjectDelete("backup", common.BackupRancherName, t.Logs)
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

	t.Logs.Info("Cleanup rancher secrets")
	Eventually(func() error {
		return common.DeleteSecret(common.VeleroNameSpace, common.RancherSecretName, t.Logs)
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

}

var _ = t.Describe("Rancher Backup and Restore Flow,", Label("f:platform-verrazzano.rancher-backup"), Serial, func() {

	t.Context("Start rancher backup", func() {
		WhenRancherBackupInstalledIt("Start rancher backup", func() {
			Eventually(func() error {
				return CreateRancherBackupObject()
			}, waitTimeout, pollingInterval).Should(BeNil(), "Create rancher backup CRD")
		})
	})

	t.Context("Check backup progress after rancher backup object was created", func() {
		WhenRancherBackupInstalledIt("Check backup progress after rancher backup object was created", func() {
			Eventually(func() error {
				return common.CheckOperatorOperationProgress("rancher", "backup", common.VeleroNameSpace, common.BackupRancherName, t.Logs)
			}, waitTimeout, pollingInterval).Should(BeNil(), "Check if rancher backup operation completed successfully")
		})
	})

	t.Context("Start restore after rancher backup is completed", func() {
		WhenRancherBackupInstalledIt("Start restore after rancher backup is completed", func() {
			Eventually(func() error {
				return CreateRancherRestoreObject()
			}, waitTimeout, pollingInterval).Should(BeNil(), "Create rancher restore CRD")
		})
	})

	t.Context("Check rancher restore progress", func() {
		WhenRancherBackupInstalledIt("Check rancher restore progress", func() {
			Eventually(func() error {
				return common.CheckOperatorOperationProgress("rancher", "restore", common.VeleroNameSpace, common.RestoreRancherName, t.Logs)
			}, waitTimeout, pollingInterval).Should(BeNil(), "Check if rancher restore operation completed successfully")
		})
	})

	t.Context("After restore is complete wait for rancher pods to come up", func() {
		WhenRancherBackupInstalledIt("After restore is complete wait for rancher pods to come up", func() {
			Eventually(func() error {
				return common.WaitForPodsShell("cattle-system", t.Logs)
			}, waitTimeout, pollingInterval).Should(BeNil(), "Check if rancher infra is up")
		})
	})

	t.Context("Get user after rancher restore is complete", func() {
		WhenRancherBackupInstalledIt("Get user after rancher restore is complete", func() {
			Eventually(func() bool {
				return VerifyRancherUsers(common.RancherToken, common.RancherURL)
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Check if rancher user has been restored successfully")
		})
	})

})
