// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package resources

import (
	"errors"
	"github.com/verrazzano/verrazzano/tools/oam-converter/pkg/resources/coherenceresources"
	"github.com/verrazzano/verrazzano/tools/oam-converter/pkg/resources/metrics"
	weblogic "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/resources/weblogicresources"
	"github.com/verrazzano/verrazzano/tools/oam-converter/pkg/types"
)

func CreateKubeResources(conversionComponents []*types.ConversionComponents) (*types.KubeRecources, error) {
	outputResources := types.KubeRecources{}
	for _, conversionComponent := range conversionComponents {
		if conversionComponent.MetricsTrait != nil {
			serviceMonitor, err := metrics.CreateMetricsChildResources(conversionComponent)
			if err != nil {
				return &outputResources,errors.New("failed to create Child resources from Metrics Trait")
			}
			outputResources.ServiceMonitors = append(outputResources.ServiceMonitors, serviceMonitor)
		}
		if conversionComponent.WeblogicworkloadMap != nil {
			//conversionComponent.AppNamespace, conversionComponent.AppName, conversionComponent.ComponentName, conversionComponent.IngressTrait, conversionComponent.WeblogicworkloadMap[conversionComponent.ComponentName]
			err := weblogic.CreateIngressChildResourcesFromWeblogic(conversionComponent)
			if err != nil {
				return &outputResources,errors.New("failed to create Child resources from Weblogic workload")

			}
		}
		if conversionComponent.Coherenceworkload != nil {
			err := coherenceresources.CreateIngressChildResourcesFromCoherence(conversionComponent)
			if err != nil {
				return &outputResources,errors.New("failed to create Child resources from Coherence workload")

			}
		}
		if conversionComponent.Helidonworkload != nil {
			////err := helidonresources.CreateIngressChildResourcesFromHelidon(conversionComponent)
			//if err != nil {
			//	return &outputResources,errors.New("failed to create Child resources from Helidon workload")
			//
			//}
		}
	}
	return &outputResources,nil
}
