// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearchoperator

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	opensearchOperatorDeploymentName = "opensearch-operator-controller-manager"
	opensearchHostName               = "opensearch.vmi.system"
	osdHostName                      = "osd.vmi.system"
	// Certificate names
	osdCertificateName = "system-tls-osd"
	osCertificateName  = "system-tls-os-ingest"
	OpsterGroup        = "opensearch.opster.io"
)

var (
	clusterCertificates = []types.NamespacedName{
		{Name: fmt.Sprintf("%s-admin-cert", clusterName), Namespace: ComponentNamespace},
		{Name: fmt.Sprintf("%s-dashboards-cert", clusterName), Namespace: ComponentNamespace},
		{Name: fmt.Sprintf("%s-master-cert", clusterName), Namespace: ComponentNamespace},
		{Name: fmt.Sprintf("%s-node-cert", clusterName), Namespace: ComponentNamespace}}

	GetControllerRuntimeClient = GetClient
	clusterGVR                 = schema.GroupVersionResource{
		Group:    OpsterGroup,
		Resource: "opensearchclusters",
		Version:  "v1",
	}

	roleGVR = schema.GroupVersionResource{
		Group:    OpsterGroup,
		Resource: "opensearchroles",
		Version:  "v1",
	}

	rolesMappingGVR = schema.GroupVersionResource{
		Group:    OpsterGroup,
		Resource: "opensearchuserrolebindings",
		Version:  "v1",
	}

	gvrList = []schema.GroupVersionResource{clusterGVR, roleGVR, rolesMappingGVR}
)

// GetOverrides gets the list of overrides
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

