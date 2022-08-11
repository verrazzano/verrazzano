// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	"context"
	"fmt"
	globalconst "github.com/verrazzano/verrazzano/pkg/constants"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	v1 "k8s.io/api/core/v1"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

const collectorZipkinPort = 9411

//configureJaeger configures Jaeger for Istio integration and returns install args for the Istio install.
// If a Jaeger collector service is detected in a Verrazzano managed namespace:
// return Istio install args for the tracing endpoint and the Istio tracing TLS mode
func configureJaeger(ctx spi.ComponentContext) ([]vzapi.InstallArgs, error) {
	// During istio bootstrap, if Jaeger operator is not enabled, or if the Jaeger services are not created yet,
	// use the collector URL of the default Jaeger instance that would eventually be created.
	collectorURL := fmt.Sprintf("%s-collector.%s.svc.cluster.local:%d",
		globalconst.JaegerInstanceName,
		constants.VerrazzanoMonitoringNamespace,
		collectorZipkinPort,
	)
	// If there is an existing Jaeger collector service already running in the cluster,
	// use that URL.
	service, err := findFirstJaegerCollectorService(ctx)
	if err != nil {
		return nil, err
	}
	if service != nil {
		port := zipkinPort(*service)
		collectorURL = fmt.Sprintf("%s.%s.svc.cluster.local:%d",
			service.Name,
			service.Namespace,
			port,
		)
	}

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
	return collectorZipkinPort
}
