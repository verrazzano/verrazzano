// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package kiali

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/bom"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/nginx"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	v1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

const (
	// ComponentName is the name of the component
	ComponentName = "kiali-server"

	kialiHostName   = "kiali.vmi.system"
	kialiSystemName = "vmi-system-kiali"
	webFQDNKey      = "server.web_fqdn"
)

// isKialiReady checks if the Kiali deployment is ready
func isKialiReady(ctx spi.ComponentContext, _ string, namespace string) bool {
	deployments := []types.NamespacedName{
		{Name: kialiSystemName, Namespace: namespace},
	}
	return status.DeploymentsReady(ctx.Log(), ctx.Client(), deployments, 1)
}

// IsEnabled returns true if the component is enabled, which is the default
func IsEnabled(comp *vzapi.KialiComponent) bool {
	if comp == nil || comp.Enabled == nil {
		return false
	}
	return *comp.Enabled
}

// AppendOverrides Build the set of Kiali overrides for the helm install
func AppendOverrides(ctx spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	hostName, err := getKialiHostName(ctx)
	if err != nil {
		return kvs, err
	}
	// Service overrides
	kvs = append(kvs, bom.KeyValue{
		Key:   webFQDNKey,
		Value: hostName,
	})
	return kvs, nil
}

func createOrUpdateKialiIngress(ctx spi.ComponentContext, namespace string) error {
	ingress := v1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: kialiSystemName, Namespace: namespace},
	}
	opResult, err := controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), &ingress, func() error {
		dnsSubDomain, err := nginx.BuildDNSDomain(ctx.Client(), ctx.EffectiveCR())
		if err != nil {
			return err
		}
		ingressTarget := fmt.Sprintf("verrazzano-ingress.%s", dnsSubDomain)

		kialiHostName := buildKialiHostnameForDomain(dnsSubDomain)

		// Overwrite the existing Kiali service definition to point to the Verrazzano authproxy
		pathType := v1.PathTypeImplementationSpecific
		ingRule := v1.IngressRule{
			Host: kialiHostName,
			IngressRuleValue: v1.IngressRuleValue{
				HTTP: &v1.HTTPIngressRuleValue{
					Paths: []v1.HTTPIngressPath{
						{
							Path:     "/()(.*)",
							PathType: &pathType,
							Backend: v1.IngressBackend{
								Service: &v1.IngressServiceBackend{
									Name: "verrazzano-authproxy",
									Port: v1.ServiceBackendPort{
										Number: 8775,
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
				Hosts:      []string{kialiHostName},
				SecretName: "system-tls",
			},
		}
		ingress.Spec.Rules = []v1.IngressRule{ingRule}

		if ingress.Annotations == nil {
			ingress.Annotations = make(map[string]string)
		}
		ingress.Annotations["nginx.ingress.kubernetes.io/proxy-body-size"] = "6M"
		ingress.Annotations["nginx.ingress.kubernetes.io/rewrite-target"] = "/$2"
		ingress.Annotations["nginx.ingress.kubernetes.io/secure-backends"] = "false"
		ingress.Annotations["nginx.ingress.kubernetes.io/backend-protocol"] = "HTTP"
		ingress.Annotations["nginx.ingress.kubernetes.io/service-upstream"] = "true"
		ingress.Annotations["nginx.ingress.kubernetes.io/upstream-vhost"] = "${service_name}.${namespace}.svc.cluster.local"
		if nginx.IsExternalDNSEnabled(ctx.EffectiveCR()) {
			ingress.Annotations["external-dns.alpha.kubernetes.io/target"] = ingressTarget
			ingress.Annotations["external-dns.alpha.kubernetes.io/ttl"] = "60"
		}
		return nil
	})
	ctx.Log().Debugf("Kiali ingress operation result: %s", opResult)
	return err
}

func getKialiHostName(context spi.ComponentContext) (string, error) {
	dnsDomain, err := nginx.BuildDNSDomain(context.Client(), context.EffectiveCR())
	if err != nil {
		return "", err
	}
	return buildKialiHostnameForDomain(dnsDomain), nil
}

func buildKialiHostnameForDomain(dnsDomain string) string {
	return fmt.Sprintf("%s.%s", kialiHostName, dnsDomain)
}
