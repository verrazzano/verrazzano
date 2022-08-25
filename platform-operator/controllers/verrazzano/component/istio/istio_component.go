// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8s/webhook"
	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"path/filepath"
	"strings"

	"github.com/verrazzano/verrazzano/pkg/bom"
	ctrlerrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
	"github.com/verrazzano/verrazzano/pkg/helm"
	"github.com/verrazzano/verrazzano/pkg/istio"
	"github.com/verrazzano/verrazzano/pkg/k8s/resource"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/secret"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
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

// IstioNamespace is the default Istio namespace
const IstioNamespace = "istio-system"

// IstioCoreDNSReleaseName is the name of the istiocoredns release
const IstioCoreDNSReleaseName = "istiocoredns"

// HelmScrtType is the secret type that helm uses to specify its releases
const HelmScrtType = "helm.sh/release.v1"

// subcompIstiod is the Istiod subcomponent in the bom
const subcompIstiod = "istiod"

// This IstioOperator YAML uses this imagePullSecret key
const imagePullSecretHelmKey = "values.global.imagePullSecrets[0]"

// istioManfiestNotInstalledError - Expected error during install when running verify-install before Istio CR is applied
const istioManfiestNotInstalledError = "Istio present but verify-install needs an IstioOperator or manifest for comparison"

const istioReaderIstioSystem = "istio-reader-istio-system"
const istiodIstioSystem = "istiod-istio-system"

const istioSidecarMutatingWebhook = "istio-sidecar-injector"

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

// Namespace returns the component namespace
func (i istioComponent) Namespace() string {
	return IstioNamespace
}

// GetJSONName returns the json name of the verrazzano component in CRD
func (i istioComponent) GetJSONName() string {
	return ComponentJSONName
}

// GetOverrides returns the Helm override sources for a component
func (i istioComponent) GetOverrides(object runtime.Object) interface{} {
	if effectiveCR, ok := object.(*vzapi.Verrazzano); ok {
		if effectiveCR.Spec.Components.Istio != nil {
			return effectiveCR.Spec.Components.Istio.ValueOverrides
		}
		return []vzapi.Overrides{}
	} else if effectiveCR, ok := object.(*installv1beta1.Verrazzano); ok {
		if effectiveCR.Spec.Components.Istio != nil {
			return effectiveCR.Spec.Components.Istio.ValueOverrides
		}
		return []installv1beta1.Overrides{}
	}

	return []vzapi.Overrides{}
}

