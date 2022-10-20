// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mysql

import (
	"bytes"
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	common "github.com/verrazzano/verrazzano/tests/e2e/backup/helpers"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework/metrics"
	"go.uber.org/zap"
	k8serror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"strings"
	"text/template"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
)

const (
	shortWaitTimeout       = 10 * time.Minute
	shortPollingInterval   = 10 * time.Second
	waitTimeout            = 15 * time.Minute
	pollingInterval        = 30 * time.Second
	mysqlPvcPrefix         = "datadir-mysql"
	mysqlChartName         = "mysql"
	mysqlInnoDBClusterName = "mysql"
	//vzMySQLChartPath       = "platform-operator/thirdparty/charts/mysql"
	vzMySQLChartPath = "../../../../platform-operator/thirdparty/charts/mysql"
)

var keycloakPods = []string{"keycloak", "mysql"}
var mysqlPods = []string{"mysql"}

var _ = t.BeforeSuite(func() {
	start := time.Now()
	common.GatherInfo()
	file, err := os.CreateTemp("", "mysql-values-")
	if err != nil {
		t.Logs.Fatal(err)
	}
	defer file.Close()
	common.MySQLBackupHelmFileName = file.Name()
	backupPrerequisites()
	metrics.Emit(t.Metrics.With("deployment_elapsed_time", time.Since(start).Milliseconds()))
})

var _ = t.AfterSuite(func() {
	start := time.Now()
	cleanUpVelero()
	os.Remove(common.MySQLBackupHelmFileName)
	metrics.Emit(t.Metrics.With("undeployment_elapsed_time", time.Since(start).Milliseconds()))
})

var t = framework.NewTestFramework("mysql-backup")

// func CreateInnoDBBackupObject() error  creates mysql operator backup CR starting the backup process
func CreateInnoDBBackupObject() error {
	var b bytes.Buffer
	template, _ := template.New("mysql-backup").Parse(common.InnoDBBackup)
	data := common.InnoDBBackupObject{
		InnoDBBackupName:                  common.BackupMySQLName,
		InnoDBNamespaceName:               constants.KeycloakNamespace,
		InnoDBClusterName:                 common.InnoDBClusterName,
		InnoDBBackupProfileName:           common.BackupResourceName,
		InnoDBBackupObjectStoreBucketName: common.OciBucketName,
		InnoDBBackupCredentialsName:       common.VeleroMySQLSecretName,
		InnoDBBackupStorageName:           common.BackupMySQLStorageName,
		InnoDBObjectStorageNamespaceName:  common.OciNamespaceName,
		InnoDBBackupRegion:                common.BackupRegion,
	}
	template.Execute(&b, data)
	err := common.DynamicSSA(context.TODO(), b.String(), t.Logs)
	if err != nil {
		t.Logs.Errorf("Error creating innodb backup object", zap.Error(err))
		return err
	}

	return nil
}

func BackupMySQLValues() error {
	t.Logs.Infof("Backing up mysql values to file '%s'", common.MySQLBackupHelmFileName)
	var cmd common.BashCommand
	var cmdArgs []string
	cmdArgs = append(cmdArgs, "/bin/sh", "-c", fmt.Sprintf("helm get values %s -n %s > %s", mysqlChartName, constants.KeycloakNamespace, common.MySQLBackupHelmFileName))
	cmd.CommandArgs = cmdArgs
	cmd.Timeout = 1 * time.Minute

	response := common.Runner(&cmd, t.Logs)
	if response.CommandError != nil {
		t.Logs.Error("Unable to get mysql helm values due to ", zap.Error(response.CommandError))
		return response.CommandError
	}
	return nil
}

