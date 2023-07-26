// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metrics

import (
	promoperapi "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	monitor "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/resources/servicemonitor"
	"github.com/verrazzano/verrazzano/tools/oam-converter/pkg/types"
)

func CreateMetricsChildResources(conversionComponent *types.ConversionComponents) (*promoperapi.ServiceMonitor, error) {

	serviceMonitor, err := monitor.CreateServiceMonitor(conversionComponent)
	if err != nil {
		return serviceMonitor, err

	}
	return serviceMonitor, nil
}
