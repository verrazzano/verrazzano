// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package virtualservice

import (
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	istio "istio.io/api/networking/v1beta1"
	"strings"
)

// getPathsFromRule gets the paths from a trait.
// If the trait has no paths a default path is returned.
func GetPathsFromRule(rule vzapi.IngressRule) []vzapi.IngressPath {
	paths := rule.Paths
	// If there are no paths create a default.
	if len(paths) == 0 {
		paths = []vzapi.IngressPath{{Path: "/", PathType: "prefix"}}
	}
	return paths
}

// createVirtualServiceMatchURIFromIngressTraitPath create the virtual service match uri map from an ingress trait path
// This is primarily used to setup defaults when either path or type are not present in the ingress path.
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
