// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package reconcile

import (
	"context"
	"strings"

	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/rancher"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// createRancherIngressAndCertCopies - creates copies of Rancher ingress and cert secret if
// they exist. If there is a DNS update in progress, we want to do this so that the old DNS
// continues to work as long as there is a good DNS entry, so that managed clusters can continue
// connecting using the old Rancher DNS in their kubeconfig, until they are updated with the new DNS.
func (r *Reconciler) createRancherIngressAndCertCopies(ctx spi.ComponentContext) {
	ok, rancherComp := registry.FindComponent(rancher.ComponentName)
	if !ok {
		return
	}

	dnsSuffix := getDNSSuffix(ctx.EffectiveCR())
	if dnsSuffix == "" {
		ctx.Log().Debug("Empty DNS suffix, skipping Rancher ingress copy")
		return
	}

	for _, ingressName := range rancherComp.GetIngressNames(ctx) {
		ing := netv1.Ingress{}
		err := r.Get(context.TODO(), ingressName, &ing)
		if err != nil {
			if !errors.IsNotFound(err) {
				ctx.Log().Infof("Cannot make a copy of Rancher ingress %s/%s - get ingress failed: %v", ingressName.Namespace, ingressName.Name, err)
			}
			continue
		}

		if len(ing.Spec.TLS) == 0 || len(ing.Spec.TLS[0].Hosts) == 0 {
			continue
		}
		if strings.HasSuffix(ing.Spec.TLS[0].Hosts[0], dnsSuffix) {
			ctx.Log().Debugf("Rancher ingress host %s has DNS suffix %s, skipping copy", ing.Spec.TLS[0].Hosts[0], dnsSuffix)
			continue
		}

		newIngressTLSList := r.createIngressCertSecretCopies(ctx, ing)
		newIngressNSN := types.NamespacedName{Name: "vz-" + ing.Name, Namespace: ingressName.Namespace}
		err = r.createIngressCopy(newIngressNSN, ing, newIngressTLSList)
		if err != nil {
			ctx.Log().Infof("Failed to create a copy of Rancher ingress %s/%s - create ingress failed: %v", ingressName.Namespace, ingressName.Name, err)
			continue
		}
		ctx.Log().Infof("Created copy of Rancher ingress %v, new ingress is %v", ingressName, newIngressNSN)
	}
}

func (r *Reconciler) createIngressCopy(newIngressName types.NamespacedName, existingIngress netv1.Ingress, newIngressTLSList []netv1.IngressTLS) error {
	ingress := &netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      newIngressName.Name,
			Namespace: newIngressName.Namespace,
		},
	}
	_, err := controllerutil.CreateOrUpdate(context.TODO(), r.Client, ingress, func() error {
		ingress.Labels = existingIngress.Labels
		ingress.Labels[constants.VerrazzanoManagedLabelKey] = "true"
		ingress.Annotations = existingIngress.Annotations
		ingress.Spec = *existingIngress.Spec.DeepCopy()
		ingress.Spec.TLS = newIngressTLSList
		return nil
	})
	return err
}

func (r *Reconciler) createIngressCertSecretCopies(ctx spi.ComponentContext, ing netv1.Ingress) []netv1.IngressTLS {
	newIngressTLSList := []netv1.IngressTLS{}
	for _, ingTLS := range ing.Spec.TLS {
		if ingTLS.SecretName != "" {
			tlsSecret := corev1.Secret{}
			tlsSecretName := types.NamespacedName{Namespace: ing.Namespace, Name: ingTLS.SecretName}
			err := r.Get(context.TODO(), tlsSecretName, &tlsSecret)
			if err == nil {
				newSecretNSN := types.NamespacedName{Name: "vz-" + tlsSecret.Name, Namespace: tlsSecret.Namespace}
				if err := r.createSecretCopy(newSecretNSN, tlsSecret); err != nil {
					ctx.Log().Infof("Failed to create copy %v of Rancher TLS secret %v", newSecretNSN, tlsSecretName)
				}
				newIngressTLS := ingTLS.DeepCopy()
				newIngressTLS.SecretName = newSecretNSN.Name
				newIngressTLSList = append(newIngressTLSList, *newIngressTLS)
			}
		}
	}
	return newIngressTLSList
}

func (r *Reconciler) createSecretCopy(newName types.NamespacedName, existingSecret corev1.Secret) error {
	newSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: newName.Namespace,
			Name:      newName.Name,
		},
	}
	_, err := controllerutil.CreateOrUpdate(context.TODO(), r.Client, newSecret, func() error {
		newSecret.Labels = existingSecret.Labels
		newSecret.Annotations = existingSecret.Annotations
		newSecret.Data = existingSecret.Data
		return nil
	})
	return err
}

func getDNSSuffix(effectiveCR *v1alpha1.Verrazzano) string {
	var dnsSuffix string

	if effectiveCR.Spec.Components.DNS == nil || effectiveCR.Spec.Components.DNS.Wildcard != nil {
		dnsSuffix = vzconfig.GetWildcardDomain(effectiveCR.Spec.Components.DNS)
	} else if effectiveCR.Spec.Components.DNS.OCI != nil {
		dnsSuffix = effectiveCR.Spec.Components.DNS.OCI.DNSZoneName
	} else if effectiveCR.Spec.Components.DNS.External != nil {
		dnsSuffix = effectiveCR.Spec.Components.DNS.External.Suffix
	}

	return dnsSuffix
}