// MonitorOverrides indicates whether monitoring of override sources is enabled for a component
func (i istioComponent) MonitorOverrides(ctx spi.ComponentContext) bool {
	if ctx.EffectiveCR().Spec.Components.Istio == nil {
		return false
	}
	if ctx.EffectiveCR().Spec.Components.Istio.MonitorChanges != nil {
		return *ctx.EffectiveCR().Spec.Components.Istio.MonitorChanges
	}
	return true
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

type istioUninstallFuncSig func(log vzlog.VerrazzanoLogger) (stdout []byte, stderr []byte, err error)

var istioUninstallFunc istioUninstallFuncSig = istio.Uninstall

func SetIstioUninstallFunction(fn istioUninstallFuncSig) {
	istioUninstallFunc = fn
}

func SetDefaultIstioUninstallFunction() {
	istioUninstallFunc = istio.Uninstall
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

func (i istioComponent) IsOperatorUninstallSupported() bool {
	return true
}

func (i istioComponent) PreUninstall(_ spi.ComponentContext) error {
	return nil
}

// Uninstall processing for Istio
func (i istioComponent) Uninstall(context spi.ComponentContext) error {
	_, _, err := istioUninstallFunc(context.Log())
	return err
}

// PostUninstall processing for Istio
func (i istioComponent) PostUninstall(context spi.ComponentContext) error {
	// Delete ClusterRoleBindings and ClusterRoles not removed with istioctl uninstall
	err := resource.Resource{
		Name:   istioReaderIstioSystem,
		Client: context.Client(),
		Object: &rbacv1.ClusterRoleBinding{},
		Log:    context.Log(),
	}.Delete()
	if err != nil {
		return err
	}

	err = resource.Resource{
		Name:   istioReaderIstioSystem,
		Client: context.Client(),
		Object: &rbacv1.ClusterRole{},
		Log:    context.Log(),
	}.Delete()
	if err != nil {
		return err
	}

	err = resource.Resource{
		Name:   istiodIstioSystem,
		Client: context.Client(),
		Object: &rbacv1.ClusterRoleBinding{},
		Log:    context.Log(),
	}.Delete()
	if err != nil {
		return err
	}

	err = resource.Resource{
		Name:   istiodIstioSystem,
		Client: context.Client(),
		Object: &rbacv1.ClusterRole{},
		Log:    context.Log(),
	}.Delete()
	if err != nil {
		return err
	}

	res := resource.Resource{
		Name:   IstioNamespace,
		Client: context.Client(),
		Object: &v1.Namespace{},
		Log:    context.Log(),
	}
	// Remove finalizers from the istio-system namespace to avoid hanging namespace deletion
	// and delete the namespace
	return res.RemoveFinalizersAndDelete()
}

// IsEnabled istio-specific enabled check for installation
func (i istioComponent) IsEnabled(effectiveCR runtime.Object) bool {
	return vzconfig.IsIstioEnabled(effectiveCR)
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
	// Validate install overrides
	if vz.Spec.Components.Istio != nil {
		if err := vzapi.ValidateInstallOverrides(vz.Spec.Components.Istio.ValueOverrides); err != nil {
			return err
		}
	}

	return i.validateForExternalIPSWithNodePort(&vz.Spec)
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (i istioComponent) ValidateUpdate(old *vzapi.Verrazzano, new *vzapi.Verrazzano) error {
	if i.IsEnabled(old) && !i.IsEnabled(new) {
		return fmt.Errorf("Disabling component %s is not allowed", ComponentJSONName)
	}
	// Validate install overrides
	if new.Spec.Components.Istio != nil {
		if err := vzapi.ValidateInstallOverrides(new.Spec.Components.Istio.ValueOverrides); err != nil {
			return err
		}
	}
	return i.validateForExternalIPSWithNodePort(&new.Spec)
}

// ValidateInstall checks if the specified Verrazzano CR is valid for this component to be installed
func (i istioComponent) ValidateUpdateV1Beta1(old *installv1beta1.Verrazzano, new *installv1beta1.Verrazzano) error {
	return nil
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (i istioComponent) ValidateInstallV1Beta1(vz *installv1beta1.Verrazzano) error {
	return nil
}

// validateForExternalIPSWithNodePort checks that externalIPs are set when Type=NodePort
func (i istioComponent) validateForExternalIPSWithNodePort(vz *vzapi.VerrazzanoSpec) error {
	// good if istio or istio.ingress is not set
	if vz.Components.Istio == nil || vz.Components.Istio.Ingress == nil {
		return nil
	}

	// good if type is not NodePort
	if vz.Components.Istio.Ingress.Type != vzapi.NodePort {
		return nil
	}

	// look for externalIPs if NodePort
	if vz.Components.Istio.Ingress.Type == vzapi.NodePort {
		return vzconfig.CheckExternalIPsArgs(vz.Components.Istio.IstioInstallArgs, vz.Components.Istio.ValueOverrides, ExternalIPArg, externalIPJsonPath, i.Name())
	}

	return nil
}

func (i istioComponent) Upgrade(context spi.ComponentContext) error {
	log := context.Log()

	// build list of temp files
	istioTempFiles, err := i.createIstioTempFiles(context)
	if err != nil {
		return err
	}

	defer removeTempFiles(context.Log())

	// build image override strings
	overrideStrings, err := getOverridesString(context)
	if err != nil {
		return err
	}
	_, _, err = upgradeFunc(log, overrideStrings, istioTempFiles...)
	if err != nil {
		return err
	}

	return err
}

func (i istioComponent) IsReady(context spi.ComponentContext) bool {
	prefix := fmt.Sprintf("Component %s", context.GetComponent())
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
	ready := status.DeploymentsAreReady(context.Log(), context.Client(), deployments, 1, prefix)
	if !ready {
		return false
	}

	verified, err := isInstalledFunc(context.Log())
	if err != nil && !isIstioManifestNotInstalledError(err) {
		context.Log().ErrorfThrottled("Unexpected error checking Istio status: %s", err)
		return false
	}
	if !verified {
		context.Log().Progressf("%s is waiting for istioctl verify-install to successfully complete", prefix)
		return false
	}

	// istioctl verify-install does not check that the Load Balancer service has an external IP address,
	// so we have to check this manually to get a useful error message
	_, err = verifyIstioIngressGatewayIP(context.Client(), context.EffectiveCR())
	if err != nil {
		// Only log for the Istio component context
		if context.GetComponent() == ComponentName {
			context.Log().Errorf("Ingress external IP pending for component %s: %v", ComponentName, err)
		}
		return false
	}
	return true
}

func isIstioManifestNotInstalledError(err error) bool {
	return strings.Contains(err.Error(), istioManfiestNotInstalledError)
}

// GetDependencies returns the dependencies of this component
func (i istioComponent) GetDependencies() []string {
	return []string{}
}

func (i istioComponent) PreUpgrade(context spi.ComponentContext) error {
	if vzconfig.IsApplicationOperatorEnabled(context.ActualCR()) {
		context.Log().Infof("Stopping WebLogic domains that are have Envoy 1.7.3 sidecar")
		if err := StopDomainsUsingOldEnvoy(context.Log(), context.Client()); err != nil {
			return err
		}
	}
	//Upgrading Istio may result in a duplicate mutating webhook configuration. Istioctl will recreate the webhook during upgrade.
	return webhook.DeleteMutatingWebhookConfiguration(context.Log(), context.Client(), istioSidecarMutatingWebhook)
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

func (i istioComponent) Reconcile(ctx spi.ComponentContext) error {
	return i.Upgrade(ctx)
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

func getOverridesString(ctx spi.ComponentContext) (string, error) {
	var kvs []bom.KeyValue
	// check for global image pull secret
	kvs, err := secret.AddGlobalImagePullSecretHelmOverride(ctx.Log(), ctx.Client(), IstioNamespace, kvs, imagePullSecretHelmKey)
	if err != nil {
		return "", err
	}

	// Build comma separated string of overrides that will be passed to
	// isioctl as --set values.
	// This include BOM image overrides as well as other overrides
	return buildOverridesString(kvs...)
}

func buildOverridesString(additionalValues ...bom.KeyValue) (string, error) {
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
