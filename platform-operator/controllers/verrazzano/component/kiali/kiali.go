// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package kiali

import (
	"context"
	"errors"
	"fmt"
	globalconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/security/password"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/verrazzano/verrazzano/pkg/bom"
	ctrlerrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
	"github.com/verrazzano/verrazzano/pkg/k8s/status"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	securityv1beta1 "istio.io/api/security/v1beta1"
	istiov1beta1 "istio.io/api/type/v1beta1"

	istioclisec "istio.io/client-go/pkg/apis/security/v1beta1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

const (
	kialiHostName         = "kiali.vmi.system"
	kialiSystemName       = "vmi-system-kiali"
	kialiServicePort      = "20001"
	kialiMetricsPort      = "9090"
	webFQDNKey            = "server.web_fqdn"
	configMapKey          = "config.yaml"
	kialiSigningKeySecret = "kiali-signing-key"
	signingKey            = "signing-key"
	signingKeyPath        = "login_token.signing_key"
)

// isKialiReady checks if the Kiali deployment is ready
func isKialiReady(ctx spi.ComponentContext) bool {
	deployments := []types.NamespacedName{
		{
			Name:      kialiSystemName,
			Namespace: ComponentNamespace,
		},
	}
	prefix := fmt.Sprintf("Component %s", ctx.GetComponent())
	return status.DeploymentsAreReady(ctx.Log(), ctx.Client(), deployments, 1, prefix)
}

// AppendOverrides Build the set of Kiali overrides for the helm install
func AppendOverrides(ctx spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	if vzconfig.IsNGINXEnabled(ctx.EffectiveCR()) {
		hostName, err := getKialiHostName(ctx)
		if err != nil {
			return kvs, err
		}
		// Service overrides
		kvs = append(kvs, bom.KeyValue{
			Key:   webFQDNKey,
			Value: hostName,
		})
	}
	signingKey, err := getKialiSigningKey(ctx)
	if err != nil {
		return kvs, err
	}
	kvs = append(kvs, bom.KeyValue{
		Key:   signingKeyPath,
		Value: signingKey,
	})
	return kvs, nil
}

// createOrUpdateKialiIngress Creates or updates the Kiali authproxy ingress
func createOrUpdateKialiIngress(ctx spi.ComponentContext, namespace string) error {
	ingress := v1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: kialiSystemName, Namespace: namespace},
	}
	_, err := controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), &ingress, func() error {
		dnsSubDomain, err := vzconfig.BuildDNSDomain(ctx.Client(), ctx.EffectiveCR())
		if err != nil {
			return ctx.Log().ErrorfNewErr("Failed building DNS domain name: %v", err)
		}
		ingressTarget := fmt.Sprintf("verrazzano-ingress.%s", dnsSubDomain)

		kialiHostName := buildKialiHostnameForDomain(dnsSubDomain)
		ingressClassName := vzconfig.GetIngressClassName(ctx.EffectiveCR())
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
									Name: constants.VerrazzanoAuthProxyServiceName,
									Port: v1.ServiceBackendPort{
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
		ingress.Spec.TLS = []v1.IngressTLS{
			{
				Hosts:      []string{kialiHostName},
				SecretName: "system-tls-kiali",
			},
		}
		ingress.Spec.Rules = []v1.IngressRule{ingRule}
		ingress.Spec.IngressClassName = &ingressClassName
		if ingress.Annotations == nil {
			ingress.Annotations = make(map[string]string)
		}
		ingress.Annotations["kubernetes.io/tls-acme"] = "true"
		ingress.Annotations["nginx.ingress.kubernetes.io/proxy-body-size"] = "6M"
		ingress.Annotations["nginx.ingress.kubernetes.io/rewrite-target"] = "/$2"
		ingress.Annotations["nginx.ingress.kubernetes.io/secure-backends"] = "false"
		ingress.Annotations["nginx.ingress.kubernetes.io/backend-protocol"] = "HTTP"
		ingress.Annotations["nginx.ingress.kubernetes.io/service-upstream"] = "true"
		ingress.Annotations["nginx.ingress.kubernetes.io/upstream-vhost"] = "${service_name}.${namespace}.svc.cluster.local"
		ingress.Annotations["cert-manager.io/common-name"] = kialiHostName
		if vzconfig.IsExternalDNSEnabled(ctx.EffectiveCR()) {
			ingress.Annotations["external-dns.alpha.kubernetes.io/target"] = ingressTarget
			ingress.Annotations["external-dns.alpha.kubernetes.io/ttl"] = "60"
		}
		return nil
	})
	if ctrlerrors.ShouldLogKubenetesAPIError(err) {
		return ctx.Log().ErrorfNewErr("Failed create/update Kiali ingress: %v", err)
	}
	return err
}

