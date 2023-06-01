// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhookoci

import (
	"fmt"

	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	vzresource "github.com/verrazzano/verrazzano/pkg/k8s/resource"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	ocidnsDeploymentName = "cert-manager-ocidns-provider"
)

// isCertManagerReady checks the state of the expected cert-manager deployments and returns true if they are in a ready state
func isCertManagerOciDNSReady(context spi.ComponentContext) bool {
	deployments := []types.NamespacedName{}
	if !vzcr.IsOCIDNSEnabled(context.EffectiveCR()) {
		context.Log().Oncef("OCI DNS is not enabled, skipping ready check")
		return true
	}
	deployments = append(deployments, types.NamespacedName{Name: ocidnsDeploymentName, Namespace: ComponentNamespace})
	prefix := fmt.Sprintf("Component %s", context.GetComponent())
	return ready.DeploymentsAreReady(context.Log(), context.Client(), deployments, 1, prefix)
}

func appendOCIDNSOverrides(ctx spi.ComponentContext, _ string, namespace string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {

	secretName, err := GetOCIDNSSecretName(ctx)
	if err != nil {
		return kvs, err
	}

	overrides := []bom.KeyValue{
		{Key: "ociAuthSecrets[0]", Value: secretName},
		{Key: "certManager.clusterResourceNamespace", Value: getClusterResourceNamespace(ctx.EffectiveCR())},
	}

	return append(kvs, overrides...), nil
}

func GetOCIDNSSecretName(ctx spi.ComponentContext) (string, error) {
	dns := ctx.EffectiveCR().Spec.Components.DNS
	if dns == nil || dns.OCI == nil {
		return "", ctx.Log().ErrorfThrottledNewErr("OCI DNS not configured")
	}
	ociDNS := dns.OCI
	if len(ociDNS.OCIConfigSecret) == 0 {
		return "", ctx.Log().ErrorfThrottledNewErr("OCI DNS auth secret not configured")
	}
	return ociDNS.OCIConfigSecret, nil
}

func (c certManagerWebhookOCIComponent) postUninstall(ctx spi.ComponentContext) error {
	// Clean up the OCI DNS secret in the clusterResourceNamespace
	dns := ctx.EffectiveCR().Spec.Components.DNS
	if dns == nil || dns.OCI == nil {
		return nil
	}
	ociDNS := dns.OCI

	err := vzresource.Resource{
		Name:      ociDNS.OCIConfigSecret,
		Namespace: getClusterResourceNamespace(ctx.EffectiveCR()),
		Client:    ctx.Client(),
		Object:    &corev1.Secret{},
		Log:       ctx.Log(),
	}.Delete()
	if err != nil {
		return err
	}
	return nil
}

func getClusterResourceNamespace(cr *vzapi.Verrazzano) string {
	if cr == nil || cr.Spec.Components.ClusterIssuer == nil {
		return constants.CertManagerNamespace
	}
	return cr.Spec.Components.ClusterIssuer.ClusterResourceNamespace
}
