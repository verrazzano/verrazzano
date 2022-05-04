// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	"context"
	"fmt"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	securityv1beta1 "istio.io/api/security/v1beta1"
	istiov1beta1 "istio.io/api/type/v1beta1"
	istioclisec "istio.io/client-go/pkg/apis/security/v1beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

func configureJaeger(ctx spi.ComponentContext) ([]vzapi.InstallArgs, error) {
	if !vzconfig.IsJaegerOperatorEnabled(ctx.EffectiveCR()) {
		return nil, nil
	}
	service, err := findFirstJaegerCollectorService(ctx)
	if err != nil {
		return nil, err
	}

	if service != nil {
		if err := createJaegerPeerAuthentication(ctx, service.Namespace); err != nil {
			return nil, err
		}

		port := zipkinPort(*service)
		collectorURL := fmt.Sprintf("%s.%s.svc.cluster.local:%d",
			service.Name,
			service.Namespace,
			port,
		)
		return []vzapi.InstallArgs{
			{
				Name:  meshConfigTracingTLSMode,
				Value: "ISTIO_MUTUAL",
			},
			{
				Name:  meshConfigTracingAddress,
				Value: collectorURL,
			},
		}, nil
	}

	return nil, nil
}

//findFirstJaegerCollectorService gets the first Jaeger collector service in the cluster that is not a headless service
func findFirstJaegerCollectorService(ctx spi.ComponentContext) (*v1.Service, error) {
	services := &v1.ServiceList{}
	selector := clipkg.MatchingLabels{
		constants.KubernetesAppLabel: constants.JaegerCollectorService,
	}
	if err := ctx.Client().List(context.TODO(), services, selector); err != nil {
		return nil, err
	}
	for idx, service := range services.Items {
		if !strings.Contains(service.Name, "headless") {
			return &services.Items[idx], nil
		}
	}
	return nil, nil
}

//zipkinPort retrieves the zipkin port from the service, if it is present. Defaults to 9411 for Jaeger collector
func zipkinPort(service v1.Service) int32 {
	for _, port := range service.Spec.Ports {
		if port.Name == "http-zipkin" {
			return port.Port
		}
	}
	return 9411
}

//createJaegerPeerAuthentication creates a PeerAuthentication resource specific for the Jaeger workload
// This resource is PERMISSIVE, to allow applications outside the Istio mesh to export traces to Jaeger
func createJaegerPeerAuthentication(ctx spi.ComponentContext, namespace string) error {
	peerAuthentication := &istioclisec.PeerAuthentication{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jaeger,
			Namespace: namespace,
		},
	}
	_, err := controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), peerAuthentication, func() error {
		peerAuthentication.Spec = securityv1beta1.PeerAuthentication{
			Selector: &istiov1beta1.WorkloadSelector{
				MatchLabels: map[string]string{
					"app": jaeger,
				},
			},
			Mtls: &securityv1beta1.PeerAuthentication_MutualTLS{
				Mode: securityv1beta1.PeerAuthentication_MutualTLS_PERMISSIVE,
			},
		}
		return nil
	})
	return err
}
