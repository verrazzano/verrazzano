package rancher

import (
	"context"
	"errors"
	"fmt"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/nginx"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//patchRancherDeployment CRI-O does not deliver MKNOD by default, until https://github.com/rancher/rancher/pull/27582 is merged we must add the capability
func patchRancherDeployment(c client.Client) error {
	deployment := appsv1.Deployment{}
	namespacedName := types.NamespacedName{Name: ComponentName, Namespace: ComponentNamespace}
	if err := c.Get(context.TODO(), namespacedName, &deployment); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	deploymentMerge := client.MergeFrom(deployment.DeepCopy())
	containers := deployment.Spec.Template.Spec.Containers
	container, ok := getRancherContainer(containers)
	if !ok {
		return errors.New("rancher container was not found")
	}
	container.SecurityContext = &v1.SecurityContext{
		Capabilities: &v1.Capabilities{
			Add: []v1.Capability{"MKNOD"},
		},
	}

	if err := c.Patch(context.TODO(), &deployment, deploymentMerge); err != nil {
		return err
	}

	return nil
}

func getRancherContainer(containers []v1.Container) (v1.Container, bool) {
	for _, container := range containers {
		if container.Name == ComponentName {
			return container, true
		}
	}

	return v1.Container{}, false
}

//patchRancherIngress annotates the Rancher ingress with environment specific values
func patchRancherIngress(c client.Client, vz *vzapi.Verrazzano) error {
	cm := vz.Spec.Components.CertManager
	if cm == nil {
		return errors.New("CertificateManager was not found in the effective CR")
	}
	dnsSuffix, err := nginx.GetDNSSuffix(c, vz)
	if err != nil {
		return err
	}
	namespacedName := types.NamespacedName{
		Namespace: ComponentNamespace,
		Name:      ComponentName,
	}
	ingress := &networking.Ingress{}
	if err := c.Get(context.TODO(), namespacedName, ingress); err != nil {
		return err
	}
	ingressMerge := client.MergeFrom(ingress.DeepCopy())
	ingress.Annotations["kubernetes.io/tls-acme"] = "true"
	if (cm.Certificate.Acme != vzapi.Acme{}) {
		addAcmeIngressAnnotations(vz.Spec.EnvironmentName, dnsSuffix, ingress)
	} else {
		addCAIngressAnnotations(vz.Spec.EnvironmentName, dnsSuffix, ingress)
	}
	return c.Patch(context.TODO(), ingress, ingressMerge)
}

//addAcmeIngressAnnotations annotate ingress with ACME specific values
func addAcmeIngressAnnotations(name, dnsSuffix string, ingress *networking.Ingress) {
	ingress.Annotations["nginx.ingress.kubernetes.io/auth-realm"] = fmt.Sprintf("%s auth", dnsSuffix)
	ingress.Annotations["external-dns.alpha.kubernetes.io/target"] = fmt.Sprintf("verrazzano-ingress.%s.%s", name, dnsSuffix)
	ingress.Annotations["cert-manager.io/issuer"] = "null"
	ingress.Annotations["cert-manager.io/issuer-kind"] = "null"
	ingress.Annotations["external-dns.alpha.kubernetes.io/ttl"] = "60"
}

//addCAIngressAnnotations annotate ingress with custom CA specific values
func addCAIngressAnnotations(name, dnsSuffix string, ingress *networking.Ingress) {
	ingress.Annotations["nginx.ingress.kubernetes.io/auth-realm"] = fmt.Sprintf("%s.%s auth", name, dnsSuffix)
	ingress.Annotations["cert-manager.io/cluster-issuer"] = "verrazzano-cluster-issuer"
}
