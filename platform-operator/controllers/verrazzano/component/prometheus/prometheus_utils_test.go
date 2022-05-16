// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package prometheus

import (
	"github.com/stretchr/testify/assert"
	"testing"

	"github.com/verrazzano/verrazzano/pkg/bom"
)

const annotation = `proxy.istio.io/config: '{"proxyMetadata":{ "OUTPUT_CERTS": "/etc/istio-output-certs"}}'
sidecar.istio.io/userVolumeMount: '[{"name": "istio-certs-dir", "mountPath": "/etc/istio-output-certs"}]'
traffic.sidecar.istio.io/includeOutboundIPRanges: ""
`
const volumeMount = `- mountPath: /etc/istio-certs
  name: istio-certs-dir
`

const volume = `- emptyDir:
    medium: Memory
  name: istio-certs-dir
`

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
					Key:   annotationKey,
					Value: annotation,
				},
				{
					Key:   volumeMountKey,
					Value: volumeMount,
				},
				{
					Key:   volumeKey,
					Value: volume,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kvs, err := AppendIstioOverrides(annotationKey, volumeMountKey, volumeKey, []bom.KeyValue{})

			assert.Equal(t, len(tt.expectAnnotations), len(kvs))

			for i := range tt.expectAnnotations {
				assert.Equal(t, tt.expectAnnotations[i], kvs[i])
			}
			assert.NoError(t, err)
		})
	}
}
