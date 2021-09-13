// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package component

import (
	"github.com/verrazzano/verrazzano/platform-operator/internal/istio"
	"go.uber.org/zap"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

type istioComponent struct {
	//componentName The name of the component
	componentName string
}

// Verify that istioComponent implements Component
var _ Component = istioComponent{}

type istioUpgradeFuncSig func(log *zap.SugaredLogger, componentName string) (stdout []byte, stderr []byte, err error)

// istioUpgradeFunc is the default upgrade function
var istioUpgradeFunc istioUpgradeFuncSig = istio.Upgrade

// Name returns the component name
func (i istioComponent) Name() string {
	return i.componentName
}

func (i istioComponent) IsOperatorInstallSupported() bool {
	return false
}

func (i istioComponent) IsInstalled(_ *zap.SugaredLogger, _ clipkg.Client, namespace string) bool {
	return false
}

func (i istioComponent) Install(log *zap.SugaredLogger, client clipkg.Client, namespace string, dryRun bool) error {
	return nil
}

func (i istioComponent) Upgrade(log *zap.SugaredLogger, client clipkg.Client, ns string, dryRun bool) error {
	_, _, err := istioUpgradeFunc(log, i.componentName)
	return err
}

func setIstioUpgradeFunc(f istioUpgradeFuncSig) {
	istioUpgradeFunc = f
}

func setIstioDefaultUpgradeFunc() {
	istioUpgradeFunc = istio.Upgrade
}

func (i istioComponent) IsReady(log *zap.SugaredLogger, client clipkg.Client, namespace string) bool {
	return true
}

// GetDependencies returns the dependencies of this component
func (i istioComponent) GetDependencies() []string {
	return []string{}
}
