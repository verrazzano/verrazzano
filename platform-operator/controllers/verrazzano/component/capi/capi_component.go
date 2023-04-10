// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package capi

import (
	"context"
	"encoding/base64"
	"fmt"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"os"
	clusterapi "sigs.k8s.io/cluster-api/cmd/clusterctl/client"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

// ComponentName is the name of the component
const ComponentName = "verrazzano-capi"

// ComponentNamespace is the namespace of the component
const ComponentNamespace = vzconst.CAPISystemNamespace

// ComponentJSONName is the JSON name of the verrazzano component in CRD
const ComponentJSONName = "verrazzano-capi"

const (
	ociDefaultSecret                    = "oci"
	capiCMDeployment                    = "capi-controller-manager"
	capiKubeadmBootstrapCMDeployment    = "capi-kubeadm-bootstrap-controller-manager"
	capiKubeadmControlPlaneCMDeployment = "capi-kubeadm-control-plane-controller-manager"
	capiOcneBootstrapCMDeployment       = "capi-ocne-bootstrap-controller-manager"
	capiOcneControlPlaneCMDeployment    = "capi-ocne-control-plane-controller-manager"
	capiociCMDeployment                 = "capoci-controller-manager"
)

var capiDeployments = []types.NamespacedName{ // TODO: add OLCNE deployments
	{
		Name:      capiCMDeployment,
		Namespace: ComponentNamespace,
	},
	{
		Name:      capiKubeadmBootstrapCMDeployment,
		Namespace: ComponentNamespace,
	},
	{
		Name:      capiKubeadmControlPlaneCMDeployment,
		Namespace: ComponentNamespace,
	},
	{
		Name:      capiOcneBootstrapCMDeployment,
		Namespace: ComponentNamespace,
	},
	{
		Name:      capiOcneControlPlaneCMDeployment,
		Namespace: ComponentNamespace,
	},
	{
		Name:      capiociCMDeployment,
		Namespace: ComponentNamespace,
	},
}

// AuthenticationType for auth
type AuthenticationType string

const (
	// UserPrincipal is default auth type
	UserPrincipal AuthenticationType = "user_principal"
	// InstancePrincipal is used for instance principle auth type
	InstancePrincipal AuthenticationType = "instance_principal"
)

// OCIAuthConfig holds connection parameters for the OCI API.
type OCIAuthConfig struct {
	Region      string             `yaml:"region"`
	Tenancy     string             `yaml:"tenancy"`
	User        string             `yaml:"user"`
	Key         string             `yaml:"key"`
	Fingerprint string             `yaml:"fingerprint"`
	Passphrase  string             `yaml:"passphrase"`
	AuthType    AuthenticationType `yaml:"authtype"`
}

// OCIConfig holds the configuration for OCI authorization.
type OCIConfig struct {
	Auth OCIAuthConfig `yaml:"auth"`
}

type capiComponent struct {
}

func NewComponent() spi.Component {
	return capiComponent{}
}

// Name returns the component name.
func (c capiComponent) Name() string {
	return ComponentName
}

// Namespace returns the component namespace.
func (c capiComponent) Namespace() string {
	return ComponentNamespace
}

// ShouldInstallBeforeUpgrade returns true if component can be installed before upgrade is done.
func (c capiComponent) ShouldInstallBeforeUpgrade() bool {
	return false
}

// GetDependencies returns the dependencies of this component.
func (c capiComponent) GetDependencies() []string {
	return []string{certmanager.ComponentName}
}

// IsReady indicates whether a component is Ready for dependency components.
func (c capiComponent) IsReady(ctx spi.ComponentContext) bool {
	prefix := fmt.Sprintf("Component %s", ctx.GetComponent())
	return ready.DeploymentsAreReady(ctx.Log(), ctx.Client(), capiDeployments, 1, prefix)
}

// IsAvailable indicates whether a component is Available for end users.
func (c capiComponent) IsAvailable(ctx spi.ComponentContext) (reason string, available v1alpha1.ComponentAvailability) {
	return (&ready.AvailabilityObjects{DeploymentNames: capiDeployments}).IsAvailable(ctx.Log(), ctx.Client())
}

// IsEnabled returns true if component is enabled for installation.
func (c capiComponent) IsEnabled(effectiveCR runtime.Object) bool {
	return true
	// TODO: uncomment when component is added to verrazzano API
	// return vzcr.IsCapiEnabled(effectiveCR)
}

// GetMinVerrazzanoVersion returns the minimum Verrazzano version required by the component
func (c capiComponent) GetMinVerrazzanoVersion() string {
	return constants.VerrazzanoVersion1_6_0
}

// GetIngressNames returns the list of ingress names associated with the component
func (c capiComponent) GetIngressNames(_ spi.ComponentContext) []types.NamespacedName {
	return []types.NamespacedName{}
}

// GetCertificateNames returns the list of expected certificates used by this component
func (c capiComponent) GetCertificateNames(_ spi.ComponentContext) []types.NamespacedName {
	return []types.NamespacedName{}
}

// GetJSONName returns the json name of the verrazzano component in CRD
func (c capiComponent) GetJSONName() string {
	return ComponentJSONName
}

// GetOverrides returns the Helm override sources for a component
func (c capiComponent) GetOverrides(object runtime.Object) interface{} {
	// TODO: update when capi component is added to Verrazzano API
	if _, ok := object.(*v1alpha1.Verrazzano); ok {
		//		if effectiveCR.Spec.Components.Capi != nil {
		//			return effectiveCR.Spec.Components.Capi.ValueOverrides
		//		}
		return []v1alpha1.Overrides{}
	}
	//effectiveCR := object.(*v1beta1.Verrazzano)
	//	if effectiveCR.Spec.Components.Capi != nil {
	//		return effectiveCR.Spec.Components.Capi.ValueOverrides
	//	}
	return []v1beta1.Overrides{}
}

// MonitorOverrides indicates whether monitoring of override sources is enabled for a component
func (c capiComponent) MonitorOverrides(_ spi.ComponentContext) bool {
	// TODO: update when capi component is added to Verrazzano API
	//	if ctx.EffectiveCR().Spec.Components.Capi == nil {
	//		return false
	//	}
	//	if ctx.EffectiveCR().Spec.Components.Capi.MonitorChanges != nil {
	//		return *ctx.EffectiveCR().Spec.Components.Istio.MonitorChanges
	//	}
	return true
}

func (c capiComponent) IsOperatorInstallSupported() bool {
	return true
}

// IsInstalled checks to see if CAPI is installed
func (c capiComponent) IsInstalled(ctx spi.ComponentContext) (bool, error) {
	daemonSet := &appsv1.Deployment{}
	err := ctx.Client().Get(context.TODO(), types.NamespacedName{Namespace: ComponentNamespace, Name: capiCMDeployment}, daemonSet)
	if errors.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		ctx.Log().Errorf("Failed to get %s/%s deployment: %v", ComponentNamespace, capiCMDeployment, err)
		return false, err
	}
	return true, nil
}

