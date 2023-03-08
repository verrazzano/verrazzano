// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mysql

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/bom"
	ctrlerrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
	k8sready "github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzpassword "github.com/verrazzano/verrazzano/pkg/security/password"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common/override"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/mysqloperator"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/mysqlcheck"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"os"
	controllerruntime "sigs.k8s.io/controller-runtime"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

const (
	rootSec                = "mysql-cluster-secret"
	helmRootPwd            = "credentials.root.password" //nolint:gosec //#gosec G101
	helmUserPwd            = "credentials.user.password" //nolint:gosec //#gosec G101
	helmUserName           = "credentials.user.name"     //nolint:gosec //#gosec G101
	mysqlUpgradeSubComp    = "mysql-upgrade"
	mySQLRootKey           = "rootPassword"
	mySQLUserKey           = "userPassword"
	secretName             = "mysql"
	secretKey              = "mysql-password"
	mySQLUsername          = "keycloak"
	rootPasswordKey        = "mysql-root-password" //nolint:gosec //#gosec G101
	legacyDBDumpClaim      = "dump-claim"
	mySQLInitFilePrefix    = "init-mysql-"
	dbLoadJobName          = "load-dump"
	dbLoadContainerName    = "mysqlsh-load-dump"
	deploymentFoundStage   = "deployment-found"
	databaseDumpedStage    = "database-dumped"
	pvcDeletedStage        = "pvc-deleted"
	pvcRecreatedStage      = "pvc-recreated"
	manualDbMigrationStage = "db-migrated"
	initdbScriptsFile      = "initdbScripts.create-db\\.sh"
	backupHookScriptsFile  = "configurationFiles.mysql-hook\\.sh"
	dbMigrationSecret      = "db-migration"
	mySQLHookFile          = "platform-operator/scripts/hooks/mysql-hook.sh"
	serverVersionKey       = "serverVersion"
	bomSubComponentName    = "mysql-upgrade"
	mysqlServerImageName   = "mysql-server"
	imageRepositoryKey     = "image.repository"
	mySQLPodName           = "mysql-0"
	mySQLContainerName     = "mysql"
	initDbScript           = `#!/bin/sh

if [[ $HOSTNAME == *-0 ]]; then
   IsRestore="${DB_RESTORE:-false}"
   rootPassword="${MYSQL_ROOT_PASSWORD}"
   mysql -u root -p${rootPassword} << EOF
CREATE USER IF NOT EXISTS keycloak IDENTIFIED BY '%s';
CREATE DATABASE IF NOT EXISTS keycloak DEFAULT CHARACTER SET utf8 DEFAULT COLLATE utf8_general_ci;
GRANT CREATE, ALTER, DROP, INDEX, REFERENCES, SELECT, INSERT, UPDATE, DELETE ON keycloak.* TO '%s'@'%%';
FLUSH PRIVILEGES;
USE keycloak;
GRANT XA_RECOVER_ADMIN ON *.* to 'keycloak'@'%%';
EOF
   if [[ $IsRestore == false ]]; then
      mysql -u root -p${rootPassword} << EOF
USE keycloak;
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
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
EOF
   fi
fi
`
	mySQLRootCommand = `/usr/bin/mysql -uroot -p%s <<EOF
GRANT ALL PRIVILEGES ON *.* TO 'root'@'localhost'; flush privileges;
EOF
`
	mySQLDbCommands = `/usr/bin/mysql -uroot -p%s <<EOF
use keycloak;
delimiter //
drop procedure if exists updatePrimaryKey //
create procedure updatePrimaryKey()
begin
   declare updateRequired INT DEFAULT 0;
   select count(*)
   into updateRequired
   from INFORMATION_SCHEMA.TABLE_CONSTRAINTS
   where CONSTRAINT_TYPE = 'PRIMARY KEY'
   and TABLE_NAME = 'DATABASECHANGELOG'
   and TABLE_SCHEMA = 'keycloak';
   if updateRequired = 0 then
      ALTER TABLE keycloak.DATABASECHANGELOG ADD PRIMARY KEY (ID,AUTHOR,FILENAME);
   end if;
end//
delimiter ;
call updatePrimaryKey();
EOF
`
	mySQLCleanup    = `rm -rf /var/lib/mysql/dump`
	mySQLShCommands = `/usr/bin/mysqlsh -uroot -p%s -S /var/lib/mysql/mysql.sock --js <<EOF
util.dumpInstance("/var/lib/mysql/dump", {ocimds: true, compatibility: ["strip_definers", "strip_restricted_grants"]})
EOF
`
	innoDBClusterStatusOnline = "ONLINE"

	// Command to grant XA_RECOVER_ADMIN to Keycloak user, during upgrade
	mySQLGrantXAAdminCommand = `/usr/bin/mysql -uroot -p%s <<EOF
use keycloak;
GRANT XA_RECOVER_ADMIN ON *.* to 'keycloak'@'%%';
EOF
`
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
	maskPw = vzpassword.MaskFunction("-p")
	// Set to true during unit testing
	unitTesting bool

	innoDBClusterGVK = schema.GroupVersionKind{
		Group:   "mysql.oracle.com",
		Version: "v2",
		Kind:    "InnoDBCluster",
	}

	innoDBClusterStatusFields = []string{"status", "cluster", "status"}
)

