// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mysql

import (
	"context"
	"fmt"
	"io/ioutil"

	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"

	"os"
	"path/filepath"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"

	"github.com/verrazzano/verrazzano/pkg/bom"
	ctrlerrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

const (
	secretName            = "mysql"
	mySQLUsernameKey      = "auth.username"
	mySQLUsername         = "keycloak"
	helmPwd               = "auth.password"     //nolint:gosec //#gosec G101
	helmRootPwd           = "auth.rootPassword" //nolint:gosec //#gosec G101
	helmCreateDb          = "auth.createDatabase"
	helmDatabase          = "auth.database"
	mySQLKey              = "mysql-password"
	mySQLRootKey          = "mysql-root-password"
	mySQLInitFilePrefix   = "init-mysql-"
	initdbScriptsFile     = "initdbScripts.create-db\\.sql"
	backupHookScriptsFile = "configurationFiles.mysql-hook\\.sh"
	persistenceEnabledKey = "primary.persistence.enabled"
	statefulsetClaimName  = "data-mysql-0"
	mySQLHookFile         = "platform-operator/scripts/hooks/mysql-hook.sh"
)

// isMySQLReady checks to see if the MySQL component is in ready state
func isMySQLReady(context spi.ComponentContext) bool {
	statefulset := []types.NamespacedName{
		{
			Name:      ComponentName,
			Namespace: ComponentNamespace,
		},
	}
	prefix := fmt.Sprintf("Component %s", context.GetComponent())
	return status.StatefulSetsAreReady(context.Log(), context.Client(), statefulset, 1, prefix)
}

// appendMySQLOverrides appends the MySQL helm overrides
func appendMySQLOverrides(compContext spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	cr := compContext.EffectiveCR()

	kvs, err := appendCustomImageOverrides(kvs)
	if err != nil {
		return kvs, ctrlerrors.RetryableError{Source: ComponentName, Cause: err}
	}

	if compContext.Init(ComponentName).GetOperation() == vzconst.UpgradeOperation {
		// create the ini file if the former MySQL deployment exists (rather than a stateful set) and there is ephemeral storage
		deployment := &appsv1.Deployment{}

		if err := compContext.Client().Get(context.TODO(), types.NamespacedName{Namespace: ComponentNamespace, Name: ComponentName}, deployment); err != nil {
			if errors.IsNotFound(err) {
				compContext.Log().Debugf("Deployment does not exist.  No need to intialize db")
			}
		}

		if deployment != nil {
			mySQLVolumeSource := getMySQLVolumeSource(compContext.EffectiveCR())
			// check for ephemeral storage
			if mySQLVolumeSource != nil && mySQLVolumeSource.EmptyDir != nil {
				// we are in the process of upgrading from a MySQL deployment using ephemeral storage, so we need to
				// provide the sql initialization file
				mySQLInitFile, err := createMySQLInitFile(compContext)
				if err != nil {
					return []bom.KeyValue{}, ctrlerrors.RetryableError{Source: ComponentName, Cause: err}
				}
				kvs = append(kvs, bom.KeyValue{Key: initdbScriptsFile, Value: mySQLInitFile, SetFile: true})
				kvs = append(kvs, bom.KeyValue{Key: backupHookScriptsFile, Value: mySQLHookFile, SetFile: true})
			}
		}

		kvs, err = appendMySQLSecret(compContext, kvs)
		if err != nil {
			return []bom.KeyValue{}, err
		}
	}

	kvs = append(kvs, bom.KeyValue{Key: mySQLUsernameKey, Value: mySQLUsername})

	if compContext.Init(ComponentName).GetOperation() == vzconst.InstallOperation {
		mySQLInitFile, err := createMySQLInitFile(compContext)
		if err != nil {
			return []bom.KeyValue{}, ctrlerrors.RetryableError{Source: ComponentName, Cause: err}
		}
		kvs = append(kvs, bom.KeyValue{Key: initdbScriptsFile, Value: mySQLInitFile, SetFile: true})
		kvs = append(kvs, bom.KeyValue{Key: backupHookScriptsFile, Value: mySQLHookFile, SetFile: true})
		kvs, err = appendMySQLSecret(compContext, kvs)
		if err != nil {
			return []bom.KeyValue{}, ctrlerrors.RetryableError{Source: ComponentName, Cause: err}
		}
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
	removeMySQLInitFile(ctx)
	return nil
}

// createMySQLInitFile creates the .sql file that gets passed to helm as an override
// this initializes the MySQL DB
func createMySQLInitFile(ctx spi.ComponentContext) (string, error) {
	file, err := os.CreateTemp(os.TempDir(), fmt.Sprintf("%s*.sql", mySQLInitFilePrefix))
	if err != nil {
		return "", err
	}
	_, err = file.Write([]byte(fmt.Sprintf(
		"CREATE DATABASE IF NOT EXISTS keycloak DEFAULT CHARACTER SET utf8 DEFAULT COLLATE utf8_general_ci;"+
			"USE keycloak;"+
			"GRANT CREATE, ALTER, DROP, INDEX, REFERENCES, SELECT, INSERT, UPDATE, DELETE ON keycloak.* TO '%s'@'%%';"+
			"FLUSH PRIVILEGES;",
		mySQLUsername,
	)))
	if err != nil {
		return "", ctx.Log().ErrorfNewErr("Failed to write to temporary file: %v", err)
	}
	// Close the file
	if err := file.Close(); err != nil {
		return "", ctx.Log().ErrorfNewErr("Failed to close temporary file: %v", err)
	}
	return file.Name(), nil
}

// removeMySQLInitFile removes any files from the OS temp dir that match the pattern of the MySQL init file
func removeMySQLInitFile(ctx spi.ComponentContext) {
	files, err := ioutil.ReadDir(os.TempDir())
	if err != nil {
		ctx.Log().Errorf("Failed reading temp directory: %v", err)
	}
	for _, file := range files {
		if !file.IsDir() && strings.HasPrefix(file.Name(), mySQLInitFilePrefix) && strings.HasSuffix(file.Name(), ".sql") {
			fullPath := filepath.Join(os.TempDir(), file.Name())
			ctx.Log().Debugf("Deleting temp MySQL init file %s", fullPath)
			if err := os.Remove(fullPath); err != nil {
				ctx.Log().Errorf("Failed deleting temp MySQL init file %s", fullPath)
			}
		}
	}
}

// generateVolumeSourceOverrides generates the appropriate persistence overrides given the component context
func generateVolumeSourceOverrides(compContext spi.ComponentContext, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	kvs, err := doGenerateVolumeSourceOverrides(compContext.EffectiveCR(), kvs)
	if err != nil {
		return kvs, err
	}

	pvList, err := common.GetPersistentVolumes(compContext, ComponentName)
	if err != nil {
		return kvs, err
	}
	if len(pvList.Items) > 0 {
		// need to use existing claim
		compContext.Log().Infof("Using existing PVC for MySQL persistence")
		kvs = append(kvs, bom.KeyValue{
			Key:       "primary.persistence.existingClaim",
			Value:     statefulsetClaimName,
			SetString: true,
		})
	}

	return kvs, err
}

// doGenerateVolumeSourceOverrides generates the appropriate persistence overrides given the effective CR
func doGenerateVolumeSourceOverrides(effectiveCR *vzapi.Verrazzano, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	mySQLVolumeSource := getMySQLVolumeSource(effectiveCR)
	// No volumes to process, return what we have
	if mySQLVolumeSource == nil {
		return kvs, nil
	}

	if mySQLVolumeSource.EmptyDir != nil {
		// EmptyDir, disable persistence
		kvs = append(kvs, bom.KeyValue{
			Key:   persistenceEnabledKey,
			Value: "false",
		})
	} else if mySQLVolumeSource.PersistentVolumeClaim != nil {
		// Configured for persistence, adapt the PVC Spec template to the appropriate Helm args
		pvcs := mySQLVolumeSource.PersistentVolumeClaim
		storageSpec, found := vzconfig.FindVolumeTemplate(pvcs.ClaimName, effectiveCR.Spec.VolumeClaimSpecTemplates)
		if !found {
			return kvs, fmt.Errorf("Failed, No VolumeClaimTemplate found for %s", pvcs.ClaimName)
		}
		storageClass := storageSpec.StorageClassName
		if storageClass != nil && len(*storageClass) > 0 {
			kvs = append(kvs, bom.KeyValue{
				Key:       "primary.persistence.storageClass",
				Value:     *storageClass,
				SetString: true,
			})
		}
		storage := storageSpec.Resources.Requests.Storage()
		if storageSpec.Resources.Requests != nil && !storage.IsZero() {
			kvs = append(kvs, bom.KeyValue{
				Key:       "primary.persistence.size",
				Value:     storage.String(),
				SetString: true,
			})
		}
		accessModes := storageSpec.AccessModes
		if len(accessModes) > 0 {
			// MySQL only allows a single AccessMode value, so just choose the first
			kvs = append(kvs, bom.KeyValue{
				Key:       "primary.persistence.accessMode",
				Value:     string(accessModes[0]),
				SetString: true,
			})
		}
		// Enable MySQL persistence
		kvs = append(kvs, bom.KeyValue{
			Key:   persistenceEnabledKey,
			Value: "true",
		})
	}
	return kvs, nil
}

// doGenerateVolumeSourceOverridesV1beta1 generates the appropriate persistence overrides given the effective CR
func doGenerateVolumeSourceOverridesV1beta1(effectiveCR *v1beta1.Verrazzano, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	mySQLVolumeSource := getMySQLVolumeSourceV1beta1(effectiveCR)
	// No volumes to process, return what we have
	if mySQLVolumeSource == nil {
		return kvs, nil
	}

	if mySQLVolumeSource.EmptyDir != nil {
		// EmptyDir, disable persistence
		kvs = append(kvs, bom.KeyValue{
			Key:   persistenceEnabledKey,
			Value: "false",
		})
	} else if mySQLVolumeSource.PersistentVolumeClaim != nil {
		// Configured for persistence, adapt the PVC Spec template to the appropriate Helm args
		pvcs := mySQLVolumeSource.PersistentVolumeClaim
		storageSpec, found := vzconfig.FindVolumeTemplateV1beta1(pvcs.ClaimName, effectiveCR.Spec.VolumeClaimSpecTemplates)
		if !found {
			return kvs, fmt.Errorf("Failed, No VolumeClaimTemplate found for %s", pvcs.ClaimName)
		}
		storageClass := storageSpec.StorageClassName
		if storageClass != nil && len(*storageClass) > 0 {
			kvs = append(kvs, bom.KeyValue{
				Key:       "primary.persistence.storageClass",
				Value:     *storageClass,
				SetString: true,
			})
		}
		storage := storageSpec.Resources.Requests.Storage()
		if storageSpec.Resources.Requests != nil && !storage.IsZero() {
			kvs = append(kvs, bom.KeyValue{
				Key:       "primary.persistence.size",
				Value:     storage.String(),
				SetString: true,
			})
		}
		accessModes := storageSpec.AccessModes
		if len(accessModes) > 0 {
			// MySQL only allows a single AccessMode value, so just choose the first
			kvs = append(kvs, bom.KeyValue{
				Key:       "primary.persistence.accessMode",
				Value:     string(accessModes[0]),
				SetString: true,
			})
		}
		// Enable MySQL persistence
		kvs = append(kvs, bom.KeyValue{
			Key:   persistenceEnabledKey,
			Value: "true",
		})
	}
	return kvs, nil
}

// getMySQLVolumeSourceV1beta1 returns the volume source from v1beta1.Verrazzano
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

// getMySQLVolumeSourceV1beta1 returns the volume source from v1beta1.Verrazzano
func getMySQLVolumeSourceV1beta1(effectiveCR *v1beta1.Verrazzano) *v1.VolumeSource {
	var mySQLVolumeSource *v1.VolumeSource
	if effectiveCR.Spec.Components.Keycloak != nil {
		mySQLVolumeSource = effectiveCR.Spec.Components.Keycloak.MySQL.VolumeSource
	}
	if mySQLVolumeSource == nil {
		mySQLVolumeSource = effectiveCR.Spec.DefaultVolumeSource
	}
	return mySQLVolumeSource
}

//appendCustomImageOverrides - Append the custom overrides for the busybox initContainer
func appendCustomImageOverrides(kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	bomFile, err := bom.NewBom(config.GetDefaultBOMFilePath())
	if err != nil {
		return kvs, ctrlerrors.RetryableError{Source: ComponentName, Cause: err}
	}

	imageOverrides, err := bomFile.BuildImageOverrides("mysql")
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
	} else if effectiveCR, ok := object.(*installv1beta1.Verrazzano); ok {
		if effectiveCR.Spec.Components.Keycloak != nil {
			return effectiveCR.Spec.Components.Keycloak.MySQL.ValueOverrides
		}
		return []installv1beta1.Overrides{}
	}

	return []vzapi.Overrides{}
}

func appendMySQLSecret(compContext spi.ComponentContext, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	secret := &v1.Secret{}
	nsName := types.NamespacedName{
		Namespace: ComponentNamespace,
		Name:      secretName}
	// Get the mysql secret
	err := compContext.Client().Get(context.TODO(), nsName, secret)
	if err != nil {
		// A secret is not expected to be found the first time around (i.e. it's an install and not an update scenario).
		// So do not return an error in this case.
		if errors.IsNotFound(err) && compContext.Init(ComponentName).GetOperation() == vzconst.InstallOperation {
			return kvs, nil
		}
		// Return an error for upgrade or update
		return []bom.KeyValue{}, compContext.Log().ErrorfNewErr("Failed getting MySQL secret: %v", err)
	}
	// Force mysql to use the initial password and root password during the upgrade or update, by specifying as helm overrides
	kvs = append(kvs, bom.KeyValue{
		Key:   helmRootPwd,
		Value: string(secret.Data[mySQLRootKey]),
	})
	kvs = append(kvs, bom.KeyValue{
		Key:   helmPwd,
		Value: string(secret.Data[mySQLKey]),
	})
	kvs = append(kvs, bom.KeyValue{
		Key:   helmCreateDb,
		Value: "false",
	})
	kvs = append(kvs, bom.KeyValue{
		Key:   helmDatabase,
		Value: "",
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
		deploymentPvc := types.NamespacedName{Namespace: ComponentNamespace, Name: DeploymentPersistentVolumeClaim}
		err := common.RetainPersistentVolume(ctx, deploymentPvc, ComponentName)
		if err != nil {
			return err
		}

		// get the current MySQL deployment
		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ComponentName,
				Namespace: ComponentNamespace,
			},
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

		ctx.Log().Debugf("Deleting PVC %v", deploymentPvc)
		if err := common.DeleteExistingVolumeClaim(ctx, deploymentPvc); err != nil {
			ctx.Log().Debugf("Unable to delete existing PVC %v", deploymentPvc)
			return err
		}

		ctx.Log().Debugf("Updating PV/PVC %v", deploymentPvc)
		if err := common.UpdateExistingVolumeClaims(ctx, deploymentPvc, StatefulsetPersistentVolumeClaim, ComponentName); err != nil {
			ctx.Log().Debugf("Unable to update PV/PVC")
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
	mySQLVolumeSource := getMySQLVolumeSource(ctx.EffectiveCR())
	if mySQLVolumeSource != nil && mySQLVolumeSource.PersistentVolumeClaim != nil {
		return common.ResetVolumeReclaimPolicy(ctx, ComponentName)
	}
	return nil
}

// convertOldInstallArgs changes persistence.* install args to primary.persistence.* to keep compatibility with the new chart
func convertOldInstallArgs(kvs []bom.KeyValue) []bom.KeyValue {
	for i, kv := range kvs {
		if strings.HasPrefix(kv.Key, "persistence") {
			kvs[i].Key = strings.Replace(kv.Key, "persistence", "primary.persistence", 1)
		}
	}
	return kvs
}
