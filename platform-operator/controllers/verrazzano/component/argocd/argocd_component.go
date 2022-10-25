// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package argocd

import (
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/istio"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/networkpolicies"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/nginx"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/secret"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"path/filepath"
)

// ComponentName is the name of the component
const ComponentName = common.ArgoCDName

// ComponentNamespace is the namespace of the component
const ComponentNamespace = constants.ArgoCDNamespace

// ComponentJSONName is the josn name of the verrazzano component in CRD
const ComponentJSONName = "argocd"

type argoCDComponent struct {
	helm.HelmComponent
}

var certificates = []types.NamespacedName{
	{Name: "tls-argocd-ingress", Namespace: ComponentNamespace},
}

func NewComponent() spi.Component {
	return argoCDComponent{
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
			Dependencies:              []string{networkpolicies.ComponentName, istio.ComponentName, nginx.ComponentName, certmanager.ComponentName},
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
		return kvs, ctx.Log().ErrorNewErr("Failed to get the BOM file for the cert-manager image overrides: ", err)
	}
	images, err := bomFile.BuildImageOverrides("argocd")
	if err != nil {
		return kvs, err
	}

	kvs = append(kvs, bom.KeyValue{Key: "global.image.repository", Value: images[1].Value})
	kvs = append(kvs, bom.KeyValue{Key: "global.image.tag", Value: images[0].Value})
	kvs = append(kvs, bom.KeyValue{Key: "server.ingress.enabled", Value: "true"})

	return kvs, nil
}

// IsEnabled ArgoCD is always enabled on admin clusters,
// and is not enabled by default on managed clusters
func (r argoCDComponent) IsEnabled(effectiveCR runtime.Object) bool {
	return vzconfig.IsArgoCDEnabled(effectiveCR)
}

// IsReady component check
func (r argoCDComponent) IsReady(ctx spi.ComponentContext) bool {
	if r.HelmComponent.IsReady(ctx) {
		return isArgoCDReady(ctx)
	}
	return false
}

//PostInstall
/* Installs the Helm chart, and patches the resulting objects
- ensure Helm chart is installed
- Patch ArgoCD ingress with NGINX/TLS annotations
*/
func (r argoCDComponent) Install(ctx spi.ComponentContext) error {
	log := ctx.Log()
	if err := r.HelmComponent.Install(ctx); err != nil {
		return log.ErrorfThrottledNewErr("Failed retrieving ArgoCD install component: %s", err.Error())
	}
	// Annotate ArgoCD ingress for NGINX/TLS
	if err := patchArgoCDIngress(ctx); err != nil {
		return log.ErrorfThrottledNewErr("Failed patching ArgoCD ingress: %s", err.Error())
	}
	log.Debugf("Patched ArgoCD ingress")

	return nil
}
