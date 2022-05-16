package prometheus

import (
	"github.com/stretchr/testify/assert"
	"testing"

	"github.com/verrazzano/verrazzano/pkg/bom"
)

// TestAppendIstioCerts tests that the istio cert annotations get applied
func TestAppendIstioAnnotations(t *testing.T) {
	annotationKey := "annKey"
	tests := []struct {
		name              string
		expectAnnotations []bom.KeyValue
	}{
		{
			name: "test expect annotations",
			expectAnnotations: []bom.KeyValue{
				{
					Key: annotationKey,
					Value: `proxy.istio.io/config: {"proxyMetadata":{ "OUTPUT_CERTS": "/etc/istio-output-certs"}}
"sidecar.istio.io/userVolumeMount": [{"name": "istio-certs-dir", "mountPath": "/etc/istio-output-certs"}]
"traffic.sidecar.istio.io/includeOutboundIPRanges": ""`,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectAnnotations[0], AppendIstioAnnotations(annotationKey, []bom.KeyValue{})[0])
		})
	}
}
