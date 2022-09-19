// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mysql

import (
	"context"
	"fmt"

	"github.com/verrazzano/verrazzano/pkg/bom"
	ctrlerrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kblabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// appendLegacyUpgradeBaseValues appends the MySQL helm values required for db migration
func appendLegacyUpgradeBaseValues(compContext spi.ComponentContext, kvs []bom.KeyValue) ([]bom.KeyValue, []byte, error) {
	var err error
	secretName := types.NamespacedName{
		Namespace: ComponentNamespace,
		Name:      ComponentName,
	}
	kvs, err = appendMySQLSecret(compContext, secretName, "mysql-root-password", kvs)
	if err != nil {
		return []bom.KeyValue{}, nil, ctrlerrors.RetryableError{Source: ComponentName, Cause: err}
	}
	userPwd, err := getLegacyUserSecret(compContext)
	if err != nil {
		return []bom.KeyValue{}, nil, ctrlerrors.RetryableError{Source: ComponentName, Cause: err}
	}
	// add user settings to enable persistence in cluster secret
	kvs = append(kvs, bom.KeyValue{Key: helmUserPwd, Value: string(userPwd)})
	kvs = append(kvs, bom.KeyValue{Key: helmUserName, Value: mySQLUsername})

	return kvs, userPwd, nil
}

// isDatabaseMigrationStageCompleted indicates whether the given migration stage is completed
func isDatabaseMigrationStageCompleted(ctx spi.ComponentContext, stage string) bool {
	secret := &v1.Secret{}
	err := ctx.Client().Get(context.TODO(), client.ObjectKey{
		Namespace: ComponentNamespace,
		Name:      dbMigrationSecret,
	}, secret)
	if err != nil {
		return false
	}
	_, ok := secret.Data[stage]
	ctx.Log().Debugf("Database migration stage %s completed: %s", stage, ok)
	return ok
}

// updateDBMigrationInProgressSecret creates or updates a secret that indicates the stage status of a DB migration
func updateDBMigrationInProgressSecret(ctx spi.ComponentContext, stage string) error {
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: dbMigrationSecret, Namespace: ComponentNamespace},
	}
	// If the secret doesn't exist, create it
	_, err := controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), secret, func() error {
		// Build the secret data
		if secret.Data == nil {
			secret.Data = make(map[string][]byte)
		}
		secret.Data[stage] = []byte("true")

		return nil
	})

	if err != nil {
		return err
	}
	return nil
}

// isLegacyPersistentDatabase indicates whether the database migration is for a database backed by a persistent store
func isLegacyPersistentDatabase(compContext spi.ComponentContext) bool {
	if isDatabaseMigrationStageCompleted(compContext, pvcDeletedStage) {
		return true
	}
	pvc := &v1.PersistentVolumeClaim{}
	err := compContext.Client().Get(context.TODO(), types.NamespacedName{Namespace: ComponentNamespace, Name: DeploymentPersistentVolumeClaim}, pvc)
	return err == nil
}

// isLegacyDatabaseUpgrade indicates whether the MySQL database being upgraded is from a legacy version
func isLegacyDatabaseUpgrade(compContext spi.ComponentContext) bool {
	deploymentFound := isDatabaseMigrationStageCompleted(compContext, deploymentFoundStage)
	compContext.Log().Debugf("is legacy upgrade based on secret: %s", deploymentFound)
	if !deploymentFound {
		// get the current MySQL deployment
		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ComponentName,
				Namespace: ComponentNamespace,
			},
		}
		compContext.Log().Debugf("Looking for deployment %s", ComponentName)
		err := compContext.Client().Get(context.TODO(), types.NamespacedName{Namespace: ComponentNamespace, Name: ComponentName}, deployment)
		if err != nil {
			compContext.Log().Infof("No legacy database deployment found")
			return false
		}

		err = updateDBMigrationInProgressSecret(compContext, deploymentFoundStage)
		return err == nil
	}

	return true
}

// appendLegacyUpgradePersistenceValues appends the helm values necessary to support a legacy persistent database upgrade
func appendLegacyUpgradePersistenceValues(kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	var err error
	kvs, err = appendCustomImageOverrides(kvs)
	if err != nil {
		return kvs, ctrlerrors.RetryableError{Source: ComponentName, Cause: err}
	}
	kvs = append(kvs, bom.KeyValue{
		Key:       "legacyUpgrade.claimName",
		Value:     "dump-claim",
		SetString: true,
	})
	kvs = append(kvs, bom.KeyValue{
		Key:       "legacyUpgrade.dumpDir",
		Value:     "dump",
		SetString: true,
	})
	return kvs, nil
}

