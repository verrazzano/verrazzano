// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package thanos

import (
	"fmt"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"strconv"

	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	// Thanos Query ingress constants
	queryHostName        = "thanos-query"
	queryCertificateName = "system-tls-thanos-query"

	// Thanos Query StoreAPI constants
	queryStoreHostName        = "thanos-query-store"
	queryStoreCertificateName = "system-tls-query-store"

	// Thanos Ruler ingress constants
	rulerHostName        = "thanos-ruler"
	rulerCertificateName = "system-tls-thanos-ruler"

	configReloaderSubcomponentName = "prometheus-config-reloader"
)

// GetOverrides gets the install overrides for the Thanos component
func GetOverrides(object runtime.Object) interface{} {
	if effectiveCR, ok := object.(*vzapi.Verrazzano); ok {
		if effectiveCR.Spec.Components.Thanos != nil {
			return effectiveCR.Spec.Components.Thanos.ValueOverrides
		}
		return []vzapi.Overrides{}
	} else if effectiveCR, ok := object.(*v1beta1.Verrazzano); ok {
		if effectiveCR.Spec.Components.Thanos != nil {
			return effectiveCR.Spec.Components.Thanos.ValueOverrides
		}
		return []v1beta1.Overrides{}
	}
	return []vzapi.Overrides{}
}

// AppendOverrides appends the default overrides for the Thanos component
func AppendOverrides(ctx spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	bomFile, err := bom.NewBom(config.GetDefaultBOMFilePath())
	if err != nil {
		return kvs, err
	}

	image, err := bomFile.BuildImageOverrides(ComponentName)
	if err != nil {
		return kvs, ctx.Log().ErrorfNewErr("Failed to build Thanos image overrides from the Verrazzano BOM: %v", err)
	}
	kvs = append(kvs, image...)

	kvs = appendVerrazzanoOverrides(ctx, kvs)

	kvs, err = appendReloaderSidecarOverrides(ctx, kvs, bomFile)
	if err != nil {
		return kvs, ctx.Log().ErrorfNewErr("Failed to build Thanos sidecar overrides from the Verrazzano BOM: %v", err)
	}

	return appendIngressOverrides(ctx, kvs)
}

// appendVerrazzanoOverrides appends overrides that are specific to Verrazzano
// i.e. .Values.verrazzano.*. To start with, there is just one (verrazzano.isIstioEnabled)
func appendVerrazzanoOverrides(ctx spi.ComponentContext, kvs []bom.KeyValue) []bom.KeyValue {
	enabled := vzcr.IsIstioInjectionEnabled(ctx.EffectiveCR())
	// isIstioEnabled is used in the Helm chart to determine whether the Thanos service monitors
	// should use Istio TLS config
	kvs = append(kvs, bom.KeyValue{Key: "verrazzano.isIstioEnabled", Value: strconv.FormatBool(enabled)})
	return kvs
}

// preInstallUpgrade handles pre-install and pre-upgrade processing for the Thanos Component
func preInstallUpgrade(ctx spi.ComponentContext) error {
	// Do nothing if dry run
	if ctx.IsDryRun() {
		ctx.Log().Debug("Thanos preInstallUpgrade dry run")
		return nil
	}

	// Create the verrazzano-monitoring namespace if not already created
	ctx.Log().Debugf("Creating namespace %s for Thanos", constants.VerrazzanoMonitoringNamespace)
	return common.EnsureVerrazzanoMonitoringNamespace(ctx)
}

