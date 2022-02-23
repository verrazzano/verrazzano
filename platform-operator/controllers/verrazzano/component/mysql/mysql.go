// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mysql

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/verrazzano/verrazzano/pkg/bom"
	ctrlerrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
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
	secretName                  = "mysql"
	mySQLUsernameKey            = "mysqlUser"
	mySQLUsername               = "keycloak"
	helmPwd                     = "mysqlPassword"
	helmRootPwd                 = "mysqlRootPassword"
	mySQLKey                    = "mysql-password"
	mySQLRootKey                = "mysql-root-password"
	mySQLInitFilePrefix         = "init-mysql-"
	mySQLExtraInitFilePrefix    = "extra-init-mysql-"
	busyboxImageNameKey         = "busybox.image"
	busyboxImageTagKey          = "busybox.tag"
)

// Define the MySQL Key:Value pair for extra init container.
// We need to replace image using the real image in the bom
const mySQLInitContainerKey = "extraInitContainers"

// Add an init container to chown the data directory to be owned by the mysql user
// (uid=27 gid=27) so that when upgrading, mysql user can definitely access data dir
// from previous volume. The Helm chart expects this to be a STRING, so using a multi-line string here
// Note: if the Helm chart for MySQL changes, this should be reviewed for correctness
const mySQLInitContainerValueTemplate = `
    - command:
      - chown
      - -R
      - 27:27
      - /var/lib/mysql
    image: {{.Image}}
    imagePullPolicy: IfNotPresent
    name: chown-data-dir
    resources:
      requests:
        cpu: 10m
        memory: 10Mi
    terminationMessagePath: /dev/termination-log
    terminationMessagePolicy: File
    volumeMounts:
      - mountPath: /var/lib/mysql
        name: data
`

// imageData needed for template rendering
type imageData struct {
	Image string
}

// isMySQLReady checks to see if the MySQL component is in ready state
func isMySQLReady(context spi.ComponentContext) bool {
	deployments := []types.NamespacedName{
		{Name: ComponentName, Namespace: ComponentNamespace},
	}
	prefix := fmt.Sprintf("Component %s", context.GetComponent())
	return status.DeploymentsReady(context.Log(), context.Client(), deployments, 1, prefix)
}

