// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package resources

import (
	"errors"
	"github.com/verrazzano/verrazzano/tools/oam-converter/pkg/resources/coherenceresources"
	"github.com/verrazzano/verrazzano/tools/oam-converter/pkg/resources/helidonresources"
	weblogic "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/resources/weblogicresources"
	"github.com/verrazzano/verrazzano/tools/oam-converter/pkg/types"
)

func CreateResources(conversionComponents []*types.ConversionComponents) error {
	for _, conversionComponent := range conversionComponents {
		if conversionComponent.WeblogicworkloadMap != nil {
			//conversionComponent.AppNamespace, conversionComponent.AppName, conversionComponent.ComponentName, conversionComponent.IngressTrait, conversionComponent.WeblogicworkloadMap[conversionComponent.ComponentName]
			err := weblogic.CreateIngressChildResourcesFromWeblogic(conversionComponent)
			if err != nil {
				return errors.New("failed to create Child resources from Weblogic workload")

			}
		}
		if conversionComponent.Coherenceworkload != nil {
			err := coherenceresources.CreateIngressChildResourcesFromCoherence(conversionComponent)
			if err != nil {
				return errors.New("failed to create Child resources from Coherence workload")

			}
		}
		if conversionComponent.Helidonworkload != nil {
			err := helidonresources.CreateIngressChildResourcesFromHelidon(conversionComponent)
			if err != nil {
				return errors.New("failed to create Child resources from Helidon workload")

			}
		}
	}
	return nil
}
