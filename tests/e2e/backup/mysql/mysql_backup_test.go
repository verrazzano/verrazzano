// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mysql

import (
	"bytes"
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/verrazzano/verrazzano/pkg/test/framework/metrics"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	common "github.com/verrazzano/verrazzano/tests/e2e/backup/helpers"
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

var keycloakPods = []string{"keycloak", "mysql"}

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

// CreateMysqlVeleroBackupObject creates velero backup CR starting the backup process
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

// CreateMysqlVeleroRestoreObject creates velero restore CR thereby starting the restore process
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

// DisplayResticInfo displays the Pvc pod volume backups/restores
func DisplayResticInfo(operation string) error {

	switch operation {
	case "backup":
		err := common.GetPodVolumeBackups(constants.VeleroNameSpace, t.Logs)
		if err != nil {
			return err
		}
		return nil
	case "restore":
		err := common.GetPodVolumeRestores(constants.VeleroNameSpace, t.Logs)
		if err != nil {
			return err
		}
		return nil
	default:
		t.Logs.Errorf("Invalid operation to display")
		return fmt.Errorf("Invalid opeartion")
	}

}

// CheckInnoDbState fetches the state if ics cluster
func CheckInnoDbState(state string) bool {
	ics, err := common.GetInnoDBCluster(constants.KeycloakNamespace, "mysql", t.Logs)
	if err != nil {
		return false
	}
	if strings.ToLower(ics.Status.Cluster.Status) == strings.ToLower(state) {
		return true
	}
	return false
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
		return KeycloakCreateUsers(common.KeycloakUserCount)
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

}

func cleanUpVelero() {
	t.Logs.Info("Cleanup backup and restore objects")

	t.Logs.Info("Cleanup restore object")
	Eventually(func() error {
		return common.CrdPruner("velero.io", "v1", common.RestoreResource, common.RestoreMySQLName, common.VeleroNameSpace, t.Logs)
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

	t.Logs.Info("Cleanup backup object")
	Eventually(func() error {
		return common.CrdPruner("velero.io", "v1", common.BackupResource, common.BackupMySQLName, common.VeleroNameSpace, t.Logs)
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

	t.Logs.Info("Cleanup backup storage object")
	Eventually(func() error {
		return common.CrdPruner("velero.io", "v1", common.BackupStorageLocationResource, common.BackupMySQLStorageName, common.VeleroNameSpace, t.Logs)
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

var _ = t.Describe("MySQL Backup and Restore,", Label("f:platform-verrazzano.mysql-backup"), Serial, func() {

	t.Context("MySQL backup", func() {
		WhenVeleroInstalledIt("MySQL backup triggered", func() {
			Eventually(func() error {
				return CreateMysqlVeleroBackupObject()
			}, waitTimeout, pollingInterval).Should(BeNil())
		})

		WhenVeleroInstalledIt("Check backup progress after velero backup object was created", func() {
			Eventually(func() error {
				return common.TrackOperationProgress("velero", common.BackupResource, common.BackupMySQLName, common.VeleroNameSpace, t.Logs)
			}, waitTimeout, pollingInterval).Should(BeNil())
		})

		WhenVeleroInstalledIt("Check pvc backup details", func() {
			Eventually(func() error {
				return DisplayResticInfo("backup")
			}, waitTimeout, pollingInterval).Should(BeNil())
		})
	})

	t.Context("Disaster simulation", func() {
		WhenVeleroInstalledIt("Delete users created as part of pre-suite", func() {
			Eventually(func() error {
				return KeycloakDeleteUsers()
			}, waitTimeout, pollingInterval).Should(BeNil())
		})

		WhenVeleroInstalledIt("Delete innodb cluster", func() {
			Eventually(func() error {
				return common.CrdPruner("mysql.oracle.com", "v2", "ics", "mysql", constants.KeycloakNamespace, t.Logs)
			}, waitTimeout, pollingInterval).Should(BeNil())
		})

		WhenVeleroInstalledIt("Delete keycloak namespace", func() {
			Eventually(func() error {
				return common.DeleteNamespace(constants.KeycloakNamespace, t.Logs)
			}, waitTimeout, pollingInterval).Should(BeNil())
		})

		WhenVeleroInstalledIt("Ensure the pods are not running before starting a restore", func() {
			Eventually(func() bool {
				return checkPodsNotRunning(constants.KeycloakNamespace, keycloakPods)
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Check if pods are down")
		})
	})

	t.Context("MySQL restore", func() {
		WhenVeleroInstalledIt(fmt.Sprintf("Start restore of mysql from backup '%s'", common.BackupMySQLName), func() {
			Eventually(func() error {
				return CreateMysqlVeleroRestoreObject()
			}, waitTimeout, pollingInterval).Should(BeNil())
		})
		WhenVeleroInstalledIt("Check MySQL restore progress", func() {
			Eventually(func() error {
				return common.TrackOperationProgress("velero", common.RestoreResource, common.RestoreMySQLName, common.VeleroNameSpace, t.Logs)
			}, waitTimeout, pollingInterval).Should(BeNil())
		})

	})

	t.Context("Flap mysql-operator", func() {
		WhenVeleroInstalledIt("scaling down of mysql operator", func() {
			Eventually(func() error {
				return common.ScaleDeployment("mysql-operator", "mysql-operator", 0, t.Logs)
			}, waitTimeout, pollingInterval).Should(BeNil())
		})

		WhenVeleroInstalledIt("mysql-operator pod down", func() {
			Eventually(func() bool {
				return checkPodsNotRunning("mysql-operator", []string{"mysql-operator"})
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})

		WhenVeleroInstalledIt("scaling up of mysql operator", func() {
			Eventually(func() error {
				return common.ScaleDeployment("mysql-operator", "mysql-operator", 1, t.Logs)
			}, waitTimeout, pollingInterval).Should(BeNil())
		})

		WhenVeleroInstalledIt("mysql-operator pod up", func() {
			Eventually(func() bool {
				return checkPodsRunning("mysql-operator", []string{"mysql-operator"})
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})
	})

	t.Context("MySQL Data and Infra verification", func() {
		WhenVeleroInstalledIt("After restore is complete wait for keycloak and mysql pods to come up", func() {
			Eventually(func() bool {
				return checkPodsRunning(constants.KeycloakNamespace, keycloakPods)
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Check if keycloak and mysql infra is up")
		})
		WhenVeleroInstalledIt("Check pvc restore details", func() {
			Eventually(func() error {
				return DisplayResticInfo("restore")
			}, waitTimeout, pollingInterval).Should(BeNil())
		})
		WhenVeleroInstalledIt("Is Restore good? Verify restore", func() {
			Eventually(func() bool {
				return KeycloakVerifyUsers()
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})

		WhenVeleroInstalledIt("InnoDB state should be 'INITIALIZING'", func() {
			Eventually(func() bool {
				return CheckInnoDbState("INITIALIZING")
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})
	})
})
