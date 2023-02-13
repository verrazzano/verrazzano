// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mysql

import (
	"bytes"
	"context"
	"fmt"
	"text/template"
	"time"

	"github.com/verrazzano/verrazzano/pkg/bom"
	ctrlerrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kblabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

// legacyDbLoadJob is the template for the db load job
const legacyDbLoadJob = `
apiVersion: batch/v1
kind: Job
metadata:
  name: load-dump
  namespace: keycloak
  labels:
    app: mysql
    component: restore-keycloak-db
spec:
  backoffLimit: 6
  template:
    spec:
      initContainers:
        - command:
            - bash
            - -c
            - chown -R 27:27 /var/lib/dump
          image: {{.InitContainerImage}}
          imagePullPolicy: IfNotPresent
          name: fixdumpdir
          resources: {}
          securityContext:
            runAsUser: 0
          terminationMessagePath: /dev/termination-log
          terminationMessagePolicy: File
      volumes:
        - name: keycloak-dump
          persistentVolumeClaim:
            claimName: {{ .ClaimName }}
      containers:
        - command: ["bash"]
          args:
            - -c
            - >-
              while ! mysqladmin ping -h"mysql.keycloak.svc.cluster.local" --silent; do sleep 1; done &&
              mysqlsh -u root -p{{ .RootPassword }} -h mysql.keycloak.svc.cluster.local -e 'util.loadDump("/var/lib/dump/{{ .DumpDir }}", {includeSchemas: ["keycloak"], includeUsers: ["keycloak"], loadUsers: true})'
          env:
            - name: MYSQL_HOST
              value: mysql
          image: {{ .ContainerImage }}
          imagePullPolicy: IfNotPresent
          name: mysqlsh-load-dump
          resources: {}
          securityContext:
            runAsUser: 0
      restartPolicy: OnFailure
`

// loadJobValues provides the parameters for the db load job
type loadJobValues struct {
	InitContainerImage string
	ContainerImage     string
	ClaimName          string
	DumpDir            string
	RootPassword       string
}

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
func appendLegacyUpgradePersistenceValues(bomFile *bom.Bom, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	var err error
	kvs, err = appendCustomImageOverrides(bomFile, kvs)
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
	if !isDatabaseMigrationStageCompleted(ctx, pvcRecreatedStage) {
		ctx.Log().Debugf("Updating PV/PVC %v", mysqlPVC)

		if err := common.UpdateExistingVolumeClaims(ctx, mysqlPVC, legacyDBDumpClaim, ComponentName); err != nil {
			ctx.Log().Debugf("Unable to update PV/PVC")
			return err
		}
		err := updateDBMigrationInProgressSecret(ctx, pvcRecreatedStage)
		if err != nil {
			return err
		}
	}

	if err := createLegacyUpgradeJob(ctx); err != nil {
		return err
	}

	// wait till the pod shows up so that it is bound to the PV
	if err := waitForJobPodRunning(ctx, time.Duration(60)*time.Second); err != nil {
		return err
	}

	return nil
}

// isJobPodRunning checks whether the job pod is in a running state
func isJobPodRunning(ctx spi.ComponentContext, jobName string) wait.ConditionFunc {
	return func() (bool, error) {

		ctx.Log().Progress("Waiting for DB load job pod to start")

		selector := &client.ListOptions{LabelSelector: kblabels.SelectorFromSet(kblabels.Set{"job-name": jobName})}
		podList := &v1.PodList{}
		if err := ctx.Client().List(context.TODO(), podList, &client.ListOptions{Namespace: ComponentNamespace}, selector); err != nil {
			return false, err
		}

		if len(podList.Items) <= 0 {
			return false, nil
		}

		switch podList.Items[0].Status.Phase {
		case v1.PodRunning, v1.PodSucceeded:
			ctx.Log().Infof("DB load job pod is running or has completed successfully")
			return true, nil
		case v1.PodFailed:
			// Check if the user manually created a pod to do the DB migration
			if len(podList.Items) > 1 {
				switch podList.Items[1].Status.Phase {
				case v1.PodRunning, v1.PodSucceeded:
					ctx.Log().Infof("DB load job pod is running or has completed successfully")
					return true, nil
				}
			}
			return false, fmt.Errorf("Job pod has completed with a failure")
		}
		return false, nil
	}
}

// waitForPodRunning polls to discover when the job pod is running
func waitForJobPodRunning(ctx spi.ComponentContext, timeout time.Duration) error {
	return wait.PollImmediate(time.Second, timeout, isJobPodRunning(ctx, dbLoadJobName))
}

// createLegacyUpgradeJob creates the job that loads the new DB with the data from the old DB
func createLegacyUpgradeJob(ctx spi.ComponentContext) error {
	job := &batchv1.Job{}
	err := ctx.Client().Get(context.TODO(), types.NamespacedName{Namespace: ComponentNamespace, Name: dbLoadJobName}, job)
	if err != nil {
		if errors.IsNotFound(err) {
			// create the job to load the DB data into new MySQL DB
			var bomFile bom.Bom
			var err error
			if bomFile, err = bom.NewBom(config.GetDefaultBOMFilePath()); err != nil {
				return err
			}
			var kvs []bom.KeyValue
			if kvs, err = appendLegacyUpgradePersistenceValues(&bomFile, []bom.KeyValue{}); err != nil {
				return err
			}

			values := loadJobValues{}
			for _, kv := range kvs {
				if kv.Key == "legacyUpgrade.container.image" {
					values.ContainerImage = kv.Value
					continue
				}
				if kv.Key == "legacyUpgrade.initContainer.image" {
					values.InitContainerImage = kv.Value
					continue
				}
				if kv.Key == "legacyUpgrade.dumpDir" {
					values.DumpDir = kv.Value
					continue
				}

				if kv.Key == "legacyUpgrade.claimName" {
					values.ClaimName = kv.Value
					continue
				}
			}
			// add the root password
			var rootPassword []byte
			if rootPassword, err = getLegacyRootPassword(ctx); err != nil {
				return err
			}
			values.RootPassword = string(rootPassword)

			var b bytes.Buffer
			template, _ := template.New("legacyUpgrade").Parse(legacyDbLoadJob)
			if err = template.Execute(&b, values); err != nil {
				return err
			}

			// create the job reference
			if err = yaml.Unmarshal(b.Bytes(), job); err != nil {
				return err
			}

			// create the job
			if err = ctx.Client().Create(context.TODO(), job); err != nil {
				return err
			}
		} else {
			return err
		}
	} else {
		// Job already exists, check its status
		// If it has failed, clean up the old job so a new one can be queued in the next try
		if isJobFailed(job) {
			// delete the job
			if err := cleanupDbMigrationJob(ctx); err != nil {
				return err
			}
			// return an error so that waitForJobPodRunning will not be executed
			return fmt.Errorf("DB load job has failed and will be retried")
		}
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
	rootPwd, err := getLegacyRootPassword(ctx)
	if err != nil {
		return err
	}

	// Root priv
	rootCmd := fmt.Sprintf(mySQLRootCommand, rootPwd)
	rootExecCmd := []string{"bash", "-c", rootCmd}
	// CHECK and ADD Primary Key Cmd
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
	// Grant root privilege =
	_, _, err = k8sutil.ExecPodNoTty(cli, cfg, mysqlPod, "mysql", rootExecCmd)
	if err != nil {
		errorMsg := maskPw(fmt.Sprintf("Failed granting root priv, err = %v", err))
		ctx.Log().Error(errorMsg)
		return fmt.Errorf("error: %s", maskPw(err.Error()))
	}
	ctx.Log().Info("Successfully updated root privileges")
	// Check and Update Primary Key
	_, _, err = k8sutil.ExecPodNoTty(cli, cfg, mysqlPod, "mysql", execCmd)
	if err != nil {
		errorMsg := maskPw(fmt.Sprintf("Failed updating table, err = %v", err))
		ctx.Log().Error(errorMsg)
		return fmt.Errorf("error: %s", maskPw(err.Error()))
	}
	ctx.Log().Info("Successfully updated keycloak table primary key")
	_, _, err = k8sutil.ExecPodNoTty(cli, cfg, mysqlPod, "mysql", cleanupCmd)
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to remove resources from previous attempts, err = %v", err)
		ctx.Log().Error(errorMsg)
		return fmt.Errorf("error: %s", err.Error())
	}
	_, _, err = k8sutil.ExecPodNoTty(cli, cfg, mysqlPod, "mysql", execShCmd)
	if err != nil {
		errorMsg := maskPw(fmt.Sprintf("Failed executing database dump, err = %v", err))
		ctx.Log().Error(errorMsg)
		return fmt.Errorf("error: %s", maskPw(err.Error()))
	}
	ctx.Log().Info("Successfully persisted database dump")
	err = updateDBMigrationInProgressSecret(ctx, databaseDumpedStage)
	if err != nil {
		return err
	}

	return nil
}

// getLegacyRootPassword returns the root password from the legacy DB secret
func getLegacyRootPassword(ctx spi.ComponentContext) ([]byte, error) {
	// retrieve root password for mysql
	rootSecret := v1.Secret{}
	if err := ctx.Client().Get(context.TODO(), client.ObjectKey{Namespace: ComponentNamespace, Name: secretName}, &rootSecret); err != nil {
		return nil, err
	}
	rootPwd := rootSecret.Data[rootPasswordKey]
	return rootPwd, nil
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

	// Check if the Db Migration was done manually in case of terminal job failure
	dbManuallyMigrated := isDatabaseMigrationStageCompleted(ctx, manualDbMigrationStage)
	if dbManuallyMigrated {
		return true
	}

	loadJob := &batchv1.Job{}
	err := ctx.Client().Get(context.TODO(), types.NamespacedName{Name: dbLoadJobName, Namespace: ComponentNamespace}, loadJob)
	if err != nil {
		return false
	}
	// check to see if job has failed and re-submit
	if loadJob.Status.Failed == 1 {
		ctx.Log().Errorf("DB load job has failed and will be retried.")
		// delete the job
		if err := cleanupDbMigrationJob(ctx); err != nil {
			return false
		}
		// resubmit
		if err := createLegacyUpgradeJob(ctx); err != nil {
			return false
		}

		return false
	}
	// get the associated pod
	dbMigrationPod, err := getDbMigrationPod(ctx)
	if err != nil {
		return false
	}
	for _, container := range dbMigrationPod.Status.ContainerStatuses {
		if container.Name == dbLoadContainerName && container.State.Terminated != nil && container.State.Terminated.ExitCode == 0 {
			ctx.Log().Info("Keycloak DB successfully migrated")
			return true
		}
	}

	return false
}

func isJobFailed(job *batchv1.Job) bool {
	if job.Status.Failed == 1 {
		return true
	}

	for _, condition := range job.Status.Conditions {
		if condition.Type == batchv1.JobFailed && condition.Status == v1.ConditionTrue {
			return true
		}
	}

	return false
}
