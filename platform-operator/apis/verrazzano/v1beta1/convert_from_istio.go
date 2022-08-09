// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Adapts code from istio_cr.go to convert the Istio component into a YAML document.

package v1beta1

import (
	"bytes"
	"fmt"
	vzyaml "github.com/verrazzano/verrazzano/pkg/yaml"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"sigs.k8s.io/yaml"
	"strings"
	"text/template"
)

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
{{- if .EgressAffinity }}
          affinity:
{{ multiLineIndent 12 .EgressAffinity }}
{{- end }}
    ingressGateways:
      - name: istio-ingressgateway
        enabled: true
        k8s:
          replicaCount: {{.IngressReplicaCount}}
          service:
            type: {{.IngressServiceType}}
            {{- if .IngressServicePorts }}
            ports:
{{ multiLineIndent 12 .IngressServicePorts }}
            {{- end}}
          {{- if .ExternalIps }}
            externalIPs:
{{.ExternalIps}}
          {{- end}}
{{- if .IngressAffinity }}
          affinity:
{{ multiLineIndent 12 .IngressAffinity }}
{{- end }}
`

const (
	externalIPArg       = "gateways.istio-ingressgateway.externalIPs"
	shortArgExternalIPs = "externalIPs"
	leftMargin          = 0
	leftMarginExtIP     = 12
)

type istioTemplateData struct {
	IngressReplicaCount uint32
	EgressReplicaCount  uint32
	IngressAffinity     string
	EgressAffinity      string
	IngressServiceType  string
	IngressServicePorts string
	ExternalIps         string
}

func convertIstioComponentToYaml(comp *v1alpha1.IstioComponent) (string, error) {
	var externalIPYAMLTemplateValue = ""
	// Build a list of YAML strings from the istioComponent initargs, one for each arg.
	expandedYamls := []string{}
	for _, arg := range comp.IstioInstallArgs {
		values := arg.ValueList
		if len(values) == 0 {
			values = []string{arg.Value}
		}

		if arg.Name == externalIPArg {
			// We want the YAML in the following format, so pass a short arg name
			// because it is going to be rendered in the go template externalIPTemplate
			//   externalIPs:
			//   - 1.2.3.4
			yamlString, err := vzyaml.Expand(leftMarginExtIP, true, shortArgExternalIPs, values...)
			if err != nil {
				return "", err
			}
			// This is handled separately
			externalIPYAMLTemplateValue = yamlString
			continue
		} else {
			var err error
			expandedYamls, err = addYAML(arg.Name, values, expandedYamls)
			if err != nil {
				return "", err
			}
		}
	}
	gatewayYaml, err := configureGateways(comp, fixExternalIPYaml(externalIPYAMLTemplateValue))
	if err != nil {
		return "", err
	}
	expandedYamls = append(expandedYamls, gatewayYaml)
	// Merge all the expanded YAMLs into a single YAML,
	// second has precedence over first, third over second, and so forth.
	merged, err := vzyaml.ReplacementMerge(expandedYamls...)
	if err != nil {
		return "", err
	}

	return merged, nil
}

func addYAML(name string, values, expandedYamls []string) ([]string, error) {
	valueName := fmt.Sprintf("spec.values.%s", name)
	yamlString, err := vzyaml.Expand(leftMargin, false, valueName, values...)
	if err != nil {
		return expandedYamls, err
	}
	return append(expandedYamls, yamlString), nil
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
func configureGateways(istioComponent *v1alpha1.IstioComponent, externalIP string) (string, error) {
	var data = istioTemplateData{}

	data.IngressReplicaCount = istioComponent.Ingress.Kubernetes.Replicas
	data.EgressReplicaCount = istioComponent.Egress.Kubernetes.Replicas

	if istioComponent.Ingress.Kubernetes.Affinity != nil {
		yml, err := yaml.Marshal(istioComponent.Ingress.Kubernetes.Affinity)
		if err != nil {
			return "", err
		}
		data.IngressAffinity = string(yml)
	}

	if istioComponent.Egress.Kubernetes.Affinity != nil {
		yml, err := yaml.Marshal(istioComponent.Egress.Kubernetes.Affinity)
		if err != nil {
			return "", err
		}
		data.EgressAffinity = string(yml)
	}

	data.IngressServiceType = string(v1alpha1.LoadBalancer)
	if istioComponent.Ingress.Type == v1alpha1.NodePort {
		data.IngressServiceType = string(v1alpha1.NodePort)
	}

	data.IngressServicePorts = ""
	if len(istioComponent.Ingress.Ports) > 0 {
		y, err := yaml.Marshal(istioComponent.Ingress.Ports)
		if err != nil {
			if err != nil {
				return "", err
			}
		}
		data.IngressServicePorts = string(y)
	}

	data.ExternalIps = ""
	if externalIP != "" {
		data.ExternalIps = externalIP
	}

	// use template to get populate template with data
	var b bytes.Buffer
	t, err := template.New("istioGateways").Funcs(template.FuncMap{
		"multiLineIndent": func(indentNum int, aff string) string {
			var b = make([]byte, indentNum)
			for i := 0; i < indentNum; i++ {
				b[i] = 32
			}
			lines := strings.SplitAfter(aff, "\n")
			for i, line := range lines {
				lines[i] = string(b) + line
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
