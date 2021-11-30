// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mysql

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/bom"
	ctrlerrrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	helmutil "github.com/verrazzano/verrazzano/platform-operator/internal/helm"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"os"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

const (
	secretName       = "mysql"
	mysqlUsernameKey = "mysqlUser"
	mysqlUsername    = "keycloak"
	helmPwd          = "mysqlPassword"
	helmRootPwd      = "mysqlRootPassword"
	mysqlKey         = "mysql-password"
	mysqlRootKey     = "mysql-root-password"
	mysqlDBFile      = "create-mysql-db.sql"
)

var pvc100Gi, _ = resource.ParseQuantity("100Gi")

func isReady(context spi.ComponentContext) bool {
	deployments := []types.NamespacedName{
		{Name: ComponentName, Namespace: vzconst.KeycloakNamespace},
	}
	return status.DeploymentsReady(context.Log(), context.Client(), deployments, 1)
}

func isEnabled(context spi.ComponentContext) bool {
	keycloak := context.EffectiveCR().Spec.Components.Keycloak
	if keycloak != nil && keycloak.Enabled != nil {
		return *keycloak.Enabled
	}
	return false
}

// appendMySQLOverrides appends the the password for database user and root user.
func appendMySQLOverrides(compContext spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	cr := compContext.EffectiveCR()
	deployed, err := helmutil.IsReleaseDeployed(ComponentName, vzconst.KeycloakNamespace)
	if err != nil {
		return []bom.KeyValue{}, ctrlerrrors.RetryableError{Source: ComponentName, Cause: err}
	}
	if deployed {
		secret := &v1.Secret{}
		nsName := types.NamespacedName{
			Namespace: vzconst.KeycloakNamespace,
			Name:      secretName}

		err = compContext.Client().Get(context.TODO(), nsName, secret)
		if err != nil {
			return []bom.KeyValue{}, ctrlerrrors.RetryableError{Source: ComponentName, Cause: err}
		}

		// Force mysql to use the initial password and root password during the upgrade, by specifying as helm overrides
		kvs = append(kvs, bom.KeyValue{
			Key:   helmRootPwd,
			Value: string(secret.Data[mysqlRootKey]),
		})
		kvs = append(kvs, bom.KeyValue{
			Key:   helmPwd,
			Value: string(secret.Data[mysqlKey]),
		})
	}
	kvs = append(kvs, bom.KeyValue{Key: mysqlUsernameKey, Value: mysqlUsername})
	if !deployed {
		err = createDBFile(compContext)
		if err != nil {
			return []bom.KeyValue{}, ctrlerrrors.RetryableError{Source: ComponentName, Cause: err}
		}
		kvs = append(kvs, bom.KeyValue{Key: "initializationFiles.create-db\\.sql", Value: os.TempDir() + "/" + mysqlDBFile, SetFile: true})
	}
	kvs, err = generateVolumeSourceOverrides(compContext, kvs)
	if err != nil {
		return []bom.KeyValue{}, ctrlerrrors.RetryableError{Source: ComponentName, Cause: err}
	}
	// Convert NGINX install-args to helm overrides
	kvs = append(kvs, helm.GetInstallArgs(getInstallArgs(cr))...)

	return kvs, nil
}

// preInstall Create and label the MySQL namespace, and create any override helm args needed
func preInstall(compContext spi.ComponentContext, namespace string) error {
	if compContext.IsDryRun() {
		compContext.Log().Infof("MySQL postInstall dry run")
		return nil
	}
	compContext.Log().Infof("Adding label needed by network policies to %s namespace", namespace)
	ns := v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
	if _, err := controllerruntime.CreateOrUpdate(context.TODO(), compContext.Client(), &ns, func() error {
		if ns.Labels == nil {
			ns.Labels = make(map[string]string)
		}
		ns.Labels["verrazzano.io/namespace"] = namespace
		ns.Labels["istio-injection"] = "enabled"
		return nil
	}); err != nil {
		return ctrlerrrors.RetryableError{Source: ComponentName, Cause: err}
	}
	return nil
}

// postInstall Patch the controller service ports based on any user-supplied overrides
func postInstall(ctx spi.ComponentContext) error {
	if ctx.IsDryRun() {
		ctx.Log().Infof("MySQL postInstall dry run")
		return nil
	}
	// Delete create-mysql-db.sql after install
	return os.Remove(os.TempDir() + "/" + mysqlDBFile)
}