// appendReloaderSidecarOverrides generates overrides for the Prometheus Config Reloader sidecar for the Thanos Ruler
func appendReloaderSidecarOverrides(ctx spi.ComponentContext, kvs []bom.KeyValue, bomFile bom.Bom) ([]bom.KeyValue, error) {
	subcomponent, err := bomFile.GetSubcomponent(configReloaderSubcomponentName)
	if err != nil {
		return kvs, ctx.Log().ErrorfNewErr("Failed to get subcomponent %s from the Verrazzano BOM: %v", configReloaderSubcomponentName, err)
	}

	if subcomponent == nil {
		return kvs, ctx.Log().ErrorfNewErr("Failed to find subcomponent %s", configReloaderSubcomponentName)
	}
	if len(subcomponent.Images) != 1 {
		return kvs, ctx.Log().ErrorfNewErr("Failed to get config reloader image from the bom, %d images returned when only one is expected", len(subcomponent.Images))
	}

	image := subcomponent.Images[0]
	imageString := constructFullImageString(bomFile, *subcomponent, image)
	sidecarPrefix := "ruler.sidecars[0]"
	kvs = append(kvs, []bom.KeyValue{
		{Key: fmt.Sprintf("%s.image", sidecarPrefix), Value: imageString},
		{Key: fmt.Sprintf("%s.name", sidecarPrefix), Value: configReloaderSubcomponentName},
		{Key: fmt.Sprintf("%s.args[0]", sidecarPrefix), Value: "--listen-address=:8080"},
		{Key: fmt.Sprintf("%s.args[1]", sidecarPrefix), Value: "--reload-url=http://127.0.0.1:10902/-/reload"},
		{Key: fmt.Sprintf("%s.args[2]", sidecarPrefix), Value: "--watched-dir=/conf/rules/"},
		{Key: fmt.Sprintf("%s.volumeMounts[0].mountPath", sidecarPrefix), Value: "/conf/rules/"},
		{Key: fmt.Sprintf("%s.volumeMounts[0].name", sidecarPrefix), Value: "ruler-config"},
		{Key: "ruler.configmapName", Value: "prometheus-prometheus-operator-kube-p-prometheus-rulefiles-0"},

		// Security setup for the sidecar
		{Key: fmt.Sprintf("%s.securityContext.privileged", sidecarPrefix), Value: "false"},
		{Key: fmt.Sprintf("%s.securityContext.capabilities.drop[0]", sidecarPrefix), Value: "ALL"},
	}...)

	return kvs, nil
}

func constructFullImageString(bomFile bom.Bom, subcomponent bom.BomSubComponent, image bom.BomImage) string {
	registry := func() string {
		if image.Registry != "" {
			return image.Registry
		}
		if subcomponent.Registry != "" {
			return subcomponent.Registry
		}
		return bomFile.GetRegistry()
	}()

	repository := func() string {
		if image.Repository != "" {
			return image.Repository
		}
		return subcomponent.Repository
	}()

	return fmt.Sprintf("%s/%s/%s:%s", registry, repository, image.ImageName, image.ImageTag)
}

// appendIngressOverrides generates overrides for ingress objects in the Thanos component
func appendIngressOverrides(ctx spi.ComponentContext, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	// If NGINX is disabled, prevent the ingresses from being created
	if !vzcr.IsNGINXEnabled(ctx.EffectiveCR()) {
		return append(kvs, []bom.KeyValue{
			{Key: "query.ingress.grpc.enabled", Value: "false"},
			{Key: "queryFrontend.ingress.enabled", Value: "false"},
		}...), nil
	}

	ingressClassName := vzconfig.GetIngressClassName(ctx.EffectiveCR())
	dnsSubDomain, err := vzconfig.BuildDNSDomain(ctx.Client(), ctx.EffectiveCR())
	if err != nil {
		return kvs, ctx.Log().ErrorfNewErr("Failed building DNS domain name for Thanos Ingress: %v", err)
	}

	frontendHostName := fmt.Sprintf("%s.%s", queryHostName, dnsSubDomain)
	frontendProps := ingressOverrideProperties{
		KeyPrefix:        "queryFrontend.ingress",
		Subdomain:        dnsSubDomain,
		HostName:         frontendHostName,
		IngressClassName: ingressClassName,
		TLSSecretName:    queryCertificateName,
		Path:             "/()(.*)",
		PathType:         netv1.PathTypeImplementationSpecific,
		ServicePort:      constants.VerrazzanoAuthProxyServicePort,
	}
	kvs = formatIngressOverrides(ctx, frontendProps, kvs)
	kvs = append(kvs, bom.KeyValue{Key: `queryFrontend.ingress.annotations.nginx\.ingress\.kubernetes\.io/session-cookie-name`, Value: frontendHostName})

	storeHostName := fmt.Sprintf("%s.%s", queryStoreHostName, dnsSubDomain)
	storeProps := ingressOverrideProperties{
		KeyPrefix:        "query.ingress.grpc",
		Subdomain:        dnsSubDomain,
		HostName:         storeHostName,
		IngressClassName: ingressClassName,
		TLSSecretName:    queryStoreCertificateName,
		Path:             "/",
		PathType:         netv1.PathTypeImplementationSpecific,
		ServicePort:      constants.VerrazzanoAuthProxyGRPCServicePort,
	}
	kvs = formatIngressOverrides(ctx, storeProps, kvs)

	rulerHostName := fmt.Sprintf("%s.%s", rulerHostName, dnsSubDomain)
	rulerProps := ingressOverrideProperties{
		KeyPrefix:        "ruler.ingress",
		Subdomain:        dnsSubDomain,
		HostName:         rulerHostName,
		IngressClassName: ingressClassName,
		TLSSecretName:    rulerCertificateName,
		Path:             "/()(.*)",
		PathType:         netv1.PathTypeImplementationSpecific,
		ServicePort:      constants.VerrazzanoAuthProxyServicePort,
	}
	kvs = append(kvs, bom.KeyValue{Key: "ruler.extraFlags[0]", Value: fmt.Sprintf("--alert.query-url=https://%s", frontendHostName)})
	return formatIngressOverrides(ctx, rulerProps, kvs), nil

}

