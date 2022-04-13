// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	k8s "github.com/verrazzano/verrazzano/platform-operator/internal/nodeport"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"

	"github.com/verrazzano/verrazzano/pkg/bom"
	ctrlerrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
	"github.com/verrazzano/verrazzano/pkg/helm"
	"github.com/verrazzano/verrazzano/pkg/istio"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

// ComponentName is the name of the component
const ComponentName = "istio"

// ComponentJSONName is the josn name of the verrazzano component in CRD
const ComponentJSONName = "istio"

// IstiodDeployment is the name of the istiod deployment
const IstiodDeployment = "istiod"

// IstioIngressgatewayDeployment is the name of the istio ingressgateway deployment
const IstioIngressgatewayDeployment = "istio-ingressgateway"

// IstioEgressgatewayDeployment is the name of the istio egressgateway deployment
const IstioEgressgatewayDeployment = "istio-egressgateway"

const istioGlobalHubKey = "global.hub"

// IstioNamespace is the default Istio namespace
const IstioNamespace = "istio-system"

// IstioCoreDNSReleaseName is the name of the istiocoredns release
const IstioCoreDNSReleaseName = "istiocoredns"

// HelmScrtType is the secret type that helm uses to specify its releases
const HelmScrtType = "helm.sh/release.v1"

// subcompIstiod is the Istiod subcomponent in the bom
const subcompIstiod = "istiod"

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

// GetJsonName returns the josn name of the verrazzano component in CRD
func (i istioComponent) GetJSONName() string {
	return ComponentJSONName
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
func (i istioComponent) IsEnabled(effectiveCR *vzapi.Verrazzano) bool {
	comp := effectiveCR.Spec.Components.Istio
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

// ValidateInstall checks if the specified Verrazzano CR is valid for this component to be installed
func (i istioComponent) ValidateInstall(vz *vzapi.Verrazzano) error {
	return k8s.ValidateForExternalIPSWithNodePort(&vz.Spec, i.Name())
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (i istioComponent) ValidateUpdate(old *vzapi.Verrazzano, new *vzapi.Verrazzano) error {
	// Do not allow any changes except to enable the component post-install
	if i.IsEnabled(old) && !i.IsEnabled(new) {
		return fmt.Errorf("Disabling component %s is not allowed", ComponentJSONName)
	}
	// Reject any other edits except IstioInstallArgs
	if !reflect.DeepEqual(i.getIngressSettings(old), i.getIngressSettings(new)) {
		return fmt.Errorf("Updates to ingress not allowed for %s", ComponentJSONName)
	}
	if !reflect.DeepEqual(i.getEgressSettings(old), i.getEgressSettings(new)) {
		return fmt.Errorf("Updates to egress not allowed for %s", ComponentJSONName)
	}
	return nil
}

func (i istioComponent) getIngressSettings(vz *vzapi.Verrazzano) *vzapi.IstioIngressSection {
	if vz != nil && vz.Spec.Components.Istio != nil {
		return vz.Spec.Components.Istio.Ingress
	}
	return nil
}

func (i istioComponent) getEgressSettings(vz *vzapi.Verrazzano) *vzapi.IstioEgressSection {
	if vz != nil && vz.Spec.Components.Istio != nil {
		return vz.Spec.Components.Istio.Egress
	}
	return nil
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
	deployments := []types.NamespacedName{
		{
			Name:      IstiodDeployment,
			Namespace: IstioNamespace,
		},
		{
			Name:      IstioIngressgatewayDeployment,
			Namespace: IstioNamespace,
		},
		{
			Name:      IstioEgressgatewayDeployment,
			Namespace: IstioNamespace,
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

	return nil
}

func (i istioComponent) Reconcile(_ spi.ComponentContext) error {
	return nil
}

// GetIngressNames returns the list of ingress names associated with the component
func (i istioComponent) GetIngressNames(_ spi.ComponentContext) []types.NamespacedName {
	return []types.NamespacedName{}
}

// GetCertificateNames returns the list of expected certificates used by this component
func (i istioComponent) GetCertificateNames(_ spi.ComponentContext) []types.NamespacedName {
	return []types.NamespacedName{}
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
