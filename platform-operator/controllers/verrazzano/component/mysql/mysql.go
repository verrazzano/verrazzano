// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mysql

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"os"
	"strings"

	"github.com/verrazzano/verrazzano/pkg/bom"
	ctrlerrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
	vzpassword "github.com/verrazzano/verrazzano/pkg/security/password"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kblabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	rootSec               = "mysql-cluster-secret"
	helmRootPwd           = "credentials.root.password" //nolint:gosec //#gosec G101
	mySQLRootKey          = "rootPassword"
	secretName            = "mysql"
	secretKey             = "mysql-password"
	mySQLUsername         = "keycloak"
	rootPasswordKey       = "rootPassword"
	statefulsetClaimName  = "dump-claim"
	mySQLInitFilePrefix   = "init-mysql-"
	initdbScriptsFile     = "initdbScripts.create-db\\.sql"
	backupHookScriptsFile = "configurationFiles.mysql-hook\\.sh"
	mySQLHookFile         = "platform-operator/scripts/hooks/mysql-hook.sh"
	initDbScript          = `CREATE USER IF NOT EXISTS keycloak IDENTIFIED BY '%s';
CREATE DATABASE IF NOT EXISTS keycloak DEFAULT CHARACTER SET utf8 DEFAULT COLLATE utf8_general_ci;
USE keycloak;
GRANT CREATE, ALTER, DROP, INDEX, REFERENCES, SELECT, INSERT, UPDATE, DELETE ON keycloak.* TO '%s'@'%%';
FLUSH PRIVILEGES;
CREATE TABLE IF NOT EXISTS DATABASECHANGELOG (
  ID varchar(255) NOT NULL,
  AUTHOR varchar(255) NOT NULL,
  FILENAME varchar(255) NOT NULL,
  DATEEXECUTED datetime NOT NULL,
  ORDEREXECUTED int(11) NOT NULL,
  EXECTYPE varchar(10) NOT NULL,
  MD5SUM varchar(35) DEFAULT NULL,
  DESCRIPTION varchar(255) DEFAULT NULL,
  COMMENTS varchar(255) DEFAULT NULL,
  TAG varchar(255) DEFAULT NULL,
  LIQUIBASE varchar(20) DEFAULT NULL,
  CONTEXTS varchar(255) DEFAULT NULL,
  LABELS varchar(255) DEFAULT NULL,
  DEPLOYMENT_ID varchar(10) DEFAULT NULL,
  PRIMARY KEY (ID,AUTHOR,FILENAME)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;`
	mySQLDbCommands = `mysql -uroot -p%s -e "USE keycloak;
ALTER TABLE DATABASECHANGELOG;
ADD PRIMARY KEY (ID,AUTHOR,FILENAME);"`
	mySQLShCommands = `mysqlsh -uroot -p%s -e "util.dumpInstance("/var/lib/mysql/dump", {ocimds: true, compatibility: ["strip_definers", "strip_restricted_grants"]})"`
)

var (
	// map of MySQL helm persistence values from previous version to existing version.  Other values will be ignored since
	// they have no equivalents in the current charts
	helmValuesMap = map[string]string{
		"persistence.accessModes":  "datadirVolumeClaimTemplate.accessModes",
		"persistence.size":         "datadirVolumeClaimTemplate.resources.requests.storage",
		"persistence.storageClass": "datadirVolumeClaimTemplate.storageClassName",
	}
	// maskPw will mask passwords in strings with '******'
	maskPw = vzpassword.MaskFunction("password ")
	// Set to true during unit testing
	unitTesting bool
)

