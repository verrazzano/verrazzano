// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"context"
	"errors"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/constants"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	adminv1 "k8s.io/api/admissionregistration/v1"
	networking "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// patchRancherIngress annotates the Rancher ingress with environment specific values
func patchRancherIngress(c client.Client, vz *vzapi.Verrazzano) error {
	clusterIssuer := vz.Spec.Components.ClusterIssuer
	if clusterIssuer == nil {
		return errors.New("CertificateManager was not found in the effective CR")
	}
	dnsSuffix, err := vzconfig.GetDNSSuffix(c, vz)
	if err != nil {
		return err
	}
	namespacedName := types.NamespacedName{
		Namespace: common.CattleSystem,
		Name:      common.RancherName,
	}
	ingress := &networking.Ingress{}
	if err := c.Get(context.TODO(), namespacedName, ingress); err != nil {
		return err
	}
	ingressMerge := client.MergeFrom(ingress.DeepCopy())
	if ingress.Annotations == nil {
		ingress.Annotations = map[string]string{}
	}
	ingress.Annotations["kubernetes.io/tls-acme"] = "true"
	ingress.Annotations["nginx.ingress.kubernetes.io/backend-protocol"] = "HTTPS"
	ingress.Annotations["nginx.ingress.kubernetes.io/force-ssl-redirect"] = "true"
	ingress.Annotations["cert-manager.io/cluster-issuer"] = constants.VerrazzanoClusterIssuerName
	ingress.Annotations["cert-manager.io/common-name"] = fmt.Sprintf("%s.%s.%s", common.RancherName, vz.Spec.EnvironmentName, dnsSuffix)

	isLEIssuer, err := clusterIssuer.IsLetsEncryptIssuer()
	if err != nil {
		return err
	}
	if isLEIssuer {
		addAcmeIngressAnnotations(vz.Spec.EnvironmentName, dnsSuffix, ingress)
	} else {
		addCAIngressAnnotations(vz.Spec.EnvironmentName, dnsSuffix, ingress)
	}

	return c.Patch(context.TODO(), ingress, ingressMerge)
}

// addAcmeIngressAnnotations annotate ingress with LetsEncrypt specific values
func addAcmeIngressAnnotations(name, dnsSuffix string, ingress *networking.Ingress) {
	ingress.Annotations["nginx.ingress.kubernetes.io/auth-realm"] = fmt.Sprintf("%s auth", dnsSuffix)
	ingress.Annotations["external-dns.alpha.kubernetes.io/target"] = fmt.Sprintf("verrazzano-ingress.%s.%s", name, dnsSuffix)
	ingress.Annotations["external-dns.alpha.kubernetes.io/ttl"] = "60"
	// Remove any existing cert manager annotations
	delete(ingress.Annotations, "cert-manager.io/issuer")
	delete(ingress.Annotations, "cert-manager.io/issuer-kind")
}

// addCAIngressAnnotations annotate ingress with custom CA specific values
func addCAIngressAnnotations(name, dnsSuffix string, ingress *networking.Ingress) {
	ingress.Annotations["nginx.ingress.kubernetes.io/auth-realm"] = fmt.Sprintf("%s.%s auth", name, dnsSuffix)
}

// cleanupRancherResources cleans up Rancher resources that are no longer supported
func cleanupRancherResources(ctx context.Context, c client.Client) error {
	di, err := getDynamicClientFunc()
	if err != nil {
		return err
	}

	// Patch dynamic schemas that we need to preserve
	for _, schema := range cloudCredentialSchemas {
		dynamicSchema, err := di.Resource(dynamicSchemaGVR).Get(ctx, schema, metav1.GetOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		} else if dynamicSchema != nil {
			// Remove owner references so cascading delete of the node driver does not also delete the dynamic schema
			dynamicSchema.SetOwnerReferences([]metav1.OwnerReference{})
			_, err = di.Resource(dynamicSchemaGVR).Update(ctx, dynamicSchema, metav1.UpdateOptions{})
			if err != nil {
				return err
			}
		}
	}

	// Delete the Rancher CAPI webhooks, which conflicts with the community CAPI webhooks
	vwhc := &adminv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: CAPIValidatingWebhook,
		},
	}
	err = c.Delete(ctx, vwhc)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	mwhc := &adminv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: CAPIMutatingWebhook,
		},
	}
	err = c.Delete(ctx, mwhc)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	// Delete any node drivers, which are no longer supported
	items, err := di.Resource(nodeDriverGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, item := range items.Items {
		err = di.Resource(nodeDriverGVR).Delete(ctx, item.GetName(), metav1.DeleteOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}
	return nil
}