// handleLegacyDatabasePreUpgrade performs the steps required to prepare a database migration
func handleLegacyDatabasePreUpgrade(ctx spi.ComponentContext) error {
	mysqlPVC := types.NamespacedName{Namespace: ComponentNamespace, Name: DeploymentPersistentVolumeClaim}
	ctx.Log().Once("Performing pre-upgrade steps required for legacy database")
	if !isDatabaseMigrationStageCompleted(ctx, pvcDeletedStage) {
		pvc := &v1.PersistentVolumeClaim{}

		if err := ctx.Client().Get(context.TODO(), mysqlPVC, pvc); err != nil {
			// no pvc so just log it and there's nothing left to do
			if errors.IsNotFound(err) {
				ctx.Log().Debugf("Did not find pvc %s. No database data migration required.", mysqlPVC)
				return nil
			}
			return err
		}
		err := common.RetainPersistentVolume(ctx, pvc, ComponentName)
		if err != nil {
			return err
		}
		if !unitTesting { // perform instance dump of MySQL
			if err := dumpDatabase(ctx); err != nil {
				ctx.Log().Debugf("Unable to perform dump of database %s", ComponentName)
				return err
			}
		}
		// delete the deployment to free up the pv/pvc
		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      ComponentName,
			},
		}
		ctx.Log().Debugf("Deleting deployment %s", ComponentName)
		if err := ctx.Client().Delete(context.TODO(), deployment); err != nil {
			if !errors.IsNotFound(err) {
				ctx.Log().Debugf("Unable to delete deployment %s", ComponentName)
				return err
			}
		} else {
			ctx.Log().Debugf("Deployment %v deleted", deployment.ObjectMeta)
		}

		ctx.Log().Debugf("Deleting PVC %v", mysqlPVC)
		if err := common.DeleteExistingVolumeClaim(ctx, mysqlPVC); err != nil {
			ctx.Log().Debugf("Unable to delete existing PVC %v", mysqlPVC)
			return err
		}
		err = updateDBMigrationInProgressSecret(ctx, pvcDeletedStage)
		if err != nil {
			return err
		}
	}
	ctx.Log().Debugf("Updating PV/PVC %v", mysqlPVC)
	if err := common.UpdateExistingVolumeClaims(ctx, mysqlPVC, StatefulsetPersistentVolumeClaim, ComponentName); err != nil {
		ctx.Log().Debugf("Unable to update PV/PVC")
		return err
	}

	return nil
}

// getLegacyUserSecret returns the legacy secret containing DB credentials
func getLegacyUserSecret(ctx spi.ComponentContext) ([]byte, error) {
	// retrieve the keycloak user password
	userSecret := v1.Secret{}
	if err := ctx.Client().Get(context.TODO(), client.ObjectKey{Namespace: ComponentNamespace, Name: secretName}, &userSecret); err != nil {
		return nil, err
	}
	userPwd := userSecret.Data[secretKey]
	return userPwd, nil
}

// getMySQLPod returns the mySQL pod that mounts the PV used for migration
func getMySQLPod(ctx spi.ComponentContext) (*v1.Pod, error) {
	appReq, _ := kblabels.NewRequirement("app", selection.Equals, []string{"mysql"})
	relReq, _ := kblabels.NewRequirement("release", selection.Equals, []string{"mysql"})
	labelSelector := kblabels.NewSelector()
	labelSelector = labelSelector.Add(*appReq, *relReq)
	mysqlPods := v1.PodList{}
	err := ctx.Client().List(context.TODO(), &mysqlPods, &client.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return nil, err
	}
	// return one of the pods
	ctx.Log().Debugf("Returning pod %s for mysql setup", mysqlPods.Items[0].Name)
	return &mysqlPods.Items[0], nil
}

// getDbMigrationPod returns the db migration pod that loads the legacy db into upgraded MySQL db
func getDbMigrationPod(ctx spi.ComponentContext) (*v1.Pod, error) {
	jobNameReq, _ := kblabels.NewRequirement("job-name", selection.Equals, []string{dbLoadJobName})
	labelSelector := kblabels.NewSelector()
	labelSelector = labelSelector.Add(*jobNameReq)
	dbMigrationPods := v1.PodList{}
	err := ctx.Client().List(context.TODO(), &dbMigrationPods, &client.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return nil, err
	}
	// return one of the pods
	return &dbMigrationPods.Items[0], nil
}

