// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package verrazzano

import (
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"reflect"
	"strconv"
)

// upgradeFuncSig is a function needed for unit test override
type upgradeFuncSig func(ctx spi.ComponentContext) error

// installFuncSig is a function needed for unit test override
type installFuncSig func(ctx spi.ComponentContext) error

// fakeComponent allows for using dummy Component implementations for controller testing
type fakeComponent struct {
	helm.HelmComponent

	upgradeFunc upgradeFuncSig
	installFunc installFuncSig
	installed   string `default:"true"`
	ready       string `default:"true"`
	enabled     string `default:"true"`
	minVersion  string
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

func (f fakeComponent) IsInstalled(_ spi.ComponentContext) (bool, error) {
	return getBool(f.installed, "installed"), nil
}

func (f fakeComponent) IsReady(x spi.ComponentContext) bool {
	if f.ReadyStatusFunc != nil {
		return f.ReadyStatusFunc(x, f.ReleaseName, f.ChartNamespace)
	}
	return getBool(f.ready, "ready")
}

func (f fakeComponent) IsEnabled(_ spi.ComponentContext) bool {
	return getBool(f.enabled, "enabled")
}

func (f fakeComponent) GetMinVerrazzanoVersion() string {
	if len(f.minVersion) > 0 {
		return f.minVersion
	}
	return constants.VerrazzanoVersion1_0_0
}

func getBool(val string, fieldName string) bool {
	// TypeOf returns type of
	// interface value passed to it
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