func NukeMySQL() error {
	var cmd common.BashCommand
	var cmdArgs []string
	cmdArgs = append(cmdArgs, "helm")
	cmdArgs = append(cmdArgs, "delete")
	cmdArgs = append(cmdArgs, mysqlChartName)
	cmdArgs = append(cmdArgs, "-n")
	cmdArgs = append(cmdArgs, constants.KeycloakNamespace)

	cmd.CommandArgs = cmdArgs
	cmd.Timeout = 5 * time.Minute

	response := common.Runner(&cmd, t.Logs)
	if response.CommandError != nil {
		t.Logs.Error("Unable to cleanup mysql due to ", zap.Error(response.CommandError))
		return response.CommandError
	}

	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		t.Logs.Errorf("Failed to get clientset with error: %v", err)
		return err
	}

	t.Logs.Infof("Deleting mysql pvc(s) from namespace '%s'", constants.KeycloakNamespace)
	for i := 0; i < 3; i++ {
		err := clientset.CoreV1().PersistentVolumeClaims(constants.KeycloakNamespace).Delete(context.TODO(), fmt.Sprintf("%s-%v", mysqlPvcPrefix, i), metav1.DeleteOptions{})
		if err != nil {
			if !k8serror.IsNotFound(err) {
				t.Logs.Errorf("Unable to delete opensearch master pvc due to '%v'", zap.Error(err))
				return err
			}
		}
	}

	return nil
}

func MySQLRestore() error {
	t.Logs.Info("Start mysql restore")

	// Get the backup folder name
	backupInfo, err := common.GetMySQLBackup(constants.KeycloakNamespace, common.BackupMySQLName, t.Logs)
	if err != nil {
		t.Logs.Errorf("Unable to fetch backup '%s' due to '%v'", common.BackupMySQLName, zap.Error(err))
		return err
	}
	backupFolderName := backupInfo.Status.Output

	var cmd common.BashCommand
	var cmdArgs []string
	cmdArgs = append(cmdArgs, "helm", "install", mysqlChartName, vzMySQLChartPath)
	cmdArgs = append(cmdArgs, "--namespace", constants.KeycloakNamespace)
	cmdArgs = append(cmdArgs, "--set", "initDB.dump.name=alpha")
	cmdArgs = append(cmdArgs, "--set", fmt.Sprintf("initDB.dump.s3.prefix=%s/%s", common.BackupMySQLStorageName, backupFolderName))
	cmdArgs = append(cmdArgs, "--set", fmt.Sprintf("initDB.dump.s3.bucketName=%s", common.OciBucketName))
	cmdArgs = append(cmdArgs, "--set", fmt.Sprintf("initDB.dump.s3.credentials=%s", common.VeleroMySQLSecretName))
	cmdArgs = append(cmdArgs, "--values", common.MySQLBackupHelmFileName)

	cmd.CommandArgs = cmdArgs
	cmd.Timeout = 10 * time.Minute

	response := common.Runner(&cmd, t.Logs)
	if response.CommandError != nil {
		t.Logs.Error("Unable to restore mysql due to ", zap.Error(response.CommandError))
		return response.CommandError
	}
	return nil
}

func RebootKeyCloak() error {
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		t.Logs.Errorf("Failed to get clientset with error: %v", err)
		return err
	}

	t.Logs.Infof("Scaling down keycloak sts")
	getScale, err := clientset.AppsV1().StatefulSets(constants.KeycloakNamespace).GetScale(context.TODO(), constants.KeycloakNamespace, metav1.GetOptions{})
	if err != nil {
		return err
	}
	replicaCount := getScale.Spec.Replicas
	scaleDown := *getScale
	scaleDown.Spec.Replicas = 0

	_, err = clientset.AppsV1().StatefulSets(constants.VerrazzanoSystemNamespace).UpdateScale(context.TODO(), constants.KeycloakNamespace, &scaleDown, metav1.UpdateOptions{})
	if err != nil {
		t.Logs.Infof("Error = %v", zap.Error(err))
		return err
	}

	scaleUp := scaleDown
	scaleUp.Spec.Replicas = replicaCount

	t.Logs.Infof("Scaling up keycloak sts")
	_, err = clientset.AppsV1().StatefulSets(constants.VerrazzanoSystemNamespace).UpdateScale(context.TODO(), constants.KeycloakNamespace, &scaleUp, metav1.UpdateOptions{})
	if err != nil {
		t.Logs.Infof("Error = %v", zap.Error(err))
		return err
	}
	return nil
}

