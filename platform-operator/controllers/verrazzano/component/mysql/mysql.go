// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mysql

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/security/password"
	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"strings"

	"github.com/verrazzano/verrazzano/pkg/bom"
	ctrlerrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

const (
	rootSec              = "mysql-cluster-secret"
	helmRootPwd          = "credentials.root.password" //nolint:gosec //#gosec G101
	mySQLRootKey         = "rootPassword"
	statefulsetClaimName = "data-mysql-0"
)

// map of MySQL helm persistence values from previous version to existing version.  Other values will be ignored since
// they have no equivalents in the current charts
var helmValuesMap = map[string]string{
	"persistence.accessModes":  "datadirVolumeClaimTemplate.accessModes",
	"persistence.size":         "datadirVolumeClaimTemplate.resources.requests.storage",
	"persistence.storageClass": "datadirVolumeClaimTemplate.storageClassName",
}

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

	kvs, err := appendMySQLSecret(compContext, kvs)
	if err != nil {
		return []bom.KeyValue{}, err
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

	if compContext.Init(ComponentName).GetOperation() == vzconst.UpgradeOperation {
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
			return kvs, fmt.Errorf("Failed, No VolumeClaimTemplate found for %s", pvcs.ClaimName)
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
			rootPwd, err := password.GeneratePassword(12)
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
		newValueKey, ok := helmValuesMap[kv.Key]
		if ok {
			kvs[i].Key = newValueKey
		}
	}
	return kvs
}
