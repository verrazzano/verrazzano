// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	"context"
	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/istio"
	"go.uber.org/zap"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"os"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

// IstioComponent represents an Istio component
type IstioComponent struct {
	ValuesFile string
}

const istioRevision = "1-10-2"

// Verify that IstioComponent implements Component
var _ spi.Component = IstioComponent{}

type upgradeFuncSig func(log *zap.SugaredLogger, overridesFiles ...string) (stdout []byte, stderr []byte, err error)

// upgradeFunc is the default upgrade function
var upgradeFunc upgradeFuncSig = istio.Upgrade

// Name returns the component name
func (i IstioComponent) Name() string {
	return "istio"
}

func (i IstioComponent) IsOperatorInstallSupported() bool {
	return false
}

func (i IstioComponent) IsInstalled(_ *zap.SugaredLogger, _ clipkg.Client, _ string) bool {
	return false
}

func (i IstioComponent) Install(log *zap.SugaredLogger, vz *installv1alpha1.Verrazzano, _ clipkg.Client, _ string, _ bool) error {
	return nil
}

func (i IstioComponent) Upgrade(log *zap.SugaredLogger, vz *installv1alpha1.Verrazzano, client clipkg.Client, _ string, _ bool) error {
	var tmpFile *os.File
	tmpFile, err := ioutil.TempFile(os.TempDir(), "values-*.yaml")
	if err != nil {
		log.Errorf("Failed to create temporary file: %v", err)
		return err
	}

	defer os.Remove(tmpFile.Name())
	if vz.Spec.Components.Istio != nil {
		istioOperatorYaml, err := BuildIstioOperatorYaml(vz.Spec.Components.Istio)
		if err != nil {
			log.Errorf("Failed to Build IstioOperator YAML: %v", err)
			return err
		}

		if _, err = tmpFile.Write([]byte(istioOperatorYaml)); err != nil {
			log.Errorf("Failed to write to temporary file: %v", err)
			return err
		}

		// Close the file
		if err := tmpFile.Close(); err != nil {
			log.Errorf("Failed to close temporary file: %v", err)
			return err
		}

		log.Infof("Created values file from Istio install args: %s", tmpFile.Name())
	}
	//_, _, err = upgradeFunc(log, i.ValuesFile, tmpFile.Name())
	_, _, err = upgradeFunc(log, i.ValuesFile)
	return err
}

func setUpgradeFunc(f upgradeFuncSig) {
	upgradeFunc = f
}

func setDefaultUpgradeFunc() {
	upgradeFunc = istio.Upgrade
}

func (i IstioComponent) IsReady(log *zap.SugaredLogger, client clipkg.Client, namespace string) bool {
	return true
}

// GetDependencies returns the dependencies of this component
func (i IstioComponent) GetDependencies() []string {
	return []string{}
}

// createVerrazzanoSystemNamespace creates the verrazzano system namespace if it does not already exist
func upgradePlatformNS(ctx context.Context, log *zap.SugaredLogger, client clipkg.Client) error {
	istioPlatformNamespaces := []string{constants.VerrazzanoSystemNamespace, constants.IngressNginxNamespace, constants.KeycloakNamespace}
	var platformNS corev1.Namespace
	for _, ns := range istioPlatformNamespaces {
		err := client.Get(ctx, types.NamespacedName{Name: ns}, &platformNS)
		if err != nil {
			if !errors.IsNotFound(err) {
				return err
			}
			nsLabels := platformNS.Labels

			// add istio.io/rev label
			nsLabels["istio.io/rev"] = istioRevision
			delete(nsLabels, "istio-injection")
			platformNS.Labels = nsLabels

			log.Infof("Relabeled namespace %v for istio upgrade", platformNS.Name)

			platformNS.GetObjectMeta()
		}
	}
	return nil
}