// KeycloakDeleteUsers helps in cleaning up test users at the end of the run
func KeycloakDeleteUsers() error {
	keycloakClient, err := pkg.NewKeycloakAdminRESTClient()
	if err != nil {
		t.Logs.Errorf("Unable to get keycloak client due to ", zap.Error(err))
		return err
	}

	for i := 0; i < len(common.KeyCloakUserIDList); i++ {
		t.Logs.Infof("Deleting user with username '%s'", common.KeyCloakUserIDList[i])
		_ = keycloakClient.DeleteUser(constants.VerrazzanoSystemNamespace, common.KeyCloakUserIDList[i])
	}

	return nil

}

// KeycloakCreateUsers helps in creating test users to populate data
func KeycloakCreateUsers(n int) error {

	keycloakClient, err := pkg.NewKeycloakAdminRESTClient()
	if err != nil {
		t.Logs.Errorf("Unable to get keycloak client due to ", zap.Error(err))
		return err
	}

	for i := 0; i < n; i++ {
		id := uuid.New().String()
		uniqueID := strings.Split(id, "-")[len(strings.Split(id, "-"))-1]
		userID := fmt.Sprintf("mysql-user-%s", uniqueID)
		t.Logs.Infof("Creating user with username '%s'", userID)
		firstName := fmt.Sprintf("john-%v", i+1)
		lastName := "doe"
		location, err := keycloakClient.CreateUser(constants.VerrazzanoSystemNamespace, userID, firstName, lastName, "hello@mysql!")
		if err != nil {
			t.Logs.Errorf("Unable to get create keycloak user due to ", zap.Error(err))
			return err
		}
		sqlUserID := strings.Split(location, "/")[len(strings.Split(location, "/"))-1]
		common.KeyCloakUserIDList = append(common.KeyCloakUserIDList, sqlUserID)
	}

	return nil

}

// KeycloakVerifyUsers helps in verifying if the user exists
func KeycloakVerifyUsers() bool {
	keycloakClient, err := pkg.NewKeycloakAdminRESTClient()
	if err != nil {
		t.Logs.Errorf("Unable to get keycloak client due to ", zap.Error(err))
		return false
	}

	for i := 0; i < len(common.KeyCloakUserIDList); i++ {
		t.Logs.Infof("Verifying user with username '%s' exists after mysql restore", common.KeyCloakUserIDList[i])
		ok, err := keycloakClient.VerifyUserExists(constants.VerrazzanoSystemNamespace, common.KeyCloakUserIDList[i])
		if err != nil {
			t.Logs.Errorf("Unable to verify keycloak user due to ", zap.Error(err))
			return false
		}
		if !ok {
			t.Logs.Errorf("User '%s' does not exist or could not be verified.", common.KeyCloakUserIDList[i])
			return false
		}
		t.Logs.Infof("User '%s' found after mysql restore!", common.KeyCloakUserIDList[i])
	}
	return true
}

