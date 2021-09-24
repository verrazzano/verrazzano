// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package verrazzano

import (
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"go.uber.org/zap"
)

// fakeComponent allows for using dummy Component implementations for controller testing
type fakeComponent struct{}

func (f fakeComponent) Name() string {
	panic("implement me")
}

func (f fakeComponent) IsOperatorInstallSupported() bool {
	return false
}

func (f fakeComponent) GetDependencies() []string {
	return []string{}
}

func (f fakeComponent) PreUpgrade(_ *zap.SugaredLogger, _ *spi.ComponentContext) error {
	return nil
}

func (f fakeComponent) Upgrade(_ *zap.SugaredLogger, _ *spi.ComponentContext) error {
	return nil
}

func (f fakeComponent) PostUpgrade(_ *zap.SugaredLogger, _ *spi.ComponentContext) error {
	return nil
}

func (f fakeComponent) PreInstall(_ *zap.SugaredLogger, _ *spi.ComponentContext) error {
	return nil
}

func (f fakeComponent) Install(_ *zap.SugaredLogger, _ *spi.ComponentContext) error {
	return nil
}

func (f fakeComponent) PostInstall(_ *zap.SugaredLogger, _ *spi.ComponentContext) error {
	return nil
}

func (f fakeComponent) IsInstalled(_ *zap.SugaredLogger, _ *spi.ComponentContext) (bool, error) {
	return true, nil
}

func (f fakeComponent) IsReady(_ *zap.SugaredLogger, _ *spi.ComponentContext) bool {
	return true
}
