// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"context"
	"fmt"

	ctrlerrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
	"github.com/verrazzano/verrazzano/pkg/vzmap"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

type IngressProperties struct {
	IngressName      string
	HostName         string
	TLSSecretName    string
	ExtraAnnotations map[string]string
}

// SameSiteCookieAnnotations creates annotations for same site cookies to enable sticky sessions on an ingress
func SameSiteCookieAnnotations(cookieName string) map[string]string {
	return map[string]string{
		"nginx.ingress.kubernetes.io/affinity":                                 "cookie",
		"nginx.ingress.kubernetes.io/session-cookie-conditional-samesite-none": "true",
		"nginx.ingress.kubernetes.io/session-cookie-expires":                   "86400",
		"nginx.ingress.kubernetes.io/session-cookie-max-age":                   "86400",
		"nginx.ingress.kubernetes.io/session-cookie-name":                      cookieName,
		"nginx.ingress.kubernetes.io/session-cookie-samesite":                  "Strict",
	}
}

// CreateOrUpdateSystemComponentIngress creates or updates an ingress for a Verrazzano system component
func CreateOrUpdateSystemComponentIngress(ctx spi.ComponentContext, props IngressProperties) error {
	// create the ingress in the same namespace as Auth Proxy, note that we cannot use authproxy.ComponentNamespace here because it creates an import cycle
	ingress := netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: props.IngressName, Namespace: constants.VerrazzanoSystemNamespace},
	}

	_, err := controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), &ingress, func() error {
		dnsSubDomain, err := vzconfig.BuildDNSDomain(ctx.Client(), ctx.EffectiveCR())
		if err != nil {
			return ctx.Log().ErrorfNewErr("Failed building DNS domain name: %v", err)
		}
		ingressClassName := vzconfig.GetIngressClassName(ctx.EffectiveCR())
		qualifiedHostName := fmt.Sprintf("%s.%s", props.HostName, dnsSubDomain)
		pathType := netv1.PathTypeImplementationSpecific

		ingRule := netv1.IngressRule{
			Host: qualifiedHostName,
			IngressRuleValue: netv1.IngressRuleValue{
				HTTP: &netv1.HTTPIngressRuleValue{
					Paths: []netv1.HTTPIngressPath{
						{
							Path:     "/()(.*)",
							PathType: &pathType,
							Backend: netv1.IngressBackend{
								Service: &netv1.IngressServiceBackend{
									Name: constants.VerrazzanoAuthProxyServiceName,
									Port: netv1.ServiceBackendPort{
										Number: constants.VerrazzanoAuthProxyServicePort,
									},
								},
								Resource: nil,
							},
						},
					},
				},
			},
		}
		ingress.Spec.TLS = []netv1.IngressTLS{
			{
				Hosts:      []string{qualifiedHostName},
				SecretName: props.TLSSecretName,
			},
		}
		ingress.Spec.Rules = []netv1.IngressRule{ingRule}
		ingress.Spec.IngressClassName = &ingressClassName
		if ingress.Annotations == nil {
			ingress.Annotations = make(map[string]string)
		}
		ingress.Annotations["kubernetes.io/tls-acme"] = "true"
		ingress.Annotations["nginx.ingress.kubernetes.io/proxy-body-size"] = "6M"
		ingress.Annotations["nginx.ingress.kubernetes.io/rewrite-target"] = "/$2"
		ingress.Annotations["nginx.ingress.kubernetes.io/service-upstream"] = "true"
		ingress.Annotations["nginx.ingress.kubernetes.io/upstream-vhost"] = "${service_name}.${namespace}.svc.cluster.local"
		ingress.Annotations["cert-manager.io/common-name"] = qualifiedHostName
		if vzconfig.IsExternalDNSEnabled(ctx.EffectiveCR()) {
			ingressTarget := fmt.Sprintf("verrazzano-ingress.%s", dnsSubDomain)
			ingress.Annotations["external-dns.alpha.kubernetes.io/target"] = ingressTarget
			ingress.Annotations["external-dns.alpha.kubernetes.io/ttl"] = "60"
		}
		ingress.Annotations = vzmap.UnionStringMaps(ingress.Annotations, props.ExtraAnnotations)
		return nil
	})
	if ctrlerrors.ShouldLogKubenetesAPIError(err) {
		return ctx.Log().ErrorfNewErr("Failed creating/updating ingress %s: %v", props.IngressName, err)
	}
	return err
}
