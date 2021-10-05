// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package nginx

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/bom"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ComponentName is the name of the component
const ComponentName = "ingress-controller"

// Namespace is the NGINX namespace for verrazzano
const ComponentNamespace = "ingress-nginx"

// ValuesFileOverride Name of the values file override for NGINX
const ValuesFileOverride = "ingress-nginx-values.yaml"

const controllerName = "ingress-controller-ingress-nginx-controller"
const backendName = "ingress-controller-ingress-nginx-defaultbackend"

func IsReady(log *zap.SugaredLogger, c client.Client, name string, namespace string) bool {
	deployments := []types.NamespacedName{
		{Name: controllerName, Namespace: namespace},
		{Name: backendName, Namespace: namespace},
	}
	return status.DeploymentsReady(log, c, deployments, 1)
}

func PreInstall(log *zap.SugaredLogger, c client.Client, cr *vzapi.Verrazzano, name string, namespace string, dir string) ([]bom.KeyValue, error) {
	log.Infof("Adding label needed by network policies to ingress-nginx namespace")
	ns := v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
	if _, err := controllerruntime.CreateOrUpdate(context.TODO(), c, &ns, func() error {
		if ns.Labels == nil {
			ns.Labels = make(map[string]string)
		}
		ns.Labels["verrazzano.io/namespace"] = "ingress-nginx"
		ns.Labels["istio-injection"] = "enabled"
		return nil
	}); err != nil {
		return []bom.KeyValue{}, err
	}

	ingressType, err := getIngressType(cr)
	if err != nil {
		return []bom.KeyValue{}, err
	}
	var kvs []bom.KeyValue
	kvs = append(kvs, bom.KeyValue{Key: "controller.service.type", Value: string(ingressType)})

	if cr.Spec.Components.DNS != nil && cr.Spec.Components.DNS.OCI != nil {
		kvs = append(kvs, bom.KeyValue{Key: "controller.service.annotations.external-dns.alpha.kubernetes.io/ttl", Value: "60", SetString: true})
		hostName := fmt.Sprintf("verrazzano-ingress.%s.%s", cr.Spec.EnvironmentName, cr.Spec.Components.DNS.OCI.DNSZoneName)
		kvs = append(kvs, bom.KeyValue{Key: "controller.service.annotations.external-dns.alpha.kubernetes.io/hostname", Value: hostName})
	}

	// Convert NGINX install-args to helm overrides
	kvs = append(kvs, helm.GetInstallArgs(getInstallArgs(cr))...)
	return kvs, nil
}

func getInstallArgs(cr *vzapi.Verrazzano) []vzapi.InstallArgs {
	if cr.Spec.Components.Ingress == nil {
		return []vzapi.InstallArgs{}
	}
	return cr.Spec.Components.Ingress.NGINXInstallArgs
}

func getIngressType(cr *vzapi.Verrazzano) (vzapi.IngressType, error) {
	ingressConfig := cr.Spec.Components.Ingress
	if ingressConfig == nil || len(ingressConfig.Type) == 0 {
		return vzapi.LoadBalancer, nil
	}
	switch ingressConfig.Type {
	case vzapi.NodePort, vzapi.LoadBalancer:
		return ingressConfig.Type, nil
	default:
		return "", fmt.Errorf("Unrecognized ingress type %s", ingressConfig.Type)
	}
}

func PostInstall(log *zap.SugaredLogger, c client.Client, cr *vzapi.Verrazzano, releaseName string, namespace string, dryRun bool) error {
	// Add any port specs needed to the service after boot
	ingressConfig := cr.Spec.Components.Ingress
	if ingressConfig == nil {
		return nil
	}
	if len(ingressConfig.Ports) == 0 {
		return nil
	}
	svc := v1.Service{}
	if err := c.Get(context.TODO(), types.NamespacedName{Name: controllerName, Namespace: ComponentNamespace}, &svc); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	patch := client.MergeFrom(svc.DeepCopy())
	svc.Spec.Ports = append(svc.Spec.Ports, ingressConfig.Ports...)
	if err := c.Patch(context.TODO(), &svc, patch); err != nil {
		return err
	}
	return nil
}
