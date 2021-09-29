// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/bom"
	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/istio"
	"go.uber.org/zap"
	"io/ioutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"os"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"time"
)

// IstioComponent represents an Istio component
type IstioComponent struct {
	// ValuesFile contains the path to the IstioOperator CR values file
	ValuesFile string

	// Revision is the istio install revision
	Revision string

	// InjectedSystemNamespaces are the system namespaces injected with istio
	InjectedSystemNamespaces []string

	// SkipUpgrade when true will skip upgrading this component in the upgrade loop
	// This is for the istio helm components
	SkipUpgrade bool
}

const IstioOperatorOverrideFile = "istio-cr.yaml"

// Verify that IstioComponent implements Component
var _ spi.Component = IstioComponent{}

type upgradeFuncSig func(log *zap.SugaredLogger, overridesFiles ...string) (stdout []byte, stderr []byte, err error)

// upgradeFunc is the default upgrade function
var upgradeFunc upgradeFuncSig = istio.Upgrade

func SetIstioUpgradeFunction(fn upgradeFuncSig) {
	upgradeFunc = fn
}

func ResetIstioUpgradeFunction() {
	upgradeFunc = istio.Upgrade
}

type installFuncSig func(log *zap.SugaredLogger, imageOverridesString string, overridesFiles ...string) (stdout []byte, stderr []byte, err error)

// installFunc is the default install function
var installFunc installFuncSig = istio.Install

func SetIstioInstallFunction(fn installFuncSig) {
	installFunc = fn
}

func ResetIstioInstallFunction() {
	installFunc = istio.Install
}

type LabelAndResartFnType func(log *zap.SugaredLogger, err error, i IstioComponent, client clipkg.Client) error

var labelAndResartFn = labelAndRestartSystemComponents

func SetLabelAndRestartFn(fn LabelAndResartFnType) {
	labelAndResartFn = fn
}

func ResetLabelAndRestartFn() {
	labelAndResartFn = labelAndRestartSystemComponents
}

// Name returns the component name
func (i IstioComponent) Name() string {
	return "istio"
}

func (i IstioComponent) IsOperatorInstallSupported() bool {
	return true
}

func (i IstioComponent) IsInstalled(_ *zap.SugaredLogger, _ clipkg.Client, _ string) (bool, error) {
	return false, nil
}

func (i IstioComponent) Install(log *zap.SugaredLogger, vz *installv1alpha1.Verrazzano, _ clipkg.Client, _ string, _ bool) error {
	var tmpFile *os.File

	// Only create override file if the CR has an Istio component
	if  vz.Spec.Components.Istio != nil {
		istioOperatorYaml, err := BuildIstioOperatorYaml(vz.Spec.Components.Istio)
		if err != nil {
			log.Errorf("Failed to Build IstioOperator YAML: %v", err)
			return err
		}

		// Write the overrides to a tmp file
		tmpFile, err := ioutil.TempFile(os.TempDir(), "istio-*.yaml")
		if err != nil {
			log.Errorf("Failed to create temporary file for Istio install: %v", err)
			return err
		}
		defer os.Remove(tmpFile.Name())
		if _, err = tmpFile.Write([]byte(istioOperatorYaml)); err != nil {
			log.Errorf("Failed to write to temporary file: %v", err)
			return err
		}
		if err := tmpFile.Close(); err != nil {
			log.Errorf("Failed to close temporary file: %v", err)
			return err
		}
		log.Infof("Created values file from Istio install args: %s", tmpFile.Name())
	}

	imageOverrides, err := buildImageOverridesString(log)
	if err != nil {
		log.Errorf("Error building image overrides from BOM for Istio: %v", err)
		return err
	}

	if tmpFile == nil {
		_, _, err := installFunc(log, imageOverrides, i.ValuesFile)
		if err != nil {
			return err
		}
	} else {
		_, _, err := installFunc(log, imageOverrides, i.ValuesFile, tmpFile.Name())
		if err != nil {
			return err
		}
	}

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

	err = labelAndResartFn(log, err, i, client)
	if err != nil {
		return err
	}

	return err
}

func labelAndRestartSystemComponents(log *zap.SugaredLogger, err error, i IstioComponent, client clipkg.Client) error {
	err = i.restartSystemNamespaceResources(log, client)
	if err != nil {
		return err
	}
	return nil
}

func setUpgradeFunc(f upgradeFuncSig) {
	upgradeFunc = f
}

func setDefaultUpgradeFunc() {
	upgradeFunc = istio.Upgrade
}

func setInstallFunc(f installFuncSig) {
	installFunc = f
}

func setDefaultInstallFunc() {
	installFunc = istio.Install
}

func (i IstioComponent) IsReady(log *zap.SugaredLogger, client clipkg.Client, namespace string) bool {
	return false
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

// createVerrazzanoSystemNamespace creates the verrazzano system namespace if it does not already exist
func (i IstioComponent) labelSystemNamespaces(log *zap.SugaredLogger, client clipkg.Client) error {
	for _, ns := range i.InjectedSystemNamespaces {
		platformNS := corev1.Namespace{}
		err := client.Get(context.TODO(), types.NamespacedName{Name: ns}, &platformNS)
		if err != nil {
			log.Infof("Namespace %v not found", ns)
			continue
		}

		nsLabels := platformNS.Labels

		// add istio.io/rev label
		nsLabels["istio.io/rev"] = i.Revision
		delete(nsLabels, "istio-injection")
		platformNS.Labels = nsLabels

		// update namespace
		err = client.Update(context.TODO(), &platformNS)
		if err != nil {
			return err
		}
		log.Infof("Relabeled namespace %v for istio upgrade", platformNS.Name)
	}
	return nil
}

// restartSystemNamespaceResources restarts all the deployments, StatefulSets, and DaemonSets
// in all of the Istio injected system namespaces
func (i IstioComponent) restartSystemNamespaceResources(log *zap.SugaredLogger, client clipkg.Client) error {
	// Restart all the deployments in the injected system namespaces
	var deploymentList appsv1.DeploymentList
	err := client.List(context.TODO(), &deploymentList)
	if err != nil {
		return err
	}
	for index := range deploymentList.Items {
		deployment := &deploymentList.Items[index]
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
	//var statefulSetList appsv1.StatefulSetList
	statefulSetList := appsv1.StatefulSetList{}
	err = client.List(context.TODO(), &statefulSetList)
	if err != nil {
		return err
	}
	for index := range statefulSetList.Items {
		statefulSet := &statefulSetList.Items[index]
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

func (i IstioComponent) GetSkipUpgrade() bool {
	return i.SkipUpgrade
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
		for i, _ := range scKvs {
			kvs = append(kvs, scKvs[i])
		}
	}

	return kvs, nil
}
