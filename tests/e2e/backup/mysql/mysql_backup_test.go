// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package backup

import (
	"bytes"
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/verrazzano/verrazzano/pkg/test/framework/metrics"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/tests/e2e/backup/common"
	"go.uber.org/zap"
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

var t = framework.NewTestFramework("mysql-backup")

// CreateMysqlVeleroBackupObject creates opaque secret from the given map of values
func CreateMysqlVeleroBackupObject() error {
	var b bytes.Buffer
	template, _ := template.New("mysql-backup").Parse(common.MySQLBackup)
	data := common.VeleroMysqlBackupObject{
		VeleroNamespaceName:          common.VeleroNameSpace,
		VeleroMysqlBackupName:        common.BackupMySQLName,
		VeleroMysqlBackupStorageName: common.BackupMySQLStorageName,
		VeleroMysqlHookResourceName:  common.BackupResourceName,
	}
	template.Execute(&b, data)
	err := common.DynamicSSA(context.TODO(), b.String(), t.Logs)
	if err != nil {
		t.Logs.Errorf("Error creating velero backup object", zap.Error(err))
		return err
	}

	return nil
}

func CreateMysqlVeleroRestoreObject() error {
	var b bytes.Buffer
	template, _ := template.New("mysql-restore").Parse(common.MySQLRestore)
	data := common.VeleroMysqlRestoreObject{
		VeleroMysqlRestore:          common.RestoreMySQLName,
		VeleroNamespaceName:         common.VeleroNameSpace,
		VeleroMysqlBackupName:       common.BackupMySQLName,
		VeleroMysqlHookResourceName: common.BackupResourceName,
	}

	template.Execute(&b, data)
	err := common.DynamicSSA(context.TODO(), b.String(), t.Logs)
	if err != nil {
		t.Logs.Errorf("Error creating velero restore object ", zap.Error(err))
		return err
	}
	return nil
}

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

func KeycloakVerifyUsers() bool {
	keycloakClient, err := pkg.NewKeycloakAdminRESTClient()
	if err != nil {
		t.Logs.Errorf("Unable to get keycloak client due to ", zap.Error(err))
		return false
	}

	for i := 0; i < len(common.KeyCloakUserIDList); i++ {
		t.Logs.Infof("Verifying user with username '%s' exists affter mysql restore", common.KeyCloakUserIDList[i])
		ok, err := keycloakClient.VerifyUserExists(constants.VerrazzanoSystemNamespace, common.KeyCloakUserIDList[i])
		if err != nil {
			t.Logs.Errorf("Unable to verify keycloak user due to ", zap.Error(err))
			return false
		}
		if !ok {
			t.Logs.Errorf("User '%s' does not exist or could not be verified.", common.KeyCloakUserIDList[i])
			return false
		}
	}
	return true
}

func DisplayResticInfo(operation string) error {
	var cmdArgs []string
	var apiResource string
	cmdArgs = append(cmdArgs, "kubectl")
	cmdArgs = append(cmdArgs, "get")
	switch operation {
	case "backup":
		apiResource = "podvolumebackups"
	case "restore":
		apiResource = "podvolumerestores"
	}
	cmdArgs = append(cmdArgs, fmt.Sprintf("%s.velero.io", apiResource))
	cmdArgs = append(cmdArgs, "-n")
	cmdArgs = append(cmdArgs, common.VeleroNameSpace)

	var kcmd common.BashCommand
	kcmd.Timeout = 1 * time.Minute
	kcmd.CommandArgs = cmdArgs
	cmdResponse := common.Runner(&kcmd, t.Logs)
	if cmdResponse.CommandError != nil {
		return cmdResponse.CommandError
	}
	return nil
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
		return common.CreateCredentialsSecretFromFile(common.VeleroNameSpace, common.VeleroMySQLSecretName, t.Logs)
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

	t.Logs.Info("Create backup storage location for velero backup objects")
	Eventually(func() error {
		return common.CreateVeleroBackupLocationObject(common.BackupMySQLStorageName, common.VeleroMySQLSecretName, t.Logs)
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

	t.Logs.Info("Create a sample keycloak user")
	Eventually(func() error {
		return KeycloakCreateUsers(10)
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

}

func cleanUpVelero() {
	t.Logs.Info("Cleanup backup and restore objects")

	t.Logs.Info("Cleanup restore object")
	Eventually(func() error {
		return common.CrdPruner("velero.io", "v1", "restores", common.RestoreMySQLName, common.VeleroNameSpace, t.Logs)
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

	t.Logs.Info("Cleanup backup object")
	Eventually(func() error {
		return common.CrdPruner("velero.io", "v1", "backups", common.BackupMySQLName, common.VeleroNameSpace, t.Logs)
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

	t.Logs.Info("Cleanup backup storage object")
	Eventually(func() error {
		return common.CrdPruner("velero.io", "v1", "backupstoragelocations", common.BackupMySQLStorageName, common.VeleroNameSpace, t.Logs)
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

	t.Logs.Info("Cleanup velero secrets")
	Eventually(func() error {
		return common.DeleteSecret(common.VeleroNameSpace, common.VeleroMySQLSecretName, t.Logs)
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

	t.Logs.Info("Delete keycloak user")
	Eventually(func() error {
		return KeycloakDeleteUsers()
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())
}

var _ = t.Describe("Backup Flow,", Label("f:platform-verrazzano.backup"), Serial, func() {

	t.Context("Start backup after velero backup storage location created", func() {
		WhenVeleroInstalledIt("Start backup after velero backup storage location created", func() {
			Eventually(func() error {
				return CreateMysqlVeleroBackupObject()
			}, waitTimeout, pollingInterval).Should(BeNil())
		})
	})

	t.Context("Check backup progress after velero backup object was created", func() {
		WhenVeleroInstalledIt("Check backup progress after velero backup object was created", func() {
			Eventually(func() error {
				return common.TrackOperationProgress(30, "velero", "backups", common.BackupMySQLName, common.VeleroNameSpace, t.Logs)
			}, waitTimeout, pollingInterval).Should(BeNil())
		})
	})

	t.Context("Check pvc backup details", func() {
		WhenVeleroInstalledIt("Check pvc backup details", func() {
			Eventually(func() error {
				return DisplayResticInfo("backup")
			}, waitTimeout, pollingInterval).Should(BeNil())
		})
	})

	t.Context("Cleanup mysql once backup is done", func() {
		WhenVeleroInstalledIt("Cleanup mysql once backup is done", func() {
			Eventually(func() error {
				return common.DeleteNamespace(constants.KeycloakNamespace, t.Logs)
			}, waitTimeout, pollingInterval).Should(BeNil())
		})

	})

	t.Context("Start restore after velero backup is completed", func() {
		WhenVeleroInstalledIt("Start restore after velero backup is completed", func() {
			Eventually(func() error {
				return CreateMysqlVeleroRestoreObject()
			}, waitTimeout, pollingInterval).Should(BeNil())
		})
	})

	t.Context("Check velero restore progress", func() {
		WhenVeleroInstalledIt("Check velero restore progress", func() {
			Eventually(func() error {
				return common.TrackOperationProgress(30, "velero", "restores", common.RestoreMySQLName, common.VeleroNameSpace, t.Logs)
			}, waitTimeout, pollingInterval).Should(BeNil())
		})
	})

	t.Context("After restore is complete wait for keycloak and mysql pods to come up", func() {
		WhenVeleroInstalledIt("After restore is complete wait for keycloak and mysql pods to come up", func() {
			Eventually(func() error {
				return common.WaitForPodsShell(constants.KeycloakNamespace, t.Logs)
			}, waitTimeout, pollingInterval).Should(BeNil(), "Check if keycloak and mysql infra is up")
		})
	})

	t.Context("Check pvc restore details", func() {
		WhenVeleroInstalledIt("Check pvc restore details", func() {
			Eventually(func() error {
				return DisplayResticInfo("restore")
			}, waitTimeout, pollingInterval).Should(BeNil())
		})
	})

	t.Context("Is Restore good? Verify restore", func() {
		WhenVeleroInstalledIt("Is Restore good? Verify restore", func() {
			Eventually(func() bool {
				return KeycloakVerifyUsers()
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})

	})

})