func createDBFile(ctx spi.ComponentContext) error {
	fmt.Println(os.Getwd())
	tmpDBFile, err := os.Create(os.TempDir() + "/" + mysqlDBFile)
	if err != nil {
		ctx.Log().Errorf("Failed to create temporary MySQL DB file: %v", err)
		return ctrlerrrors.RetryableError{Source: ComponentName, Cause: err}
	}

	_, err = tmpDBFile.Write([]byte(fmt.Sprintf(
		"CREATE DATABASE IF NOT EXISTS keycloak DEFAULT CHARACTER SET utf8 DEFAULT COLLATE utf8_general_ci;"+
			"USE keycloak;"+
			"GRANT CREATE, ALTER, DROP, INDEX, REFERENCES, SELECT, INSERT, UPDATE, DELETE ON keycloak.* TO '%s'@'%%';"+
			"FLUSH PRIVILEGES;",
		mysqlUsername,
	)))
	if err != nil {
		ctx.Log().Errorf("Failed to write to temporary file: %v", err)
		return ctrlerrrors.RetryableError{Source: ComponentName, Cause: err}
	}

	// Close the file
	if err := tmpDBFile.Close(); err != nil {
		ctx.Log().Errorf("Failed to close temporary file: %v", err)
		return ctrlerrrors.RetryableError{Source: ComponentName, Cause: err}
	}
	return nil
}

func generateVolumeSourceOverrides(compContext spi.ComponentContext, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	effectiveCR := compContext.EffectiveCR()
	defaultVolumeSpec := effectiveCR.Spec.DefaultVolumeSource

	// keycloak was not specified in CR so return defaults
	if effectiveCR.Spec.Components.Keycloak == nil {
		if defaultVolumeSpec != nil && defaultVolumeSpec.EmptyDir != nil {
			kvs = append(kvs, bom.KeyValue{
				Key:   "persistence.enabled",
				Value: "false",
			})
		}
		return kvs, nil
	}

	// Use a volume source specified in the Keycloak config, otherwise use the default spec
	mysqlVolumeSource := effectiveCR.Spec.Components.Keycloak.MySQL.VolumeSource
	if mysqlVolumeSource == nil {
		mysqlVolumeSource = defaultVolumeSpec
	}

	// No volumes to process, return what we have
	if mysqlVolumeSource == nil {
		return kvs, nil
	}

	if mysqlVolumeSource.EmptyDir != nil {
		// EmptyDir, disable persistence
		kvs = append(kvs, bom.KeyValue{
			Key:   "persistence.enabled",
			Value: "false",
		})
	} else if mysqlVolumeSource.PersistentVolumeClaim != nil {
		// Configured for persistence, adapt the PVC Spec template to the appropriate Helm args
		pvcs := mysqlVolumeSource.PersistentVolumeClaim
		storageSpec, found := findVolumeTemplate(pvcs.ClaimName, effectiveCR.Spec.VolumeClaimSpecTemplates)
		if !found {
			err := fmt.Errorf("No VolumeClaimTemplate found for %s", pvcs.ClaimName)
			return kvs, err
		}
		storageClass := storageSpec.StorageClassName
		if storageClass != nil && len(*storageClass) > 0 {
			kvs = append(kvs, bom.KeyValue{
				Key:       "persistence.storageClass",
				Value:     *storageClass,
				SetString: true,
			})
		}
		storage := storageSpec.Resources.Requests.Storage()
		if storageSpec.Resources.Requests != nil && !storage.IsZero() {
			kvs = append(kvs, bom.KeyValue{
				Key:       "persistence.size",
				Value:     storage.String(),
				SetString: true,
			})
		}
		accessModes := storageSpec.AccessModes
		if len(accessModes) > 0 {
			// MySQL only allows a single AccessMode value, so just choose the first
			kvs = append(kvs, bom.KeyValue{
				Key:       "persistence.accessMode",
				Value:     string(accessModes[0]),
				SetString: true,
			})
		}
		// Enable MySQL persistence
		kvs = append(kvs, bom.KeyValue{
			Key:   "persistence.enabled",
			Value: "true",
		})
	}
	return kvs, nil
}

// findVolumeTemplate Find a named VolumeClaimTemplate in the list
func findVolumeTemplate(templateName string, templates []vzapi.VolumeClaimSpecTemplate) (*v1.PersistentVolumeClaimSpec, bool) {
	for i, template := range templates {
		if templateName == template.Name {
			return &templates[i].Spec, true
		}
	}
	return nil, false
}

// getInstallArgs get the install args for MySQL
func getInstallArgs(cr *vzapi.Verrazzano) []vzapi.InstallArgs {
	if cr.Spec.Components.Keycloak == nil {
		return []vzapi.InstallArgs{}
	}
	return cr.Spec.Components.Keycloak.MySQL.MySQLInstallArgs
}