// isMySQLReady checks to see if the MySQL component is in ready state
func (c mysqlComponent) isMySQLReady(ctx spi.ComponentContext) bool {
	deployment := []types.NamespacedName{
		{
			Name:      fmt.Sprintf("%s-router", ComponentName),
			Namespace: ComponentNamespace,
		},
	}
	prefix := fmt.Sprintf("Component %s", ctx.GetComponent())
	serverReplicas := 1
	routerReplicas := 0
	overrides, err := override.GetInstallOverridesYAML(ctx, GetOverrides(ctx.EffectiveCR()).([]vzapi.Overrides))
	if err != nil {
		return false
	}
	for _, overrideYaml := range overrides {
		if strings.Contains(overrideYaml, "serverInstances:") {
			value, err := override.ExtractValueFromOverrideString(overrideYaml, "serverInstances")
			if err != nil {
				return false
			}
			serverReplicas = int(value.(float64))
		}
		if strings.Contains(overrideYaml, "routerInstances:") {
			value, err := override.ExtractValueFromOverrideString(overrideYaml, "routerInstances")
			if err != nil {
				return false
			}
			routerReplicas = int(value.(float64))
		}
	}
	ready := k8sready.StatefulSetsAreReady(ctx.Log(), ctx.Client(), c.AvailabilityObjects.StatefulsetNames, int32(serverReplicas), prefix)

	if ready && routerReplicas > 0 {
		ready = k8sready.DeploymentsAreReady(ctx.Log(), ctx.Client(), deployment, int32(routerReplicas), prefix)
	}

	return ready && checkDbMigrationJobCompletion(ctx) && isInnoDBClusterOnline(ctx)
}

// isInnoDBClusterOnline returns true if the InnoDBCluster resource cluster status is online
func isInnoDBClusterOnline(ctx spi.ComponentContext) bool {
	ctx.Log().Debug("Checking if the InnoDBCluster is online")

	innoDBCluster := unstructured.Unstructured{}
	innoDBCluster.SetGroupVersionKind(innoDBClusterGVK)

	// the InnoDBCluster resource name is the helm release name
	nsn := types.NamespacedName{Namespace: ComponentNamespace, Name: helmReleaseName}
	if err := ctx.Client().Get(context.Background(), nsn, &innoDBCluster); err != nil {
		ctx.Log().Errorf("Error retrieving InnoDBCluster %v: %v", nsn, err)
		return false
	}

	clusterStatus, exists, err := unstructured.NestedString(innoDBCluster.UnstructuredContent(), innoDBClusterStatusFields...)
	if err != nil {
		ctx.Log().Errorf("Error retrieving InnoDBCluster %v clusterStatus: %v", nsn, err)
		return false
	}
	if exists {
		if clusterStatus == innoDBClusterStatusOnline {
			ctx.Log().Debugf("InnoDBCluster %v is online", nsn)
			return true
		}
		ctx.Log().Progressf("Waiting for InnoDBCluster %v to be online, cluster status is: %s", nsn, clusterStatus)
		return false
	}

	ctx.Log().Debugf("InnoDBCluster %v clusterStatus not found", nsn)
	return false
}

