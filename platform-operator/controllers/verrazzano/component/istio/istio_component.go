// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/istio"
	"go.uber.org/zap"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

// IstioComponent represents an Istio component
type IstioComponent struct{}

// Verify that IstioComponent implements Component
var _ spi.Component = IstioComponent{}

type istioUpgradeFuncSig func(log *zap.SugaredLogger, overridesFiles ...string) (stdout []byte, stderr []byte, err error)

// istioUpgradeFunc is the default upgrade function
var istioUpgradeFunc istioUpgradeFuncSig = istio.Upgrade

// Name returns the component name
func (i IstioComponent) Name() string {
	return "istio"
}

func (i IstioComponent) IsOperatorInstallSupported() bool {
	return false
}

func (i IstioComponent) IsInstalled(_ *zap.SugaredLogger, _ clipkg.Client, namespace string) (bool, error) {
	return false, nil
}

func (i IstioComponent) Install(log *zap.SugaredLogger, client clipkg.Client, namespace string, dryRun bool) error {
	return nil
}

func (i IstioComponent) Upgrade(log *zap.SugaredLogger, client clipkg.Client, ns string, dryRun bool) error {
	_, _, err := istioUpgradeFunc(log, i.Name())
	return err
}

func setIstioUpgradeFunc(f istioUpgradeFuncSig) {
	istioUpgradeFunc = f
}

func setIstioDefaultUpgradeFunc() {
	istioUpgradeFunc = istio.Upgrade
}

func (i IstioComponent) IsReady(log *zap.SugaredLogger, client clipkg.Client, namespace string) bool {
	return true
}

// GetDependencies returns the dependencies of this component
func (i IstioComponent) GetDependencies() []string {
	return []string{}
}

func (i IstioComponent) PreUpgrade(log *zap.SugaredLogger, client clipkg.Client, namespace string, dryRun bool) error {
	return nil
}

func (i IstioComponent) PostUpgrade(log *zap.SugaredLogger, client clipkg.Client, namespace string, dryRun bool) error {
	return nil
}

func (i IstioComponent) PreInstall(log *zap.SugaredLogger, client clipkg.Client, namespace string, dryRun bool) error {
	return nil
}

func (i IstioComponent) PostInstall(log *zap.SugaredLogger, client clipkg.Client, namespace string, dryRun bool) error {
	return nil
}
