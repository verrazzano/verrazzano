// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package verrazzano

import (
	"k8s.io/apimachinery/pkg/runtime"
	"reflect"
	"strconv"

	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
)

// upgradeFuncSig is a function needed for unit test override
type upgradeFuncSig func(ctx spi.ComponentContext) error

// installFuncSig is a function needed for unit test override
type installFuncSig func(ctx spi.ComponentContext) error

// isInstalledFuncSig is a function needed for unit test override
type isInstalledFuncSig func(ctx spi.ComponentContext) (bool, error)

// fakeComponent allows for using dummy Component implementations for controller testing
type fakeComponent struct {
	helm.HelmComponent

	upgradeFunc     upgradeFuncSig
	installFunc     installFuncSig
	isInstalledFunc isInstalledFuncSig
	installed       string `default:"true"`
	ready           string `default:"true"`
	enabled         string `default:"true"`
	monitorChanges  string `default:"true"`
	minVersion      string
}

func (f fakeComponent) Name() string {
	return f.ReleaseName
}

func (f fakeComponent) GetSkipUpgrade() bool {
	return f.SkipUpgrade
}

func (f fakeComponent) IsOperatorInstallSupported() bool {
	return f.SupportsOperatorInstall
}

func (f fakeComponent) IsOperatorUninstallSupported() bool {
	return f.SupportsOperatorUninstall
}

func (f fakeComponent) GetDependencies() []string {
	return f.Dependencies
}

func (f fakeComponent) PreUpgrade(ctx spi.ComponentContext) error {
	if f.PreUpgradeFunc != nil {
		return f.PreUpgrade(ctx)
	}
	return nil
}

func (f fakeComponent) Upgrade(ctx spi.ComponentContext) error {
	if f.upgradeFunc != nil {
		return f.upgradeFunc(ctx)
	}
	return nil
}

func (f fakeComponent) PostUpgrade(_ spi.ComponentContext) error {
	return nil
}

func (f fakeComponent) PreInstall(ctx spi.ComponentContext) error {
	if f.PreInstallFunc != nil {
		return f.PreInstallFunc(ctx, f.ReleaseName, f.ChartNamespace, f.ChartDir)
	}
	return nil
}

func (f fakeComponent) Install(ctx spi.ComponentContext) error {
	if f.installFunc != nil {
		return f.installFunc(ctx)
	}
	return nil
}

func (f fakeComponent) PostInstall(ctx spi.ComponentContext) error {
	if f.PostInstallFunc != nil {
		return f.PostInstallFunc(ctx, f.ReleaseName, f.ChartNamespace)
	}
	return nil
}

func (f fakeComponent) IsInstalled(ctx spi.ComponentContext) (bool, error) {
	if f.isInstalledFunc != nil {
		return f.isInstalledFunc(ctx)
	}
	return getBool(f.installed, "installed"), nil
}

func (f fakeComponent) IsReady(x spi.ComponentContext) bool {
	return getBool(f.ready, "ready")
}

func (f fakeComponent) IsEnabled(effectiveCR runtime.Object) bool {
	return getBool(f.enabled, "enabled")
}

func (f fakeComponent) GetMinVerrazzanoVersion() string {
	if len(f.minVersion) > 0 {
		return f.minVersion
	}
	return constants.VerrazzanoVersion1_0_0
}

func (f fakeComponent) MonitorOverrides(ctx spi.ComponentContext) bool {
	return getBool(f.monitorChanges, "monitorChanges")
}

// getBool implements defaults for boolean fields
func getBool(val string, fieldName string) bool {
	typ := reflect.TypeOf(fakeComponent{})
	// checking if null string
	if val == "" {
		// returns the struct field
		// with the given parameter "name"
		f, _ := typ.FieldByName(fieldName)

		// returns the value associated
		// with key in the tag string
		// and returns empty string if
		// no such key in tag
		val = f.Tag.Get("default")
	}
	boolVal, _ := strconv.ParseBool(val)
	return boolVal
}