// appendMySQLOverrides appends the MySQL helm overrides
func appendMySQLOverrides(compContext spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	cr := compContext.EffectiveCR()

	var err error
	var bomFile bom.Bom

	bomFile, err = bom.NewBom(config.GetDefaultBOMFilePath())
	if err != nil {
		return kvs, ctrlerrors.RetryableError{Source: ComponentName, Cause: err}
	}

	if compContext.Init(ComponentName).GetOperation() == vzconst.UpgradeOperation {
		var userPwd []byte
		if isLegacyDatabaseUpgrade(compContext) {
			kvs, userPwd, err = appendLegacyUpgradeBaseValues(compContext, kvs)
			if err != nil {
				return []bom.KeyValue{}, ctrlerrors.RetryableError{Source: ComponentName, Cause: err}
			}
			if !isLegacyPersistentDatabase(compContext) {
				// we are in the process of upgrading from a MySQL deployment using ephemeral storage, so we need to
				// provide the sql initialization file
				kvs, err = appendDatabaseInitializationValues(compContext, userPwd, kvs)
				if err != nil {
					return []bom.KeyValue{}, ctrlerrors.RetryableError{Source: ComponentName, Cause: err}
				}
			}
		}
	}

	if compContext.Init(ComponentName).GetOperation() == vzconst.InstallOperation {
		userPwd, err := getOrCreateDBUserPassword(compContext)
		if err != nil {
			return []bom.KeyValue{}, ctrlerrors.RetryableError{Source: ComponentName, Cause: err}
		}
		// persist the user password in the cluster secret
		kvs = append(kvs, bom.KeyValue{Key: helmUserPwd, Value: userPwd})
		kvs = append(kvs, bom.KeyValue{Key: helmUserName, Value: mySQLUsername})
		kvs, err = appendDatabaseInitializationValues(compContext, []byte(userPwd), kvs)
		if err != nil {
			return []bom.KeyValue{}, ctrlerrors.RetryableError{Source: ComponentName, Cause: err}
		}
		secretName := types.NamespacedName{
			Namespace: ComponentNamespace,
			Name:      rootSec,
		}
		kvs, err = appendMySQLSecret(compContext, secretName, mySQLRootKey, kvs)
		if err != nil {
			return []bom.KeyValue{}, ctrlerrors.RetryableError{Source: ComponentName, Cause: err}
		}
	}

	// Apply the version of the MySQL operator to the InnoDB cluster instance for Verrazzano Components
	mySQLVersion, err := getMySQLVersion(&bomFile)
	if err != nil {
		return kvs, err
	}
	kvs = append(kvs, mySQLVersion)

	// Apply overrides for which mysql-operator image to use in containers
	kvs, err = generateMySQLOperatorOverrides(&bomFile, kvs)
	if err != nil {
		return kvs, err
	}

	repositorySetting, err := getRegistrySettings(&bomFile)
	if err != nil {
		return kvs, err
	}
	kvs = append(kvs, repositorySetting)

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

func getRegistrySettings(bomFile *bom.Bom) (bom.KeyValue, error) {
	sc, err := bomFile.GetSubcomponent(bomSubComponentName)
	if err != nil {
		return bom.KeyValue{}, err
	}
	img, err := bomFile.FindImage(sc, mysqlServerImageName)
	if err != nil {
		return bom.KeyValue{}, err
	}
	resolvedRegistry := bomFile.ResolveRegistry(sc, img)
	resolvedRepo := bomFile.ResolveRepo(sc, img)
	return bom.KeyValue{Key: imageRepositoryKey, Value: fmt.Sprintf("%s/%s", resolvedRegistry, resolvedRepo)}, nil
}

func getMySQLVersion(bomFile *bom.Bom) (bom.KeyValue, error) {
	version, err := bomFile.GetComponentVersion(ComponentName)
	if err != nil {
		return bom.KeyValue{}, err
	}
	return bom.KeyValue{Key: serverVersionKey, Value: version}, nil
}

func generateMySQLOperatorOverrides(bomFile *bom.Bom, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	_, images, err := bomFile.BuildImageStrings(mysqloperator.ComponentName)
	if err != nil {
		return kvs, err
	}
	if len(images) != 1 {
		return kvs, fmt.Errorf("expected one image for %s, found %d", mysqloperator.ComponentName, len(images))
	}
	kvs = append(kvs, bom.KeyValue{Key: "mysqlOperator.image", Value: images[0]})
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
	var err error
	convertedVZ := v1beta1.Verrazzano{}
	if err = common.ConvertVerrazzanoCR(compContext.EffectiveCR(), &convertedVZ); err != nil {
		return nil, err
	}

	mySQLVolumeSource := getMySQLVolumeSource(&convertedVZ)
	if mySQLVolumeSource != nil && mySQLVolumeSource.EmptyDir != nil {
		compContext.Log().Info("EmptyDir currently not supported for MySQL server.  A default persistent volume will be used.")
	} else {
		kvs, err = doGenerateVolumeSourceOverrides(&convertedVZ, kvs)
		if err != nil {
			return kvs, err
		}
	}

	return kvs, nil
}

// doGenerateVolumeSourceOverrides generates the appropriate persistence overrides given the effective CR
func doGenerateVolumeSourceOverrides(effectiveCR *v1beta1.Verrazzano, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	mySQLVolumeSource := getMySQLVolumeSource(effectiveCR)

	if mySQLVolumeSource != nil && mySQLVolumeSource.PersistentVolumeClaim != nil {
		// Configured for persistence, adapt the PVC Spec template to the appropriate Helm args
		pvcs := mySQLVolumeSource.PersistentVolumeClaim
		storageSpec, found := vzconfig.FindVolumeTemplate(pvcs.ClaimName, effectiveCR)
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

// getMySQLVolumeSourceV1beta1 returns the volume source from v1beta1.Verrazzano
func getMySQLVolumeSource(effectiveCR *v1beta1.Verrazzano) *v1.VolumeSource {
	var mySQLVolumeSource *v1.VolumeSource
	if effectiveCR.Spec.Components.Keycloak != nil {
		mySQLVolumeSource = effectiveCR.Spec.Components.Keycloak.MySQL.VolumeSource
	}
	if mySQLVolumeSource == nil {
		mySQLVolumeSource = effectiveCR.Spec.DefaultVolumeSource
	}
	return mySQLVolumeSource
}

// appendCustomImageOverrides - Append the custom overrides for the busybox initContainer
func appendCustomImageOverrides(bomFile *bom.Bom, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	imageOverrides, err := bomFile.BuildImageOverrides(mysqlUpgradeSubComp)
	if err != nil {
		return kvs, ctrlerrors.RetryableError{Source: ComponentName, Cause: err}
	}

	kvs = append(kvs, imageOverrides...)
	return kvs, nil
}

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
	} else if effectiveCR, ok := object.(*v1beta1.Verrazzano); ok {
		if effectiveCR.Spec.Components.Keycloak != nil {
			return effectiveCR.Spec.Components.Keycloak.MySQL.ValueOverrides
		}
		return []v1beta1.Overrides{}
	}

	return []vzapi.Overrides{}
}

func appendMySQLSecret(compContext spi.ComponentContext, secretName types.NamespacedName, rootKey string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	rootSecret := &v1.Secret{}
	// use self-signed
	kvs = append(kvs, bom.KeyValue{
		Key:   "tls.useSelfSigned",
		Value: "true",
	})
	// Get the mysql root secret
	err := compContext.Client().Get(context.TODO(), secretName, rootSecret)
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
		return []bom.KeyValue{}, compContext.Log().ErrorfNewErr("Failed getting MySQL root secret: %v", err)
	}
	// Force mysql to use the initial password and root password during the upgrade or update, by specifying as helm overrides
	kvs = append(kvs, bom.KeyValue{
		Key:   helmRootPwd,
		Value: string(rootSecret.Data[rootKey]),
	})
	return kvs, nil
}

