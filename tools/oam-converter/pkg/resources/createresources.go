package resources

import (
	"fmt"
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
				fmt.Printf("Failed to create Child resources from Weblogic workload", err)
				return err
			}
		}
		if conversionComponent.Coherenceworkload != nil {
			err := coherenceresources.CreateIngressChildResourcesFromCoherence(conversionComponent)
			if err != nil {
				fmt.Printf("Failed to create Child resources from Coherence workload", err)
				return err
			}
		}
		if conversionComponent.Helidonworkload != nil {
			err := helidonresources.CreateIngressChildResourcesFromHelidon(conversionComponent)
			if err != nil {
				fmt.Printf("Failed to create Child resources from Helidon workload", err)
				return err
			}
		}
	}
	return nil
}
