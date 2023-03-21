// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package thanos

import (
	"fmt"

	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	// Thanos Query ingress constants
	queryHostName        = "thanos-query"
	queryCertificateName = "system-tls-thanos-query"

	// Thanos Query StoreAPI constants
	queryStoreHostName        = "query-store"
	queryStoreCertificateName = "system-tls-query-store"
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
		return kvs, err
	}
	kvs = append(kvs, image...)

	ingresses, err := appendIngressOverrides(ctx, kvs)
	if err != nil {
		return kvs, err
	}
	kvs = append(kvs, ingresses...)

	return kvs, nil
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
	}
	return formatIngressOverrides(ctx, storeProps, kvs), nil
}

// ingressOverrideProperties creates a structure to host Override property strings for Thanos ingresses
type ingressOverrideProperties struct {
	KeyPrefix        string
	Subdomain        string
	HostName         string
	IngressClassName string
	TLSSecretName    string
}

// formatIngressOverrides appends the correct overrides to a given ingress prefix based on generated properties for Ingress values
func formatIngressOverrides(ctx spi.ComponentContext, props ingressOverrideProperties, kvs []bom.KeyValue) []bom.KeyValue {
	kvs = append(kvs, []bom.KeyValue{
		{Key: fmt.Sprintf("%s.namespace", props.KeyPrefix), Value: constants.VerrazzanoSystemNamespace},
		{Key: fmt.Sprintf("%s.ingressClassName", props.KeyPrefix), Value: props.IngressClassName},
		{Key: fmt.Sprintf("%s.extraRules[0].host", props.KeyPrefix), Value: props.HostName},
		{Key: fmt.Sprintf("%s.extraTls[0].hosts[0]", props.KeyPrefix), Value: props.HostName},
		{Key: fmt.Sprintf("%s.extraTls[0].secretName", props.KeyPrefix), Value: props.TLSSecretName},
	}...)
	if vzcr.IsExternalDNSEnabled(ctx.EffectiveCR()) {
		ingressTarget := fmt.Sprintf("verrazzano-ingress.%s", props.Subdomain)
		kvs = append(kvs, []bom.KeyValue{
			{Key: fmt.Sprintf(`%s.annotations.external-dns\.alpha\.kubernetes\.io/target`, props.KeyPrefix), Value: ingressTarget},
			{Key: fmt.Sprintf(`%s.annotations.external-dns\.alpha\.kubernetes\.io/ttl`, props.KeyPrefix), Value: "60"},
		}...)
	}
	return kvs
}
