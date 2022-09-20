// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	vzyaml "github.com/verrazzano/verrazzano/pkg/yaml"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"sigs.k8s.io/yaml"
)

const (
	//ExternalIPArg is used in a special case where Istio helm chart no longer supports ExternalIPs.
	// Put external IPs into the IstioOperator YAML, which does support it
	ExternalIPArg            = "gateways.istio-ingressgateway.externalIPs"
	specServiceJSONPath      = "spec.components.ingressGateways.0.k8s.service"
	externalIPJsonPathSuffix = "externalIPs.0"
	typeJSONPathSuffix       = "type"
	externalIPJsonPath       = specServiceJSONPath + "." + externalIPJsonPathSuffix
)

type ReplicaData struct {
	IngressReplicaCount uint32
	EgressReplicaCount  uint32
	IngressAffinity     string
	EgressAffinity      string
	IngressServiceType  string
	IngressServicePorts string
	ExternalIps         string
}

// BuildIstioOperatorYaml builds the IstioOperator CR YAML that will be passed as an override to istioctl
// Transform the Verrazzano CR istioComponent provided by the user onto an IstioOperator formatted YAML
func BuildIstioOperatorYaml(ctx spi.ComponentContext, comp *v1beta1.IstioComponent) (string, error) {
	// Build a list of YAML strings from the istioComponent initargs, one for each arg.
	expandedYamls := []string{}
	// get istio overrides for Jaeger tracing
	jaegerTracingYaml, err := configureJaegerTracing()
	if err != nil {
		return "", err
	}
	for _, arg := range comp.ValueOverrides {
		values := arg.Values
		overrideYaml, err := yaml.JSONToYAML(values.Raw)
		if err != nil {
			return "", err
		}
		expandedYamls = append([]string{string(overrideYaml)}, expandedYamls...)
		if err != nil {
			return "", err
		}
	}
	expandedYamls = append([]string{jaegerTracingYaml}, expandedYamls...)
	for _, yamlContent := range expandedYamls {
		ctx.Log().Infof("ISTIO: BuildOperator YAML contents %s", yamlContent)
	}
	// Merge all of the expanded YAMLs into a single YAML,
	// second has precedence over first, third over second, and so forth.
	merged, err := vzyaml.ReplacementMerge(expandedYamls...)
	ctx.Log().Infof("ISTIO: Merged YAML contents %s", merged)
	if err != nil {
		return "", err
	}
	return merged, nil
}
