// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"
	"fmt"
	"path/filepath"

	globalconst "github.com/verrazzano/verrazzano/pkg/constants"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/authproxy"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/istio"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/nginx"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// ComponentName is the name of the component
	ComponentName = "verrazzano"

	// ComponentNamespace is the namespace of the component
	ComponentNamespace = constants.VerrazzanoSystemNamespace

	// vzImagePullSecretKeyName is the Helm key name for the VZ chart image pull secret
	vzImagePullSecretKeyName = "global.imagePullSecrets[0]"

	// Certificate names
	verrazzanoCertificateName = "verrazzano-tls"

	prometheusCertificateName = "system-tls-prometheus"

	// ES secret keys
	esUsernameKey = "username"
	esPasswordKey = "password"
)

// ComponentJSONName is the josn name of the verrazzano component in CRD
const ComponentJSONName = "verrazzano"

var getControllerRuntimeClient = getClient

type verrazzanoComponent struct {
	helm.HelmComponent
}

func NewComponent() spi.Component {
	return verrazzanoComponent{
		helm.HelmComponent{
			ReleaseName:             ComponentName,
			JSONName:                ComponentJSONName,
			ChartDir:                filepath.Join(config.GetHelmChartsDir(), ComponentName),
			ChartNamespace:          ComponentNamespace,
			IgnoreNamespaceOverride: true,
			ResolveNamespaceFunc:    resolveVerrazzanoNamespace,
			AppendOverridesFunc:     appendVerrazzanoOverrides,
			ImagePullSecretKeyname:  vzImagePullSecretKeyName,
			SupportsOperatorInstall: true,
			Dependencies:            []string{istio.ComponentName, nginx.ComponentName, certmanager.ComponentName, authproxy.ComponentName},
			GetInstallOverridesFunc: GetOverrides,
		},
	}
}

// PreInstall Verrazzano component pre-install processing; create and label required namespaces, copy any
// required secrets
func (c verrazzanoComponent) PreInstall(ctx spi.ComponentContext) error {
	if vzconfig.IsVMOEnabled(ctx.EffectiveCR()) {
		// Make sure the VMI CRD is installed since the Verrazzano component may create/update
		// a VMI CR
		if err := common.ApplyCRDYaml(ctx, config.GetHelmVMOChartsDir()); err != nil {
			return err
		}
	}
	// create or update  VMI secret
	if err := common.EnsureVMISecret(ctx.Client()); err != nil {
		return err
	}
	// create or update  backup secret
	if err := common.EnsureBackupSecret(ctx.Client()); err != nil {
		return err
	}
	ctx.Log().Debug("Verrazzano pre-install")
	if err := createAndLabelNamespaces(ctx); err != nil {
		return ctx.Log().ErrorfNewErr("Failed creating/labeling namespaces for Verrazzano: %v", err)
	}
	if err := loggingPreInstall(ctx); err != nil {
		return ctx.Log().ErrorfNewErr("Failed copying logging secrets for Verrazzano: %v", err)
	}
	return nil
}

// Install Verrazzano component install processing
func (c verrazzanoComponent) Install(ctx spi.ComponentContext) error {
	if err := c.HelmComponent.Install(ctx); err != nil {
		return err
	}
	return common.CreateOrUpdateVMI(ctx, updateFunc)
}

// PreUpgrade Verrazzano component pre-upgrade processing
func (c verrazzanoComponent) PreUpgrade(ctx spi.ComponentContext) error {
	if vzconfig.IsVMOEnabled(ctx.EffectiveCR()) {
		if err := common.ExportVMOHelmChart(ctx); err != nil {
			return err
		}
		if err := common.ApplyCRDYaml(ctx, config.GetHelmVMOChartsDir()); err != nil {
			return err
		}
	}
	return verrazzanoPreUpgrade(ctx, ComponentNamespace)
}

// Upgrade Verrazzano component upgrade processing
func (c verrazzanoComponent) Upgrade(ctx spi.ComponentContext) error {
	if err := c.HelmComponent.Upgrade(ctx); err != nil {
		return err
	}
	return common.CreateOrUpdateVMI(ctx, updateFunc)
}

// IsReady component check
func (c verrazzanoComponent) IsReady(ctx spi.ComponentContext) bool {
	if c.HelmComponent.IsReady(ctx) {
		return isVerrazzanoReady(ctx)
	}
	return false
}

