// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package thanos

import (
	"context"
	"fmt"

	"github.com/verrazzano/verrazzano/pkg/bom"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	ctrlerrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	securityv1beta1 "istio.io/api/security/v1beta1"
	istiov1beta1 "istio.io/api/type/v1beta1"
	istioclisec "istio.io/client-go/pkg/apis/security/v1beta1"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	// Thanos Query Frontend ingress constants
	frontendHostName        = "thanos-query-frontend"
	frontendCertificateName = "system-tls-thanos-query-frontend"

	thanosAuthPolicyName = "thanos-query-authzpol"
	thanosNetPolicyName  = "thanos-query"
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
func AppendOverrides(_ spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	bomFile, err := bom.NewBom(config.GetDefaultBOMFilePath())
	if err != nil {
		return kvs, err
	}
	image, err := bomFile.BuildImageOverrides(ComponentName)
	if err != nil {
		return kvs, err
	}
	return append(kvs, image...), nil
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

// preInstallUpgrade handles post-install and post-upgrade processing for the Thanos Component
func postInstallUpgrade(ctx spi.ComponentContext) error {
	// Do nothing if dry run
	if ctx.IsDryRun() {
		ctx.Log().Debug("Thanos postInstallUpgrade dry run")
		return nil
	}

	if err := createOrUpdateComponentIngress(ctx); err != nil {
		return err
	}
	if err := createOrUpdateNetworkPolicies(ctx); err != nil {
		return err
	}
	return createOrUpdateComponentAuthPolicy(ctx)
}

func createOrUpdateComponentIngress(ctx spi.ComponentContext) error {
	// If NGINX is not enabled, skip the ingress creation
	if !vzcr.IsNGINXEnabled(ctx.EffectiveCR()) {
		return nil
	}

	// Create the Thanos Query Frontend Ingress
	thanosProps := common.IngressProperties{
		IngressName:      constants.ThanosQueryFrontendIngress,
		HostName:         frontendHostName,
		TLSSecretName:    frontendCertificateName,
		ExtraAnnotations: common.SameSiteCookieAnnotations(frontendHostName),
	}
	return common.CreateOrUpdateSystemComponentIngress(ctx, thanosProps)
}

// createOrUpdateComponentAuthPolicy creates the Istio authorization policy for Thanos
func createOrUpdateComponentAuthPolicy(ctx spi.ComponentContext) error {
	// if Istio is explicitly disabled, do not attempt to create the auth policy
	if !vzcr.IsIstioEnabled(ctx.EffectiveCR()) {
		ctx.Log().Debug("Skipping Authorization Policy creation for Thanos, Istio is disabled.")
		return nil
	}

	authPol := istioclisec.AuthorizationPolicy{
		ObjectMeta: metav1.ObjectMeta{Namespace: ComponentNamespace, Name: thanosAuthPolicyName},
	}
	_, err := controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), &authPol, func() error {
		authPol.Spec = newAuthPolicySpec()
		return nil
	})
	if ctrlerrors.ShouldLogKubenetesAPIError(err) {
		return ctx.Log().ErrorfNewErr("Failed creating/updating Thanos Authorization Policy: %v", err)
	}
	return err
}

// createOrUpdateNetworkPolicies creates or updates network policies for this component
func createOrUpdateNetworkPolicies(ctx spi.ComponentContext) error {
	netPolicy := &netv1.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: thanosNetPolicyName, Namespace: ComponentNamespace}}

	_, err := controllerutil.CreateOrUpdate(context.TODO(), ctx.Client(), netPolicy, func() error {
		netPolicy.Spec = newNetworkPolicySpec()
		return nil
	})

	return err
}

// newAuthPolicySpec returns an Authorization Policy spec for the Thanos component ingress
func newAuthPolicySpec() securityv1beta1.AuthorizationPolicy {
	return securityv1beta1.AuthorizationPolicy{
		Selector: &istiov1beta1.WorkloadSelector{
			MatchLabels: map[string]string{
				"app.kubernetes.io/name": ComponentName,
			},
		},
		Action: securityv1beta1.AuthorizationPolicy_ALLOW,
		Rules: []*securityv1beta1.Rule{
			{
				// allow Auth Proxy to access Thanos query frontend
				From: []*securityv1beta1.Rule_From{{
					Source: &securityv1beta1.Source{
						Principals: []string{
							fmt.Sprintf("cluster.local/ns/%s/sa/verrazzano-authproxy", constants.VerrazzanoSystemNamespace),
						},
						Namespaces: []string{constants.VerrazzanoSystemNamespace},
					},
				}},
				To: []*securityv1beta1.Rule_To{{
					Operation: &securityv1beta1.Operation{
						Ports: []string{"10902"},
					},
				}},
			},
		},
	}
}

// newNetworkPolicy returns a populated NetworkPolicySpec with ingress rules for Thanos
func newNetworkPolicySpec() netv1.NetworkPolicySpec {
	tcpProtocol := corev1.ProtocolTCP
	port := intstr.FromInt(10902)

	return netv1.NetworkPolicySpec{
		PodSelector: metav1.LabelSelector{
			MatchLabels: map[string]string{
				"app.kubernetes.io/name": ComponentName,
			},
		},
		PolicyTypes: []netv1.PolicyType{
			netv1.PolicyTypeIngress,
		},
		Ingress: []netv1.NetworkPolicyIngressRule{
			{
				// allow ingress to port 9090 from Auth Proxy
				From: []netv1.NetworkPolicyPeer{
					{
						NamespaceSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								vzconst.LabelVerrazzanoNamespace: constants.VerrazzanoSystemNamespace,
							},
						},
						PodSelector: &metav1.LabelSelector{
							MatchExpressions: []metav1.LabelSelectorRequirement{
								{
									Key:      "app",
									Operator: metav1.LabelSelectorOpIn,
									Values: []string{
										"verrazzano-authproxy",
									},
								},
							},
						},
					},
				},
				Ports: []netv1.NetworkPolicyPort{
					{
						Protocol: &tcpProtocol,
						Port:     &port,
					},
				},
			},
		},
	}
}
