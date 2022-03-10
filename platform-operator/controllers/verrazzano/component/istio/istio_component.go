// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/verrazzano/verrazzano/pkg/bom"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	ctrlerrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
	"github.com/verrazzano/verrazzano/pkg/helm"
	"github.com/verrazzano/verrazzano/pkg/istio"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzString "github.com/verrazzano/verrazzano/pkg/string"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

// ComponentName is the name of the component
const ComponentName = "istio"

// IstiodDeployment is the name of the istiod deployment
const IstiodDeployment = "istiod"

// IstioInressgatewayDeployment is the name of the istio ingressgateway deployment
const IstioInressgatewayDeployment = "istio-ingressgateway"

// IstioEressgatewayDeployment is the name of the istio egressgateway deployment
const IstioEressgatewayDeployment = "istio-egressgateway"

const istioGlobalHubKey = "global.hub"

// IstioNamespace is the default Istio namespace
const IstioNamespace = "istio-system"

// IstioCoreDNSReleaseName is the name of the istiocoredns release
const IstioCoreDNSReleaseName = "istiocoredns"

// HelmScrtType is the secret type that helm uses to specify its releases
const HelmScrtType = "helm.sh/release.v1"

// istioComponent represents an Istio component
type istioComponent struct {
	// ValuesFile contains the path to the IstioOperator CR values file
	ValuesFile string

	// Revision is the istio install revision
	Revision string

	// InjectedSystemNamespaces are the system namespaces injected with istio
	InjectedSystemNamespaces []string

	// Internal monitor object for peforming `istioctl` operations in the background
	monitor installMonitor
}

type upgradeFuncSig func(log vzlog.VerrazzanoLogger, imageOverrideString string, overridesFiles ...string) (stdout []byte, stderr []byte, err error)

// upgradeFunc is the default upgrade function
var upgradeFunc upgradeFuncSig = istio.Upgrade

func SetIstioUpgradeFunction(fn upgradeFuncSig) {
	upgradeFunc = fn
}

func SetDefaultIstioUpgradeFunction() {
	upgradeFunc = istio.Upgrade
}

type helmUninstallFuncSig func(log vzlog.VerrazzanoLogger, releaseName string, namespace string, dryRun bool) (stdout []byte, stderr []byte, err error)

var helmUninstallFunction helmUninstallFuncSig = helm.Uninstall

func SetHelmUninstallFunction(fn helmUninstallFuncSig) {
	helmUninstallFunction = fn
}

func SetDefaultHelmUninstallFunction() {
	helmUninstallFunction = helm.Uninstall
}

func NewComponent() spi.Component {
	return istioComponent{
		ValuesFile:               filepath.Join(config.GetHelmOverridesDir(), "istio-cr.yaml"),
		InjectedSystemNamespaces: config.GetInjectedSystemNamespaces(),
		monitor:                  &installMonitorType{},
	}
}

// IsEnabled istio-specific enabled check for installation
func (i istioComponent) IsEnabled(ctx spi.ComponentContext) bool {
	comp := ctx.EffectiveCR().Spec.Components.Istio
	if comp == nil || comp.Enabled == nil {
		return true
	}
	return *comp.Enabled
}

// GetMinVerrazzanoVersion returns the minimum Verrazzano version required by the component
func (i istioComponent) GetMinVerrazzanoVersion() string {
	return constants.VerrazzanoVersion1_0_0
}

// Name returns the component name
func (i istioComponent) Name() string {
	return ComponentName
}

func (i istioComponent) Upgrade(context spi.ComponentContext) error {

	log := context.Log()

	// temp file to contain override values from istio install args
	var tmpFile *os.File
	tmpFile, err := ioutil.TempFile(os.TempDir(), "values-*.yaml")
	if err != nil {
		return log.ErrorfNewErr("Failed to create temporary file: %v", err)
	}

	vz := context.EffectiveCR()
	defer os.Remove(tmpFile.Name())
	if vz.Spec.Components.Istio != nil {
		istioOperatorYaml, err := BuildIstioOperatorYaml(vz.Spec.Components.Istio)
		if err != nil {
			return log.ErrorfNewErr("Failed to Build IstioOperator YAML: %v", err)
		}

		if _, err = tmpFile.Write([]byte(istioOperatorYaml)); err != nil {
			return log.ErrorfNewErr("Failed to write to temporary file: %v", err)
		}

		// Close the file
		if err := tmpFile.Close(); err != nil {
			return log.ErrorfNewErr("Failed to close temporary file: %v", err)
		}

		log.Debugf("Created values file from Istio install args: %s", tmpFile.Name())
	}

	// images overrides to get passed into the istioctl command
	imageOverrides, err := buildImageOverridesString(log)
	if err != nil {
		return log.ErrorfNewErr("Error building image overrides from BOM for Istio: %v", err)
	}
	_, _, err = upgradeFunc(log, imageOverrides, i.ValuesFile, tmpFile.Name())
	if err != nil {
		return err
	}

	return err
}

