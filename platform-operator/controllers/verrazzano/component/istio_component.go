package component

import (
	"github.com/verrazzano/verrazzano/platform-operator/internal/istio"
	"go.uber.org/zap"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

type istioComponent struct {
	//componentName The name of the component
	componentName string

	//supportsOperatorInstall Indicates whether or not the component supports install via the operator
	supportsOperatorInstall bool
}

// Verify that helmComponent implements Component
var _ Component = helmComponent{}

type istioUpgradeFuncSig func() error

// helmUpgradeFunc is the default upgrade function
var istioUpgradeFunc istioUpgradeFuncSig = istio.Upgrade

// Name returns the component name
func (i istioComponent) Name() string {
	return i.componentName
}

func (i istioComponent) IsOperatorInstallSupported() bool {
	return i.supportsOperatorInstall
}

func (i istioComponent) IsInstalled() bool {
	return false
}

func (i istioComponent) Install(log *zap.SugaredLogger, client clipkg.Client, namespace string, dryRun bool) error {
	return nil
}

func (i istioComponent) Upgrade(log *zap.SugaredLogger, client clipkg.Client, ns string, dryRun bool) error {
	return nil
}