// dumpDatabase uses the mySQL Shell utility to dump the instance to its mounted PV
func dumpDatabase(ctx spi.ComponentContext) error {
	if isDatabaseMigrationStageCompleted(ctx, databaseDumpedStage) {
		return nil
	}
	// retrieve root password for mysql
	rootSecret := v1.Secret{}
	if err := ctx.Client().Get(context.TODO(), client.ObjectKey{Namespace: ComponentNamespace, Name: secretName}, &rootSecret); err != nil {
		return err
	}
	rootPwd := rootSecret.Data[rootPasswordKey]

	// ADD Primary Key Cmd
	sqlCmd := fmt.Sprintf(mySQLDbCommands, rootPwd)
	execCmd := []string{"bash", "-c", sqlCmd}
	// util.dumpInstance() Cmd
	cleanupCmd := []string{"bash", "-c", mySQLCleanup}
	sqlShCmd := fmt.Sprintf(mySQLShCommands, rootPwd)
	execShCmd := []string{"bash", "-c", sqlShCmd}
	cfg, cli, err := k8sutil.ClientConfig()
	if err != nil {
		return err
	}
	mysqlPod, err := getMySQLPod(ctx)
	if err != nil {
		return err
	}
	_, _, err = k8sutil.ExecPod(cli, cfg, mysqlPod, "mysql", execCmd)
	if err != nil {
		errorMsg := maskPw(fmt.Sprintf("Failed updating table, err = %v", err))
		ctx.Log().Error(errorMsg)
		return fmt.Errorf("error: %s", maskPw(err.Error()))
	}
	ctx.Log().Debug("Successfully updated keycloak table primary key")
	_, _, err = k8sutil.ExecPod(cli, cfg, mysqlPod, "mysql", cleanupCmd)
	_, _, err = k8sutil.ExecPod(cli, cfg, mysqlPod, "mysql", execShCmd)
	if err != nil {
		errorMsg := maskPw(fmt.Sprintf("Failed executing database dump, err = %v", err))
		ctx.Log().Error(errorMsg)
		return fmt.Errorf("error: %s", maskPw(err.Error()))
	}
	ctx.Log().Debug("Successfully persisted database dump")
	err = updateDBMigrationInProgressSecret(ctx, databaseDumpedStage)
	if err != nil {
		return err
	}

	return nil
}

// deleteDbMigrationSecret deletes the secret tracing the db migration stages
func deleteDbMigrationSecret(ctx spi.ComponentContext) error {
	migrationSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ComponentNamespace,
			Name:      dbMigrationSecret,
		},
	}

	if err := ctx.Client().Delete(context.TODO(), migrationSecret); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	} else {
		ctx.Log().Debugf("Secret %v deleted", migrationSecret.ObjectMeta)
	}

	return nil
}

// cleanupDbMigrationJob cleans up the db migration job and associated resources (pod)
func cleanupDbMigrationJob(ctx spi.ComponentContext) error {
	jobFound := &batchv1.Job{}
	err := ctx.Client().Get(context.TODO(), types.NamespacedName{Name: dbLoadJobName, Namespace: ComponentNamespace}, jobFound)
	if err == nil {
		propagationPolicy := metav1.DeletePropagationBackground
		deleteOptions := &client.DeleteOptions{PropagationPolicy: &propagationPolicy}
		err = ctx.Client().Delete(context.TODO(), jobFound, deleteOptions)
		if err != nil {
			return err
		}
	}
	return nil
}

// checkDbMigrationJobCompletion checks whether a db migration job exists and it has completed
func checkDbMigrationJobCompletion(ctx spi.ComponentContext) bool {
	// check for existence of db restoration job.  If it exists, wait for its completion
	ctx.Log().Progress("Checking status of keycloak DB restoration")
	loadJob := &batchv1.Job{}
	err := ctx.Client().Get(context.TODO(), types.NamespacedName{Name: dbLoadJobName, Namespace: ComponentNamespace}, loadJob)
	if err != nil {
		return errors.IsNotFound(err)
	}
	// get the associated pod
	dbMigrationPod, err := getDbMigrationPod(ctx)
	if err != nil {
		return false
	}
	for _, container := range dbMigrationPod.Status.ContainerStatuses {
		if container.Name == dbLoadContainerName && container.State.Terminated != nil && container.State.Terminated.ExitCode == 0 {
			return true
		}
	}

	return false
}
