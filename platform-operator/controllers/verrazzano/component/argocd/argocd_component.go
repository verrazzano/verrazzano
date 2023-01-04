// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package argocd

import (
	"fmt"
	"path/filepath"

	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/istio"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/keycloak"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/networkpolicies"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/nginx"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/secret"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

// ComponentName is the name of the component
const ComponentName = common.ArgoCDName

// ComponentNamespace is the namespace of the component
const ComponentNamespace = constants.ArgoCDNamespace

// ComponentJSONName is the josn name of the verrazzano component in CRD
const ComponentJSONName = "argocd"

type argoCDComponent struct {
	keycloak.ArgoClientSecretProvider
	helm.HelmComponent
}

var certificates = []types.NamespacedName{
	{Name: "tls-argocd-ingress", Namespace: ComponentNamespace},
}

func NewComponent() spi.Component {
	return argoCDComponent{
		ArgoClientSecretProvider: keycloak.DefaultArgoClientSecretProvider{},
		HelmComponent: helm.HelmComponent{
			ReleaseName:               common.ArgoCDName,
			JSONName:                  ComponentJSONName,
			ChartDir:                  filepath.Join(config.GetThirdPartyDir(), common.ArgoCDName),
			ChartNamespace:            ComponentNamespace,
			IgnoreNamespaceOverride:   true,
			SupportsOperatorInstall:   true,
			SupportsOperatorUninstall: true,
			ImagePullSecretKeyname:    secret.DefaultImagePullSecretKeyName,
			ValuesFile:                filepath.Join(config.GetHelmOverridesDir(), "argocd-values.yaml"),
			AppendOverridesFunc:       AppendOverrides,
			Certificates:              certificates,
			Dependencies:              []string{networkpolicies.ComponentName, istio.ComponentName, nginx.ComponentName, certmanager.ComponentName, keycloak.ComponentName},
			AvailabilityObjects: &ready.AvailabilityObjects{
				DeploymentNames: []types.NamespacedName{
					{
						Name:      common.ArgoCDApplicationSetController,
						Namespace: constants.ArgoCDNamespace,
					},
					{
						Name:      common.ArgoCDNotificationController,
						Namespace: constants.ArgoCDNamespace,
					},
					{
						Name:      common.ArgoCDRedis,
						Namespace: constants.ArgoCDNamespace,
					},
					{
						Name:      common.ArgoCDRepoServer,
						Namespace: constants.ArgoCDNamespace,
					},
					{
						Name:      common.ArgoCDServer,
						Namespace: constants.ArgoCDNamespace,
					},
				},
			},
			IngressNames: []types.NamespacedName{
				{
					Namespace: ComponentNamespace,
					Name:      constants.ArgoCDIngress,
				},
			},
			GetInstallOverridesFunc: GetOverrides,
		},
	}
}

// AppendOverrides set the ArgoCD overrides for Helm
func AppendOverrides(ctx spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	bomFile, err := bom.NewBom(config.GetDefaultBOMFilePath())
	if err != nil {
		return nil, err
	}
	images, err := bomFile.BuildImageOverrides("argocd")
	if err != nil {
		return kvs, err
	}

	kvs = append(kvs, bom.KeyValue{Key: "global.image.repository", Value: images[1].Value})
	kvs = append(kvs, bom.KeyValue{Key: "global.image.tag", Value: images[0].Value})

	images, err = bomFile.BuildImageOverrides("redis")
	if err != nil {
		return kvs, err
	}

	kvs = append(kvs, bom.KeyValue{Key: "redis.image.repository", Value: images[1].Value})
	kvs = append(kvs, bom.KeyValue{Key: "redis.image.tag", Value: images[0].Value})

	return kvs, nil
}

// IsEnabled ArgoCD is disabled by default on admin clusters,
// and is not enabled by default on managed clusters
func (c argoCDComponent) IsEnabled(effectiveCR runtime.Object) bool {
	return vzcr.IsArgoCDEnabled(effectiveCR)
}

// IsReady component check
func (c argoCDComponent) IsReady(ctx spi.ComponentContext) bool {
	if c.HelmComponent.IsReady(ctx) {
		return isArgoCDReady(ctx)
	}
	return false
}

// PreInstall applies the Application Operator CRDs
func (c argoCDComponent) PreInstall(ctx spi.ComponentContext) error {
	if err := common.ApplyCRDYaml(ctx, config.GetHelmAppOpChartsDir()); err != nil {
		return err
	}
	return c.HelmComponent.PreInstall(ctx)
}

//Install
/* Installs the Helm chart, and patches the resulting objects
- ensure Helm chart is installed
- Patch ArgoCD ingress with NGINX/TLS annotations
*/
func (c argoCDComponent) Install(ctx spi.ComponentContext) error {
	log := ctx.Log()
	if err := c.HelmComponent.Install(ctx); err != nil {
		return log.ErrorfThrottledNewErr("Failed retrieving ArgoCD install component: %s", err.Error())
	}
	// Annotate ArgoCD ingress for NGINX/TLS
	if err := patchArgoCDIngress(ctx); err != nil {
		return log.ErrorfThrottledNewErr("Failed patching ArgoCD ingress: %s", err.Error())
	}
	log.Debugf("Patched ArgoCD ingress")

	return nil
}

// PostInstall
/* Additional setup for ArgoCD after the component is installed
- Patch argocd-rbac-cm by providing role admin to verrazzano-admins group
- Patch argocd-cm with tls-argocd-ingress secret root ca
- Patch argocd-secret with the keycloak client secret
*/
func (c argoCDComponent) PostInstall(ctx spi.ComponentContext) error {
	log := ctx.Log()
	if err := c.HelmComponent.PostInstall(ctx); err != nil {
		return log.ErrorfThrottledNewErr("Failed retrieving ArgoCD post install component: %s", err.Error())
	}

	return nil
}

// ConfigureKeycloakOIDC
// +configures Keycloak as OIDC provider for ArgoCD.
// - Patch argocd-secret with the keycloak client secret.
// - Patch argocd-cm with the oidc configuration to enable keycloak authentication.
// - Patch argocd-rbac-cm by providing role admin to verrazzano-admins group
func ConfigureKeycloakOIDC(ctx spi.ComponentContext) error {
	log := ctx.Log()

	if vzcr.IsKeycloakEnabled(ctx.EffectiveCR()) && vzcr.IsArgoCDEnabled(ctx.EffectiveCR()) {

		ctx.Log().Oncef("Configuring Keycloak as a ArgoCD authentication provider")
		component := NewComponent().(argoCDComponent)
		if err := patchArgoCDSecret(component, ctx); err != nil {
			return log.ErrorfThrottledNewErr("Failed patching ArgoCD secret: %s", err.Error())
		}
		log.Debugf("Patched ArgoCD secret")

		if err := patchArgoCDConfigMap(ctx); err != nil {
			return err
		}
		log.Debugf("Patched ArgoCD configmap")

		if err := patchArgoCDRbacConfigMap(ctx); err != nil {
			return err
		}
		log.Debugf("Patched ArgoCD RBac configmap")

		if err := restartArgoCDServerDeploy(ctx); err != nil {
			return err
		}
	}

	return nil
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (c argoCDComponent) ValidateUpdate(old *vzapi.Verrazzano, new *vzapi.Verrazzano) error {
	// Block all changes for now, particularly around storage changes
	if c.IsEnabled(old) && !c.IsEnabled(new) {
		return fmt.Errorf("Disabling component %s is not allowed", ComponentJSONName)
	}
	return c.HelmComponent.ValidateUpdate(old, new)
}

// ValidateUpdateV1Beta1 checks if the specified new Verrazzano CR is valid for this component to be updated
func (c argoCDComponent) ValidateUpdateV1Beta1(old *installv1beta1.Verrazzano, new *installv1beta1.Verrazzano) error {
	// Block all changes for now, particularly around storage changes
	if c.IsEnabled(old) && !c.IsEnabled(new) {
		return fmt.Errorf("Disabling component %s is not allowed", ComponentJSONName)
	}
	return c.HelmComponent.ValidateUpdateV1Beta1(old, new)
}

// ValidateInstall checks if the specified Verrazzano CR is valid for this component to be installed
func (c argoCDComponent) ValidateInstall(vz *vzapi.Verrazzano) error {
	vzV1Beta1 := &installv1beta1.Verrazzano{}

	if err := vz.ConvertTo(vzV1Beta1); err != nil {
		return err
	}

	return c.ValidateInstallV1Beta1(vzV1Beta1)
}

// ValidateInstallV1Beta1 checks if the specified Verrazzano CR is valid for this component to be installed
func (c argoCDComponent) ValidateInstallV1Beta1(vz *installv1beta1.Verrazzano) error {
	return c.HelmComponent.ValidateInstallV1Beta1(vz)
}
