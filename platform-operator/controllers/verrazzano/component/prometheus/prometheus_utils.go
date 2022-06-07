// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package prometheus

import (
	v8oconst "github.com/verrazzano/verrazzano/pkg/constants"
	vpoconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetVerrazzanoMonitoringNamespace provides the namespace for the Monitoring subcomponents in one location
func GetVerrazzanoMonitoringNamespace() *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: vpoconst.VerrazzanoMonitoringNamespace,
			Labels: map[string]string{
				v8oconst.LabelIstioInjection:      "enabled",
				v8oconst.LabelVerrazzanoNamespace: vpoconst.VerrazzanoMonitoringNamespace,
			},
		},
	}
}
