package prometheus

import (
	"github.com/verrazzano/verrazzano/pkg/bom"
)

// appendIstioAnnotations appends Istio annotations necessary for Prometheus in Istio
func AppendIstioAnnotations(annotationsKey string, kvs []bom.KeyValue) []bom.KeyValue {
	annotations := `proxy.istio.io/config: {"proxyMetadata":{ "OUTPUT_CERTS": "/etc/istio-output-certs"}}
"sidecar.istio.io/userVolumeMount": [{"name": "istio-certs-dir", "mountPath": "/etc/istio-output-certs"}]
"traffic.sidecar.istio.io/includeOutboundIPRanges": ""`

	return append(kvs, bom.KeyValue{Key: annotationsKey, Value: annotations})
}
