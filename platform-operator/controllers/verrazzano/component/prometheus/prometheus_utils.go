// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package prometheus

import (
	vmoconst "github.com/verrazzano/verrazzano-monitoring-operator/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/bom"
	v8oconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

const istioVolumeName = "istio-certs-dir"

var VerrazzanoMonitoringNamespace = corev1.Namespace{
	ObjectMeta: metav1.ObjectMeta{
		Name: constants.VerrazzanoMonitoringNamespace,
		Labels: map[string]string{
			v8oconst.LabelIstioInjection: "enabled",
		},
	},
}

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
			Name:      istioVolumeName,
			MountPath: vmoconst.IstioCertsMountPath,
		},
	})
	if err != nil {
		return kvs, err
	}

	// Volume annotation for certs
	volumeData, err := yaml.Marshal([]corev1.Volume{
		{
			Name: istioVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{
					Medium: corev1.StorageMediumMemory,
				},
			},
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
