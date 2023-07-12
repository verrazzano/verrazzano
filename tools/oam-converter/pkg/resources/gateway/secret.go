package gateway

import (
	coallateHosts "github.com/verrazzano/verrazzano/pkg/ingresstrait"
	"github.com/verrazzano/verrazzano/tools/oam-converter/pkg/types"
	vsapi "istio.io/client-go/pkg/apis/networking/v1beta1"
)

func CreateCertificateAndSecret(conversionComponent *types.ConversionComponents) (*vsapi.Gateway, []string, error) {

	// If there are no rules, create a single default rule

	// Create a list of unique hostnames across all rules in the trait
	allHostsForTrait, err := coallateHosts.CoallateAllHostsForTrait(conversionComponent.IngressTrait, conversionComponent.AppName, conversionComponent.AppNamespace)
	if err != nil {
		print(err)
		return nil, nil, err
	}
	// Generate the certificate and secret for all hosts in the trait rules
	secretName := CreateGatewaySecret(conversionComponent.IngressTrait, allHostsForTrait, conversionComponent.AppNamespace, conversionComponent.AppName)
	if secretName != "" {
		gwName, err := BuildGatewayName(conversionComponent.IngressTrait, conversionComponent.ComponentName, conversionComponent.AppNamespace)
		if err != nil {
			print(err)
			return nil, nil, err
		}
		gateway, err := CreateGateway(conversionComponent.ComponentName, conversionComponent.IngressTrait, allHostsForTrait, gwName, secretName)
		if err != nil {
			print(err)
			return nil, nil, err
		}
		return gateway, allHostsForTrait, nil
	}
	return nil, nil, nil
}
