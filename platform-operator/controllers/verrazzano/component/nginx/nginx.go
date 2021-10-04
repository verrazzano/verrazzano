// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package nginx

import (
	"context"
	"github.com/verrazzano/verrazzano/pkg/bom"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ComponentName is the name of the component
const ComponentName = "ingress-controller"

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
	ingressConfig := cr.Spec.Components.Ingress
	if ingressConfig == nil {
		return []bom.KeyValue{}, nil
	}
	ingressType := vzapi.LoadBalancer
	switch ingressConfig.Type {
	case vzapi.NodePort:
		ingressType = vzapi.NodePort
	}
	var kvs []bom.KeyValue
	kvs = append(kvs, bom.KeyValue{Key: "controller.service.type", Value: string(ingressType)})
	kvs = append(kvs, helm.GetInstallArgs(ingressConfig.NGINXInstallArgs)...)
	return kvs, nil
}

func PostInstall(log *zap.SugaredLogger, c client.Client, cr *vzapi.Verrazzano, releaseName string, namespace string, dryRun bool) error {
	log.Infof("Adding label needed by network policies to ingress-nginx namespace")
	ns := v1.Namespace{}
	if err := c.Get(context.TODO(), types.NamespacedName{Namespace: namespace}, &ns); err != nil {
		return err
	}
	if ns.Labels == nil {
		ns.Labels = make(map[string]string)
	}
	ns.Labels["verrazzano.io/namespace"] = "ingress-nginx"
	if err := c.Update(context.TODO(), &ns); err != nil {
		return err
	}

	// Add any port specs needed to the service after boot
	ingressConfig := cr.Spec.Components.Ingress
	if ingressConfig == nil {
		return nil
	}
	if len(ingressConfig.Ports) == 0 {
		return nil
	}
	svc := v1.Service{}
	if err := c.Get(context.TODO(), types.NamespacedName{Name: controllerName, Namespace: "ingress-nginx"}, &svc); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	svc.Spec.Ports = append(svc.Spec.Ports, ingressConfig.Ports...)
	if err := c.Update(context.TODO(), &svc); err != nil {
		return err
	}
	return nil
}
