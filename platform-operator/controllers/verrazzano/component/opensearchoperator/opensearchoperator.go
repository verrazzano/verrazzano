// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearchoperator

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

const (
	opensearchOperatorDeploymentName = "opensearch-operator-controller-manager"
	opensearchHostName               = "opensearch.vmi.system"
	osdHostName                      = "osd.vmi.system"
	osIngressName                    = "opensearch"
	osdIngressName                   = "opensearch-dashboards"

	opsterOSService = "opensearch"

	opsterOSDService = "opensearch-dashboards"

	securityconfigSecretName = "securityconfig-secret"
)

func (o opensearchOperatorComponent) isReady(ctx spi.ComponentContext) bool {
	return ready.DeploymentsAreReady(ctx.Log(), ctx.Client(), getDeploymentList(), 1, getPrefix(ctx))
}

// GetOverrides gets the install overrides
func GetOverrides(object runtime.Object) interface{} {
	if effectiveCR, ok := object.(*vzapi.Verrazzano); ok {
		if effectiveCR.Spec.Components.OpenSearchOperator != nil {
			return effectiveCR.Spec.Components.OpenSearchOperator.ValueOverrides
		}
		return []vzapi.Overrides{}
	} else if effectiveCR, ok := object.(*installv1beta1.Verrazzano); ok {
		if effectiveCR.Spec.Components.OpenSearchOperator != nil {
			return effectiveCR.Spec.Components.OpenSearchOperator.ValueOverrides
		}
		return []installv1beta1.Overrides{}
	}

	return []vzapi.Overrides{}
}

func AppendOverrides(ctx spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	// TODO: Image overrides once the BFS images are done

	// Bootstrap pod overrides
	kvs = append(kvs, bom.KeyValue{
		Key:   `openSearchCluster.bootstrap.additionalConfig.cluster\.initial_master_nodes`,
		Value: fmt.Sprintf("%s-%s-0", clusterName, getMasterNode(ctx)),
	})

	kvs, err := buildIngressOverrides(ctx, kvs)
	if err != nil {
		return kvs, ctx.Log().ErrorfNewErr("Failed to build ingress overrides: %v", err)
	}

	// NodePool Overrides
	// Merge configuration from OpenSearch and OpenSearchDashboards components

	return kvs, nil
}

func buildIngressOverrides(ctx spi.ComponentContext, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	if vzcr.IsNGINXEnabled(ctx.EffectiveCR()) {

		dnsSubDomain, err := vzconfig.BuildDNSDomain(ctx.Client(), ctx.EffectiveCR())
		if err != nil {
			return kvs, ctx.Log().ErrorfNewErr("Failed to build DNS subdomain: %v", err)
		}
		ingressClassName := vzconfig.GetIngressClassName(ctx.EffectiveCR())
		ingressTarget := fmt.Sprintf("verrazzano-ingress.%s", dnsSubDomain)

		ingressAnnotations := make(map[string]string)
		ingressAnnotations[`kubernetes\.io/tls-acme`] = "true"
		ingressAnnotations[`nginx\.ingress\.kubernetes\.io/proxy-body-size`] = "65M"
		ingressAnnotations[`nginx\.ingress\.kubernetes\.io/rewrite-target`] = "/$2"
		ingressAnnotations[`nginx\.ingress\.kubernetes\.io/service-upstream`] = "true"
		ingressAnnotations[`nginx\.ingress\.kubernetes\.io/upstream-vhost`] = "${service_name}.${namespace}.svc.cluster.local"
		ingressAnnotations[`cert-manager\.io/cluster-issuer`] = constants.VerrazzanoClusterIssuerName
		if vzcr.IsExternalDNSEnabled(ctx.EffectiveCR()) {
			ingressAnnotations[`external-dns\.alpha\.kubernetes\.io/target`] = ingressTarget
			ingressAnnotations[`external-dns\.alpha\.kubernetes\.io/ttl`] = "60"
		}

		kvs, err = appendOSIngressOverrides(ingressAnnotations, dnsSubDomain, ingressClassName, kvs)
		kvs, err = appendOSDIngressOverrides(ingressAnnotations, dnsSubDomain, ingressClassName, kvs)

	} else {
		kvs = append(kvs, bom.KeyValue{
			Key:   "ingress.openSearch.enable",
			Value: "false",
		})
		kvs = append(kvs, bom.KeyValue{
			Key:   "ingress.openSearchDashboards.enable",
			Value: "false",
		})
	}

	return kvs, nil
}