// isMySQLReady checks to see if the MySQL component is in ready state
func isMySQLReady(context spi.ComponentContext) bool {
	statefulset := []types.NamespacedName{
		{
			Name:      ComponentName,
			Namespace: ComponentNamespace,
		},
	}
	deployment := []types.NamespacedName{
		{
			Name:      fmt.Sprintf("%s-router", ComponentName),
			Namespace: ComponentNamespace,
		},
	}
	prefix := fmt.Sprintf("Component %s", context.GetComponent())
	serverReplicas := 1
	routerReplicas := 0
	overrides, err := common.GetInstallOverridesYAML(context, GetOverrides(context.EffectiveCR()).([]vzapi.Overrides))
	if err != nil {
		return false
	}
	for _, overrideYaml := range overrides {
		if strings.Contains(overrideYaml, "serverInstances:") {
			value, err := common.ExtractValueFromOverrideString(overrideYaml, "serverInstances")
			if err != nil {
				return false
			}
			serverReplicas = int(value.(float64))
		}
		if strings.Contains(overrideYaml, "routerInstances:") {
			value, err := common.ExtractValueFromOverrideString(overrideYaml, "routerInstances")
			if err != nil {
				return false
			}
			routerReplicas = int(value.(float64))
		}
	}
	ready := status.StatefulSetsAreReady(context.Log(), context.Client(), statefulset, int32(serverReplicas), prefix)
	if routerReplicas > 0 {
		return ready && status.DeploymentsAreReady(context.Log(), context.Client(), deployment, int32(routerReplicas), prefix)
	}
	return ready
}

// appendMySQLOverrides appends the MySQL helm overrides
func appendMySQLOverrides(compContext spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	cr := compContext.EffectiveCR()

	// Pending private regsitry testing as to how to handle the mysql-server and mysql-router images managed
	// by the operator
	//kvs, err := appendCustomImageOverrides(kvs)
	//if err != nil {
	//	return kvs, ctrlerrors.RetryableError{Source: ComponentName, Cause: err}
	//}

	if compContext.Init(ComponentName).GetOperation() == vzconst.InstallOperation {
		err := createKeycloakDBSecret(compContext)
		if err != nil {
			return []bom.KeyValue{}, ctrlerrors.RetryableError{Source: ComponentName, Cause: err}
		}
		mySQLInitFile, err := createMySQLInitFile(compContext)
		if err != nil {
			return []bom.KeyValue{}, ctrlerrors.RetryableError{Source: ComponentName, Cause: err}
		}
		kvs = append(kvs, bom.KeyValue{Key: initdbScriptsFile, Value: mySQLInitFile, SetFile: true})
		kvs = append(kvs, bom.KeyValue{Key: backupHookScriptsFile, Value: mySQLHookFile, SetFile: true})
	}
	kvs, err := appendMySQLSecret(compContext, kvs)
	if err != nil {
		return []bom.KeyValue{}, ctrlerrors.RetryableError{Source: ComponentName, Cause: err}
	}

	// generate the MySQl PV overrides
	kvs, err = generateVolumeSourceOverrides(compContext, kvs)
	if err != nil {
		compContext.Log().Error(err)
		return []bom.KeyValue{}, ctrlerrors.RetryableError{Source: ComponentName, Cause: err}
	}

	// Convert MySQL install-args to helm overrides
	kvs = append(kvs, convertOldInstallArgs(helm.GetInstallArgs(getInstallArgs(cr)))...)

	return kvs, nil
}

// preInstall creates and label the MySQL namespace
func preInstall(compContext spi.ComponentContext, namespace string) error {
	if compContext.IsDryRun() {
		compContext.Log().Debug("MySQL PreInstall dry run")
		return nil
	}
	compContext.Log().Debugf("Adding label needed by network policies to %s namespace", namespace)
	ns := v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
	if _, err := controllerruntime.CreateOrUpdate(context.TODO(), compContext.Client(), &ns, func() error {
		if ns.Labels == nil {
			ns.Labels = make(map[string]string)
		}
		ns.Labels["verrazzano.io/namespace"] = namespace
		istio := compContext.EffectiveCR().Spec.Components.Istio
		if istio != nil && istio.IsInjectionEnabled() {
			ns.Labels["istio-injection"] = "enabled"
		}
		return nil
	}); err != nil {
		return ctrlerrors.RetryableError{Source: ComponentName, Cause: err}
	}
	return nil
}

// postInstall removes the MySQL Init file
func postInstall(ctx spi.ComponentContext) error {
	if ctx.IsDryRun() {
		ctx.Log().Debug("MySQL PostInstall dry run")
		return nil
	}
	// Delete create-mysql-db.sql after install
	return nil
}