// preUpgrade handles the re-association of a previous MySQL deployment PV/PVC with the new MySQL statefulset (if needed)
func preUpgrade(ctx spi.ComponentContext) error {
	if ctx.IsDryRun() {
		ctx.Log().Debug("MySQL pre upgrade dry run")
		return nil
	}
	// vz > 1.3 uses statefulsets, not deployments
	// no migration is needed if vz >= 1.4
	if isLegacyDatabaseUpgrade(ctx) {
		if err := handleLegacyDatabasePreUpgrade(ctx); err != nil {
			return err
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
	// cleanup db migration job if it exists
	if err := cleanupDbMigrationJob(ctx); err != nil {
		return err
	}

	//delete db migration secret if it exists
	if err := deleteDbMigrationSecret(ctx); err != nil {
		return err
	}

	// grant XA_RECOVER_ADMIN to Keycloak user
	if err := grantXARecoverAdmin(ctx); err != nil {
		return err
	}

	return common.ResetVolumeReclaimPolicy(ctx, ComponentName)
}

// PostUpgradeCleanup - Clean up any remaining resources after a successful upgrade
func PostUpgradeCleanup(log vzlog.VerrazzanoLogger, client clipkg.Client) error {
	// Clean up the dump volume claim used during the database upgrade
	log.Progressf("MySQL post-upgrade cleanup")
	pvc := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      legacyDBDumpClaim,
			Namespace: ComponentNamespace,
		},
	}
	if err := client.Delete(context.TODO(), pvc); err != nil {
		if !errors.IsNotFound(err) {
			log.Progressf("Error deleting temporary database upgrade volume claim %v: %v", clipkg.ObjectKeyFromObject(pvc), err.Error())
			return err
		}
	}
	log.Oncef("Deleted temporary legacy database volume claim %v", clipkg.ObjectKeyFromObject(pvc))
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
func createMySQLInitFile(ctx spi.ComponentContext, userPwd []byte) (string, error) {
	file, err := os.CreateTemp(os.TempDir(), fmt.Sprintf("%s*.sh", mySQLInitFilePrefix))
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

// getRootDBPassword returns the password to access the DB as root
func getRootDBPassword(compContext spi.ComponentContext) ([]byte, error) {
	secretName := types.NamespacedName{
		Namespace: ComponentNamespace,
		Name:      rootSec,
	}
	dbSecret := &v1.Secret{}
	err := compContext.Client().Get(context.TODO(), secretName, dbSecret)
	if err != nil {
		return nil, err
	}
	return dbSecret.Data[mySQLRootKey], nil
}

// getOrCreateDBUserPassword creates or updates a secret containing the password used by keycloak to access the DB
func getOrCreateDBUserPassword(compContext spi.ComponentContext) (string, error) {
	secretName := types.NamespacedName{
		Namespace: ComponentNamespace,
		Name:      rootSec,
	}
	dbSecret := &v1.Secret{}
	err := compContext.Client().Get(context.TODO(), secretName, dbSecret)
	if err != nil {
		if errors.IsNotFound(err) {
			password, err := vzpassword.GeneratePassword(12)
			if err != nil {
				return "", err
			}
			return password, nil
		}
		return "", err
	}
	return string(dbSecret.Data[mySQLUserKey]), nil
}

func appendDatabaseInitializationValues(compContext spi.ComponentContext, userPwd []byte, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	compContext.Log().Debug("Adding database initialization values to MySQL helm values")
	mySQLInitFile, err := createMySQLInitFile(compContext, userPwd)
	if err != nil {
		return []bom.KeyValue{}, ctrlerrors.RetryableError{Source: ComponentName, Cause: err}
	}
	kvs = append(kvs, bom.KeyValue{Key: initdbScriptsFile, Value: mySQLInitFile, SetFile: true})
	kvs = append(kvs, bom.KeyValue{Key: backupHookScriptsFile, Value: mySQLHookFile, SetFile: true})

	return kvs, nil
}

// This is needed for unit testing
func initUnitTesting() {
	unitTesting = true
}

// postUninstall performs additional actions after the uninstall step
func (c mysqlComponent) postUninstall(ctx spi.ComponentContext) error {
	return mysqlcheck.RepairICStuckDeleting(ctx)
}

// grantXARecoverAdmin grants XA_RECOVER_ADMIN to Keycloak user
func grantXARecoverAdmin(ctx spi.ComponentContext) error {
	if unitTesting {
		return nil
	}
	// Get root password, required to connect to MySQL pod
	rootPassword, err := getRootDBPassword(ctx)
	if err != nil {
		return err
	}

	sqlCmd := fmt.Sprintf(mySQLGrantXAAdminCommand, rootPassword)
	execCmd := []string{"bash", "-c", sqlCmd}
	cfg, cli, err := k8sutil.ClientConfig()
	if err != nil {
		return err
	}
	mysqlPod := mySQLPod()

	// Run script defined by mySQLGrantXAAdminCommand from the MySQL pod as root user to grant XA_RECOVER_ADMIN
	_, _, err = k8sutil.ExecPodNoTty(cli, cfg, mysqlPod, mySQLContainerName, execCmd)
	if err != nil {
		errorMsg := maskPw(fmt.Sprintf("Failed granting XA_RECOVER_ADMIN, err = %v", err))
		ctx.Log().Error(errorMsg)
		return fmt.Errorf("error: %s", maskPw(err.Error()))
	}
	ctx.Log().Debug("Granted XA_RECOVER_ADMIN to Keycloak user successfully")
	return nil
}

func mySQLPod() *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mySQLPodName,
			Namespace: ComponentNamespace,
		},
	}
}
