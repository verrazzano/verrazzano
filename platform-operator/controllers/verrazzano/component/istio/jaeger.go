// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	"fmt"
	globalconst "github.com/verrazzano/verrazzano/pkg/constants"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
)

const collectorZipkinPort = 9411

//configureJaeger configures Jaeger for Istio integration and returns install args for the Istio install.
// return Istio install args for the tracing endpoint and the Istio tracing TLS mode
func configureJaeger(ctx spi.ComponentContext) ([]vzapi.InstallArgs, error) {
	// During istio bootstrap, if Jaeger operator is not enabled, or if the Jaeger services are not created yet,
	// use the collector URL of the default Jaeger instance that would eventually be created.
	collectorURL := fmt.Sprintf("%s-%s.%s.svc.cluster.local.:%d",
		globalconst.JaegerInstanceName,
		globalconst.JaegerCollectorComponentName,
		constants.VerrazzanoMonitoringNamespace,
		collectorZipkinPort,
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
