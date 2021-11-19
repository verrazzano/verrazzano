// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mysql

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/bom"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"os"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

// ComponentName is the name of the component
const (
	secretName    = "mysql"
	mysqlUsername = "keycloak"
	helmPwd       = "mysqlPassword"
	helmRootPwd   = "mysqlRootPassword"
	mysqlKey      = "mysql-password"
	mysqlRootKey  = "mysql-root-password"
	mysqlDBFile   = "create-mysql-db.sql"
)

func IsReady(context spi.ComponentContext, name string, namespace string) bool {
	deployments := []types.NamespacedName{
		{Name: name, Namespace: namespace},
	}
	return status.DeploymentsReady(context.Log(), context.Client(), deployments, 1)
}

// AppendMySQLOverrides appends the the password for database user and root user.
func AppendMySQLOverrides(compContext spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	cr := compContext.EffectiveCR()
	//secret := &v1.Secret{}
	//nsName := types.NamespacedName{
	//	Namespace: vzconst.KeycloakNamespace,
	//	Name:      secretName}
	//
	//err := compContext.Client().Get(context.TODO(), nsName, secret)
	//if err != nil {
	//	return []bom.KeyValue{}, err
	//}
	//
	//// Force mysql to use the initial password and root password during the upgrade, by specifying as helm overrides
	//kvs = append(kvs, bom.KeyValue{
	//	Key:   helmPwd,
	//	Value: string(secret.Data[mysqlKey]),
	//})
	//kvs = append(kvs, bom.KeyValue{
	//	Key:   helmRootPwd,
	//	Value: string(secret.Data[mysqlRootKey]),
	//})
	kvs = append(kvs, bom.KeyValue{Key: "mysqlUser", Value: mysqlUsername})
	err := createDBFile(compContext)
	if err != nil {
		return []bom.KeyValue{}, err
	}
	kvs = append(kvs, bom.KeyValue{Key: "initializationFiles.create-db\\.sql", Value: os.TempDir() + "/" + mysqlDBFile, SetFile: true})
	kvs, err = generateVolumeSourceOverrides(compContext, kvs)
	if err != nil {
		return []bom.KeyValue{}, err
	}
	// Convert NGINX install-args to helm overrides
	kvs = append(kvs, helm.GetInstallArgs(getInstallArgs(cr))...)

	return kvs, nil
}

// PreInstall Create and label the NGINX namespace, and create any override helm args needed
func PreInstall(compContext spi.ComponentContext, name string, namespace string, dir string) error {
	if compContext.IsDryRun() {
		compContext.Log().Infof("MySQL PostInstall dry run")
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
		return err
	}
	return nil
}

// PostInstall Patch the controller service ports based on any user-supplied overrides
func PostInstall(ctx spi.ComponentContext, _ string, _ string) error {
	if ctx.IsDryRun() {
		ctx.Log().Infof("MySQL PostInstall dry run")
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
		return err
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
		return err
	}

	// Close the file
	if err := tmpDBFile.Close(); err != nil {
		ctx.Log().Errorf("Failed to close temporary file: %v", err)
		return err
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
