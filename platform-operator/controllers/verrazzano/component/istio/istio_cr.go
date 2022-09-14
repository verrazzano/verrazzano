// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"

	"sigs.k8s.io/yaml"

	vzyaml "github.com/verrazzano/verrazzano/pkg/yaml"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
)

const (
	//ExternalIPArg is used in a special case where Istio helm chart no longer supports ExternalIPs.
	// Put external IPs into the IstioOperator YAML, which does support it
	ExternalIPArg            = "gateways.istio-ingressgateway.externalIPs"
	specServiceJSONPath      = "spec.components.ingressGateways.0.k8s.service"
	externalIPJsonPathSuffix = "externalIPs.0"
	typeJSONPathSuffix       = "type"
	externalIPJsonPath       = specServiceJSONPath + "." + externalIPJsonPathSuffix

	//meshConfigTracingAddress is the Jaeger collector address
	meshConfigTracingAddress = "meshConfig.defaultConfig.tracing.zipkin.address"

	//meshConfigTracingTLSMode is the TLS mode for Istio-Jaeger communication
	meshConfigTracingTLSMode = "meshConfig.defaultConfig.tracing.tlsSettings.mode"

	leftMargin      = 0
	leftMarginExtIP = 12
)

// Define the IstioOperator template which is used to insert the generated YAML values.
//
// NOTE: The go template rendering doesn't properly indent the multi-line YAML value
// For example, the template fragment only indents the fist line of values
//
//	global:
//	  {{.Values}}
//
// so the result is
//
//	global:
//	  line1:
//
// line2:
//
//	line3:
//
// etc...
//
// A solution is to pre-indent each line of the values then insert it at column 0 as follows:
//
//	global:
//
// {{.Values}}
// See the leftMargin usage in the code
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
          affinity:
{{ multiLineIndent 12 .EgressAffinity }}
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
          affinity:
{{ multiLineIndent 12 .IngressAffinity }}
`

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
func BuildIstioOperatorYaml(ctx spi.ComponentContext, comp *vzapi.IstioComponent) (string, error) {

	var externalIPYAMLTemplateValue = ""

	// Build a list of YAML strings from the istioComponent initargs, one for each arg.
	expandedYamls := []string{}

	// get any install args for Jaeger
	jaegerArgs, err := configureJaeger(ctx)
	if err != nil {
		return "", err
	}

	for _, arg := range append(jaegerArgs, comp.IstioInstallArgs...) {
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
			yamlString, err := vzyaml.Expand(leftMarginExtIP, true, shortArgExternalIPs, values...)
			if err != nil {
				return "", err
			}
			// This is handled seperately
			externalIPYAMLTemplateValue = yamlString
			continue
		} else {
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
	// Merge all of the expanded YAMLs into a single YAML,
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
//
//	     externalIPs
//	     - 1.2.3.4
//	     - 1.3.4.6
//
//	to
//	     - 1.2.3.4
//	     - 1.3.4.6
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

	data.IngressServiceType = string(vzapi.LoadBalancer)
	if istioComponent.Ingress.Type == vzapi.NodePort {
		data.IngressServiceType = string(vzapi.NodePort)
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
