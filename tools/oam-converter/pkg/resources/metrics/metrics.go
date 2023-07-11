// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metrics

import (
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	monitor "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/resources/servicemonitor"
)

func CreateMetricsChildResources(metricstrait *vzapi.MetricsTrait) {
	//createService()
	monitor.CreateServiceMonitor(metricstrait)
}
