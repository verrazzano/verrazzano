// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	"bytes"
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	istioclisec "istio.io/client-go/pkg/apis/security/v1beta1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"
)

var testZipkinNamespace = "foo"
var enabledFlag = true

var testScheme = runtime.NewScheme()

func init() {
	_ = istioclisec.AddToScheme(testScheme)
	_ = corev1.AddToScheme(testScheme)
}

var testZipkinService = corev1.Service{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: testZipkinNamespace,
		Name:      "jaeger-collector",
		Labels: map[string]string{
			constants.KubernetesAppLabel: constants.JaegerCollectorService,
		},
	},
	Spec: corev1.ServiceSpec{
		Ports: []corev1.ServicePort{
			{
				Name: "foo",
				Port: 1,
			},
			{
				Name: "http-zipkin",
				Port: 5555,
			},
		},
	},
}

var expectedYaml = `
apiVersion: install.istio.io/v1alpha1
kind: IstioOperator
spec:
  meshConfig:
    defaultConfig:
      tracing:
        tlsSettings:
          mode: "ISTIO_MUTUAL"
        zipkin:
          address: "jaeger-operator-jaeger-collector.verrazzano-monitoring.svc.cluster.local.:9411"
`

var jaegerTracingJSONTemplate = `{
    "apiVersion": "install.istio.io/v1alpha1",
    "kind": "IstioOperator",
    "spec": {
        "meshConfig": {
            "defaultConfig": {
                "tracing": {
{{- if .TracingEnabled }}
                    "enabled": {{.TracingEnabled}},
{{- end}}
{{- if (gt .Sampling 0) }}
                    "sampling": {{.Sampling}},
{{- end}}
{{- if (ne .TLSMode "") }}
                    "tlsSettings": {
                        "mode": "{{.TLSMode}}"
                    },
{{- end}}
{{- if (ne .CollectorAddress "") }}
                    "zipkin": {
                        "address": "{{.CollectorAddress}}"
                    }
{{- end}}
                }
            }
        }
    }
}`

type JaegerTracingConfig struct {
	TracingEnabled   bool
	Sampling         int
	CollectorAddress string
	TLSMode          string
}

var jaegerEnabledCR = &vzapi.Verrazzano{
	Spec: vzapi.VerrazzanoSpec{
		Components: vzapi.ComponentSpec{
			JaegerOperator: &vzapi.JaegerOperatorComponent{
				Enabled: &enabledFlag,
			},
		},
	},
}

func TestConfigureJaeger(t *testing.T) {
	ctxNoService := spi.NewFakeContext(fake.NewClientBuilder().WithScheme(testScheme).Build(), jaegerEnabledCR, nil, false)
	ctxWithServiceAndUnmanagedNamespace := spi.NewFakeContext(fake.NewClientBuilder().
		WithObjects(&testZipkinService).
		WithScheme(testScheme).
		Build(), jaegerEnabledCR, nil, false)

	var tests = []struct {
		name    string
		ctx     spi.ComponentContext
		numArgs int
	}{
		{
			"2 args (tls mode and zipkin address) returned when Jaeger operator is disabled",
			spi.NewFakeContext(fake.NewClientBuilder().Build(), &vzapi.Verrazzano{}, nil, false),
			2,
		},
		{
			"2 args (tls mode and zipkin address) returned when service is not present",
			ctxNoService,
			2,
		},
		{
			"2 args (tls mode and zipkin address) returned when service is present",
			ctxWithServiceAndUnmanagedNamespace,
			2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yamlString, err := configureJaegerTracing()
			assert.NoError(t, err)
			assert.YAMLEq(t, expectedYaml, yamlString)
		})
	}
}

func TestBuildJaegerTracingYaml(t *testing.T) {
	testV1Alpha1Istio := vzapi.IstioComponent{
		IstioInstallArgs: []vzapi.InstallArgs{
			{
				Name:  "meshConfig.defaultConfig.tracing.sampling",
				Value: "90",
			},
			{
				Name:  "meshConfig.defaultConfig.tracing.zipkin.address",
				Value: "abc.xyz:9411",
			},
		},
	}
	convertedIstioComponent, _ := vzapi.ConvertIstioToV1Beta1(&testV1Alpha1Istio)
	fakeContext := spi.NewFakeContext(fake.NewClientBuilder().WithScheme(runtime.NewScheme()).Build(), &vzapi.Verrazzano{}, nil, false)
	var tests = []struct {
		name         string
		istio        v1beta1.IstioComponent
		expectedYaml string
	}{
		{
			"when no overrides, then default yaml is returned",
			v1beta1.IstioComponent{},
			getJaegerTracingConfigAsYAML(JaegerTracingConfig{
				TLSMode:          "ISTIO_MUTUAL",
				CollectorAddress: "jaeger-operator-jaeger-collector.verrazzano-monitoring.svc.cluster.local.:9411",
			}),
		},
		{
			"user provided sampling rate and collector address",
			v1beta1.IstioComponent{
				InstallOverrides: v1beta1.InstallOverrides{
					ValueOverrides: []v1beta1.Overrides{
						{
							Values: &v1.JSON{
								Raw: getJaegerTracingConfigAsJSON(JaegerTracingConfig{
									Sampling:         90,
									CollectorAddress: "abc.xyz:9411",
								}),
							},
						},
					},
				},
			},
			getJaegerTracingConfigAsYAML(JaegerTracingConfig{
				Sampling:         90,
				TLSMode:          "ISTIO_MUTUAL",
				CollectorAddress: "abc.xyz:9411",
			}),
		},
		{
			"user provided sampling rate and collector address in v1alpha1 does not get merged as it uses spec.values.meshConfig",
			*convertedIstioComponent,
			getJaegerTracingConfigAsYAML(JaegerTracingConfig{
				TLSMode:          "ISTIO_MUTUAL",
				CollectorAddress: "jaeger-operator-jaeger-collector.verrazzano-monitoring.svc.cluster.local.:9411",
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNilf(t, tt.istio, "expected yaml cannot be nil")
			yamlString, err := buildJaegerTracingYaml(fakeContext, &tt.istio, "default")
			assert.NoError(t, err)
			assert.NotEmpty(t, tt.expectedYaml, "expected yaml cannot be empty yaml string")
			assert.YAMLEq(t, tt.expectedYaml, yamlString)
		})
	}
}

func getJaegerTracingConfigAsJSON(configData JaegerTracingConfig) []byte {
	var b bytes.Buffer
	t, err := template.New("jaeger-config-data-json").Parse(jaegerTracingJSONTemplate)
	if err != nil {
		return nil
	}

	err = t.Execute(&b, &configData)
	if err != nil {
		return nil
	}
	return b.Bytes()
}

func getJaegerTracingConfigAsYAML(configData JaegerTracingConfig) string {
	tracingDataYaml, err := yaml.JSONToYAML(getJaegerTracingConfigAsJSON(configData))
	if err != nil {
		return ""
	}
	return string(tracingDataYaml)
}
