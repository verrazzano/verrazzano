// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package backup

import (
	"bytes"
	"context"
	"fmt"
	"github.com/Jeffail/gabs/v2"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/verrazzano/verrazzano/pkg/httputil"
	"github.com/verrazzano/verrazzano/pkg/test/framework/metrics"
	"github.com/verrazzano/verrazzano/tests/e2e/backup"
	"go.uber.org/zap"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"os"
	"strconv"
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
	rancherUserName      = "cowboy"
	rancherFullName      = "Lone Ranger"
	rancherPassword      = "rancher@newstack"
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

var t = framework.NewTestFramework("rancher-backup")

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
	template.Execute(&b, data)
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

	t.Logs.Infof("Rancher backup filename = %s", backup.RancherBackupFileName)

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
	template.Execute(&b, data)
	err = backup.DynamicSSA(context.TODO(), b.String(), t.Logs)
	if err != nil {
		t.Logs.Errorf("Error creating rancher backup object", zap.Error(err))
		return err
	}
	t.Logs.Infof("Rancher backup filename = %s", backup.RancherBackupFileName)
	return nil
}

func GetRancherLoginToken() string {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		t.Logs.Errorf("Failed to get kubeconfigPath with error: %v", err)
		return ""
	}
	api := pkg.EventuallyGetAPIEndpoint(kubeconfigPath)
	rancherURL := pkg.EventuallyGetRancherURL(t.Logs, api)
	httpClient := pkg.EventuallyVerrazzanoRetryableHTTPClient()
	token := pkg.GetRancherAdminToken(t.Logs, httpClient, rancherURL)
	return token
}

func CreateRancherUserFromShell() error {
	var b bytes.Buffer
	template, _ := template.New("rancher-user").Parse(backup.RancherUserTemplate)
	data := backup.RancherUser{
		FullName: strconv.Quote(rancherFullName),
		Username: strconv.Quote(rancherUserName),
		Password: strconv.Quote(rancherPassword),
	}
	template.Execute(&b, data)

	url, err := backup.GetRancherURL(t.Logs)
	if err != nil {
		return err
	}

	os.WriteFile("test.json", b.Bytes(), 0644)
	defer os.Remove("test.json")

	var cmdArgs []string
	apiPath := "v3/users"
	curlCmd := fmt.Sprintf("curl -ks %s -u %s -X POST -H 'Accept: application/json' -H 'Content-Type: application/json' -d @test.json", strconv.Quote(fmt.Sprintf("%s/%s", url, apiPath)), strconv.Quote(backup.RancherToken))
	cmdArgs = append(cmdArgs, "/bin/sh")
	cmdArgs = append(cmdArgs, "-c")
	cmdArgs = append(cmdArgs, curlCmd)

	var kcmd backup.BashCommand
	kcmd.Timeout = 2 * time.Minute
	kcmd.CommandArgs = cmdArgs

	curlResponse := backup.Runner(&kcmd, t.Logs)
	if curlResponse.CommandError != nil {
		return err
	}

	return nil

}

func DeleteRancherUserFromShell() error {
	url, err := backup.GetRancherURL(t.Logs)
	if err != nil {
		return err
	}

	var cmdArgs []string
	apiPath := "v3/users"
	curlCmd := fmt.Sprintf("curl -ks %s -u %s -X DELETE", strconv.Quote(fmt.Sprintf("%s/%s", url, apiPath)), strconv.Quote(backup.RancherToken))
	cmdArgs = append(cmdArgs, "/bin/sh")
	cmdArgs = append(cmdArgs, "-c")
	cmdArgs = append(cmdArgs, curlCmd)

	var kcmd backup.BashCommand
	kcmd.Timeout = 2 * time.Minute
	kcmd.CommandArgs = cmdArgs

	curlResponse := backup.Runner(&kcmd, t.Logs)
	if curlResponse.CommandError != nil {
		return err
	}

	return nil

}

func GetRancherUser() string {
	url, err := backup.GetRancherURL(t.Logs)
	if err != nil {
		return ""
	}
	httpClient := pkg.EventuallyVerrazzanoRetryableHTTPClient()

	apiPath := fmt.Sprintf("v3/users?username=%s", rancherUserName)
	req, err := retryablehttp.NewRequest("GET", fmt.Sprintf("%s/%s", url, apiPath), nil)
	if err != nil {
		t.Logs.Error(fmt.Sprintf("error creating rancher api request for %s: %v", apiPath, err))
		return ""
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", backup.RancherToken))
	req.Header.Set("Accept", "application/json")
	response, err := httpClient.Do(req)
	if err != nil {
		t.Logs.Error(fmt.Sprintf("error invoking rancher api request %s: %v", apiPath, err))
		return ""
	}
	defer response.Body.Close()

	err = httputil.ValidateResponseCode(response, http.StatusOK)
	if err != nil {
		return ""
	}
	// extract the response body
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		t.Logs.Errorf("Failed to read Rancher token response: %v", err)
		return ""
	}

	jsonParsed, err := gabs.ParseJSON(body)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%s", jsonParsed.Path("data.0.username").Data())
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

	t.Logs.Info("Fetching rancher login Token")
	Eventually(func() string {
		backup.RancherToken = GetRancherLoginToken()
		return backup.RancherToken
	}, waitTimeout, pollingInterval).ShouldNot(BeEmpty())

	t.Logs.Info("Creating a user with the retrieved login token")
	Eventually(func() error {
		return CreateRancherUserFromShell()
	}, waitTimeout, pollingInterval).Should(BeNil())
}

func cleanUpRancher() {
	t.Logs.Info("Cleanup backup and restore objects")

	t.Logs.Infof("Cleanup user '%s' as part of cleanup", rancherUserName)
	Eventually(func() error {
		return DeleteRancherUserFromShell()
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

	t.Logs.Info("Cleanup restore object")
	Eventually(func() error {
		return backup.RancherObjectDelete("restore", backup.RestoreRancherName, t.Logs)
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

	t.Logs.Info("Cleanup backup object")
	Eventually(func() error {
		return backup.RancherObjectDelete("backup", backup.BackupRancherName, t.Logs)
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

	t.Logs.Info("Cleanup rancher secrets")
	Eventually(func() error {
		return DeleteSecret(backup.VeleroNameSpace, backup.RancherSecretName)
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
				return backup.CheckOperatorOperationProgress("rancher", "backup", backup.VeleroNameSpace, backup.BackupRancherName, t.Logs)
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
				return backup.CheckOperatorOperationProgress("rancher", "restore", backup.VeleroNameSpace, backup.RestoreRancherName, t.Logs)
			}, waitTimeout, pollingInterval).Should(BeNil(), "Check if rancher restore operation completed successfully")
		})
	})

	t.Context("After restore is complete wait for rancher pods to come up", func() {
		WhenRancherBackupInstalledIt("After restore is complete wait for rancher pods to come up", func() {
			Eventually(func() error {
				return backup.WaitForPodsShell("cattle-system", t.Logs)
			}, waitTimeout, pollingInterval).Should(BeNil(), "Check if rancher infra is up")
		})
	})

	t.Context("Get user after rancher restore is complete", func() {
		WhenRancherBackupInstalledIt("Get user after rancher restore is complete", func() {
			Eventually(func() string {
				return GetRancherUser()
			}, waitTimeout, pollingInterval).Should(Equal(rancherUserName), "Check if rancher user has been restored successfully")
		})
	})

})