// IsInstalled component check
func (c verrazzanoComponent) IsInstalled(ctx spi.ComponentContext) (bool, error) {
	installed, _ := c.HelmComponent.IsInstalled(ctx)
	if installed {
		return doesPromExist(ctx), nil
	}
	return false, nil
}

// PostInstall - post-install, clean up temp files
func (c verrazzanoComponent) PostInstall(ctx spi.ComponentContext) error {
	cleanTempFiles(ctx)
	// populate the ingress and certificate names before calling PostInstall on Helm component because those will be needed there
	c.HelmComponent.IngressNames = c.GetIngressNames(ctx)
	c.HelmComponent.Certificates = c.GetCertificateNames(ctx)
	return c.HelmComponent.PostInstall(ctx)
}

// PostUpgrade Verrazzano-post-upgrade processing
func (c verrazzanoComponent) PostUpgrade(ctx spi.ComponentContext) error {
	ctx.Log().Debugf("Verrazzano component post-upgrade")
	cleanTempFiles(ctx)
	c.HelmComponent.IngressNames = c.GetIngressNames(ctx)
	c.HelmComponent.Certificates = c.GetCertificateNames(ctx)
	if vzconfig.IsVMOEnabled(ctx.EffectiveCR()) {
		if err := common.ReassociateVMOResources(ctx); err != nil {
			return err
		}
	}
	return c.HelmComponent.PostUpgrade(ctx)
}

// IsEnabled verrazzano-specific enabled check for installation
func (c verrazzanoComponent) IsEnabled(effectiveCR *vzapi.Verrazzano) bool {
	comp := effectiveCR.Spec.Components.Verrazzano
	if comp == nil || comp.Enabled == nil {
		return true
	}
	return *comp.Enabled
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (c verrazzanoComponent) ValidateUpdate(old *vzapi.Verrazzano, new *vzapi.Verrazzano) error {
	// Do not allow disabling active components
	if err := c.checkEnabled(old, new); err != nil {
		return err
	}
	// Reject any other edits except InstallArgs
	// Do not allow any updates to storage settings via the volumeClaimSpecTemplates/defaultVolumeSource
	if err := common.CompareStorageOverrides(old, new, ComponentJSONName); err != nil {
		return err
	}
	if err := validateFluentd(new); err != nil {
		return err
	}
	return c.HelmComponent.ValidateUpdate(old, new)
}

// ValidateInstall checks if the specified Verrazzano CR is valid for this component to be installed
func (c verrazzanoComponent) ValidateInstall(vz *vzapi.Verrazzano) error {
	if err := validateFluentd(vz); err != nil {
		return err
	}
	return c.HelmComponent.ValidateInstall(vz)
}

// existing Fluentd mount paths can be found at platform-operator/helm_config/charts/verrazzano/templates/verrazzano-logging.yaml
var existingFluentdMountPaths = [7]string{
	"/fluentd/cacerts", "/fluentd/secret", "/fluentd/etc",
	"/root/.oci", "/var/log", "/var/lib", "/run/log/journal"}

func validateFluentd(vz *vzapi.Verrazzano) error {
	fluentd := vz.Spec.Components.Fluentd
	if fluentd == nil {
		return nil
	}
	if err := validateExtraVolumeMounts(fluentd); err != nil {
		return err
	}
	if err := validateLogCollector(fluentd); err != nil {
		return err
	}
	return nil
}

func (c verrazzanoComponent) checkEnabled(old *vzapi.Verrazzano, new *vzapi.Verrazzano) error {
	// Do not allow disabling of any component post-install for now
	if c.IsEnabled(old) && !c.IsEnabled(new) {
		return fmt.Errorf("Disabling component %s is not allowed", ComponentJSONName)
	}
	if vzconfig.IsConsoleEnabled(old) && !vzconfig.IsConsoleEnabled(new) {
		return fmt.Errorf("Disabling component console not allowed")
	}
	if vzconfig.IsPrometheusEnabled(old) && !vzconfig.IsPrometheusEnabled(new) {
		return fmt.Errorf("Disabling component prometheus not allowed")
	}
	return nil
}

// GetIngressNames - gets the names of the ingresses associated with this component
func (c verrazzanoComponent) GetIngressNames(ctx spi.ComponentContext) []types.NamespacedName {
	var ingressNames []types.NamespacedName

	if vzconfig.IsPrometheusEnabled(ctx.EffectiveCR()) {
		ingressNames = append(ingressNames, types.NamespacedName{
			Namespace: ComponentNamespace,
			Name:      constants.PrometheusIngress,
		})
	}

	return ingressNames
}

// GetCertificateNames - gets the names of the ingresses associated with this component
func (c verrazzanoComponent) GetCertificateNames(ctx spi.ComponentContext) []types.NamespacedName {
	var certificateNames []types.NamespacedName

	certificateNames = append(certificateNames, types.NamespacedName{
		Namespace: ComponentNamespace,
		Name:      verrazzanoCertificateName,
	})

	if vzconfig.IsPrometheusEnabled(ctx.EffectiveCR()) {
		certificateNames = append(certificateNames, types.NamespacedName{
			Namespace: ComponentNamespace,
			Name:      prometheusCertificateName,
		})
	}

	return certificateNames
}

// getClient returns a controller runtime client for the Verrazzano resource
func getClient() (client.Client, error) {
	config, err := ctrl.GetConfig()
	if err != nil {
		return nil, err
	}
	return client.New(config, client.Options{Scheme: newScheme()})
}

// newScheme creates a new scheme that includes this package's object for use by client
func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	vzapi.AddToScheme(scheme)
	clientgoscheme.AddToScheme(scheme)
	return scheme
}

