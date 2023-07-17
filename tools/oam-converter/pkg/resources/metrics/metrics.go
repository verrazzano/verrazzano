// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metrics

import (
	_ "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	monitor "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/resources/servicemonitor"
	"github.com/verrazzano/verrazzano/tools/oam-converter/pkg/types"

)

func CreateMetricsChildResources(conversionComponent *types.ConversionComponents) error {

	err := monitor.CreateServiceMonitor(conversionComponent)
	if err != nil {
		return err

	}
	return nil
}
