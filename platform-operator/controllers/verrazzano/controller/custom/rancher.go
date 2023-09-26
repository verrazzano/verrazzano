// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package custom

import (
	"context"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/result"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/rancher"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	componentspi "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strings"
)

// CreateRancherIngressAndCertCopies - creates copies of Rancher ingress and cert secret if
// they exist. If there is a DNS update in progress, we want to do this so that the old DNS
// continues to work as long as there is a good DNS entry, so that managed clusters can continue
// connecting using the old Rancher DNS in their kubeconfig, until they are updated with the new DNS.
func CreateRancherIngressAndCertCopies(componentCtx componentspi.ComponentContext) {
	ok, rancherComp := registry.FindComponent(rancher.ComponentName)
	if !ok {
		return
	}

	dnsSuffix := getDNSSuffix(componentCtx.EffectiveCR())
	if dnsSuffix == "" {
		componentCtx.Log().Debug("Empty DNS suffix, skipping Rancher ingress copy")
		return
	}

	for _, ingressName := range rancherComp.GetIngressNames(componentCtx) {
		ing := netv1.Ingress{}
		err := componentCtx.Client().Get(context.TODO(), ingressName, &ing)
		if err != nil {
			if !errors.IsNotFound(err) {
				componentCtx.Log().Infof("Cannot make a copy of Rancher ingress %s/%s - get ingress failed: %v", ingressName.Namespace, ingressName.Name, err)
			}
			continue
		}

		if len(ing.Spec.TLS) == 0 || len(ing.Spec.TLS[0].Hosts) == 0 {
			continue
		}
		if strings.HasSuffix(ing.Spec.TLS[0].Hosts[0], dnsSuffix) {
			componentCtx.Log().Debugf("Rancher ingress host %s has DNS suffix %s, skipping copy", ing.Spec.TLS[0].Hosts[0], dnsSuffix)
			continue
		}

		newIngressTLSList := createIngressCertSecretCopies(componentCtx, ing)
		newIngressNSN := types.NamespacedName{Name: "vz-" + ing.Name, Namespace: ingressName.Namespace}
		err = createIngressCopy(componentCtx.Client(), newIngressNSN, ing, newIngressTLSList)
		if err != nil {
			componentCtx.Log().Infof("Failed to create a copy of Rancher ingress %s/%s - create ingress failed: %v", ingressName.Namespace, ingressName.Name, err)
			continue
		}
		componentCtx.Log().Infof("Created copy of Rancher ingress %v, new ingress is %v", ingressName, newIngressNSN)
	}
}

func createIngressCopy(cli client.Client, newIngressName types.NamespacedName, existingIngress netv1.Ingress, newIngressTLSList []netv1.IngressTLS) error {
	ingress := &netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      newIngressName.Name,
			Namespace: newIngressName.Namespace,
		},
	}
	_, err := controllerutil.CreateOrUpdate(context.TODO(), cli, ingress, func() error {
		ingress.Labels = existingIngress.Labels
		if ingress.Labels == nil {
			ingress.Labels = map[string]string{}
		}
		ingress.Labels[constants.VerrazzanoManagedLabelKey] = "true"
		ingress.Annotations = existingIngress.Annotations
		ingress.Spec = *existingIngress.Spec.DeepCopy()
		ingress.Spec.TLS = newIngressTLSList
		return nil
	})
	return err
}

func createIngressCertSecretCopies(componentCtx componentspi.ComponentContext, ing netv1.Ingress) []netv1.IngressTLS {
	newIngressTLSList := []netv1.IngressTLS{}
	for _, ingTLS := range ing.Spec.TLS {
		if ingTLS.SecretName != "" {
			tlsSecret := corev1.Secret{}
			tlsSecretName := types.NamespacedName{Namespace: ing.Namespace, Name: ingTLS.SecretName}
			err := componentCtx.Client().Get(context.TODO(), tlsSecretName, &tlsSecret)
			if err == nil {
				newSecretNSN := types.NamespacedName{Name: "vz-" + tlsSecret.Name, Namespace: tlsSecret.Namespace}
				if err := createSecretCopy(componentCtx.Client(), newSecretNSN, tlsSecret); err != nil {
					componentCtx.Log().Infof("Failed to create copy %v of Rancher TLS secret %v", newSecretNSN, tlsSecretName)
				}
				newIngressTLS := ingTLS.DeepCopy()
				newIngressTLS.SecretName = newSecretNSN.Name
				newIngressTLSList = append(newIngressTLSList, *newIngressTLS)
			}
		}
	}
	return newIngressTLSList
}

func createSecretCopy(cli client.Client, newName types.NamespacedName, existingSecret corev1.Secret) error {
	newSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: newName.Namespace,
			Name:      newName.Name,
		},
	}
	_, err := controllerutil.CreateOrUpdate(context.TODO(), cli, newSecret, func() error {
		newSecret.Labels = existingSecret.Labels
		newSecret.Annotations = existingSecret.Annotations
		newSecret.Data = existingSecret.Data
		return nil
	})
	return err
}

func getDNSSuffix(effectiveCR *vzv1alpha1.Verrazzano) string {
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

// preUninstallRancherLocal does Rancher pre-uninstall
func PreUninstallRancher(cli client.Client, log vzlog.VerrazzanoLogger, actualCR *vzv1alpha1.Verrazzano, effectiveCR *vzv1alpha1.Verrazzano) result.Result {
	rancherProvisioned, err := rancher.IsClusterProvisionedByRancher()
	if err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}

	// Don't remove Rancher local namespace if cluster was provisioned by Rancher (for example RKE2).  Removing
	// will cause cluster corruption.
	if rancherProvisioned {
		return result.NewResult()
	}
	// If Rancher is installed, then delete local cluster
	found, comp := registry.FindComponent(rancher.ComponentName)
	if !found {
		return result.NewResult()
	}

	spicomponentCtx, err := componentspi.NewContext(log, cli, actualCR, nil, false)
	if err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}
	compContext := spicomponentCtx.Init(rancher.ComponentName).Operation(vzconst.UninstallOperation)
	installed, err := comp.IsInstalled(compContext)
	if err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}
	if !installed {
		return result.NewResult()
	}
	rancher.DeleteLocalCluster(log, cli)

	if err := DeleteMCResources(spicomponentCtx); err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}
	return result.NewResult()
}

func RunRancherPostUninstall(componentCtx componentspi.ComponentContext) error {
	// Look up the Rancher component and call PostUninstall explicitly, without checking if it's installed;
	// this is to catch any lingering managed cluster resources
	if found, comp := registry.FindComponent(rancher.ComponentName); found {
		err := comp.PostUninstall(componentCtx.Init(rancher.ComponentName).Operation(vzconst.UninstallOperation))
		if err != nil {
			componentCtx.Log().Once("Waiting for Rancher post-uninstall cleanup to be done")
			return err
		}
	}
	return nil
}
