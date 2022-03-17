// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"sigs.k8s.io/yaml"

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
    pilot:
      k8s:
        replicaCount: {{.PilotReplicaCount}}
        affinity:
{{ multiLineIndent 10 .PilotAffinity }}
        service:
{{ multiLineIndent 10 .PilotService }}
        serviceAnnotations:
{{ multiLineIndent 10 .PilotServiceAnnotations }}
        podAnnotations:
{{ multiLineIndent 10 .PilotPodAnnotations }}
        resources:
{{ multiLineIndent 10 .PilotResources }}
    egressGateways:
      - name: istio-egressgateway
        enabled: true
        k8s:
          replicaCount: {{.EgressReplicaCount}}
          affinity:
{{ multiLineIndent 12 .EgressAffinity }}
          service:
{{ multiLineIndent 12 .EgressService }}
          serviceAnnotations:
{{ multiLineIndent 12 .EgressServiceAnnotations }}
          podAnnotations:
{{ multiLineIndent 12 .EgressPodAnnotations }}
          resources:
{{ multiLineIndent 12 .EgressResources }}
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
          affinity:
{{ multiLineIndent 12 .IngressAffinity }}
          service:
{{ multiLineIndent 12 .IngressService }}
          serviceAnnotations:
{{ multiLineIndent 12 .IngressServiceAnnotations }}
          podAnnotations:
{{ multiLineIndent 12 .IngressPodAnnotations }}
          resources:
{{ multiLineIndent 12 .IngressResources }}
`

type ReplicaData struct {
	PilotReplicaCount         uint32
	PilotResources            string
	PilotService              string
	PilotPodAnnotations       string
	PilotServiceAnnotations   string
	PilotAffinity             string
	IngressReplicaCount       uint32
	IngressResources          string
	IngressPodAnnotations     string
	IngressService            string
	IngressServiceAnnotations string
	IngressAffinity           string
	EgressReplicaCount        uint32
	EgressResources           string
	EgressPodAnnotations      string
	EgressService             string
	EgressServiceAnnotations  string
	EgressAffinity            string
	ExternalIps               string
}

// BuildIstioOperatorYaml builds the IstioOperator CR YAML that will be passed as an override to istioctl
// Transform the Verrazzano CR istioComponent provided by the user onto an IstioOperator formatted YAML
func BuildIstioOperatorYaml(comp *vzapi.IstioComponent) (string, error) {
	// All generated YAML will be indented 6 spaces
	const leftMargin = 0
	const leftMarginExtIP = 12

	var externalIPYAMLTemplateValue = ""
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
	gatewayYaml, err := configureGateways(comp, fixExternalIPYaml(externalIPYAMLTemplateValue))
	if err != nil {
		return "", err
	}
	expandedYamls = append(expandedYamls, gatewayYaml)
	// Merge all of the expanded YAMLs into a single YAML,
	// second has precedence over first, third over second, and so forth.
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
func configureGateways(istioComponent *vzapi.IstioComponent, externalIP string) (string, error) {
	var data = ReplicaData{}

	data.PilotReplicaCount = istioComponent.Pilot.Kubernetes.Replicas
	data.IngressReplicaCount = istioComponent.Ingress.Kubernetes.Replicas
	data.EgressReplicaCount = istioComponent.Egress.Kubernetes.Replicas

	if istioComponent.Pilot.Kubernetes.Affinity != nil {
		yml, err := yaml.Marshal(istioComponent.Pilot.Kubernetes.Affinity)
		if err != nil {
			return "", err
		}
		data.PilotAffinity = string(yml)
	}

	if istioComponent.Pilot.Kubernetes.Service != nil {
		if istioComponent.Pilot.Kubernetes.Service.Spec != nil {
			yml, err := yaml.Marshal(istioComponent.Pilot.Kubernetes.Service.Spec)
			if err != nil {
				return "", err
			}
			data.PilotService = string(yml)
		}
		if len(istioComponent.Pilot.Kubernetes.Service.Annotations) > 0 {
			yml, err := yaml.Marshal(istioComponent.Pilot.Kubernetes.Service.Annotations)
			if err != nil {
				return "", err
			}
			data.PilotServiceAnnotations = string(yml)
		}
	}

	if istioComponent.Pilot.Kubernetes.Service != nil {
		yml, err := yaml.Marshal(istioComponent.Pilot.Kubernetes.Service.Spec)
		if err != nil {
			return "", err
		}
		data.PilotService = string(yml)
	}

	if istioComponent.Pilot.Kubernetes.Resources != nil {
		yml, err := yaml.Marshal(istioComponent.Pilot.Kubernetes.Resources)
		if err != nil {
			return "", err
		}
		data.PilotResources = string(yml)
	}

	if istioComponent.Pilot.Kubernetes.Affinity != nil {
		yml, err := yaml.Marshal(istioComponent.Pilot.Kubernetes.Affinity)
		if err != nil {
			return "", err
		}
		data.PilotAffinity = string(yml)
	}

	if istioComponent.Ingress.Kubernetes.Service != nil {
		if istioComponent.Ingress.Kubernetes.Service.Spec != nil {
			yml, err := yaml.Marshal(istioComponent.Ingress.Kubernetes.Service.Spec)
			if err != nil {
				return "", err
			}
			data.IngressService = string(yml)
		}
		if len(istioComponent.Ingress.Kubernetes.Service.Annotations) > 0 {
			yml, err := yaml.Marshal(istioComponent.Ingress.Kubernetes.Service.Annotations)
			if err != nil {
				return "", err
			}
			data.IngressServiceAnnotations = string(yml)
		}
	}

	if istioComponent.Ingress.Kubernetes.Resources != nil {
		yml, err := yaml.Marshal(istioComponent.Ingress.Kubernetes.Resources)
		if err != nil {
			return "", err
		}
		data.IngressResources = string(yml)
	}

	if istioComponent.Egress.Kubernetes.Affinity != nil {
		yml, err := yaml.Marshal(istioComponent.Egress.Kubernetes.Affinity)
		if err != nil {
			return "", err
		}
		data.EgressAffinity = string(yml)
	}

	if istioComponent.Egress.Kubernetes.Service != nil {
		if istioComponent.Egress.Kubernetes.Service.Spec != nil {
			yml, err := yaml.Marshal(istioComponent.Egress.Kubernetes.Service.Spec)
			if err != nil {
				return "", err
			}
			data.EgressService = string(yml)
		}
		if len(istioComponent.Egress.Kubernetes.Service.Annotations) > 0 {
			yml, err := yaml.Marshal(istioComponent.Egress.Kubernetes.Service.Annotations)
			if err != nil {
				return "", err
			}
			data.EgressServiceAnnotations = string(yml)
		}
	}

	if istioComponent.Egress.Kubernetes.Resources != nil {
		yml, err := yaml.Marshal(istioComponent.Egress.Kubernetes.Resources)
		if err != nil {
			return "", err
		}
		data.EgressResources = string(yml)
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
