// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	"context"
	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/istio"
	"go.uber.org/zap"
	"io/ioutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"os"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

// IstioComponent represents an Istio component
type IstioComponent struct {
	// ValuesFile contains the path to the IstioOperator CR values file
	ValuesFile string

	// Revision is the istio install revision
	Revision string

	// InjectedSystemNamespaces are the system namespaces injected with istio
	InjectedSystemNamespaces []string
}

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
	_, _, err = upgradeFunc(log, i.ValuesFile, tmpFile.Name())
	if err != nil {
		return err
	}
	err = i.labelSystemNamespaces(log, client)
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
func (i IstioComponent) labelSystemNamespaces(log *zap.SugaredLogger, client clipkg.Client) error {
	var platformNS corev1.Namespace
	//var deploymentList appsv1.DeploymentList
	//var statefulSetList appsv1.DeploymentList
	//var daemonSetList appsv1.DeploymentList
	for _, ns := range i.InjectedSystemNamespaces {
		err := client.Get(context.TODO(), types.NamespacedName{Name: ns}, &platformNS)
		if err != nil {
			if !errors.IsNotFound(err) {
				return err
			}
			nsLabels := platformNS.Labels

			// add istio.io/rev label
			nsLabels["istio.io/rev"] = i.Revision
			delete(nsLabels, "istio-injection")
			platformNS.Labels = nsLabels
			client.Update(context.TODO(), platformNS.DeepCopyObject())
			log.Infof("Relabeled namespace %v for istio upgrade", platformNS.Name)
		}
	}
	return nil
}

func (i IstioComponent) restartSystemNamespaceResources(log *zap.SugaredLogger, client clipkg.Client) error {
	deploymentList := &appsv1.DeploymentList{}
	err := client.List(context.TODO(), deploymentList)
	if err != nil {
		return err
	}
	for _, deploy := range deploymentList.Items {
		if contains(i.InjectedSystemNamespaces, deploy.Namespace) {
			client.Update(context.TODO(), deploy.DeepCopyObject())
		}
	}
}

// contains is a helper function that should be a build-in
func contains(arr []string, s string) bool {
	for _, str := range arr {
		if str == s {
			return true
		}
	}
	return false
}