// generateVolumeSourceOverrides generates the appropriate persistence overrides given the component context
func generateVolumeSourceOverrides(compContext spi.ComponentContext, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	mySQLVolumeSource := getMySQLVolumeSource(compContext.EffectiveCR())
	// No volumes to process, return what we have
	if mySQLVolumeSource == nil {
		return kvs, nil
	}

	if mySQLVolumeSource.EmptyDir != nil {
		// EmptyDir currently not support with mysql operator
		compContext.Log().Info("EmptyDir currently not supported for MySQL server.  A default persistent volume will be used.")
	} else {
		var err error
		kvs, err = doGenerateVolumeSourceOverrides(compContext.EffectiveCR(), kvs)
		if err != nil {
			return kvs, err
		}
	}

	return kvs, nil
}

// doGenerateVolumeSourceOverrides generates the appropriate persistence overrides given the effective CR
func doGenerateVolumeSourceOverrides(effectiveCR *vzapi.Verrazzano, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	mySQLVolumeSource := getMySQLVolumeSource(effectiveCR)

	if mySQLVolumeSource != nil && mySQLVolumeSource.PersistentVolumeClaim != nil {
		// Configured for persistence, adapt the PVC Spec template to the appropriate Helm args
		pvcs := mySQLVolumeSource.PersistentVolumeClaim
		storageSpec, found := vzconfig.FindVolumeTemplate(pvcs.ClaimName, effectiveCR.Spec.VolumeClaimSpecTemplates)
		if !found {
			return kvs, fmt.Errorf("failed, No VolumeClaimTemplate found for %s", pvcs.ClaimName)
		}
		storageClass := storageSpec.StorageClassName
		if storageClass != nil && len(*storageClass) > 0 {
			kvs = append(kvs, bom.KeyValue{
				Key:       "datadirVolumeClaimTemplate.storageClassName",
				Value:     *storageClass,
				SetString: true,
			})
		}
		storage := storageSpec.Resources.Requests.Storage()
		if storageSpec.Resources.Requests != nil && !storage.IsZero() {
			kvs = append(kvs, bom.KeyValue{
				Key:       "datadirVolumeClaimTemplate.resources.requests.storage",
				Value:     storage.String(),
				SetString: true,
			})
		}
		accessModes := storageSpec.AccessModes
		if len(accessModes) > 0 {
			// MySQL only allows a single AccessMode value, so just choose the first
			kvs = append(kvs, bom.KeyValue{
				Key:       "datadirVolumeClaimTemplate.accessModes",
				Value:     string(accessModes[0]),
				SetString: true,
			})
		}
	}
	return kvs, nil
}

func getMySQLVolumeSource(effectiveCR *vzapi.Verrazzano) *v1.VolumeSource {
	var mySQLVolumeSource *v1.VolumeSource
	if effectiveCR.Spec.Components.Keycloak != nil {
		mySQLVolumeSource = effectiveCR.Spec.Components.Keycloak.MySQL.VolumeSource
	}
	if mySQLVolumeSource == nil {
		mySQLVolumeSource = effectiveCR.Spec.DefaultVolumeSource
	}
	return mySQLVolumeSource
}

////appendCustomImageOverrides - Append the custom overrides for the busybox initContainer
//func appendCustomImageOverrides(kvs []bom.KeyValue) ([]bom.KeyValue, error) {
//	bomFile, err := bom.NewBom(config.GetDefaultBOMFilePath())
//	if err != nil {
//		return kvs, ctrlerrors.RetryableError{Source: ComponentName, Cause: err}
//	}
//
//	imageOverrides, err := bomFile.BuildImageOverrides("mysql")
//	if err != nil {
//		return kvs, ctrlerrors.RetryableError{Source: ComponentName, Cause: err}
//	}
//
//	kvs = append(kvs, imageOverrides...)
//	return kvs, nil
//}

// getInstallArgs get the install args for MySQL
func getInstallArgs(cr *vzapi.Verrazzano) []vzapi.InstallArgs {
	if cr.Spec.Components.Keycloak == nil {
		return []vzapi.InstallArgs{}
	}
	return cr.Spec.Components.Keycloak.MySQL.MySQLInstallArgs
}

