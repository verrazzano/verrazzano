// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package reconcile

import (
	"context"

	"github.com/google/uuid"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/rancher"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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
	for _, ingressName := range rancherComp.GetIngressNames(ctx) {
		ing := netv1.Ingress{}
		err := r.Get(context.TODO(), ingressName, &ing)
		if err != nil {
			if !errors.IsNotFound(err) {
				ctx.Log().Infof("Cannot make a copy of Rancher ingress %s/%s - get ingress failed: %v", ingressName.Namespace, ingressName.Name, err)
			}
			continue
		}
		newIngressTLSList := r.createIngressCertSecretCopies(ctx, ing)
		// generate a new name for the ingress copy
		newIngName := generateNewName(ingressName.Name)
		newIngressNSN := types.NamespacedName{Name: newIngName, Namespace: ingressName.Namespace}
		err = r.createIngressCopy(newIngressNSN, ing, newIngressTLSList)
		if err != nil {
			ctx.Log().Infof("Failed to create a copy of Rancher ingress %s/%s - create ingress failed: %v", ingressName.Namespace, ingressName.Name, err)
			continue
		}
		ctx.Log().Infof("Created copy of Rancher ingress %v, new ingress is %v", ingressName, newIngressNSN)
	}
}

func (r *Reconciler) createIngressCopy(newIngressName types.NamespacedName, existingIngress netv1.Ingress, newIngressTLSList []netv1.IngressTLS) error {
	newIngSpec := existingIngress.Spec.DeepCopy()
	newIngSpec.TLS = newIngressTLSList
	newIng := netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   newIngressName.Namespace,
			Name:        newIngressName.Name,
			Labels:      existingIngress.Labels,
			Annotations: existingIngress.Annotations,
		},
		Spec: *newIngSpec,
	}
	return r.Create(context.TODO(), &newIng)
}

func (r *Reconciler) createIngressCertSecretCopies(ctx spi.ComponentContext, ing netv1.Ingress) []netv1.IngressTLS {
	newIngressTLSList := []netv1.IngressTLS{}
	for _, ingTLS := range ing.Spec.TLS {
		if ingTLS.SecretName != "" {
			tlsSecret := corev1.Secret{}
			tlsSecretName := types.NamespacedName{Namespace: ing.Namespace, Name: ingTLS.SecretName}
			err := r.Get(context.TODO(), tlsSecretName, &tlsSecret)
			if err == nil {
				newSecretNSN := types.NamespacedName{Name: generateNewName(tlsSecret.Name), Namespace: tlsSecret.Namespace}
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
	newSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   newName.Namespace,
			Name:        newName.Name,
			Labels:      existingSecret.Labels,
			Annotations: existingSecret.Annotations,
		},
		Data: existingSecret.Data,
	}
	return r.Create(context.TODO(), &newSecret)
}

func generateNewName(existingName string) string {
	return existingName + "-" + uuid.NewString()[:7]
}
