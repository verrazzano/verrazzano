// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package adapter

import (
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/keycloak"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/weblogic"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	corev1 "k8s.io/api/core/v1"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

const valuesYaml = "values.yaml"

func ApplyComponentAsModule(client clipkg.Client, vz *vzapi.Verrazzano, componentName string) error {
	adapter := componentAdapters[componentName]
	if adapter != nil {
		return adapter(vz).createOrUpdate(client)
	}
	return nil
}

var componentAdapters = map[string]func(*vzapi.Verrazzano) *componentAdapter{
	// Keycloak Adapter
	keycloak.ComponentName: func(vz *vzapi.Verrazzano) *componentAdapter {
		adapter := NewAdapter(vzconfig.IsKeycloakEnabled(vz))
		if adapter.IsEnabled {
			adapter.Name = keycloak.ComponentName
			adapter.Namespace = vz.Namespace
			adapter.ChartNamespace = keycloak.ComponentNamespace
			adapter.ChartPath = keycloak.ComponentName
			kc := vz.Spec.Components.Keycloak
			if kc != nil {
				adapter.InstallOverrides = kc.InstallOverrides
				override := vzapi.Overrides{
					ConfigMapRef: &corev1.ConfigMapKeySelector{
						Key: valuesYaml,
						LocalObjectReference: corev1.LocalObjectReference{
							Name: keycloak.ConfigMapName,
						},
					},
				}
				adapter.InstallOverrides.ValueOverrides = append([]vzapi.Overrides{override}, kc.ValueOverrides...)
			}
		}
		return adapter
	},

	// Weblogic Operator Adapter
	weblogic.ComponentName: func(vz *vzapi.Verrazzano) *componentAdapter {
		adapter := NewAdapter(vzconfig.IsWeblogicOperatorEnabled(vz))
		if adapter.IsEnabled {
			wko := vz.Spec.Components.WebLogicOperator
			adapter.Name = weblogic.ComponentName
			adapter.Namespace = vz.Namespace
			adapter.ChartNamespace = weblogic.ComponentNamespace
			adapter.ChartPath = weblogic.ComponentName
			if wko != nil {
				adapter.InstallOverrides = wko.InstallOverrides
				override := vzapi.Overrides{
					ConfigMapRef: &corev1.ConfigMapKeySelector{
						Key: valuesYaml,
						LocalObjectReference: corev1.LocalObjectReference{
							Name: weblogic.ConfigMapName,
						},
					},
				}
				adapter.InstallOverrides.ValueOverrides = append([]vzapi.Overrides{override}, wko.InstallOverrides.ValueOverrides...)
			}
		}
		return adapter
	},
}
