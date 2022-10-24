// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package argocd

import (
	"context"
	"errors"
	"fmt"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	networkv1 "k8s.io/api/networking/v1"
	v1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

// patchArgoCDIngress annotates the ArgoCD ingress with environment specific values
func patchArgoCDIngress(ctx spi.ComponentContext) error {
	ingress := networkv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: "argocd-server", Namespace: "argocd"},
	}
	_, err := controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), &ingress, func() error {
		dnsSubDomain, err := vzconfig.BuildDNSDomain(ctx.Client(), ctx.EffectiveCR())
		if err != nil {
			return ctx.Log().ErrorfNewErr("Failed building DNS domain name: %v", err)
		}
		argoCDHostName := buildArgoCDHostNameForDomain(dnsSubDomain)
		ingressClassName := vzconfig.GetIngressClassName(ctx.EffectiveCR())
		pathType := v1.PathTypeImplementationSpecific
		ingRule := v1.IngressRule{
			Host: argoCDHostName,
			IngressRuleValue: v1.IngressRuleValue{
				HTTP: &v1.HTTPIngressRuleValue{
					Paths: []v1.HTTPIngressPath{
						{
							PathType: &pathType,
							Backend: v1.IngressBackend{
								Service: &v1.IngressServiceBackend{
									Name: common.ArgoCDService,
									Port: v1.ServiceBackendPort{
										Name: "http",
									},
								},
								Resource: nil,
							},
						},
					},
				},
			},
		}
		ingress.Spec.TLS = []v1.IngressTLS{
			{
				Hosts:      []string{argoCDHostName},
				SecretName: "tls-argocd-ingress",
			},
		}
		ingress.Spec.Rules = []v1.IngressRule{ingRule}
		ingress.Spec.IngressClassName = &ingressClassName
		dnsSuffix, _ := vzconfig.GetDNSSuffix(ctx.Client(), ctx.EffectiveCR())
		if ingress.Annotations == nil {
			ingress.Annotations = map[string]string{}
		}
		ingress.Annotations["cert-manager.io/common-name"] = fmt.Sprintf("%s.%s.%s",
			ComponentName, ctx.EffectiveCR().Spec.EnvironmentName, dnsSuffix)
		ingress.Annotations["kubernetes.io/tls-acme"] = "true"
		ingress.Annotations["nginx.ingress.kubernetes.io/backend-protocol"] = "HTTP"
		ingress.Annotations["nginx.ingress.kubernetes.io/force-ssl-redirect"] = "true"
		ingress.Annotations["nginx.ingress.kubernetes.io/affinity"] = "cookie"
		ingress.Annotations["nginx.ingress.kubernetes.io/proxy-connect-timeout"] = "30"
		ingress.Annotations["nginx.ingress.kubernetes.io/proxy-read-timeout"] = "1800"
		ingress.Annotations["nginx.ingress.kubernetes.io/proxy-send-timeout"] = "1800"
		ingress.Annotations["nginx.ingress.kubernetes.io/service-upstream"] = "true"
		ingress.Annotations["nginx.ingress.kubernetes.io/session-cookie-conditional-samesite-none"] = "true"
		ingress.Annotations["nginx.ingress.kubernetes.io/session-cookie-expires"] = "86400"
		ingress.Annotations["nginx.ingress.kubernetes.io/session-cookie-max-age"] = "86400"
		ingress.Annotations["nginx.ingress.kubernetes.io/session-cookie-name"] = "route"
		ingress.Annotations["nginx.ingress.kubernetes.io/session-cookie-samesite"] = "Strict"
		ingress.Annotations["nginx.ingress.kubernetes.io/upstream-vhost"] = "${service_name}.${namespace}.svc.cluster.local"
		cm := ctx.EffectiveCR().Spec.Components.CertManager
		if cm == nil {
			return errors.New("CertificateManager was not found in the effective CR")
		}
		if (cm.Certificate.Acme != vzapi.Acme{}) {
			addAcmeIngressAnnotations(ctx.EffectiveCR().Spec.EnvironmentName, dnsSuffix, ingress)
		} else {
			addCAIngressAnnotations(ctx.EffectiveCR().Spec.EnvironmentName, dnsSuffix, ingress)
		}
		return nil
	})
	ctx.Log().Debugf("patchArgoCDIngress: ArgoCD ingress operation result: %v", err)
	return err
}

// addAcmeIngressAnnotations annotate ingress with ACME specific values
func addAcmeIngressAnnotations(name, dnsSuffix string, ingress networkv1.Ingress) {
	ingress.Annotations["nginx.ingress.kubernetes.io/auth-realm"] = fmt.Sprintf("%s auth", dnsSuffix)
	ingress.Annotations["external-dns.alpha.kubernetes.io/target"] = fmt.Sprintf("verrazzano-ingress.%s.%s", name, dnsSuffix)
	ingress.Annotations["external-dns.alpha.kubernetes.io/ttl"] = "60"
	// Remove any existing cert manage annotations
	delete(ingress.Annotations, "cert-manager.io/issuer")
	delete(ingress.Annotations, "cert-manager.io/issuer-kind")
}

// addCAIngressAnnotations annotate ingress with custom CA specific values
func addCAIngressAnnotations(name, dnsSuffix string, ingress networkv1.Ingress) {
	ingress.Annotations["nginx.ingress.kubernetes.io/auth-realm"] = fmt.Sprintf("%s.%s auth", name, dnsSuffix)
	ingress.Annotations["cert-manager.io/cluster-issuer"] = "verrazzano-cluster-issuer"
	ingress.Annotations["cert-manager.io/common-name"] = fmt.Sprintf("%s.%s.%s", common.ArgoCDName, name, dnsSuffix)
}

// buildArgoCDHostNameForDomain - builds the hostname for ArgocD ingress
func buildArgoCDHostNameForDomain(dnsDomain string) string {
	return fmt.Sprintf("%s.%s", common.ArgoCDName, dnsDomain)
}
