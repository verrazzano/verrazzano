// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"context"
	"fmt"

	globalconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/nginxutil"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/namespace"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateAndLabelNamespaces(ctx spi.ComponentContext) error {
	if err := LabelKubeSystemNamespace(ctx.Client()); err != nil {
		return err
	}

	if err := CreateAndLabelVMINamespaces(ctx); err != nil {
		return err
	}

	if err := namespace.CreateVerrazzanoMultiClusterNamespace(ctx.Client()); err != nil {
		return err
	}

	// Set istio injection flag.  This will be false if Istio disabled or injections explictiy disabled
	istio := ctx.EffectiveCR().Spec.Components.Istio
	istioInject := istio != nil && istio.IsInjectionEnabled()

	if vzcr.IsCertManagerEnabled(ctx.EffectiveCR()) {
		if err := namespace.CreateCertManagerNamespace(ctx.Client()); err != nil {
			return ctx.Log().ErrorfNewErr("Failed creating namespace %s: %v", globalconst.CertManagerNamespace, err)
		}
	}

	IngressNGINXNamespace := nginxutil.IngressNGINXNamespace()
	if err := namespace.CreateIngressNginxNamespace(ctx.Client(), istioInject, IngressNGINXNamespace); err != nil {
		return ctx.Log().ErrorfNewErr("Failed creating namespace %s: %v", IngressNGINXNamespace, err)
	}

	if vzcr.IsIstioEnabled(ctx.EffectiveCR()) {
		if err := namespace.CreateIstioNamespace(ctx.Client()); err != nil {
			return ctx.Log().ErrorfNewErr("Failed creating namespace %s: %v", globalconst.IstioSystemNamespace, err)
		}
	}

	if vzcr.IsKeycloakEnabled(ctx.EffectiveCR()) {
		if err := namespace.CreateKeycloakNamespace(ctx.Client(), istioInject); err != nil {
			return ctx.Log().ErrorfNewErr("Failed creating namespace %s: %v", globalconst.KeycloakNamespace, err)
		}
	}

	if vzcr.IsMySQLOperatorEnabled(ctx.EffectiveCR()) {
		if err := namespace.CreateMysqlOperator(ctx.Client(), istioInject); err != nil {
			return ctx.Log().ErrorfNewErr("Failed creating namespace %s: %v", globalconst.MySQLOperatorNamespace, err)
		}
	}

	if vzcr.IsArgoCDEnabled(ctx.EffectiveCR()) {
		if err := namespace.CreateArgoCDNamespace(ctx.Client(), istioInject); err != nil {
			return ctx.Log().ErrorfNewErr("Failed creating namespace %s: %v", constants.ArgoCDNamespace, err)
		}
	}

	if vzcr.IsClusterAPIEnabled(ctx.EffectiveCR()) {
		if err := namespace.CreateVerrazzanoCapiNamespace(ctx.Client()); err != nil {
			return ctx.Log().ErrorfNewErr("Failed creating namespace %s: %v", globalconst.VerrazzanoCAPINamespace, err)
		}
	}

	// cattle-system NS must be created since the rancher NetworkPolicy, which is always installed, requires it
	if err := namespace.CreateRancherNamespace(ctx.Client()); err != nil {
		return ctx.Log().ErrorfNewErr("Failed creating namespace %s: %v", globalconst.RancherSystemNamespace, err)
	}

	if err := namespace.CreateVerrazzanoMonitoringNamespace(ctx.Client(), istioInject); err != nil {
		return ctx.Log().ErrorfNewErr("Failed creating namespace %s: %v", constants.VerrazzanoMonitoringNamespace, err)
	}

	if vzcr.IsVeleroEnabled(ctx.EffectiveCR()) {
		if err := namespace.CreateVeleroNamespace(ctx.Client()); err != nil {
			return ctx.Log().ErrorfNewErr("Failed creating namespace %s: %v", constants.VeleroNameSpace, err)
		}
	}

	return nil
}

// LabelKubeSystemNamespace adds the label needed by network polices to kube-system
func LabelKubeSystemNamespace(client clipkg.Client) error {
	const KubeSystemNamespace = "kube-system"
	ns := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: KubeSystemNamespace}}
	if _, err := controllerruntime.CreateOrUpdate(context.TODO(), client, &ns, func() error {
		if ns.Labels == nil {
			ns.Labels = make(map[string]string)
		}
		ns.Labels[globalconst.LabelVerrazzanoNamespace] = KubeSystemNamespace
		return nil
	}); err != nil {
		return err
	}
	return nil
}

// CheckExistingNamespace checks namespaces if there is an existing namespace that is not created by Verrazzano
// It compares the label of the namespace with the Verrazzano namespace label i.e verrazzano.io/namespace
func CheckExistingNamespace(ns []corev1.Namespace, includeNamespace func(*corev1.Namespace) bool) error {
	for i := range ns {
		if includeNamespace(&ns[i]) {
			for l := range ns[i].Labels {
				if l == globalconst.LabelVerrazzanoNamespace {
					return nil
				}
			}
			return fmt.Errorf("found existing namespace %s not created by Verrazzano", ns[i].Name)
		}
	}
	return nil
}
