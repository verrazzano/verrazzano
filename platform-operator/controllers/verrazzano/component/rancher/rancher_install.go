// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"context"
	"errors"
	"fmt"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// patchRancherDeployment CRI-O does not deliver MKNOD by default, until https://github.com/rancher/rancher/pull/27582 is merged we must add the capability
func patchRancherDeployment(c client.Client) error {
	deployment := appsv1.Deployment{}
	namespacedName := types.NamespacedName{Name: common.RancherName, Namespace: common.CattleSystem}
	if err := c.Get(context.TODO(), namespacedName, &deployment); err != nil {
		return err
	}
	deploymentMerge := client.MergeFrom(deployment.DeepCopy())
	ok := false
	for i, container := range deployment.Spec.Template.Spec.Containers {
		if container.Name == common.RancherName {
			container.SecurityContext = &v1.SecurityContext{
				Capabilities: &v1.Capabilities{
					Add: []v1.Capability{"MKNOD"},
				},
			}
			ok = true
			deployment.Spec.Template.Spec.Containers[i] = container
		}
	}
	if !ok {
		return errors.New("container 'rancher' was not found")
	}

	return c.Patch(context.TODO(), &deployment, deploymentMerge)
}

// patchRancherIngress annotates the Rancher ingress with environment specific values
func patchRancherIngress(c client.Client, vz *vzapi.Verrazzano) error {
	cm := vz.Spec.Components.CertManager
	if cm == nil {
		return errors.New("CertificateManager was not found in the effective CR")
	}
	dnsSuffix, err := vzconfig.GetDNSSuffix(c, vz)
	if err != nil {
		return err
	}
	namespacedName := types.NamespacedName{
		Namespace: common.CattleSystem,
		Name:      common.RancherName,
	}
	ingress := &networking.Ingress{}
	if err := c.Get(context.TODO(), namespacedName, ingress); err != nil {
		return err
	}
	ingressMerge := client.MergeFrom(ingress.DeepCopy())
	ingress.Annotations["kubernetes.io/tls-acme"] = "true"
	ingress.Annotations["nginx.ingress.kubernetes.io/backend-protocol"] = "HTTPS"
	ingress.Annotations["nginx.ingress.kubernetes.io/force-ssl-redirect"] = "true"
	if (cm.Certificate.Acme != vzapi.Acme{}) {
		addAcmeIngressAnnotations(vz.Spec.EnvironmentName, dnsSuffix, ingress)
	} else {
		addCAIngressAnnotations(vz.Spec.EnvironmentName, dnsSuffix, ingress)
	}
	return c.Patch(context.TODO(), ingress, ingressMerge)
}

// addAcmeIngressAnnotations annotate ingress with ACME specific values
func addAcmeIngressAnnotations(name, dnsSuffix string, ingress *networking.Ingress) {
	ingress.Annotations["nginx.ingress.kubernetes.io/auth-realm"] = fmt.Sprintf("%s auth", dnsSuffix)
	ingress.Annotations["external-dns.alpha.kubernetes.io/target"] = fmt.Sprintf("verrazzano-ingress.%s.%s", name, dnsSuffix)
	ingress.Annotations["external-dns.alpha.kubernetes.io/ttl"] = "60"
	// Remove any existing cert manage annotations
	delete(ingress.Annotations, "cert-manager.io/issuer")
	delete(ingress.Annotations, "cert-manager.io/issuer-kind")
}

// addCAIngressAnnotations annotate ingress with custom CA specific values
func addCAIngressAnnotations(name, dnsSuffix string, ingress *networking.Ingress) {
	ingress.Annotations["nginx.ingress.kubernetes.io/auth-realm"] = fmt.Sprintf("%s.%s auth", name, dnsSuffix)
	ingress.Annotations["cert-manager.io/cluster-issuer"] = "verrazzano-cluster-issuer"
	ingress.Annotations["cert-manager.io/common-name"] = fmt.Sprintf("%s.%s.%s", common.RancherName, name, dnsSuffix)
}
