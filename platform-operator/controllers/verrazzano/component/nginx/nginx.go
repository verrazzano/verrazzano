// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package nginx

import (
	"context"
	"fmt"

	"github.com/verrazzano/verrazzano/pkg/bom"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vpoconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// ValuesFileOverride Name of the values file override for NGINX
	ValuesFileOverride = "ingress-nginx-values.yaml"

	ControllerName = vpoconst.NGINXControllerServiceName
	backendName    = "ingress-controller-ingress-nginx-defaultbackend"
)

func isNginxReady(context spi.ComponentContext) bool {
	deployments := []types.NamespacedName{
		{Name: ControllerName, Namespace: ComponentNamespace},
		{Name: backendName, Namespace: ComponentNamespace},
	}
	prefix := fmt.Sprintf("Component %s", context.GetComponent())
	return status.DeploymentsReady(context.Log(), context.Client(), deployments, 1, prefix)
}

func AppendOverrides(context spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	cr := context.EffectiveCR()
	ingressType, err := vzconfig.GetServiceType(cr)
	if err != nil {
		return []bom.KeyValue{}, err
	}

	newKvs := append(kvs, bom.KeyValue{Key: "controller.service.type", Value: string(ingressType)})

	if cr.Spec.Components.DNS != nil && cr.Spec.Components.DNS.OCI != nil {
		newKvs = append(newKvs, bom.KeyValue{Key: "controller.service.annotations.external-dns\\.alpha\\.kubernetes\\.io/ttl", Value: "60", SetString: true})
		hostName := fmt.Sprintf("verrazzano-ingress.%s.%s", cr.Spec.EnvironmentName, cr.Spec.Components.DNS.OCI.DNSZoneName)
		newKvs = append(newKvs, bom.KeyValue{Key: "controller.service.annotations.external-dns\\.alpha\\.kubernetes\\.io/hostname", Value: hostName})
	}

	// Convert NGINX install-args to helm overrides
	newKvs = append(newKvs, helm.GetInstallArgs(getInstallArgs(cr))...)
	return newKvs, nil
}

// PreInstall Create and label the NGINX namespace, and create any override helm args needed
func PreInstall(compContext spi.ComponentContext, name string, namespace string, dir string) error {
	if compContext.IsDryRun() {
		compContext.Log().Debug("NGINX PostInstall dry run")
		return nil
	}
	compContext.Log().Debug("Adding label needed by network policies to ingress-nginx namespace")
	ns := v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
	if _, err := controllerruntime.CreateOrUpdate(context.TODO(), compContext.Client(), &ns, func() error {
		if ns.Labels == nil {
			ns.Labels = make(map[string]string)
		}
		ns.Labels["verrazzano.io/namespace"] = "ingress-nginx"
		ns.Labels["istio-injection"] = "enabled"
		return nil
	}); err != nil {
		return err
	}
	return nil
}

// PostInstall Patch the controller service ports based on any user-supplied overrides
func PostInstall(ctx spi.ComponentContext, _ string, _ string) error {
	if ctx.IsDryRun() {
		ctx.Log().Debug("NGINX PostInstall dry run")
		return nil
	}
	// Add any port specs needed to the service after boot
	ingressConfig := ctx.EffectiveCR().Spec.Components.Ingress
	if ingressConfig == nil {
		return nil
	}
	if len(ingressConfig.Ports) == 0 {
		return nil
	}

	c := ctx.Client()
	svcPatch := v1.Service{}
	if err := c.Get(context.TODO(), types.NamespacedName{Name: ControllerName, Namespace: ComponentNamespace}, &svcPatch); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	mergeFromSvc := client.MergeFrom(svcPatch.DeepCopy())
	svcPatch.Spec.Ports = ingressConfig.Ports
	if err := c.Patch(context.TODO(), &svcPatch, mergeFromSvc); err != nil {
		return err
	}
	return nil
}

// getInstallArgs get the install args for NGINX
func getInstallArgs(cr *vzapi.Verrazzano) []vzapi.InstallArgs {
	if cr.Spec.Components.Ingress == nil {
		return []vzapi.InstallArgs{}
	}
	return cr.Spec.Components.Ingress.NGINXInstallArgs
}