// 'It' Wrapper to only run spec if the Velero is supported on the current Verrazzano version
func WhenMySQLOpInstalledIt(description string, f func()) {
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
	if !pkg.IsMySQLOperatorEnabled(kubeconfigPath) {
		supported = false
	}
	if supported {
		t.It(description, f)
	} else {
		t.Logs.Infof("Skipping check '%v', the MySQL operator not enabled or minimum version detection failed", description)
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

func backupPrerequisites() {
	t.Logs.Info("Setup backup pre-requisites")
	t.Logs.Info("Create backup secret for innodb backup objects")

	Eventually(func() error {
		return BackupMySQLValues()
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

	Eventually(func() error {
		return common.CreateMySQLCredentialsSecretFromFile(constants.KeycloakNamespace, common.VeleroMySQLSecretName, t.Logs)
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

	t.Logs.Info("Create a sample keycloak user")
	Eventually(func() error {
		return KeycloakCreateUsers(common.KeycloakUserCount)
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

}

func cleanUpVelero() {
	t.Logs.Info("Cleanup backup and restore objects")

	t.Logs.Info("Cleanup backup object")
	Eventually(func() error {
		return common.CrdPruner("mysql.oracle.com", "v2", "mysqlbackups", common.BackupMySQLName, constants.KeycloakNamespace, t.Logs)
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

	t.Logs.Info("Cleanup mysql backup secrets")
	Eventually(func() error {
		return common.DeleteSecret(constants.KeycloakNamespace, common.VeleroMySQLSecretName, t.Logs)
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

	t.Logs.Info("Delete keycloak user")
	Eventually(func() error {
		return KeycloakDeleteUsers()
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())
}

var _ = t.Describe("MySQL Backup and Restore,", Label("f:platform-verrazzano.mysql-backup"), Serial, func() {

	t.Context("MySQL backup operator", func() {
		WhenMySQLOpInstalledIt("MySQL backup triggered", func() {
			Eventually(func() error {
				return CreateInnoDBBackupObject()
			}, waitTimeout, pollingInterval).Should(BeNil())
		})

		WhenMySQLOpInstalledIt("Check backup progress after mysql backup object was created", func() {
			Eventually(func() error {
				return common.TrackOperationProgress("mysql", common.BackupResource, common.BackupMySQLName, constants.KeycloakNamespace, t.Logs)
			}, waitTimeout, pollingInterval).Should(BeNil())
		})
	})

	t.Context("Disaster simulation", func() {
		WhenMySQLOpInstalledIt("Delete users created as part of pre-suite", func() {
			Eventually(func() error {
				return KeycloakDeleteUsers()
			}, waitTimeout, pollingInterval).Should(BeNil())
		})

		WhenMySQLOpInstalledIt("Delete innodb cluster", func() {
			Eventually(func() error {
				return NukeMySQL()
			}, waitTimeout, pollingInterval).Should(BeNil())
		})

		WhenMySQLOpInstalledIt("Ensure the pods are not running before starting a restore", func() {
			Eventually(func() bool {
				return checkPodsNotRunning(constants.KeycloakNamespace, mysqlPods)
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Check if pods are down")
		})
	})

	t.Context("MySQL restore", func() {
		WhenMySQLOpInstalledIt(fmt.Sprintf("Start restore of mysql from backup '%s'", common.BackupMySQLName), func() {
			Eventually(func() error {
				return MySQLRestore()
			}, waitTimeout, pollingInterval).Should(BeNil())
		})
		WhenMySQLOpInstalledIt("Check MySQL restore progress", func() {
			Eventually(func() error {
				return common.TrackOperationProgress("mysql", common.RestoreResource, mysqlInnoDBClusterName, constants.KeycloakNamespace, t.Logs)
			}, waitTimeout, pollingInterval).Should(BeNil())
		})

	})

	t.Context("MySQL Data and Infra verification", func() {
		WhenMySQLOpInstalledIt("After restore is complete reboot keycloak", func() {
			Eventually(func() error {
				return RebootKeyCloak()
			}, waitTimeout, pollingInterval).Should(BeNil())
		})

		WhenMySQLOpInstalledIt("After restore is complete wait for keycloak and mysql pods to come up", func() {
			Eventually(func() bool {
				return checkPodsRunning(constants.KeycloakNamespace, keycloakPods)
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Check if keycloak and mysql infra is up")
		})

		WhenMySQLOpInstalledIt("Is Restore good? Verify restore", func() {
			Eventually(func() bool {
				return KeycloakVerifyUsers()
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})

	})
})
