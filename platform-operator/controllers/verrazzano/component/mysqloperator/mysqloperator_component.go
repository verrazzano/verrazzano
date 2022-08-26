// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mysqloperator

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	"k8s.io/apimachinery/pkg/runtime"
	"path/filepath"

	"github.com/verrazzano/verrazzano/pkg/constants"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vpocons "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/secret"
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
			ImagePullSecretKeyname:    secret.DefaultImagePullSecretKeyName,
			MinVerrazzanoVersion:      vpocons.VerrazzanoVersion1_4_0,
			ValuesFile:                filepath.Join(config.GetHelmOverridesDir(), "mysql-operator-values.yaml"),
			Dependencies:              []string{},
			GetInstallOverridesFunc:   getOverrides,
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

// ValidateInstall checks if the specified Verrazzano CR is valid for this component to be installed
func (c mysqlOperatorComponent) ValidateInstall(vz *vzapi.Verrazzano) error {
	return c.validateMySQLOperator(vz)
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (c mysqlOperatorComponent) ValidateUpdate(old *vzapi.Verrazzano, new *vzapi.Verrazzano) error {
	if c.IsEnabled(old) && !c.IsEnabled(new) {
		return fmt.Errorf("disabling component %s is not allowed", ComponentJSONName)
	}
	return c.validateMySQLOperator(new)
}
