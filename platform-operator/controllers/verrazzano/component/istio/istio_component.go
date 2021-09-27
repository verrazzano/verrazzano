// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	"context"
	"fmt"
	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/istio"
	"go.uber.org/zap"
	"io/ioutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"os"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
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
	return false
}

func (i IstioComponent) IsInstalled(_ *zap.SugaredLogger, _ clipkg.Client, _ string) (bool, error) {
	return false, nil
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

	err = labelAndResartFn(log, err, i, client)
	if err != nil {
		return err
	}

	return err
}

func labelAndRestartSystemComponents(log *zap.SugaredLogger, err error, i IstioComponent, client clipkg.Client) error {
	err = i.labelSystemNamespaces(log, client)
	if err != nil {
		return err
	}

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

// createVerrazzanoSystemNamespace creates the verrazzano system namespace if it does not already exist
func (i IstioComponent) labelSystemNamespaces(log *zap.SugaredLogger, client clipkg.Client) error {
	for _, ns := range i.InjectedSystemNamespaces {
		platformNS := corev1.Namespace{}
		err := client.Get(context.TODO(), types.NamespacedName{Name: ns}, &platformNS)
		if err != nil {
			log.Infof("Namespace %v not found", ns)
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
	for index, deployment := range deploymentList.Items {
		if contains(i.InjectedSystemNamespaces, deployment.Namespace) {
			if deployment.Spec.Paused {
				return fmt.Errorf("Deployment %v can't be restarted because it is paused", deployment.Name)
			}
			if deployment.Spec.Template.ObjectMeta.Annotations == nil {
				deployment.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
			}
			deployment.Spec.Template.ObjectMeta.Annotations["verrazzano.io/restartedAt"] = time.Now().Format(time.RFC3339)
			if err := client.Update(context.TODO(), &deploymentList.Items[index]); err != nil {
				return err
			}
		}
	}
	log.Info("Restarted system deployments in istio injected namespaces")

	// Restart all the StatefulSet in the injected system namespaces
	//var statefulSetList appsv1.StatefulSetList
	statefulSetList := appsv1.StatefulSetList{}
	err = client.List(context.TODO(), &statefulSetList)
	if err != nil {
		return err
	}
	for index, statefulSet := range statefulSetList.Items {
		if contains(i.InjectedSystemNamespaces, statefulSet.Namespace) {
			if statefulSet.Spec.Template.ObjectMeta.Annotations == nil {
				statefulSet.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
			}
			statefulSet.Spec.Template.ObjectMeta.Annotations["verrazzano.io/restartedAt"] = time.Now().Format(time.RFC3339)
			if err := client.Update(context.TODO(), &statefulSetList.Items[index]); err != nil {
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
	for index, daemonSet := range daemonSetList.Items {
		if contains(i.InjectedSystemNamespaces, daemonSet.Namespace) {
			if daemonSet.Spec.Template.ObjectMeta.Annotations == nil {
				daemonSet.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
			}
			daemonSet.Spec.Template.ObjectMeta.Annotations["verrazzano.io/restartedAt"] = time.Now().Format(time.RFC3339)
			if err := client.Update(context.TODO(), &daemonSetList.Items[index]); err != nil {
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
