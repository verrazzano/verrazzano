// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	"bytes"
	"fmt"
	"sigs.k8s.io/yaml"
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
const istioGatewayTemplate = `
apiVersion: install.istio.io/v1alpha1
kind: IstioOperator
spec:
  components:
    egressGateways:
      - name: istio-egressgateway
        enabled: true
        k8s:
          replicaCount: {{.EgressReplicaCount}}
{{- if .Affinity }}
          affinity:
{{ format .Affinity }}
{{- end}}
    ingressGateways:
      - name: istio-ingressgateway
        enabled: true
        k8s:
          replicaCount: {{.IngressReplicaCount}}
          {{- if .ExternalIps }}
          service:
            externalIPs:
              {{.ExternalIps}}
          {{- end}}
{{- if .Affinity }}
          affinity:
{{ format .Affinity }}
{{- end}}
`

type ReplicaData struct {
	IngressReplicaCount uint32
	EgressReplicaCount  uint32
	ExternalIps         string
	Affinity            string
}

// BuildIstioOperatorYaml builds the IstioOperator CR YAML that will be passed as an override to istioctl
// Transform the Verrazzano CR istioComponent provided by the user onto an IstioOperator formatted YAML
func BuildIstioOperatorYaml(comp *vzapi.IstioComponent) (string, error) {
	// All generated YAML will be indented 6 spaces
	const leftMargin = 0
	const leftMarginExtIP = 12

	var externalIPYAMLTemplateValue string = ""
	fmt.Printf("CDD Istio Component = %+v\n", *comp)
	// Build a list of YAML strings from the istioComponent initargs, one for each arg.
	expandedYamls := []string{}
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
			externalIPYAMLTemplateValue = yaml
			continue
		} else {
			valueName := fmt.Sprintf("spec.values.%s", arg.Name)
			yaml, err := vzyaml.Expand(leftMargin, false, valueName, values...)
			if err != nil {
				return "", err
			}
			expandedYamls = append(expandedYamls, yaml)
		}
	}
	gatewayYaml, err := configureGateways(comp.Kubernetes, fixExternalIPYaml(externalIPYAMLTemplateValue))
	if err != nil {
		return "", err
	}
	//	expandedYamls = append(expandedYamls, gatewayYaml)
	expandedYamls = append(expandedYamls, gatewayYaml)
	// Merge all of the expanded YAMLs into a single YAML,
	// second has precedence over first, third over second, and so forth.
	//	merged, err := vzyaml.ReplacementMerge(expandedYamls...)
	merged, err := vzyaml.ReplacementMerge(expandedYamls...)
	if err != nil {
		return "", err
	}

	return merged, nil
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

// value replicas and create Istio gateway yaml
func configureGateways(k8sConfig *vzapi.IstioKubernetesSection, externalIP string) (string, error) {
	var data = ReplicaData{}
	fmt.Printf("CDD Istio Component Kubernetes = %+v\n", k8sConfig)
	fmt.Printf("CDD Istio Component Kubernetes Replicas = %d\n", k8sConfig.Replicas)

	data.IngressReplicaCount = k8sConfig.Replicas
	data.EgressReplicaCount = k8sConfig.Replicas

	if k8sConfig.Affinity != nil {
		//		yml, err := yaml.Marshal(k8sConfig.Affinity.PodAntiAffinity)
		yml, err := yaml.Marshal(k8sConfig.Affinity)
		if err != nil {
			return "", err
		}
		data.Affinity = string(yml)
		fmt.Printf("CDD Affinity Yaml = %s\n", yml)
	}
	data.ExternalIps = ""
	if externalIP != "" {
		data.ExternalIps = externalIP
	}

	// use template to get populate template with data
	var b bytes.Buffer
	t, err := template.New("istioGateways").Funcs(template.FuncMap{
		"format": func(aff string) string {
			const indent = "            " // 12 spaces
			lines := strings.SplitAfter(aff, "\n")
			for i, line := range lines {
				lines[i] = indent + line
			}
			return strings.Join(lines[:], "")
		},
	}).Parse(istioGatewayTemplate)
	if err != nil {
		return "", err
	}

	err = t.Execute(&b, &data)
	if err != nil {
		return "", err
	}

	return b.String(), nil
}