// appendOverrides appends the additional overrides for install
func appendOverrides(ctx spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {

	kvs, err := buildIngressOverrides(ctx, kvs)
	if err != nil {
		return kvs, ctx.Log().ErrorfNewErr("Failed to build ingress overrides: %v", err)
	}
	clusterResourceNamespace := constants.CertManagerNamespace
	if ctx.EffectiveCR().Spec.Components.ClusterIssuer != nil && ctx.EffectiveCR().Spec.Components.ClusterIssuer.ClusterResourceNamespace != "" {
		clusterResourceNamespace = ctx.EffectiveCR().Spec.Components.ClusterIssuer.ClusterResourceNamespace
	}
	kvs = append(kvs, bom.KeyValue{
		Key:   "clusterResourceNamespace",
		Value: clusterResourceNamespace,
	})
	return kvs, nil
}

// deleteRelatedResource deletes the resources handled by the opensearchOperator
// Like OpenSearchRoles, OpenSearchUserRolesBindings
// Since the operator adds a finalizer to these resources, they need to deleted before the operator is uninstalled
func (o opensearchOperatorComponent) deleteRelatedResource() error {
	client, err := k8sutil.GetDynamicClient()
	if err != nil {
		return fmt.Errorf("failed to create dynamic client: %v", err)
	}

	for _, gvr := range gvrList {
		resourceClient := client.Resource(gvr)
		objList, err := resourceClient.Namespace(ComponentNamespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return fmt.Errorf("failed to list %s: %v", gvr.String(), err)
		}

		for _, obj := range objList.Items {
			err = resourceClient.Namespace(ComponentNamespace).Delete(context.TODO(), obj.GetName(), metav1.DeleteOptions{})
			if err != nil {
				return fmt.Errorf("failed to delete %s: %v", gvr.String(), err)
			}
		}
	}
	return nil
}

// areRelatedResourcesDeleted checks if the related resources are deleted or not
func (o opensearchOperatorComponent) areRelatedResourcesDeleted() error {
	client, err := k8sutil.GetDynamicClient()
	if err != nil {
		return err
	}

	for _, gvr := range gvrList {
		resourceClient := client.Resource(gvr)
		objList, err := resourceClient.Namespace(ComponentNamespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return err
		}
		if len(objList.Items) > 0 {
			return fmt.Errorf("waiting for all %s to be deleted", gvr.String())
		}
	}
	return nil
}

// buildIngressOverrides builds the overrides required for the OpenSearch and OpenSearchDashboards ingresses
func buildIngressOverrides(ctx spi.ComponentContext, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	if vzcr.IsNGINXEnabled(ctx.EffectiveCR()) {
		dnsSubDomain, err := vzconfig.BuildDNSDomain(ctx.Client(), ctx.EffectiveCR())
		if err != nil {
			return kvs, ctx.Log().ErrorfNewErr("Failed to build DNS subdomain: %v", err)
		}
		ingressClassName := vzconfig.GetIngressClassName(ctx.EffectiveCR())
		ingressTarget := fmt.Sprintf("verrazzano-ingress.%s", dnsSubDomain)

		ingressAnnotations := make(map[string]string)
		ingressAnnotations[`cert-manager\.io/cluster-issuer`] = constants.VerrazzanoClusterIssuerName
		if vzcr.IsExternalDNSEnabled(ctx.EffectiveCR()) {
			ingressAnnotations[`external-dns\.alpha\.kubernetes\.io/target`] = ingressTarget
			ingressAnnotations[`external-dns\.alpha\.kubernetes\.io/ttl`] = "60"
		}

		path := "ingress.opensearchDashboards"
		kvs = appendIngressOverrides(ingressAnnotations, path, buildOSDHostnameForDomain(dnsSubDomain), osdCertificateName, ingressClassName, kvs)

		path = "ingress.opensearch"
		kvs = appendIngressOverrides(ingressAnnotations, path, buildOSHostnameForDomain(dnsSubDomain), osCertificateName, ingressClassName, kvs)

	} else {
		kvs = append(kvs, bom.KeyValue{
			Key:   "ingress.opensearch.enable",
			Value: "false",
		})
		kvs = append(kvs, bom.KeyValue{
			Key:   "ingress.opensearchDashboards.enable",
			Value: "false",
		})
	}

	return kvs, nil
}

// appendIngressOverrides appends the required overrides for the ingresses
func appendIngressOverrides(ingressAnnotations map[string]string, path, hostName, tlsSecret, ingressClassName string, kvs []bom.KeyValue) []bom.KeyValue {
	ingressAnnotations[`cert-manager\.io/common-name`] = hostName

	kvs = append(kvs, bom.KeyValue{
		Key:   fmt.Sprintf("%s.ingressClassName", path),
		Value: ingressClassName,
	})
	kvs = append(kvs, bom.KeyValue{
		Key:   fmt.Sprintf("%s.host", path),
		Value: hostName,
	})
	annotationsKey := fmt.Sprintf("%s.annotations", path)
	for key, value := range ingressAnnotations {
		kvs = append(kvs, bom.KeyValue{
			Key:       fmt.Sprintf("%s.%s", annotationsKey, key),
			Value:     value,
			SetString: true,
		})
	}
	kvs = append(kvs, bom.KeyValue{
		Key:   fmt.Sprintf("%s.tls[0].secretName", path),
		Value: tlsSecret,
	})

	kvs = append(kvs, bom.KeyValue{
		Key:   fmt.Sprintf("%s.tls[0].hosts[0]", path),
		Value: hostName,
	})
	return kvs
}

// isReady checks if all the sts and deployments for OpenSearch are ready or not
func (o opensearchOperatorComponent) isReady(ctx spi.ComponentContext) bool {
	deployments := getEnabledDeployments(ctx)
	return ready.DeploymentsAreReady(ctx.Log(), ctx.Client(), deployments, 1, getPrefix(ctx))
}

// GetClient returns a controller runtime client for the Verrazzano resource
func GetClient() (clipkg.Client, error) {
	runtimeConfig, err := k8sutil.GetConfigFromController()
	if err != nil {
		return nil, err
	}
	return clipkg.New(runtimeConfig, clipkg.Options{Scheme: newScheme()})
}

// newScheme creates a new scheme that includes this package's object for use by client
func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = vzapi.AddToScheme(scheme)
	_ = installv1beta1.AddToScheme(scheme)
	_ = clientgoscheme.AddToScheme(scheme)
	return scheme
}

// getEnabledDeployments returns the enabled deployments for this component
func getEnabledDeployments(ctx spi.ComponentContext) []types.NamespacedName {
	return []types.NamespacedName{
		{
			Name:      opensearchOperatorDeploymentName,
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
