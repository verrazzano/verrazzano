package destinationRule

import (
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	istio "istio.io/api/networking/v1beta1"
)

// createDestinationFromRuleOrService creates a destination from either the rule or the service.
// If the rule contains destination information that is used.
func CreateDestinationFromRule(rule vzapi.IngressRule) (*istio.HTTPRouteDestination, error) {
	dest := &istio.HTTPRouteDestination{Destination: &istio.Destination{Host: rule.Destination.Host}}
	if rule.Destination.Port != 0 {
		dest.Destination.Port = &istio.PortSelector{Number: rule.Destination.Port}
	}
	return dest, nil

}
