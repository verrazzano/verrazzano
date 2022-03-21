// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	"bytes"
	"fmt"
	corev1 "k8s.io/api/core/v1"
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
    egressGateways:
      - name: istio-egressgateway
        enabled: true
        k8s:
          {{- if .EgressReplicaCount }}
          replicaCount: {{.EgressReplicaCount}}
          {{- end }}
          {{- if .EgressAffinity }}
          affinity:
{{ multiLineIndent 12 .EgressAffinity }}
          {{- end}}
          {{- if .EgressService }}
          service:
{{ multiLineIndent 12 .EgressService }}
          {{- end}}
          {{- if .EgressServiceAnnotations }}
          serviceAnnotations:
{{ multiLineIndent 12 .EgressServiceAnnotations }}
          {{- end}}
          {{- if .EgressResources }}
          resources:
{{ multiLineIndent 12 .EgressResources }}
          {{- end}}
    ingressGateways:
      - name: istio-ingressgateway
        enabled: true
        k8s:
        {{- if .IngressReplicaCount }}
          replicaCount: {{.IngressReplicaCount}}
        {{- end }}
        {{- if .IngressAffinity }}
          affinity:
{{ multiLineIndent 12 .IngressAffinity }}
        {{- end }}
        {{- if .IngressService }}
          service:
{{ multiLineIndent 12 .IngressService }}
        {{- end }}
        {{- if .IngressServiceAnnotations }}
          serviceAnnotations:
{{ multiLineIndent 12 .IngressServiceAnnotations }}
        {{- end }}
        {{- if .IngressResources }}
          resources:
{{ multiLineIndent 12 .IngressResources }}
        {{- end }}
{{- if or .PilotReplicaCount .PilotAffinity .PilotService .PilotServiceAnnotations }}
    pilot:
      k8s:
        {{- if .PilotReplicaCount }}
        replicaCount: {{.PilotReplicaCount}}
        {{- end }}
        {{- if .PilotAffinity }}
        affinity:
{{ multiLineIndent 10 .PilotAffinity }}
        {{- end }}
        {{- if .PilotService }}
        service:
{{ multiLineIndent 10 .PilotService }}
        {{- end }}
        {{- if .PilotServiceAnnotations }}
        serviceAnnotations:
{{ multiLineIndent 10 .PilotServiceAnnotations }}
        {{- end }}
        {{- if .PilotResources }}
        resources:
{{ multiLineIndent 10 .PilotResources }}
        {{- end }}
{{- end }}
`

// replicaData is a temp structure used to fill in the template
type replicaData struct {
	PilotReplicaCount         uint32
	PilotResources            string
	PilotService              string
	PilotServiceAnnotations   string
	PilotAffinity             string
	IngressReplicaCount       uint32
	IngressResources          string
	IngressService            string
	IngressServiceAnnotations string
	IngressAffinity           string
	EgressReplicaCount        uint32
	EgressResources           string
	EgressService             string
	EgressServiceAnnotations  string
	EgressAffinity            string
}

// buildIstioOperatorYaml builds the IstioOperator CR YAML that will be passed as an override to istioctl
// Transform the Verrazzano CR istioComponent provided by the user onto an IstioOperator formatted YAML
func buildIstioOperatorYaml(comp *vzapi.IstioComponent) (string, error) {
	// All generated YAML will be indented 6 spaces
	const leftMargin = 0

	//var externalIPYAMLTemplateValue = ""
	externalIPs := []string{}
	// Build a list of YAML strings from the istioComponent initargs, one for each arg.
	expandedInstallArgYamls := []string{}
	for _, arg := range comp.IstioInstallArgs {
		values := arg.ValueList
		if len(values) == 0 {
			values = []string{arg.Value}
		}
		if arg.Name == ExternalIPArg {
			externalIPs = values
			continue
		} else {
			valueName := fmt.Sprintf("spec.values.%s", arg.Name)
			yaml, err := vzyaml.Expand(leftMargin, false, valueName, values...)
			if err != nil {
				return "", err
			}
			expandedInstallArgYamls = append(expandedInstallArgYamls, yaml)
		}
	}
	overridesYaml, err := processIstioOverrides(comp, externalIPs)
	if err != nil {
		return "", err
	}
	expandedInstallArgYamls = append(expandedInstallArgYamls, overridesYaml)
	// Merge all of the expanded YAMLs into a single YAML,
	// second has precedence over first, third over second, and so forth.
	merged, err := vzyaml.ReplacementMerge(expandedInstallArgYamls...)
	if err != nil {
		return "", err
	}

	return merged, nil
}

// value replicas and create Istio gateway yaml
func processIstioOverrides(istioComponent *vzapi.IstioComponent, externalIPs []string) (string, error) {
	var data = replicaData{}

	if istioComponent.Pilot != nil && istioComponent.Pilot.Kubernetes != nil {
		kubernetes := istioComponent.Pilot.Kubernetes
		data.PilotReplicaCount = kubernetes.Replicas
		if kubernetes.Affinity != nil {
			yml, err := yaml.Marshal(kubernetes.Affinity)
			if err != nil {
				return "", err
			}
			data.PilotAffinity = string(yml)
		}

		if kubernetes.Service != nil {
			if len(kubernetes.Service.Ports) > 0 {
				svc := &corev1.ServiceSpec{
					Ports: kubernetes.Service.Ports,
				}
				yml, err := yaml.Marshal(svc)
				if err != nil {
					return "", err
				}
				data.PilotService = string(yml)
			}
			if len(kubernetes.Service.Annotations) > 0 {
				yml, err := yaml.Marshal(kubernetes.Service.Annotations)
				if err != nil {
					return "", err
				}
				data.PilotServiceAnnotations = string(yml)
			}
		}
	}

	// Process externalIPs from installArgs first; if the Kubernetes.Service field is set, it will overwrite this
	// - Webhook should prevent this
	if len(externalIPs) > 0 {
		// Only execute this if the Kubernetes.Service field is not set, and externalIPs was provided through installArgs
		yml, err := yaml.Marshal(&corev1.ServiceSpec{ExternalIPs: externalIPs})
		if err != nil {
			return "", err
		}
		data.IngressService = string(yml)
	}

	if istioComponent.Ingress != nil && istioComponent.Ingress.Kubernetes != nil {
		kubernetes := istioComponent.Ingress.Kubernetes
		data.IngressReplicaCount = kubernetes.Replicas
		if kubernetes.Service != nil {
			if len(kubernetes.Service.Ports) > 0 {
				svc := &corev1.ServiceSpec{
					Ports: kubernetes.Service.Ports,
				}
				yml, err := yaml.Marshal(svc)
				if err != nil {
					return "", err
				}
				data.IngressService = string(yml)
			}
			if len(kubernetes.Service.Annotations) > 0 {
				yml, err := yaml.Marshal(kubernetes.Service.Annotations)
				if err != nil {
					return "", err
				}
				data.IngressServiceAnnotations = string(yml)
			}
		}
		if kubernetes.Resources != nil {
			yml, err := yaml.Marshal(kubernetes.Resources)
			if err != nil {
				return "", err
			}
			data.IngressResources = string(yml)
		}

		if kubernetes.Affinity != nil {
			yml, err := yaml.Marshal(kubernetes.Affinity)
			if err != nil {
				return "", err
			}
			data.IngressAffinity = string(yml)
		}
	}

	if istioComponent.Egress != nil && istioComponent.Egress.Kubernetes != nil {
		egressSettings := istioComponent.Egress.Kubernetes
		data.EgressReplicaCount = egressSettings.Replicas
		if egressSettings.Service != nil {
			if len(egressSettings.Service.Ports) > 0 {
				svc := &corev1.ServiceSpec{
					Ports: egressSettings.Service.Ports,
				}
				yml, err := yaml.Marshal(svc)
				if err != nil {
					return "", err
				}
				data.EgressService = string(yml)
			}
			if len(egressSettings.Service.Annotations) > 0 {
				yml, err := yaml.Marshal(egressSettings.Service.Annotations)
				if err != nil {
					return "", err
				}
				data.EgressServiceAnnotations = string(yml)
			}
		}
		if egressSettings.Affinity != nil {
			yml, err := yaml.Marshal(egressSettings.Affinity)
			if err != nil {
				return "", err
			}
			data.EgressAffinity = string(yml)
		}
		if egressSettings.Resources != nil {
			yml, err := yaml.Marshal(egressSettings.Resources)
			if err != nil {
				return "", err
			}
			data.EgressResources = string(yml)
		}
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
