// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/internal/helm"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/istio"

	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
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

type upgradeFuncSig func(log *zap.SugaredLogger, imageOverrideString string, overridesFiles ...string) (stdout []byte, stderr []byte, err error)

// upgradeFunc is the default upgrade function
var upgradeFunc upgradeFuncSig = istio.Upgrade

func SetIstioUpgradeFunction(fn upgradeFuncSig) {
	upgradeFunc = fn
}

func ResetIstioUpgradeFunction() {
	upgradeFunc = istio.Upgrade
}

type restartComponentsFnType func(log *zap.SugaredLogger, err error, i IstioComponent, client clipkg.Client) error

var restartComponentsFn = restartComponents

func SetRestartComponentsFn(fn restartComponentsFnType) {
	restartComponentsFn = fn
}

func ResetRestartComponentsFn() {
	restartComponentsFn = restartComponents
}

// Name returns the component name
func (i IstioComponent) Name() string {
	return "istio"
}

func (i IstioComponent) IsOperatorInstallSupported() bool {
	return false
}

func (i IstioComponent) IsInstalled(_ spi.ComponentContext) (bool, error) {
	return false, nil
}

func (i IstioComponent) Install(_ spi.ComponentContext) error {
	return nil
}

func (i IstioComponent) Upgrade(context spi.ComponentContext) error {

	log := context.Log()

	// temp file to contain override values from istio install args
	var tmpFile *os.File
	tmpFile, err := ioutil.TempFile(os.TempDir(), "values-*.yaml")
	if err != nil {
		log.Errorf("Failed to create temporary file: %v", err)
		return err
	}

	vz := context.EffectiveCR()
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

	// images overrides to get passed into the istioctl command
	imageOverrides, err := buildImageOverridesString(log)
	if err != nil {
		log.Errorf("Error building image overrides from BOM for Istio: %v", err)
		return err
	}
	_, _, err = upgradeFunc(log, imageOverrides, i.ValuesFile, tmpFile.Name())
	if err != nil {
		return err
	}

	err = restartComponentsFn(log, err, i, context.Client())
	if err != nil {
		return err
	}

	return err
}

func setUpgradeFunc(f upgradeFuncSig) {
	upgradeFunc = f
}

func setDefaultUpgradeFunc() {
	upgradeFunc = istio.Upgrade
}

func (i IstioComponent) IsReady(_ spi.ComponentContext) bool {
	return true
}

// GetDependencies returns the dependencies of this component
func (i IstioComponent) GetDependencies() []string {
	return []string{}
}

func (i IstioComponent) PreUpgrade(_ spi.ComponentContext) error {
	return nil
}

func (i IstioComponent) PostUpgrade(context spi.ComponentContext) error {
	// Check if the component is installed before trying to upgrade
	found, err := helm.IsReleaseInstalled("istiocoredns", constants.IstioSystemNamespace)
	if err != nil {
		return err
	}
	if found {
		_, _, err = helm.Uninstall(context.Log(), "istiocoredns", constants.IstioSystemNamespace, context.IsDryRun())
	}
	return err
}

func (i IstioComponent) PreInstall(_ spi.ComponentContext) error {
	return nil
}

func (i IstioComponent) PostInstall(_ spi.ComponentContext) error {
	return nil
}

// restartComponents restarts all the deployments, StatefulSets, and DaemonSets
// in all of the Istio injected system namespaces
func restartComponents(log *zap.SugaredLogger, err error, i IstioComponent, client clipkg.Client) error {

	// Restart all the deployments in the injected system namespaces
	var deploymentList appsv1.DeploymentList
	err = client.List(context.TODO(), &deploymentList)
	if err != nil {
		return err
	}
	for index := range deploymentList.Items {
		deployment := &deploymentList.Items[index]

		// Check if deployment is in an Istio injected system namespace
		if contains(i.InjectedSystemNamespaces, deployment.Namespace) {
			if deployment.Spec.Paused {
				return fmt.Errorf("Deployment %v can't be restarted because it is paused", deployment.Name)
			}
			if deployment.Spec.Template.ObjectMeta.Annotations == nil {
				deployment.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
			}
			deployment.Spec.Template.ObjectMeta.Annotations["verrazzano.io/restartedAt"] = time.Now().Format(time.RFC3339)
			if err := client.Update(context.TODO(), deployment); err != nil {
				return err
			}
		}
	}
	log.Info("Restarted system Deployments in istio injected namespaces")

	// Restart all the StatefulSet in the injected system namespaces
	statefulSetList := appsv1.StatefulSetList{}
	err = client.List(context.TODO(), &statefulSetList)
	if err != nil {
		return err
	}
	for index := range statefulSetList.Items {
		statefulSet := &statefulSetList.Items[index]

		// Check if StatefulSet is in an Istio injected system namespace
		if contains(i.InjectedSystemNamespaces, statefulSet.Namespace) {
			if statefulSet.Spec.Template.ObjectMeta.Annotations == nil {
				statefulSet.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
			}
			statefulSet.Spec.Template.ObjectMeta.Annotations["verrazzano.io/restartedAt"] = time.Now().Format(time.RFC3339)
			if err := client.Update(context.TODO(), statefulSet); err != nil {
				return err
			}
		}
	}
	log.Info("Restarted system Statefulsets in istio injected namespaces")

	// Restart all the DaemonSets in the injected system namespaces
	var daemonSetList appsv1.DaemonSetList
	err = client.List(context.TODO(), &daemonSetList)
	if err != nil {
		return err
	}
	for index := range daemonSetList.Items {
		daemonSet := &daemonSetList.Items[index]

		// Check if DaemonSet is in an Istio injected system namespace
		if contains(i.InjectedSystemNamespaces, daemonSet.Namespace) {
			if daemonSet.Spec.Template.ObjectMeta.Annotations == nil {
				daemonSet.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
			}
			daemonSet.Spec.Template.ObjectMeta.Annotations["verrazzano.io/restartedAt"] = time.Now().Format(time.RFC3339)
			if err := client.Update(context.TODO(), daemonSet); err != nil {
				return err
			}
		}
	}
	log.Info("Restarted system DaemonSets in istio injected namespaces")
	return nil
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

func buildImageOverridesString(log *zap.SugaredLogger) (string, error) {
	// Get the image overrides from the BOM
	var kvs []bom.KeyValue
	var err error
	kvs, err = getImageOverrides()
	if err != nil {
		return "", err
	}

	// If there are overridesString the create a comma separated string
	var overridesString string
	if len(kvs) > 0 {
		bldr := strings.Builder{}
		for i, kv := range kvs {
			if i > 0 {
				bldr.WriteString(",")
			}
			bldr.WriteString(fmt.Sprintf("%s=%s", kv.Key, kv.Value))
		}
		overridesString = bldr.String()
	}
	return overridesString, nil
}

// Get the image overrides from the BOM
func getImageOverrides() ([]bom.KeyValue, error) {

	const subcompIstiod = "istiod-1.10.2"
	subComponentNames := []string{subcompIstiod}

	// Create a Bom and get the Key Value overrides
	bomFile, err := bom.NewBom(config.GetDefaultBOMFilePath())
	if err != nil {
		return nil, err
	}

	var kvs []bom.KeyValue
	for _, scName := range subComponentNames {
		scKvs, err := bomFile.BuildImageOverrides(scName)
		if err != nil {
			return nil, err
		}
		for i := range scKvs {
			kvs = append(kvs, scKvs[i])
		}
	}

	return kvs, nil
}