func (i istioComponent) IsReady(context spi.ComponentContext) bool {
	deployments := []status.PodReadyCheck{
		{
			NamespacedName: types.NamespacedName{
				Name:      IstiodDeployment,
				Namespace: IstioNamespace,
			},
			LabelSelector: labels.Set{"app": IstiodDeployment}.AsSelector(),
		},
		{
			NamespacedName: types.NamespacedName{
				Name:      IstioInressgatewayDeployment,
				Namespace: IstioNamespace,
			},
			LabelSelector: labels.Set{"app": IstioInressgatewayDeployment}.AsSelector(),
		},
		{
			NamespacedName: types.NamespacedName{
				Name:      IstioEressgatewayDeployment,
				Namespace: IstioNamespace,
			},
			LabelSelector: labels.Set{"app": IstioEressgatewayDeployment}.AsSelector(),
		},
	}
	prefix := fmt.Sprintf("Component %s", context.GetComponent())
	return status.DeploymentsAreReady(context.Log(), context.Client(), deployments, 1, prefix)
}

// GetDependencies returns the dependencies of this component
func (i istioComponent) GetDependencies() []string {
	return []string{}
}

func (i istioComponent) PreUpgrade(context spi.ComponentContext) error {
	context.Log().Infof("Stopping WebLogic domains that are have Envoy 1.7.3 sidecar")
	return StopDomainsUsingOldEnvoy(context.Log(), context.Client())
}

func (i istioComponent) PostUpgrade(context spi.ComponentContext) error {
	err := deleteIstioCoreDNS(context)
	if err != nil {
		return err
	}
	err = removeIstioHelmSecrets(context)
	if err != nil {
		return err
	}

	// Generate a restart version that will not change for this Verrazzano version
	// Valid labels cannot contain + sign
	restartVersion := context.EffectiveCR().Spec.Version + "-upgrade"
	restartVersion = strings.ReplaceAll(restartVersion, "+", "-")

	// Start WebLogic domains that were shutdown
	context.Log().Infof("Starting WebLogic domains that were stopped pre-upgrade")
	if err := StartDomainsStoppedByUpgrade(context.Log(), context.Client(), restartVersion); err != nil {
		return err
	}

	// Restart all other apps
	context.Log().Infof("Restarting all applications so they can get the new Envoy sidecar")
	if err := RestartAllApps(context.Log(), context.Client(), restartVersion); err != nil {
		return err
	}
	return nil
}

func (i istioComponent) Reconcile(_ spi.ComponentContext) error {
	return nil
}

// GetIngressNames returns the list of ingress names associated with the component
func (i istioComponent) GetIngressNames(_ spi.ComponentContext) []types.NamespacedName {
	return []types.NamespacedName{}
}

