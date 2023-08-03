//Copyright (c) 2023, Oracle and/or its affiliates.
//Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package virtualservice

import (
	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	consts "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/constants"
	istio "istio.io/api/networking/v1beta1"
	vsapi "istio.io/client-go/pkg/apis/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestGetPathsFromRule(t *testing.T) {
	// Create a sample IngressRule with paths
	ruleWithPaths := vzapi.IngressRule{
		Paths: []vzapi.IngressPath{
			{Path: "/api/v1", PathType: "prefix"},
			{Path: "/app", PathType: "exact"},
		},
	}

	// Call the function with the sample IngressRule
	pathsWithRule := GetPathsFromRule(ruleWithPaths)

	// Check if the function returns the correct paths
	assert.Equal(t, ruleWithPaths.Paths, pathsWithRule, "Unexpected paths returned when rule has paths")

	// Create a sample IngressRule with no paths
	ruleWithoutPaths := vzapi.IngressRule{}

	// Call the function with the sample IngressRule
	pathsWithoutRule := GetPathsFromRule(ruleWithoutPaths)

	// Check if the function returns the default path when there are no paths in the rule
	expectedDefaultPath := []vzapi.IngressPath{{Path: "/", PathType: "prefix"}}
	assert.Equal(t, expectedDefaultPath, pathsWithoutRule, "Unexpected paths returned when rule has no paths")
}

func TestCreateVirtualServiceMatchURIFromIngressTraitPath(t *testing.T) {
	// Create a sample IngressPath with path set to "/api/v1" and pathType set to "prefix"
	path := vzapi.IngressPath{
		Path:     "/api/v1",
		PathType: "prefix",
	}

	// Call the function with the sample IngressPath
	match := CreateVirtualServiceMatchURIFromIngressTraitPath(path)

	// Check if the function returns the correct string match for the given path and pathType
	expectedMatch := &istio.StringMatch{MatchType: &istio.StringMatch_Prefix{Prefix: "/api/v1"}}
	assert.Equal(t, expectedMatch, match, "Unexpected StringMatch for path with prefix")

	// Create a sample IngressPath with path set to "/app" and pathType set to "exact"
	path = vzapi.IngressPath{
		Path:     "/app",
		PathType: "exact",
	}

	// Call the function with the sample IngressPath
	match = CreateVirtualServiceMatchURIFromIngressTraitPath(path)

	// Check if the function returns the correct string match for the given path and pathType
	expectedMatch = &istio.StringMatch{MatchType: &istio.StringMatch_Exact{Exact: "/app"}}
	assert.Equal(t, expectedMatch, match, "Unexpected StringMatch for path with exact")

	// Create a sample IngressPath with an empty path and pathType not specified
	path = vzapi.IngressPath{
		Path: "",
	}

	// Call the function with the sample IngressPath
	match = CreateVirtualServiceMatchURIFromIngressTraitPath(path)

	// Check if the function returns the default StringMatch for an empty path (prefix type)
	expectedMatch = &istio.StringMatch{MatchType: &istio.StringMatch_Prefix{Prefix: "/"}}
	assert.Equal(t, expectedMatch, match, "Unexpected StringMatch for empty path (prefix)")

	// Create a sample IngressPath with path set to "/" and pathType set to "regex"
	path = vzapi.IngressPath{
		Path:     "/",
		PathType: "regex",
	}

	// Call the function with the sample IngressPath
	match = CreateVirtualServiceMatchURIFromIngressTraitPath(path)

	// Check if the function returns the correct string match for the given path and pathType
	expectedMatch = &istio.StringMatch{MatchType: &istio.StringMatch_Regex{Regex: "/"}}
	assert.Equal(t, expectedMatch, match, "Unexpected StringMatch for path with regex")
}

func TestCreateVirtualService(t *testing.T) {
	// Create a sample IngressTrait, IngressRule, allHostsForTrait, name, and Gateway
	ingressTrait := &vzapi.IngressTrait{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-namespace",
		},
	}
	rule := vzapi.IngressRule{
		Paths: []vzapi.IngressPath{
			{Path: "/api/v1", PathType: "prefix"},
		},
	}
	allHostsForTrait := []string{"example.com"}
	name := "test-virtualservice"
	gateway := &vsapi.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-gateway",
		},
	}

	// Call the function with the sample inputs
	virtualService, err := CreateVirtualService(ingressTrait, rule, allHostsForTrait, name, gateway)

	// Check if the function returns the VirtualService with the correct settings
	expectedVirtualService := &vsapi.VirtualService{
		TypeMeta: metav1.TypeMeta{
			APIVersion: consts.VirtualServiceAPIVersion,
			Kind:       "VirtualService",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-namespace",
			Name:      "test-virtualservice",
		},
	}

	matches := []*istio.HTTPMatchRequest{
		{
			Uri: &istio.StringMatch{MatchType: &istio.StringMatch_Prefix{Prefix: "/api/v1"}},
		},
	}
	route := []*istio.HTTPRoute{
		{
			Match: matches,
			Route: []*istio.HTTPRouteDestination{
				{

					Destination: &istio.Destination{
						Host: "",
					},
				},
			},
		},
	}
	expectedVirtualService.Spec.Gateways = []string{"test-gateway"}
	expectedVirtualService.Spec.Hosts = []string{"example.com"}
	expectedVirtualService.Spec.Http = route
	assert.Nil(t, err, "Unexpected error returned from CreateVirtualService")
	assert.Equal(t, expectedVirtualService, virtualService, "Unexpected VirtualService returned")
}
