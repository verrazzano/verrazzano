// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/Jeffail/gabs/v2"
	globalconst "github.com/verrazzano/verrazzano/pkg/constants"
	vzyaml "github.com/verrazzano/verrazzano/pkg/yaml"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"sigs.k8s.io/yaml"
)

const collectorZipkinPort = 9411

const istioTracingTemplate = `
apiVersion: install.istio.io/v1alpha1
kind: IstioOperator
spec:
  meshConfig:
    defaultConfig:
      tracing:
        tlsSettings:
          mode: "{{.TracingTLSMode}}"
        zipkin:
          address: "{{.JaegerCollectorURL}}"
`

type istioTracingTemplateData struct {
	JaegerCollectorURL string
	TracingTLSMode     string
}

// buildJaegerTracingYaml builds the IstioOperator CR YAML that will contain the default system generated Jaeger
// configurations merged with any user provided overrides.
func buildJaegerTracingYaml(comp *v1beta1.IstioComponent) (string, error) {
	// Build a list of YAML strings from the istioComponent initargs, one for each arg.
	// get istio overrides for Jaeger tracing
	jaegerTracingYaml, err := configureJaegerTracing()
	if err != nil {
		return "", err
	}
	expandedYamls := []string{jaegerTracingYaml}
	for _, arg := range comp.ValueOverrides {
		values := arg.Values
		if values == nil {
			continue
		}
		// If user provided overrided already contains Jaeger tracing related settings, then merge it with
		// default values and use that else use the default values as a system added override.
		if containsJaegerTracingOverrides(values.Raw) {
			overrideYaml, err := yaml.JSONToYAML(values.Raw)
			if err != nil {
				return "", err
			}
			expandedYamls = append(expandedYamls, string(overrideYaml))
		}
	}
	// Merge all of the expanded YAMLs into a single YAML,
	// second has precedence over first, third over second, and so forth.
	merged, err := vzyaml.ReplacementMerge(expandedYamls...)
	if err != nil {
		return "", err
	}
	return merged, nil
}

func containsJaegerTracingOverrides(jsonOverride []byte) bool {
	jsonString, err := gabs.ParseJSON(jsonOverride)
	if err != nil {
		return false
	}
	return jsonString.ExistsP(meshConfigTracingPath)
}

// configureJaegerTracing configures Jaeger for Istio integration and returns install args for the Istio install.
// return Istio install args for the tracing endpoint and the Istio tracing TLS mode
func configureJaegerTracing() (string, error) {
	collectorURL := fmt.Sprintf("%s-%s.%s.svc.cluster.local.:%d",
		globalconst.JaegerInstanceName,
		globalconst.JaegerCollectorComponentName,
		constants.VerrazzanoMonitoringNamespace,
		collectorZipkinPort,
	)
	t, err := template.New("tracing_template").Parse(istioTracingTemplate)
	if err != nil {
		return "", err
	}
	var b bytes.Buffer
	var data = istioTracingTemplateData{}
	data.JaegerCollectorURL = collectorURL
	data.TracingTLSMode = "ISTIO_MUTUAL"

	err = t.Execute(&b, &data)
	if err != nil {
		return "", err
	}
	return b.String(), nil
}