// RestartComponents restarts all the deployments, StatefulSets, and DaemonSets
// in all of the Istio injected system namespaces
func RestartComponents(log vzlog.VerrazzanoLogger, namespaces []string, client clipkg.Client) error {
	// Restart all the deployments in the injected system namespaces
	var deploymentList appsv1.DeploymentList
	err := client.List(context.TODO(), &deploymentList)
	if err != nil {
		return err
	}
	for index := range deploymentList.Items {
		deployment := &deploymentList.Items[index]

		// Check if deployment is in an Istio injected system namespace
		if vzString.SliceContainsString(namespaces, deployment.Namespace) {
			if deployment.Spec.Paused {
				return log.ErrorfNewErr("Failed, deployment %s can't be restarted because it is paused", deployment.Name)
			}
			if deployment.Spec.Template.ObjectMeta.Annotations == nil {
				deployment.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
			}
			deployment.Spec.Template.ObjectMeta.Annotations[vzconst.VerrazzanoRestartAnnotation] = time.Now().Format(time.RFC3339)
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
		if vzString.SliceContainsString(namespaces, statefulSet.Namespace) {
			if statefulSet.Spec.Template.ObjectMeta.Annotations == nil {
				statefulSet.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
			}
			statefulSet.Spec.Template.ObjectMeta.Annotations[vzconst.VerrazzanoRestartAnnotation] = time.Now().Format(time.RFC3339)
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
		if vzString.SliceContainsString(namespaces, daemonSet.Namespace) {
			if daemonSet.Spec.Template.ObjectMeta.Annotations == nil {
				daemonSet.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
			}
			daemonSet.Spec.Template.ObjectMeta.Annotations[vzconst.VerrazzanoRestartAnnotation] = time.Now().Format(time.RFC3339)
			if err := client.Update(context.TODO(), daemonSet); err != nil {
				return err
			}
		}
	}
	log.Info("Restarted system DaemonSets in istio injected namespaces")
	return nil
}

func deleteIstioCoreDNS(context spi.ComponentContext) error {
	// Check if the component is installed before trying to upgrade
	found, err := helm.IsReleaseInstalled(IstioCoreDNSReleaseName, constants.IstioSystemNamespace)
	if err != nil {
		return context.Log().ErrorfNewErr("Failed searching for release: %v", err)
	}
	if found {
		_, _, err = helmUninstallFunction(context.Log(), IstioCoreDNSReleaseName, constants.IstioSystemNamespace, context.IsDryRun())
		if err != nil {
			return context.Log().ErrorfNewErr("Failed trying to uninstall istiocoredns: %v", err)
		}
	}
	return err
}

// removeIstioHelmSecrets deletes the release metadata that helm uses to access to access and control the releases
// this is sufficient to prevent helm from trying to operator on deployments it doesn't control anymore
// however it does not delete the underlying resources
func removeIstioHelmSecrets(compContext spi.ComponentContext) error {
	client := compContext.Client()
	var secretList v1.SecretList
	listOptions := clipkg.ListOptions{Namespace: constants.IstioSystemNamespace}
	err := client.List(context.TODO(), &secretList, &listOptions)
	if err != nil {
		return compContext.Log().ErrorfNewErr("Error retrieving list of secrets in the istio-system namespace: %v", err)
	}
	for index := range secretList.Items {
		secret := &secretList.Items[index]
		secretName := secret.Name
		if secret.Type == HelmScrtType && !strings.Contains(secretName, IstioCoreDNSReleaseName) {
			err = client.Delete(context.TODO(), secret)
			if err != nil {
				if ctrlerrors.ShouldLogKubenetesAPIError(err) {
					compContext.Log().Errorf("Error deleting helm secret %s: %v", secretName, err)
				}
				return err
			}
			compContext.Log().Debugf("Deleted helm secret %s", secretName)
		}
	}
	return nil
}

func buildImageOverridesString(_ vzlog.VerrazzanoLogger) (string, error) {
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

// AppendIstioOverrides appends the Keycloak theme for the Key keycloak.extraInitContainers.
// A go template is used to replace the image in the init container spec.
func AppendIstioOverrides(_ spi.ComponentContext, releaseName string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	// Create a Bom and get the Key Value overrides
	bomFile, err := bom.NewBom(config.GetDefaultBOMFilePath())
	if err != nil {
		return nil, err
	}

	// Get the istio component
	sc, err := bomFile.GetSubcomponent(releaseName)
	if err != nil {
		return nil, err
	}

	registry := bomFile.ResolveRegistry(sc, bom.BomImage{})
	repo := bomFile.ResolveRepo(sc, bom.BomImage{})

	// Override the global.hub if either of the 2 env vars were defined
	if registry != bomFile.GetRegistry() || repo != sc.Repository {
		// Return a new Key:Value pair with the rendered Value
		kvs = append(kvs, bom.KeyValue{
			Key:   istioGlobalHubKey,
			Value: registry + "/" + repo,
		})
	}

	return kvs, nil
}

func buildOverridesString(log vzlog.VerrazzanoLogger, client clipkg.Client, namespace string, additionalValues ...bom.KeyValue) (string, error) {
	// Get the image overrides from the BOM
	kvs, err := getImageOverrides()
	if err != nil {
		return "", err
	}

	// Append any special overrides passed in
	if len(additionalValues) > 0 {
		kvs = append(kvs, additionalValues...)
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
	const subcompIstiod = "istiod"
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
