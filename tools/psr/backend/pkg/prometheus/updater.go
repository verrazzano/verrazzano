// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package prometheus

import (
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

const (
	AlertmanagerCMName = "alertmanager-config-override"
	AlertmanagerCMKey  = "config"
)

type AlertmanagerConfigModifier struct {
}

func (u AlertmanagerConfigModifier) ModifyCR(cr *vzapi.Verrazzano) {
	selector := &corev1.ConfigMapKeySelector{
		LocalObjectReference: corev1.LocalObjectReference{
			Name: AlertmanagerCMName,
		},
		Key: AlertmanagerCMKey,
	}
	if cr.Spec.Components.PrometheusOperator == nil {
		cr.Spec.Components.PrometheusOperator = &vzapi.PrometheusOperatorComponent{}
	}
	overrides := cr.Spec.Components.PrometheusOperator.ValueOverrides
	if overrides == nil {
		overrides = []vzapi.Overrides{}
	} else {
		for _, override := range overrides {
			if override.ConfigMapRef == selector {
				return
			}
		}
	}
	cr.Spec.Components.PrometheusOperator.ValueOverrides = append(overrides, vzapi.Overrides{ConfigMapRef: selector})
}