// GetOverrides gets the list of overrides
func GetOverrides(object runtime.Object) interface{} {
	if effectiveCR, ok := object.(*vzapi.Verrazzano); ok {
		if effectiveCR.Spec.Components.Keycloak != nil {
			return effectiveCR.Spec.Components.Keycloak.MySQL.ValueOverrides
		}
		return []vzapi.Overrides{}
	} else if effectiveCR, ok := object.(*installv1beta1.Verrazzano); ok {
		if effectiveCR.Spec.Components.Keycloak != nil {
			return effectiveCR.Spec.Components.Keycloak.MySQL.ValueOverrides
		}
		return []installv1beta1.Overrides{}
	}

	return []vzapi.Overrides{}
}

func appendMySQLSecret(compContext spi.ComponentContext, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	rootSecret := &v1.Secret{}
	nsName := types.NamespacedName{
		Namespace: ComponentNamespace,
		Name:      rootSec,
	}
	// use self signed
	kvs = append(kvs, bom.KeyValue{
		Key:   "tls.useSelfSigned",
		Value: "true",
	})
	// Get the mysql userSecret
	err := compContext.Client().Get(context.TODO(), nsName, rootSecret)
	if err != nil {
		// A secret is not expected to be found the first time around (i.e. it's an install and not an update scenario).
		// So do not return an error in this case.
		if errors.IsNotFound(err) && compContext.Init(ComponentName).GetOperation() == vzconst.InstallOperation {
			rootPwd, err := vzpassword.GeneratePassword(12)
			if err != nil {
				return kvs, ctrlerrors.RetryableError{Source: ComponentName, Cause: err}
			}
			// setup root password
			kvs = append(kvs, bom.KeyValue{
				Key:       helmRootPwd,
				Value:     rootPwd,
				SetString: true,
			})
			return kvs, nil
		}
		// Return an error for upgrade or update
		return []bom.KeyValue{}, compContext.Log().ErrorfNewErr("Failed getting MySQL userSecret: %v", err)
	}
	// Force mysql to use the initial password and root password during the upgrade or update, by specifying as helm overrides
	kvs = append(kvs, bom.KeyValue{
		Key:   helmRootPwd,
		Value: string(rootSecret.Data[mySQLRootKey]),
	})
	return kvs, nil
}

// preUpgrade handles the re-association of a previous MySQL deployment PV/PVC with the new MySQL statefulset (if needed)
func preUpgrade(ctx spi.ComponentContext) error {
	if ctx.IsDryRun() {
		ctx.Log().Debug("MySQL pre upgrade dry run")
		return nil
	}

	// following steps are only needed for persistent storage
	mySQLVolumeSource := getMySQLVolumeSource(ctx.EffectiveCR())
	if mySQLVolumeSource != nil && mySQLVolumeSource.PersistentVolumeClaim != nil {
		// get the current MySQL deployment
		deployment, err := getMySQLDeployment(ctx)
		if err != nil {
			return err
		}
		// vz > 1.3 uses statefulsets, not deployments
		// no migration is needed if vz >= 1.4
		if deployment != nil {
			// change the ReclaimPolicy of the PV to Reclaim
			mysqlPVC := types.NamespacedName{Namespace: ComponentNamespace, Name: DeploymentPersistentVolumeClaim}
			err := common.RetainPersistentVolume(ctx, mysqlPVC, ComponentName)
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

			ctx.Log().Debugf("Updating PV/PVC %v", mysqlPVC)
			if err := common.UpdateExistingVolumeClaims(ctx, mysqlPVC, StatefulsetPersistentVolumeClaim, ComponentName); err != nil {
				ctx.Log().Debugf("Unable to update PV/PVC")
				return err
			}
		}
	}
	return nil
}

// postUpgrade perform operations required after the helm upgrade completes
func postUpgrade(ctx spi.ComponentContext) error {
	if ctx.IsDryRun() {
		ctx.Log().Debug("MySQL post upgrade dry run")
		return nil
	}
	mySQLVolumeSource := getMySQLVolumeSource(ctx.EffectiveCR())
	if mySQLVolumeSource != nil && mySQLVolumeSource.PersistentVolumeClaim != nil {
		return common.ResetVolumeReclaimPolicy(ctx, ComponentName)
	}
	return nil
}

