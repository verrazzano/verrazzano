// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mysqloperator

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/mysql"
	"path/filepath"
	"strconv"

	ctrlerrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	controllerruntime "sigs.k8s.io/controller-runtime"

	"github.com/verrazzano/verrazzano/pkg/constants"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vpocons "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	// ComponentName is the name of the component
	ComponentName = "mysql-operator"

	// ComponentNamespace is the namespace of the component
	ComponentNamespace = constants.MySQLOperatorNamespace

	// ComponentJSONName is the json name of the component in the CRD
	ComponentJSONName = "mySQLOperator"
)

type mysqlOperatorComponent struct {
	helm.HelmComponent
}

func NewComponent() spi.Component {
	return mysqlOperatorComponent{
		helm.HelmComponent{
			ReleaseName:               ComponentName,
			JSONName:                  ComponentJSONName,
			ChartDir:                  filepath.Join(config.GetThirdPartyDir(), ComponentName),
			ChartNamespace:            ComponentNamespace,
			IgnoreNamespaceOverride:   true,
			SupportsOperatorInstall:   true,
			SupportsOperatorUninstall: true,
			ImagePullSecretKeyname:    "image.pullSecrets.secretName",
			MinVerrazzanoVersion:      vpocons.VerrazzanoVersion1_4_0,
			ValuesFile:                filepath.Join(config.GetHelmOverridesDir(), "mysql-operator-values.yaml"),
			AppendOverridesFunc:       AppendOverrides,
			Dependencies:              []string{},
			GetInstallOverridesFunc:   getOverrides,
			InstallBeforeUpgrade:      true,
		},
	}
}

// IsEnabled returns true if the component is enabled for install
func (c mysqlOperatorComponent) IsEnabled(effectiveCR runtime.Object) bool {
	return vzconfig.IsMySQLOperatorEnabled(effectiveCR)
}

// IsReady - component specific ready-check
func (c mysqlOperatorComponent) IsReady(context spi.ComponentContext) bool {
	if c.HelmComponent.IsReady(context) {
		return isReady(context)
	}
	return false
}

// IsInstalled returns true if the component is installed
func (c mysqlOperatorComponent) IsInstalled(ctx spi.ComponentContext) (bool, error) {
	return isInstalled(ctx), nil
}

// PreInstall runs before components are installed
func (c mysqlOperatorComponent) PreInstall(compContext spi.ComponentContext) error {
	cli := compContext.Client()
	log := compContext.Log()

	// create namespace
	ns := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ComponentNamespace}}
	if _, err := controllerutil.CreateOrUpdate(context.TODO(), cli, &ns, func() error {
		return nil
	}); err != nil {
		return log.ErrorfNewErr("Failed to create or update the %s namespace: %v", ComponentNamespace, err)
	}

	return nil
}

// PreUpgrade recycles the MySql operator pods to make sure the latest Istio sidecar exists.
// This needs to be done since MySQL operator is installed before upgrade
func (c mysqlOperatorComponent) PreUpgrade(compContext spi.ComponentContext) error {
	compContext.Log().Oncef("Restarting MySQL operator so that it picks up Istio proxy sidecar")
	// Annotate the deployment to cause the restart
	var deployment appsv1.Deployment
	deployment.Namespace = ComponentNamespace
	deployment.Name = ComponentName
	if _, err := controllerruntime.CreateOrUpdate(context.TODO(), compContext.Client(), &deployment, func() error {
		if deployment.Spec.Template.ObjectMeta.Annotations == nil {
			deployment.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
		}
		// Annotate using the generation so we don't restart twice
		deployment.Spec.Template.ObjectMeta.Annotations[constants.VerrazzanoRestartAnnotation] = strconv.Itoa(int(deployment.Generation))
		return nil
	}); err != nil {
		return ctrlerrors.RetryableError{Source: ComponentName, Cause: err}
	}
	return nil
}

// PreUninstall waits until MySQL has been uninstalled. This is needed since the pods have finalizers
// that are processed by the MySQL operator
func (c mysqlOperatorComponent) PreUninstall(compContext spi.ComponentContext) error {
	// Keycloak enabled determines if MySQL is enabled
	if !vzconfig.IsKeycloakEnabled(compContext.EffectiveCR()) {
		return nil
	}

	// Create a component context for mysql
	spiCtx, err := spi.NewContext(compContext.Log(), compContext.Client(), compContext.ActualCR(), compContext.ActualCRV1Beta1(), compContext.IsDryRun())
	if err != nil {
		spiCtx.Log().Errorf("Failed to create component context: %v", err)
		return err
	}
	mySQLContext := spiCtx.Init(mysql.ComponentName).Operation(vpocons.UninstallOperation)

	// Check if MySQL is installed
	mySQLComp := mysql.NewComponent()
	installed, err := mySQLComp.IsInstalled(mySQLContext)
	if err != nil {
		spiCtx.Log().Errorf("Failed to check if MySQL is installed: %v", err)
		return err
	}
	if installed {
		spiCtx.Log().Progressf("MySQL operator uninstall is waiting for MySQL to be uninstalled")
		return ctrlerrors.RetryableError{Source: ComponentName}
	}

	// MySQL is not installed, safe to uninstall MySQL operator
	return nil
}

// ValidateInstall checks if the specified Verrazzano CR is valid for this component to be installed
func (c mysqlOperatorComponent) ValidateInstall(vz *vzapi.Verrazzano) error {
	convertedVZ := v1beta1.Verrazzano{}
	if err := common.ConvertVerrazzanoCR(vz, &convertedVZ); err != nil {
		return err
	}
	return c.validateMySQLOperator(&convertedVZ)
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (c mysqlOperatorComponent) ValidateUpdate(old *vzapi.Verrazzano, new *vzapi.Verrazzano) error {
	if c.IsEnabled(old) && !c.IsEnabled(new) {
		return fmt.Errorf("disabling component %s is not allowed", ComponentJSONName)
	}
	convertedVZ := v1beta1.Verrazzano{}
	if err := common.ConvertVerrazzanoCR(new, &convertedVZ); err != nil {
		return err
	}
	return c.validateMySQLOperator(&convertedVZ)
}

// ValidateInstallV1Beta1 checks if the specified Verrazzano CR is valid for this component to be installed
func (c mysqlOperatorComponent) ValidateInstallV1Beta1(vz *v1beta1.Verrazzano) error {
	return c.validateMySQLOperator(vz)
}

// ValidateUpdateV1Beta1 checks if the specified new Verrazzano CR is valid for this component to be updated
func (c mysqlOperatorComponent) ValidateUpdateV1Beta1(old *v1beta1.Verrazzano, new *v1beta1.Verrazzano) error {
	if c.IsEnabled(old) && !c.IsEnabled(new) {
		return fmt.Errorf("disabling component %s is not allowed", ComponentJSONName)
	}
	return c.validateMySQLOperator(new)
}
