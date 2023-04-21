// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package nginx

import (
	ctx "context"
	"fmt"
	helm2 "github.com/verrazzano/verrazzano/pkg/helm"
	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/pkg/k8s/resource"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	vpoconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/istio"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/networkpolicies"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/secret"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"path/filepath"
	"sigs.k8s.io/yaml"
)

// ComponentName is the name of the component
const ComponentName = "ingress-controller"

// ComponentNamespace is the namespace of the component
const ComponentNamespace = vpoconst.IngressNginxNamespace

// ComponentJSONName is the JSON name of the verrazzano component in CRD
const ComponentJSONName = "ingressNGINX"

// nginxExternalIPKey is the nginxInstallArgs key for externalIPs
const nginxExternalIPKey = "controller.service.externalIPs"

const nginxExternalIPJsonPath = "controller.service.externalIPs.0"

// nginxComponent represents an Nginx component
type nginxComponent struct {
	helm.HelmComponent
}

// Verify that nginxComponent implements Component
var _ spi.Component = nginxComponent{}

// NewComponent returns a new Nginx component
func NewComponent() spi.Component {
	return nginxComponent{
		helm.HelmComponent{
			ReleaseName:               ComponentName,
			JSONName:                  ComponentJSONName,
			ChartDir:                  filepath.Join(config.GetThirdPartyDir(), "ingress-nginx"), // Note name is different than release name
			ChartNamespace:            ComponentNamespace,
			IgnoreNamespaceOverride:   true,
			SupportsOperatorInstall:   true,
			SupportsOperatorUninstall: true,
			ImagePullSecretKeyname:    secret.DefaultImagePullSecretKeyName,
			ValuesFile:                filepath.Join(config.GetHelmOverridesDir(), ValuesFileOverride),
			PreInstallFunc:            PreInstall,
			AppendOverridesFunc:       AppendOverrides,
			PostInstallFunc:           PostInstall,
			Dependencies:              []string{networkpolicies.ComponentName, istio.ComponentName},
			GetInstallOverridesFunc:   GetOverrides,
			AvailabilityObjects: &ready.AvailabilityObjects{
				DeploymentNames: []types.NamespacedName{
					{
						Name:      ControllerName,
						Namespace: ComponentNamespace,
					},
					{
						Name:      backendName,
						Namespace: ComponentNamespace,
					},
				},
			},
		},
	}
}

// IsInstalled Indicates whether the component is installed
func (c nginxComponent) IsInstalled(context spi.ComponentContext) (bool, error) {
	// Check if Verrazzano NGINX is installed in the ingress-nginx namespace
	installed, err := c.isNGINXInstalledInOldNamespace(context)
	if err != nil {
		context.Log().ErrorfNewErr("Error checking if the old ingress-nginx chart %s/%s is installed error: %v", vpoconst.OldIngressNginxNamespace, c.ReleaseName, err.Error())
	}
	if installed {
		return true, nil
	}

	return c.HelmComponent.IsInstalled(context)
}

// IsEnabled nginx-specific enabled check for installation
func (c nginxComponent) IsEnabled(effectiveCR runtime.Object) bool {
	return vzcr.IsNGINXEnabled(effectiveCR)
}