func createOrUpdateAuthPolicy(ctx spi.ComponentContext) error {
	authPol := istioclisec.AuthorizationPolicy{
		ObjectMeta: metav1.ObjectMeta{Namespace: constants.VerrazzanoSystemNamespace, Name: "vmi-system-kiali-authzpol"},
	}
	_, err := controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), &authPol, func() error {
		authPol.Spec = securityv1beta1.AuthorizationPolicy{
			Selector: &istiov1beta1.WorkloadSelector{
				MatchLabels: map[string]string{
					"app": "kiali",
				},
			},
			Action: securityv1beta1.AuthorizationPolicy_ALLOW,
			Rules: []*securityv1beta1.Rule{
				{
					From: []*securityv1beta1.Rule_From{{
						Source: &securityv1beta1.Source{
							Principals: []string{fmt.Sprintf("cluster.local/ns/%s/sa/verrazzano-authproxy", constants.VerrazzanoSystemNamespace)},
							Namespaces: []string{constants.VerrazzanoSystemNamespace},
						},
					}},
					To: []*securityv1beta1.Rule_To{{
						Operation: &securityv1beta1.Operation{
							Ports: []string{kialiServicePort},
						},
					}},
				},
				{
					From: []*securityv1beta1.Rule_From{{
						Source: &securityv1beta1.Source{
							Principals: []string{fmt.Sprintf("cluster.local/ns/%s/sa/verrazzano-monitoring-operator", constants.VerrazzanoSystemNamespace)},
							Namespaces: []string{constants.VerrazzanoSystemNamespace},
						},
					}},
					To: []*securityv1beta1.Rule_To{{
						Operation: &securityv1beta1.Operation{
							Ports: []string{kialiMetricsPort},
						},
					}},
				},
			},
		}
		return nil
	})
	if ctrlerrors.ShouldLogKubenetesAPIError(err) {
		return ctx.Log().ErrorfNewErr("Failed create/update Kiali auth policy: %v", err)
	}
	return err
}

func getKialiHostName(context spi.ComponentContext) (string, error) {
	dnsDomain, err := vzconfig.BuildDNSDomain(context.Client(), context.EffectiveCR())
	if err != nil {
		return "", err
	}
	return buildKialiHostnameForDomain(dnsDomain), nil
}

func buildKialiHostnameForDomain(dnsDomain string) string {
	return fmt.Sprintf("%s.%s", kialiHostName, dnsDomain)
}

// getKialiSigningKey this secret holds the data for the Kiali signing key
func getKialiSigningKey(ctx spi.ComponentContext) (string, error) {
	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kialiSigningKeySecret,
			Namespace: globalconst.VerrazzanoSystemNamespace,
		},
		Data: map[string][]byte{},
	}
	err := ctx.Client().Get(context.TODO(), clipkg.ObjectKey{
		Namespace: constants.VerrazzanoSystemNamespace,
		Name:      kialiSigningKeySecret,
	}, &secret)
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return "", ctx.Log().ErrorfThrottledNewErr("Unexpected error getting secret %s, : %V", kialiSigningKeySecret, err)
		}
		pw, err := password.GeneratePassword(16)
		if err != nil {
			return "", err
		}
		_, err = controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), &secret, func() error {
			if secret.Data[signingKey] == nil {
				pw, err := password.GeneratePassword(16)
				if err != nil {
					return err
				}
				secret.Data[signingKey] = []byte(pw)
			}
			return nil
		})
		return pw, err
	}
	signingKeyData, ok := secret.Data[signingKey]
	if !ok {
		return "", errors.New(fmt.Sprintf("Error retrieving signing key from secret %s", kialiSigningKeySecret))
	}
	return string(signingKeyData), nil
}

// GetOverrides returns the Kiali specific install overrides from v1beta1.Verrazzano CR
func GetOverrides(object runtime.Object) interface{} {
	if effectiveCR, ok := object.(*v1alpha1.Verrazzano); ok {
		if effectiveCR.Spec.Components.Kiali != nil {
			return effectiveCR.Spec.Components.Kiali.ValueOverrides
		}
		return []v1alpha1.Overrides{}
	} else if effectiveCR, ok := object.(*installv1beta1.Verrazzano); ok {
		if effectiveCR.Spec.Components.Kiali != nil {
			return effectiveCR.Spec.Components.Kiali.ValueOverrides
		}
		return []installv1beta1.Overrides{}
	}
	return []v1alpha1.Overrides{}
}
