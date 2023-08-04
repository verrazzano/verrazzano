// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package virtualservice

import (
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	consts "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/constants"
	destination "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/resources/destinationrule"
	istio "istio.io/api/networking/v1beta1"
	vsapi "istio.io/client-go/pkg/apis/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
)

// GetPathsFromRule gets the paths from a trait.
// If the trait has no paths a default path is returned.
func GetPathsFromRule(rule vzapi.IngressRule) []vzapi.IngressPath {
	paths := rule.Paths
	// If there are no paths create a default.
	if len(paths) == 0 {
		paths = []vzapi.IngressPath{{Path: "/", PathType: "prefix"}}
	}
	return paths
}

// CreateVirtualServiceMatchURIFromIngressTraitPath create the virtual service match uri map from an ingress trait path
// This is primarily used to set up defaults when either path or type are not present in the ingress path.
// If the provided ingress path doesn't contain a path it is default to /
// If the provided ingress path doesn't contain a type it is defaulted to prefix if path is / and exact otherwise.
func CreateVirtualServiceMatchURIFromIngressTraitPath(path vzapi.IngressPath) *istio.StringMatch {
	// Default path to /
	p := strings.TrimSpace(path.Path)
	if p == "" {
		p = "/"
	}
	// If path is / default type to prefix
	// If path is not / default to exact
	t := strings.ToLower(strings.TrimSpace(path.PathType))
	if t == "" {
		if p == "/" {
			t = "prefix"
		} else {
			t = "exact"
		}
	}

	switch t {
	case "regex":
		return &istio.StringMatch{MatchType: &istio.StringMatch_Regex{Regex: p}}
	case "prefix":
		return &istio.StringMatch{MatchType: &istio.StringMatch_Prefix{Prefix: p}}
	default:
		return &istio.StringMatch{MatchType: &istio.StringMatch_Exact{Exact: p}}
	}
}

// CreateVirtualService creates the VirtualService child resource of the trait.
func CreateVirtualService(ingresstrait *vzapi.IngressTrait, rule vzapi.IngressRule,
	allHostsForTrait []string, name string, gateway *vsapi.Gateway) (*vsapi.VirtualService, error) {
	virtualService := &vsapi.VirtualService{
		TypeMeta: metav1.TypeMeta{
			APIVersion: consts.VirtualServiceAPIVersion,
			Kind:       "VirtualService",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ingresstrait.Namespace,
			Name:      name,
		},
	}
	return mutateVirtualService(virtualService, rule, allHostsForTrait, gateway)
}

// mutateVirtualService mutates the output virtual service resource
func mutateVirtualService(virtualService *vsapi.VirtualService, rule vzapi.IngressRule, allHostsForTrait []string, gateway *vsapi.Gateway) (*vsapi.VirtualService, error) {
	virtualService.Spec.Gateways = []string{gateway.Name}
	virtualService.Spec.Hosts = allHostsForTrait
	matches := []*istio.HTTPMatchRequest{}
	paths := GetPathsFromRule(rule)
	for _, path := range paths {
		matches = append(matches, &istio.HTTPMatchRequest{
			Uri: CreateVirtualServiceMatchURIFromIngressTraitPath(path)})
	}

	dest, err := destination.CreateDestinationFromRule(rule)

	if err != nil {

		return nil, err
	}
	route := istio.HTTPRoute{
		Match: matches,
		Route: []*istio.HTTPRouteDestination{dest}}
	virtualService.Spec.Http = []*istio.HTTPRoute{&route}

	return virtualService, nil
}