// appendMySQLOverrides appends the MySQL helm overrides
func appendMySQLOverrides(compContext spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	cr := compContext.EffectiveCR()

	kvs, err := appendCustomImageOverrides(compContext, kvs)
	if err != nil {
		return kvs, ctrlerrors.RetryableError{Source: ComponentName, Cause: err}
	}

	if compContext.Init(ComponentName).GetOperation() == vzconst.UpgradeOperation {
		secret := &v1.Secret{}
		nsName := types.NamespacedName{
			Namespace: ComponentNamespace,
			Name:      secretName}
		// Get the mysql secret
		err := compContext.Client().Get(context.TODO(), nsName, secret)
		if err != nil {
			return []bom.KeyValue{}, compContext.Log().ErrorfNewErr("Failed getting MySQL secret: %v", err)
		}
		// Force mysql to use the initial password and root password during the upgrade, by specifying as helm overrides
		kvs = append(kvs, bom.KeyValue{
			Key:   helmRootPwd,
			Value: string(secret.Data[mySQLRootKey]),
		})
		kvs = append(kvs, bom.KeyValue{
			Key:   helmPwd,
			Value: string(secret.Data[mySQLKey]),
		})
	}

	kvs = append(kvs, bom.KeyValue{Key: mySQLUsernameKey, Value: mySQLUsername})

	if compContext.Init(ComponentName).GetOperation() == vzconst.InstallOperation {
		mySQLInitFile, err := createMySQLInitFile(compContext)
		if err != nil {
			return []bom.KeyValue{}, ctrlerrors.RetryableError{Source: ComponentName, Cause: err}
		}
		kvs = append(kvs, bom.KeyValue{Key: "initializationFiles.create-db\\.sql", Value: mySQLInitFile, SetFile: true})
	}

	// generate the MySQl PV overrides
	kvs, err = generateVolumeSourceOverrides(compContext, kvs)
	if err != nil {
		return []bom.KeyValue{}, ctrlerrors.RetryableError{Source: ComponentName, Cause: err}
	}

	// Convert MySQL install-args to helm overrides
	kvs = append(kvs, helm.GetInstallArgs(getInstallArgs(cr))...)

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
		ns.Labels["istio-injection"] = "enabled"
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
	removeExtraInitContainersFile(ctx)
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

// generateVolumeSourceOverrides generates the appropriate persistence overrides given the effective CR
func generateVolumeSourceOverrides(compContext spi.ComponentContext, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	effectiveCR := compContext.EffectiveCR()
	var mySQLVolumeSource *v1.VolumeSource
	if effectiveCR.Spec.Components.Keycloak != nil {
		mySQLVolumeSource = effectiveCR.Spec.Components.Keycloak.MySQL.VolumeSource
	}
	if mySQLVolumeSource == nil {
		mySQLVolumeSource = effectiveCR.Spec.DefaultVolumeSource
	}

	// No volumes to process, return what we have
	if mySQLVolumeSource == nil {
		return kvs, nil
	}

	if mySQLVolumeSource.EmptyDir != nil {
		// EmptyDir, disable persistence
		kvs = append(kvs, bom.KeyValue{
			Key:   "persistence.enabled",
			Value: "false",
		})
	} else if mySQLVolumeSource.PersistentVolumeClaim != nil {
		// Configured for persistence, adapt the PVC Spec template to the appropriate Helm args
		pvcs := mySQLVolumeSource.PersistentVolumeClaim
		storageSpec, found := vzconfig.FindVolumeTemplate(pvcs.ClaimName, effectiveCR.Spec.VolumeClaimSpecTemplates)
		if !found {
			return kvs, compContext.Log().ErrorfNewErr("Failed, No VolumeClaimTemplate found for %s", pvcs.ClaimName)
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

//appendCustomImageOverrides - Append the custom overrides for the busybox initContainer
func appendCustomImageOverrides(compContext spi.ComponentContext, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	bomFile, err := bom.NewBom(config.GetDefaultBOMFilePath())
	if err != nil {
		return kvs, ctrlerrors.RetryableError{Source: ComponentName, Cause: err}
	}

	imageOverrides, err := bomFile.BuildImageOverrides("oraclelinux")
	if err != nil {
		return kvs, ctrlerrors.RetryableError{Source: ComponentName, Cause: err}
	}

	if len(imageOverrides) < 2 {
		return nil, fmt.Errorf("Component MySQL failed, expected atleast 2 image override keys for oraclelinux, found %v", len(imageOverrides))
	}

	mySQLExtraInitFile, err := createExtraInitContainersFile(compContext, imageOverrides)
	if err != nil {
		return []bom.KeyValue{}, ctrlerrors.RetryableError{Source: ComponentName, Cause: err}
	}
	kvs = append(kvs, bom.KeyValue{Key: mySQLInitContainerKey, Value: mySQLExtraInitFile, SetFile: true})

	kvs = append(kvs, imageOverrides...)
	return kvs, nil
}

func getInitImage(imageOverrides []bom.KeyValue) string {
	busyboxImageName, busyboxImageTag := "", ""
	for _, imageOverride := range imageOverrides {
		if imageOverride.Key == busyboxImageNameKey {
			busyboxImageName = imageOverride.Value
		}
		if imageOverride.Key == busyboxImageTagKey {
			busyboxImageTag = imageOverride.Value
		}
	}
	return busyboxImageName + ":" + busyboxImageTag
}

func createExtraInitContainersFile(ctx spi.ComponentContext, imageOverrides []bom.KeyValue) (string, error) {
	file, err := os.CreateTemp(os.TempDir(), fmt.Sprintf("%s*.yaml", mySQLExtraInitFilePrefix))
	if err != nil {
		return "", err
	}

	// use template to get populate template with image:tag
	var b bytes.Buffer
	t, err := template.New("image").Parse(mySQLInitContainerValueTemplate)
	if err != nil {
		return "", err
	}

	// Render the template
	data := imageData{Image: getInitImage(imageOverrides)}
	err = t.Execute(&b, data)
	if err != nil {
		return "", err
	}

	_, err = file.Write(b.Bytes())
	if err != nil {
		return "", ctx.Log().ErrorfNewErr("Failed to write to temporary file: %v", err)
	}

	// TODO: troubleshooting - remove later
	//Print the contents of the file
	fileContents, err := ioutil.ReadFile(file.Name())
	if err != nil {
		return "", ctx.Log().ErrorfNewErr("Failed to read from temporary file: %v", err)
	}
	ctx.Log().Infof("MySQL extra init file contents: %s", string(fileContents))

	// Close the file
	if err := file.Close(); err != nil {
		return "", ctx.Log().ErrorfNewErr("Failed to close temporary file: %v", err)
	}
	return file.Name(), nil
}

func removeExtraInitContainersFile(ctx spi.ComponentContext) {
	files, err := ioutil.ReadDir(os.TempDir())
	if err != nil {
		ctx.Log().Errorf("Failed reading temp directory: %v", err)
	}
	for _, file := range files {
		if !file.IsDir() && strings.HasPrefix(file.Name(), mySQLExtraInitFilePrefix) && strings.HasSuffix(file.Name(), ".yaml") {
			fullPath := filepath.Join(os.TempDir(), file.Name())
			ctx.Log().Debugf("Deleting temp MySQL extra init containers file %s", fullPath)
			if err := os.Remove(fullPath); err != nil {
				ctx.Log().Errorf("Failed deleting temp MySQL extra init containers file %s", fullPath)
			}
		}
	}
}

// getInstallArgs get the install args for MySQL
func getInstallArgs(cr *vzapi.Verrazzano) []vzapi.InstallArgs {
	if cr.Spec.Components.Keycloak == nil {
		return []vzapi.InstallArgs{}
	}
	return cr.Spec.Components.Keycloak.MySQL.MySQLInstallArgs
}