// convertOldInstallArgs changes persistence.* install args to primary.persistence.* to keep compatibility with the new chart
func convertOldInstallArgs(kvs []bom.KeyValue) []bom.KeyValue {
	for i, kv := range kvs {
		newValueKey, ok := helmValuesMap[kv.Key]
		if ok {
			kvs[i].Key = newValueKey
		}
	}
	return kvs
}

// createMySQLInitFile creates the .sql file that gets passed to helm as an override
// this initializes the MySQL DB
func createMySQLInitFile(ctx spi.ComponentContext) (string, error) {
	// retrieve the keycloak user password
	userSecret := v1.Secret{}
	if err := ctx.Client().Get(context.TODO(), client.ObjectKey{Namespace: ComponentNamespace, Name: secretName}, &userSecret); err != nil {
		return "", err
	}
	userPwd := userSecret.Data[secretKey]
	file, err := os.CreateTemp(os.TempDir(), fmt.Sprintf("%s*.sql", mySQLInitFilePrefix))
	if err != nil {
		return "", err
	}
	_, err = file.Write([]byte(fmt.Sprintf(initDbScript, userPwd, mySQLUsername)))
	if err != nil {
		return "", ctx.Log().ErrorfNewErr("Failed to write to temporary file: %v", err)
	}
	// Close the file
	if err := file.Close(); err != nil {
		return "", ctx.Log().ErrorfNewErr("Failed to close temporary file: %v", err)
	}
	return file.Name(), nil
}

// createKeycloakDBSecret creates or updates a secret containing the password used by keycloak to access the DB
func createKeycloakDBSecret(ctx spi.ComponentContext) error {
	// create MySQL keycloak user secret
	keycloakSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ComponentNamespace,
			Name:      secretName,
		},
	}
	err := ctx.Client().Get(context.TODO(), client.ObjectKey{
		Namespace: ComponentNamespace,
		Name:      secretName,
	}, keycloakSecret)

	if err != nil {
		password, err := vzpassword.GeneratePassword(12)
		if err != nil {
			return err
		}
		_, err = controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), keycloakSecret, func() error {
			keycloakSecret.Data = map[string][]byte{
				secretKey: []byte(password),
			}
			return nil
		})
		if err != nil {
			return err
		}
		ctx.Log().Once("Component Keycloak successfully created the keycloak db secret")
	}
	return nil
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
	ctx.Log().Infof("Returning pod %s for mysql setup", mysqlPods.Items[0].Name)
	return &mysqlPods.Items[0], nil
}

// getMySQLDeployment returns the deployment if it exists
func getMySQLDeployment(ctx spi.ComponentContext) (*appsv1.Deployment, error) {
	mysqlDeployment := &appsv1.Deployment{}
	if err := ctx.Client().Get(context.TODO(), client.ObjectKey{
		Namespace: ComponentNamespace,
		Name:      ComponentName,
	}, mysqlDeployment); err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, ctx.Log().ErrorfNewErr("Failed getting mysql deployment: %v", err)
	}
	return mysqlDeployment, nil
}

// dumpDatabase uses the mySQL Shell utility to dump the instance to its mounted PV
func dumpDatabase(ctx spi.ComponentContext) error {
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
	stdOut, stdErr, err := k8sutil.ExecPod(cli, cfg, mysqlPod, "mysql", execCmd)
	if err != nil {
		errorMsg := maskPw(fmt.Sprintf("Failed logging into mysql: stdout = %s: stderr = %s, err = %v", stdOut, stdErr, err))
		ctx.Log().Error(errorMsg)
		return fmt.Errorf("error: %s", maskPw(err.Error()))
	}
	stdOut, stdErr, err = k8sutil.ExecPod(cli, cfg, mysqlPod, "mysql", execShCmd)
	if err != nil {
		errorMsg := maskPw(fmt.Sprintf("Failed logging into mysql: stdout = %s: stderr = %s, err = %v", stdOut, stdErr, err))
		ctx.Log().Error(errorMsg)
		return fmt.Errorf("error: %s", maskPw(err.Error()))
	}
	return nil
}

// This is needed for unit testing
func initUnitTesting() {
	unitTesting = true
}
