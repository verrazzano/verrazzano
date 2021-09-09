// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package component

import (
	"github.com/verrazzano/verrazzano/platform-operator/internal/istio"
	"go.uber.org/zap"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

type IstioComponent struct {
	//componentName The name of the component
	componentName string

	//supportsOperatorInstall Indicates whether or not the component supports install via the operator
	supportsOperatorInstall bool
}

// Verify that HelmComponent implements Component
var _ Component = HelmComponent{}

type istioUpgradeFuncSig func(log *zap.SugaredLogger) error

// helmUpgradeFunc is the default upgrade function
var istioUpgradeFunc istioUpgradeFuncSig = istio.Upgrade

// Name returns the component name
func (i IstioComponent) Name() string {
	return i.componentName
}

func (i IstioComponent) IsOperatorInstallSupported() bool {
	return i.supportsOperatorInstall
}

func (i IstioComponent) IsInstalled() bool {
	return false
}

func (i IstioComponent) Install(log *zap.SugaredLogger, client clipkg.Client, namespace string, dryRun bool) error {
	return nil
}

func (i IstioComponent) Upgrade(log *zap.SugaredLogger, client clipkg.Client, ns string, dryRun bool) error {
	err := istioUpgradeFunc(log)
	return err
}
