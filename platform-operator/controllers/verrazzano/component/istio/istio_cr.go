// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	"bytes"
	"strings"
	"text/template"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzyaml "github.com/verrazzano/verrazzano/platform-operator/internal/yaml"
)

// ExternalIPArg is used in a special case where Istio helm chart no longer supports ExternalIPs.
// Put external IPs into the IstioOperator YAML, which does support it
const ExternalIPArg = "gateways.istio-ingressgateway.externalIPs"

// Define the IstioOperator template which is used to insert the generated YAML values.
//
// NOTE: The go template rendering doesn't properly indent the multi-line YAML value
// For example, the template fragment only indents the fist line of values
//    global:
//      {{.Values}}
// so the result is
//    global:
//      line1:
// line2:
//   line3:
// etc...
//
// A solution is to pre-indent each line of the values then insert it at column 0 as follows:
//    global:
// {{.Values}}
// See the leftMargin usage in the code
//
const istioHelmValuesTempate = `
apiVersion: install.istio.io/v1alpha1
kind: IstioOperator
spec:
  components:
    egressGateways:
      - name: istio-egressgateway
        enabled: true

  # Global values passed through to helm global.yaml.
  # Please keep this in sync with manifests/charts/global.yaml
  values:
    global:
{{.Values}}
`

// Template for merging externalIp YAML
const externalIPTemplate = `
apiVersion: install.istio.io/v1alpha1
kind: IstioOperator
spec:
  components:
    ingressGateways:
      - name: istio-ingressgateway
        enabled: true
        k8s:
          service:
            type: ClusterIP
            externalIPs:
{{.ExternalIps}}
`

// templateValuesIstioHelm is needed for template rendering of helm values
type templateValuesIstioHelm struct {
	Values string
}

// templateValuesExternalIPs needed for template rendering of external IPs.
type templateValuesExternalIPs struct {
	ExternalIps string
}

// BuildIstioOperatorYaml builds the IstioOperator CR YAML that will be passed as an override to istioctl
// Transform the Verrazzano CR IstioComponent provided by the user onto an IstioOperator formatted YAML
func BuildIstioOperatorYaml(comp *vzapi.IstioComponent) (string, error) {
	// All generated YAML will be indented 6 spaces
	const leftMargin = 6
	const leftMarginExtIP = 12

	var externalIPYAML string
	var resultYaml string

	// Build a list of YAML strings from the IstioComponent initargs, one for each arg.
	var expandedYamls []string
	for _, arg := range comp.IstioInstallArgs {
		values := arg.ValueList
		if len(values) == 0 {
			values = []string{arg.Value}
		}

		if arg.Name == ExternalIPArg {
			// We want the YAML in the following format, so pass a short arg name
			// because it is going to be rendered in the go template externalIPTemplate
			//   externalIPs:
			//   - 1.2.3.4
			//
			const shortArgExternalIPs = "externalIPs"
			yaml, err := vzyaml.Expand(leftMarginExtIP, true, shortArgExternalIPs, values...)
			if err != nil {
				return "", err
			}
			// This is handled seperately
			externalIPYAML = yaml
			continue
		} else {
			yaml, err := vzyaml.Expand(leftMargin, false, arg.Name, values...)
			if err != nil {
				return "", err
			}
			expandedYamls = append(expandedYamls, yaml)
		}
	}

	// Merge all of the expanded YAMLs into a single YAML,
	// second has precedence over first, third over second, and so forth.
	merged, err := vzyaml.ReplacementMerge(expandedYamls...)
	if err != nil {
		return "", err
	}

	// Render the IstioOperator YAML with the values YAML
	resultYaml, err = renderHelmValues(merged)
	if err != nil {
		return "", err
	}

	// If the externalIPs exists, the render that YAML and merge it
	if len(externalIPYAML) > 0 {
		// Render the IstioOperator YAML with the external IPs
		extYaml, err := renderExternalIPYAML(externalIPYAML)
		if err != nil {
			return "", err
		}
		// Now merge the 2 IstioOperator YAMLs
		firstResultYaml := resultYaml
		resultYaml, err = vzyaml.ReplacementMerge(firstResultYaml, extYaml)
		if err != nil {
			return "", err
		}
	}
	return resultYaml, nil
}

// Render the helm values using the template, return the YAML
func renderHelmValues(yaml string) (string, error) {
	t, err := template.New("values").Parse(istioHelmValuesTempate)
	if err != nil {
		return "", err
	}
	var rendered bytes.Buffer
	tInput := templateValuesIstioHelm{Values: yaml}
	err = t.Execute(&rendered, tInput)
	if err != nil {
		return "", err
	}
	return rendered.String(), nil
}

// Render the externalIP values using the template, return the YAML
func renderExternalIPYAML(yaml string) (string, error) {
	t, err := template.New("externalIP").Parse(externalIPTemplate)
	if err != nil {
		return "", err
	}
	var rendered bytes.Buffer
	tInput := templateValuesExternalIPs{ExternalIps: fixExternalIPYaml(yaml)}
	err = t.Execute(&rendered, tInput)
	if err != nil {
		return "", err
	}
	return rendered.String(), nil
}

// Change the YAML from
//       externalIPs
//       - 1.2.3.4
//       - 1.3.4.6
//
//  to
//       - 1.2.3.4
//       - 1.3.4.6
//
func fixExternalIPYaml(yaml string) string {
	segs := strings.SplitN(yaml, "\n", 2)
	if len(segs) == 2 {
		return segs[1]
	}
	return ""
}
