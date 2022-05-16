package prometheus

import (
	"github.com/stretchr/testify/assert"
	"testing"

	"github.com/verrazzano/verrazzano/pkg/bom"
)

// TestAppendIstioCerts tests that the istio cert annotations get applied
func TestAppendIstioOverrides(t *testing.T) {
	annotationKey := "annKey"
	volumeMountKey := "vmKey"
	volumeKey := "volKey"
	tests := []struct {
		name              string
		expectAnnotations []bom.KeyValue
	}{
		{
			name: "test expect annotations",
			expectAnnotations: []bom.KeyValue{
				{
					Key: annotationKey,
					Value: `proxy.istio.io/config: '{"proxyMetadata":{ "OUTPUT_CERTS": "/etc/istio-output-certs"}}'
sidecar.istio.io/userVolumeMount: '[{"name": "istio-certs-dir", "mountPath": "/etc/istio-output-certs"}]'
traffic.sidecar.istio.io/includeOutboundIPRanges: ""
`,
				},
				{
					Key:   volumeMountKey,
					Value: "",
				},
				{
					Key:   volumeKey,
					Value: "",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kvs, err := AppendIstioOverrides(annotationKey, volumeMountKey, volumeKey, []bom.KeyValue{})

			assert.Equal(t, len(tt.expectAnnotations), len(kvs))

			for i, _ := range tt.expectAnnotations {
				assert.Equal(t, tt.expectAnnotations[i], kvs[i])
			}
			assert.NoError(t, err)
		})
	}
}
