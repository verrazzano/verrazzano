// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package operator

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/bom"
	ctrlerrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	"path"
	"strings"

	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	v1 "k8s.io/api/core/v1"
	networkv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

const (
	deploymentName        = "jaeger-operator"
	templateFile          = "/jaeger/jaeger-operator.yaml"
	jaegerHostName        = "jaeger"
	jaegerCertificateName = "jaeger-tls"
)

var subcomponentNames = []string{
	"jaeger-ingester",
	"jaeger-agent",
	"jaeger-query",
	"jaeger-collector",
	"jaeger-operator",
}

func componentInstall(ctx spi.ComponentContext) error {
	args, err := buildInstallArgs()
	if err != nil {
		return err
	}

	// Apply Jaeger Operator
	yamlApplier := k8sutil.NewYAMLApplier(ctx.Client(), "")
	if err := yamlApplier.ApplyFT(path.Join(config.GetThirdPartyManifestsDir(), templateFile), args); err != nil {
		return ctx.Log().ErrorfNewErr("Failed to install Jaeger Operator: %v", err)
	}
	return nil
}

func buildInstallArgs() (map[string]interface{}, error) {
	args := map[string]interface{}{
		"namespace": constants.VerrazzanoMonitoringNamespace,
	}
	bomFile, err := bom.NewBom(config.GetDefaultBOMFilePath())
	if err != nil {
		return args, err
	}
	for _, subcomponent := range subcomponentNames {
		if err := setImageOverride(args, bomFile, subcomponent); err != nil {
			return args, err
		}
	}
	return args, nil
}

func setImageOverride(args map[string]interface{}, bomFile bom.Bom, subcomponent string) error {
	images, err := bomFile.GetImageNameList(subcomponent)
	if err != nil {
		return err
	}
	if len(images) != 1 {
		return fmt.Errorf("expected 1 %s image, got %d", subcomponent, len(images))
	}

	args[strings.ReplaceAll(subcomponent, "-", "")] = images[0]
	return nil
}

// isJaegerOperatorReady checks if the Jaeger operator deployment is ready
func isJaegerOperatorReady(context spi.ComponentContext) bool {
	return status.DeploymentsAreReady(context.Log(), context.Client(), deployments, 1, componentPrefix)
}

func ensureVerrazzanoMonitoringNamespace(ctx spi.ComponentContext) error {
	// Create the verrazzano-monitoring namespace
	ctx.Log().Debugf("Creating namespace %s for the Jaeger Operator", ComponentNamespace)
	namespace := v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ComponentNamespace}}
	if _, err := controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), &namespace, func() error {
		return nil
	}); err != nil {
		return ctx.Log().ErrorfNewErr("Failed to create or update the %s namespace: %v", ComponentNamespace, err)
	}
	return nil
}

// createOrUpdateJaegerIngress Creates or updates the Jaeger authproxy ingress
func createOrUpdateJaegerIngress(ctx spi.ComponentContext, namespace string) error {
	ingress := networkv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: constants.JaegerIngress, Namespace: namespace},
	}
	_, err := controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), &ingress, func() error {
		dnsSubDomain, err := vzconfig.BuildDNSDomain(ctx.Client(), ctx.EffectiveCR())
		if err != nil {
			return ctx.Log().ErrorfNewErr("Failed building DNS domain name: %v", err)
		}

		jaegerHostName := buildJaegerHostnameForDomain(dnsSubDomain)

		// Overwrite the existing Jaeger service definition to point to the Verrazzano authproxy
		pathType := networkv1.PathTypeImplementationSpecific
		ingRule := networkv1.IngressRule{
			Host: jaegerHostName,
			IngressRuleValue: networkv1.IngressRuleValue{
				HTTP: &networkv1.HTTPIngressRuleValue{
					Paths: []networkv1.HTTPIngressPath{
						{
							Path:     "/()(.*)",
							PathType: &pathType,
							Backend: networkv1.IngressBackend{
								Service: &networkv1.IngressServiceBackend{
									Name: constants.VerrazzanoAuthProxyServiceName,
									Port: networkv1.ServiceBackendPort{
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
		ingress.Spec.TLS = []networkv1.IngressTLS{
			{
				Hosts:      []string{jaegerHostName},
				SecretName: "jaeger-tls",
			},
		}
		ingress.Spec.Rules = []networkv1.IngressRule{ingRule}

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
		ingress.Annotations["cert-manager.io/common-name"] = jaegerHostName
		if vzconfig.IsExternalDNSEnabled(ctx.EffectiveCR()) {
			ingressTarget := fmt.Sprintf("verrazzano-ingress.%s", dnsSubDomain)
			ingress.Annotations["external-dns.alpha.kubernetes.io/target"] = ingressTarget
			ingress.Annotations["external-dns.alpha.kubernetes.io/ttl"] = "60"
		}
		return nil
	})
	if ctrlerrors.ShouldLogKubenetesAPIError(err) {
		return ctx.Log().ErrorfNewErr("Failed create/update Jaeger ingress: %v", err)
	}
	return err
}

func buildJaegerHostnameForDomain(dnsDomain string) string {
	return fmt.Sprintf("%s.%s", jaegerHostName, dnsDomain)
}