func appendOSDIngressOverrides(ingressAnnotations map[string]string, dnsSubDomain, ingressClassName string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	osdHostName := buildOSDHostnameForDomain(dnsSubDomain)
	ingressAnnotations[`cert-manager\.io/common-name`] = osdHostName

	kvs = append(kvs, bom.KeyValue{
		Key:   "ingress.openSearchDashboards.ingressClassName",
		Value: ingressClassName,
	})
	kvs = append(kvs, bom.KeyValue{
		Key:   "ingress.openSearchDashboards.host",
		Value: osdHostName,
	})

	annotationsKey := "ingress.openSearchDashboards.annotations"
	for key, value := range ingressAnnotations {
		kvs = append(kvs, bom.KeyValue{
			Key:       fmt.Sprintf("%s.%s", annotationsKey, key),
			Value:     value,
			SetString: true,
		})
	}

	kvs = append(kvs, bom.KeyValue{
		Key:   "ingress.openSearchDashboards.tls[0].secretName",
		Value: "system-tls-osd",
	})

	kvs = append(kvs, bom.KeyValue{
		Key:   "ingress.openSearchDashboards.tls[0].hosts[0]",
		Value: osdHostName,
	})

	return kvs, nil
}

func appendOSIngressOverrides(ingressAnnotations map[string]string, dnsSubDomain, ingressClassName string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	opensearchHostName := buildOSHostnameForDomain(dnsSubDomain)
	ingressAnnotations[`cert-manager\.io/common-name`] = opensearchHostName

	kvs = append(kvs, bom.KeyValue{
		Key:   "ingress.openSearch.ingressClassName",
		Value: ingressClassName,
	})
	kvs = append(kvs, bom.KeyValue{
		Key:   "ingress.openSearch.host",
		Value: opensearchHostName,
	})

	annotationsKey := "ingress.openSearch.annotations"
	for key, value := range ingressAnnotations {
		kvs = append(kvs, bom.KeyValue{
			Key:       fmt.Sprintf("%s.%s", annotationsKey, key),
			Value:     value,
			SetString: true,
		})
	}

	kvs = append(kvs, bom.KeyValue{
		Key:   "ingress.openSearch.tls[0].secretName",
		Value: "system-tls-os-ingest",
	})

	kvs = append(kvs, bom.KeyValue{
		Key:   "ingress.openSearch.tls[0].hosts[0]",
		Value: opensearchHostName,
	})

	return kvs, nil
}

func getMasterNode(ctx spi.ComponentContext) string {
	nodes := ctx.EffectiveCR().Spec.Components.Elasticsearch.Nodes

	for _, node := range nodes {
		for _, role := range node.Roles {
			if node.Replicas != nil && *node.Replicas <= 0 {
				continue
			}
			if role == "master" || role == "cluster_manager" {
				return node.Name
			}
		}
	}
	return ""
}

func GetMergedNodes(ctx spi.ComponentContext) {
	//legacyNodes := getNodePoolFromNodes(ctx.EffectiveCR().Spec.Components.Elasticsearch.Nodes)
	//currentNodes := getNodesFromOverrides(ctx.EffectiveCR().Spec.Components.OpenSearchOperator)
}

func getDeploymentList() []types.NamespacedName {
	return []types.NamespacedName{
		{
			Name:      opensearchOperatorDeploymentName,
			Namespace: ComponentNamespace,
		},
	}
}

func getIngressList() []types.NamespacedName {
	return []types.NamespacedName{
		{
			Name:      osIngressName,
			Namespace: ComponentNamespace,
		},
		{
			Name:      osdIngressName,
			Namespace: ComponentNamespace,
		},
	}
}

func buildOSHostnameForDomain(dnsDomain string) string {
	return fmt.Sprintf("%s.%s", opensearchHostName, dnsDomain)
}

func buildOSDHostnameForDomain(dnsDomain string) string {
	return fmt.Sprintf("%s.%s", osdHostName, dnsDomain)
}

func getPrefix(ctx spi.ComponentContext) string {
	return fmt.Sprintf("Component %s", ctx.GetComponent())
}