// ingressOverrideProperties creates a structure to host Override property strings for Thanos ingresses
type ingressOverrideProperties struct {
	KeyPrefix        string
	Subdomain        string
	HostName         string
	IngressClassName string
	TLSSecretName    string
	Path             string
	PathType         netv1.PathType
	ServicePort      int
}

// formatIngressOverrides appends the correct overrides to a given ingress prefix based on generated properties for Ingress values
func formatIngressOverrides(ctx spi.ComponentContext, props ingressOverrideProperties, kvs []bom.KeyValue) []bom.KeyValue {
	kvs = append(kvs, []bom.KeyValue{
		{Key: fmt.Sprintf("%s.namespace", props.KeyPrefix), Value: constants.VerrazzanoSystemNamespace},
		{Key: fmt.Sprintf("%s.ingressClassName", props.KeyPrefix), Value: props.IngressClassName},
		{Key: fmt.Sprintf("%s.extraRules[0].host", props.KeyPrefix), Value: props.HostName},
		{Key: fmt.Sprintf("%s.extraRules[0].http.paths[0].backend.service.name", props.KeyPrefix), Value: constants.VerrazzanoAuthProxyServiceName},
		{Key: fmt.Sprintf("%s.extraRules[0].http.paths[0].backend.service.port.number", props.KeyPrefix), Value: strconv.Itoa(props.ServicePort)},
		{Key: fmt.Sprintf("%s.extraRules[0].http.paths[0].path", props.KeyPrefix), Value: props.Path},
		{Key: fmt.Sprintf("%s.extraRules[0].http.paths[0].pathType", props.KeyPrefix), Value: string(props.PathType)},
		{Key: fmt.Sprintf("%s.extraTls[0].hosts[0]", props.KeyPrefix), Value: props.HostName},
		{Key: fmt.Sprintf("%s.extraTls[0].secretName", props.KeyPrefix), Value: props.TLSSecretName},
	}...)
	ingressTarget := fmt.Sprintf("verrazzano-ingress.%s", props.Subdomain)
	if vzcr.IsExternalDNSEnabled(ctx.EffectiveCR()) {
		kvs = append(kvs, []bom.KeyValue{
			{Key: fmt.Sprintf(`%s.annotations.external-dns\.alpha\.kubernetes\.io/target`, props.KeyPrefix), Value: ingressTarget},
			{Key: fmt.Sprintf(`%s.annotations.external-dns\.alpha\.kubernetes\.io/ttl`, props.KeyPrefix), Value: "60", SetString: true},
		}...)
	}
	kvs = append(kvs, []bom.KeyValue{
		{Key: fmt.Sprintf(`%s.annotations.cert-manager\.io/common-name`, props.KeyPrefix), Value: props.HostName},
		{Key: fmt.Sprintf(`%s.annotations.cert-manager\.io/cluster-issuer`, props.KeyPrefix), Value: vzconst.VerrazzanoClusterIssuerName},
	}...)
	return kvs
}
