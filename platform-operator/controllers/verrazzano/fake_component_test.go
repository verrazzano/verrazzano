// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package verrazzano

import (
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
)

// fakeComponent allows for using dummy Component implementations for controller testing
type fakeComponent struct{}

func (f fakeComponent) Name() string {
	return "fake"
}

func (f fakeComponent) GetSkipUpgrade() bool {
	return false
}

func (f fakeComponent) IsOperatorInstallSupported() bool {
	return false
}

func (f fakeComponent) GetDependencies() []string {
	return []string{}
}

func (f fakeComponent) PreUpgrade(_ spi.ComponentContext) error {
	return nil
}

func (f fakeComponent) Upgrade(_ spi.ComponentContext) error {
	return nil
}

func (f fakeComponent) PostUpgrade(_ spi.ComponentContext) error {
	return nil
}

func (f fakeComponent) PreInstall(_ spi.ComponentContext) error {
	return nil
}

func (f fakeComponent) Install(_ spi.ComponentContext) error {
	return nil
}

func (f fakeComponent) PostInstall(_ spi.ComponentContext) error {
	return nil
}

func (f fakeComponent) IsInstalled(_ spi.ComponentContext) (bool, error) {
	return true, nil
}

func (f fakeComponent) IsReady(_ spi.ComponentContext) bool {
	return true
}

func (f fakeComponent) IsEnabled(_ spi.ComponentContext) bool {
	return true
}

func (i fakeComponent) GetMinVerrazzanoVersion() string {
	return constants.VerrazzanoVersion_1_0_0
}