// IsReady component check
func (c nginxComponent) IsReady(ctx spi.ComponentContext) bool {
	if c.HelmComponent.IsReady(ctx) {
		return c.isNginxReady(ctx)
	}
	return false
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (c nginxComponent) ValidateUpdate(old *vzapi.Verrazzano, new *vzapi.Verrazzano) error {
	if c.IsEnabled(old) && !c.IsEnabled(new) {
		return fmt.Errorf("Disabling component %s is not allowed", ComponentJSONName)
	}
	if err := c.HelmComponent.ValidateUpdate(old, new); err != nil {
		return err
	}
	return c.validateForExternalIPSWithNodePort(new)
}

// ValidateInstall checks if the specified Verrazzano CR is valid for this component to be installed
func (c nginxComponent) ValidateInstall(vz *vzapi.Verrazzano) error {
	if err := c.HelmComponent.ValidateInstall(vz); err != nil {
		return err
	}
	return c.validateForExternalIPSWithNodePort(vz)
}

// ValidateInstallV1Beta1 checks if the specified Verrazzano CR is valid for this component to be installed
func (c nginxComponent) ValidateInstallV1Beta1(vz *v1beta1.Verrazzano) error {
	if err := c.HelmComponent.ValidateInstallV1Beta1(vz); err != nil {
		return err
	}
	return c.validateForExternalIPSWithNodePortV1Beta1(vz)
}

// ValidateUpdateV1Beta1 checks if the specified new Verrazzano CR is valid for this component to be updated
func (c nginxComponent) ValidateUpdateV1Beta1(old *v1beta1.Verrazzano, new *v1beta1.Verrazzano) error {
	if c.IsEnabled(old) && !c.IsEnabled(new) {
		return fmt.Errorf("disabling component %s is not allowed", ComponentJSONName)
	}
	if err := c.HelmComponent.ValidateUpdateV1Beta1(old, new); err != nil {
		return err
	}
	return c.validateForExternalIPSWithNodePortV1Beta1(new)
}

// validateForExternalIPSWithNodePort checks that externalIPs are set when Type=NodePort
func (c nginxComponent) validateForExternalIPSWithNodePort(vz *vzapi.Verrazzano) error {
	// good if ingress is not set
	if vz.Spec.Components.Ingress == nil {
		return nil
	}

	// good if type is not NodePort
	if vz.Spec.Components.Ingress.Type != vzapi.NodePort {
		return nil
	}

	// look for externalIPs if NodePort
	if vz.Spec.Components.Ingress.Type == vzapi.NodePort {
		return vzconfig.CheckExternalIPsArgs(vz.Spec.Components.Ingress.NGINXInstallArgs, vz.Spec.Components.Ingress.ValueOverrides, nginxExternalIPKey, nginxExternalIPJsonPath, c.Name(), vz.Namespace)
	}

	return nil
}

// validateForExternalIPSWithNodePort checks that externalIPs are set when Type=NodePort
func (c nginxComponent) validateForExternalIPSWithNodePortV1Beta1(vz *v1beta1.Verrazzano) error {
	// good if ingress is not set
	if vz.Spec.Components.IngressNGINX == nil {
		return nil
	}

	// good if type is not NodePort
	if vz.Spec.Components.IngressNGINX.Type != v1beta1.NodePort {
		return nil
	}

	// look for externalIPs if NodePort
	if vz.Spec.Components.IngressNGINX.Type == v1beta1.NodePort {
		return vzconfig.CheckExternalIPsOverridesArgs(vz.Spec.Components.IngressNGINX.ValueOverrides, nginxExternalIPJsonPath, c.Name(), vz.Namespace)
	}

	return nil
}

// MonitorOverrides checks whether monitoring of install overrides is enabled or not
func (c nginxComponent) MonitorOverrides(ctx spi.ComponentContext) bool {
	if ctx.EffectiveCR().Spec.Components.Ingress != nil {
		if ctx.EffectiveCR().Spec.Components.Ingress.MonitorChanges != nil {
			return *ctx.EffectiveCR().Spec.Components.Ingress.MonitorChanges
		}
		return true
	}
	return false
}

// PostUninstall processing for NGINX
func (c nginxComponent) PostUninstall(context spi.ComponentContext) error {
	res := resource.Resource{
		Name:   ComponentNamespace,
		Client: context.Client(),
		Object: &corev1.Namespace{},
		Log:    context.Log(),
	}
	// Remove finalizers from the ingress-nginx namespace to avoid hanging namespace deletion
	// and delete the namespace
	return res.RemoveFinalizersAndDelete()
}

// PreUpgrade processing for NGINX
func (c nginxComponent) PreUpgrade(context spi.ComponentContext) error {
	// Check if Verrazzano NGINX is installed in the ingress-nginx namespace
	installed, err := c.isNGINXInstalledInOldNamespace(context)
	if err != nil {
		context.Log().ErrorfNewErr("Error checking if the old ingress-nginx chart %s/%s is installed error: %v", vpoconst.OldIngressNginxNamespace, c.ReleaseName, err.Error())
	}
	if !installed {
		return nil
	}

	// The old verrazzano ingress-nginx chart is installed in ingress-nginx.  Uninstall it.
	err = helm2.Uninstall(context.Log(), c.ReleaseName, vpoconst.OldIngressNginxNamespace, context.IsDryRun())
	if err != nil {
		context.Log().Errorf("Error uninstalling the old ingress-nginx chart %s/%s, error: %v", vpoconst.OldIngressNginxNamespace, c.ReleaseName, err.Error())
		return err
	}
	context.Log().Progressf("Uninstalled the old Verrazzano ingress-nginx chart %s/%s", vpoconst.OldIngressNginxNamespace, c.ReleaseName)

	// Remove the old network policies
	if err := c.deleteNetworkPolicy(context, vpoconst.OldIngressNginxNamespace, "ingress-nginx-controller"); err != nil {
		return err
	}
	if err := c.deleteNetworkPolicy(context, vpoconst.OldIngressNginxNamespace, " ingress-nginx-default-backend"); err != nil {
		return err
	}
	return nil
}

func (c nginxComponent) deleteNetworkPolicy(context spi.ComponentContext, namespace string, name string) error {
	// Remove the old network policies
	net := netv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	err := context.Client().Delete(ctx.TODO(), &net)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return context.Log().ErrorfNewErr("Failed to delete the old network policy %s/%s: %v", namespace, name, err)
	}
	return nil
}

// See if Verrazzano NGINX is installed in ingress-nginx
func (c nginxComponent) isNGINXInstalledInOldNamespace(context spi.ComponentContext) (bool, error) {
	const vzClass = "verrazzano-nginx"

	type YamlConfig struct {
		Controller struct {
			IngressClassResource struct {
				Name string `json:"name"`
			}
		}
	}

	// See if NGINX is installed in the ingress-nginx namespace
	found, err := helm2.IsReleaseInstalled(c.ReleaseName, vpoconst.OldIngressNginxNamespace)
	if err != nil {
		context.Log().ErrorfNewErr("Error checking if the old ingress-nginx chart %s/%s is installed error: %v", vpoconst.OldIngressNginxNamespace, c.ReleaseName, err.Error())
	}
	if found {
		valMap, err := helm2.GetValuesMap(context.Log(), c.ReleaseName, vpoconst.OldIngressNginxNamespace)
		if err != nil {
			return false, err
		}
		b, err := yaml.Marshal(&valMap)
		if err != nil {
			return false, err
		}
		yml := YamlConfig{}
		if err := yaml.Unmarshal(b, &yml); err != nil {
			return false, err
		}
		if yml.Controller.IngressClassResource.Name == vzClass {
			return true, nil
		}

		return false, nil
	}
	return false, nil
}
