// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	"bytes"
	"fmt"
	"text/template"

	globalconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
)

const collectorZipkinPort = 9411

const istioTracingTemplate = `
apiVersion: install.istio.io/v1alpha1
kind: IstioOperator
spec:
  values:
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

//configureJaeger configures Jaeger for Istio integration and returns install args for the Istio install.
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
