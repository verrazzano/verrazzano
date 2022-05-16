package prometheus

import (
	"github.com/verrazzano/verrazzano-monitoring-operator/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/bom"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

// appendIstioAnnotations appends Istio annotations necessary for Prometheus in Istio
func AppendIstioOverrides(annotationsKey, volumeMountKey, volumeKey string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	// Istio annotations for certs
	annotations, err := yaml.Marshal(map[string]string{
		`proxy.istio.io/config`:                            `{"proxyMetadata":{ "OUTPUT_CERTS": "/etc/istio-output-certs"}}`,
		"sidecar.istio.io/userVolumeMount":                 `[{"name": "istio-certs-dir", "mountPath": "/etc/istio-output-certs"}]`,
		"traffic.sidecar.istio.io/includeOutboundIPRanges": "",
	})
	if err != nil {
		return kvs, err
	}

	// Volume mount annotation for certs
	volumeMountData, err := yaml.Marshal([]corev1.VolumeMount{
		{
			Name:      "istio-certs-dir",
			MountPath: constants.IstioCertsMountPath,
		},
	})
	if err != nil {
		return kvs, err
	}

	// Volume annotation for certs
	volumeData, err := yaml.Marshal([]corev1.VolumeMount{
		{
			Name:      "istio-certs-dir",
			MountPath: constants.IstioCertsMountPath,
		},
	})
	if err != nil {
		return kvs, err
	}

	// Append the new Istio annotations
	kvs = append(kvs, []bom.KeyValue{
		{
			Key:   annotationsKey,
			Value: string(annotations),
		},
		{
			Key:   volumeMountKey,
			Value: string(volumeMountData),
		},
		{
			Key:   volumeKey,
			Value: string(volumeData),
		},
	}...)
	return kvs, nil
}