func getInstallSecret(client client.Client, secretName string, secret *corev1.Secret) error {
	err := client.Get(context.TODO(), types.NamespacedName{Name: secretName, Namespace: constants.VerrazzanoInstallNamespace}, secret)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return fmt.Errorf("secret \"%s\" must be created in the \"%s\" namespace", secretName, constants.VerrazzanoInstallNamespace)
		}
		return err
	}
	return nil
}

func validateExtraVolumeMounts(fluentd *vzapi.FluentdComponent) error {
	if len(fluentd.ExtraVolumeMounts) > 0 {
		for _, vm := range fluentd.ExtraVolumeMounts {
			mountPath := vm.Source
			if vm.Destination != "" {
				mountPath = vm.Destination
			}
			for _, existing := range existingFluentdMountPaths {
				if mountPath == existing {
					return fmt.Errorf("duplicate mount path found: %s; Fluentd by default has mount paths: %v", mountPath, existingFluentdMountPaths)
				}
			}
		}
	}
	return nil
}

func validateLogCollector(fluentd *vzapi.FluentdComponent) error {
	if fluentd.OCI != nil && fluentd.ElasticsearchURL != globalconst.DefaultOpensearchURL && fluentd.ElasticsearchURL != "" {
		return fmt.Errorf("fluentd config does not allow both OCI %v and external Opensearch %v", fluentd.OCI, fluentd.ElasticsearchURL)
	}
	if err := validateLogCollectorSecret(fluentd); err != nil {
		return err
	}
	return nil
}

func validateLogCollectorSecret(fluentd *vzapi.FluentdComponent) error {
	if len(fluentd.ElasticsearchSecret) > 0 && fluentd.ElasticsearchSecret != globalconst.VerrazzanoESInternal {
		cli, err := getControllerRuntimeClient()
		if err != nil {
			return err
		}
		secret := &corev1.Secret{}
		if err := getInstallSecret(cli, fluentd.ElasticsearchSecret, secret); err != nil {
			return err
		}
		if err := validateEntryExist(secret, esUsernameKey); err != nil {
			return err
		}
		return validateEntryExist(secret, esPasswordKey)
	}
	return nil
}

func validateEntryExist(secret *corev1.Secret, entry string) error {
	secretName := secret.Name
	_, ok := secret.Data[entry]
	if !ok {
		return fmt.Errorf("invalid Fluentd configuration, missing %s entry in secret \"%s\"", entry, secretName)
	}
	return nil
}

// MonitorOverrides checks whether monitoring of install overrides is enabled or not
func (c verrazzanoComponent) MonitorOverrides(ctx spi.ComponentContext) bool {
	if ctx.EffectiveCR().Spec.Components.Verrazzano != nil {
		if ctx.EffectiveCR().Spec.Components.Verrazzano.MonitorChanges != nil {
			return *ctx.EffectiveCR().Spec.Components.Verrazzano.MonitorChanges
		}
		return true
	}
	return false
}