func (c capiComponent) PreInstall(ctx spi.ComponentContext) error {
	// Get OCI credentials from secret in the verrazzano-install namespace
	ociSecret := corev1.Secret{}
	// TODO: use secret name from API when available
	if err := ctx.Client().Get(context.TODO(), client.ObjectKey{Name: ociDefaultSecret, Namespace: constants.VerrazzanoInstallNamespace}, &ociSecret); err != nil {
		return ctx.Log().ErrorfNewErr("Failed to find secret %s in the %s namespace: %v", ociDefaultSecret, constants.VerrazzanoInstallNamespace, err)
	}

	var ociYaml []byte
	for key := range ociSecret.Data {
		if key == "oci.yaml" {
			ociYaml = ociSecret.Data[key]
			break
		}
	}

	if ociYaml == nil {
		return ctx.Log().ErrorfNewErr("Failed to find oci.yaml in secret %s in the %s namespace", ociDefaultSecret, constants.VerrazzanoInstallNamespace)
	}

	cfg := OCIConfig{}
	if err := yaml.Unmarshal(ociYaml, &cfg); err != nil {
		return ctx.Log().ErrorfNewErr("Failed to parse oci.yaml in secret %s in the %s namespace", ociDefaultSecret, constants.VerrazzanoInstallNamespace)
	}

	if cfg.Auth.AuthType == UserPrincipal {
		os.Setenv("OCI_TENANCY_ID_B64", base64.StdEncoding.EncodeToString([]byte(cfg.Auth.Tenancy)))
		os.Setenv("OCI_CREDENTIALS_FINGERPRINT_B64", base64.StdEncoding.EncodeToString([]byte(cfg.Auth.Fingerprint)))
		os.Setenv("OCI_USER_ID_B64", base64.StdEncoding.EncodeToString([]byte(cfg.Auth.User)))
		os.Setenv("OCI_REGION_B64", base64.StdEncoding.EncodeToString([]byte(cfg.Auth.Region)))
		os.Setenv("OCI_CREDENTIALS_KEY_B64", base64.StdEncoding.EncodeToString([]byte(cfg.Auth.Key)))
		if len(cfg.Auth.Passphrase) != 0 {
			os.Setenv("OCI_PASSPHRASE_B64", base64.StdEncoding.EncodeToString([]byte(cfg.Auth.Passphrase)))
		}
	} else if cfg.Auth.AuthType == InstancePrincipal {
		os.Setenv("USE_INSTANCE_PRINCIPAL_B64", base64.StdEncoding.EncodeToString([]byte("true")))
	} else {
		return ctx.Log().ErrorfNewErr("Invalid authtype value %s found for oci.yaml in secret %s in the %s namespace", cfg.Auth.AuthType, ociDefaultSecret, constants.VerrazzanoInstallNamespace)
	}

	return nil
}

func (c capiComponent) Install(_ spi.ComponentContext) error {
	capiClient, err := clusterapi.New("")
	if err != nil {
		return err
	}

	// TODO: version of providers should come from the BOM. Is kubeadm optional?
	// Set up the init options for the CAPI init.
	initOptions := clusterapi.InitOptions{
		BootstrapProviders:      []string{"ocne:v0.1.0", "kubeadm"},
		ControlPlaneProviders:   []string{"ocne:v0.1.0", "kubeadm"},
		InfrastructureProviders: []string{"oci:v0.8.0"},
		TargetNamespace:         ComponentNamespace,
	}

	_, err = capiClient.Init(initOptions)
	return err
}

func (c capiComponent) PostInstall(_ spi.ComponentContext) error {
	return nil
}

func (c capiComponent) IsOperatorUninstallSupported() bool {
	return true
}

func (c capiComponent) PreUninstall(_ spi.ComponentContext) error {
	return nil
}

func (c capiComponent) Uninstall(_ spi.ComponentContext) error {
	capiClient, err := clusterapi.New("")
	if err != nil {
		return err
	}

	// Set up the init options for the CAPI init.
	deleteOptions := clusterapi.DeleteOptions{
		DeleteAll:        true,
		IncludeNamespace: true,
	}
	return capiClient.Delete(deleteOptions)
}

func (c capiComponent) PostUninstall(_ spi.ComponentContext) error {
	return nil
}

func (c capiComponent) PreUpgrade(_ spi.ComponentContext) error {
	return nil
}

func (c capiComponent) Upgrade(_ spi.ComponentContext) error {
	return nil
}

func (c capiComponent) PostUpgrade(_ spi.ComponentContext) error {
	return nil
}

func (c capiComponent) ValidateInstall(vz *v1alpha1.Verrazzano) error {
	return nil
}

func (c capiComponent) ValidateUpdate(old *v1alpha1.Verrazzano, new *v1alpha1.Verrazzano) error {
	return nil
}

func (c capiComponent) ValidateInstallV1Beta1(vz *v1beta1.Verrazzano) error {
	return nil
}

func (c capiComponent) ValidateUpdateV1Beta1(old *v1beta1.Verrazzano, new *v1beta1.Verrazzano) error {
	return nil
}

// Reconcile reconciles the CAPI component
func (c capiComponent) Reconcile(ctx spi.ComponentContext) error {
	return nil
}
