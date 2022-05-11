// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package secrets

import (
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testNS         = "default"
	testSecretName = "po-val"
	testVZName     = "test-vz"
)

var compStatusMap = makeVerrazzanoComponentStatusMap()

var testSecret = corev1.Secret{
	TypeMeta: metav1.TypeMeta{
		Kind: "Secret",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      testSecretName,
		Namespace: testNS,
	},
	Immutable: nil,
	Data:      map[string][]byte{"override": []byte("true")},
}

var testVZ = vzapi.Verrazzano{
	TypeMeta: metav1.TypeMeta{
		APIVersion: "install.verrazzano.io/v1alpha1",
		Kind:       "Verrazzano",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      testVZName,
		Namespace: testNS,
	},
	Spec: vzapi.VerrazzanoSpec{
		Components: vzapi.ComponentSpec{PrometheusOperator: &vzapi.PrometheusOperatorComponent{
			Enabled: True(),
			HelmValueOverrides: vzapi.HelmValueOverrides{
				MonitorChanges: True(),
				ValueOverrides: []vzapi.Overrides{
					{
						SecretRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: testSecretName,
							},
							Key:      "",
							Optional: nil,
						},
					},
				},
			},
		}},
	},
	Status: vzapi.VerrazzanoStatus{
		State: vzapi.VzStateReady,
		Conditions: []vzapi.Condition{
			{
				Type: vzapi.CondInstallComplete,
			},
		},
		Components: compStatusMap,
	},
}

// create component status map required for testing
func makeVerrazzanoComponentStatusMap() vzapi.ComponentStatusMap {
	statusMap := make(vzapi.ComponentStatusMap)
	for _, comp := range registry.GetComponents() {
		if comp.IsOperatorInstallSupported() {
			statusMap[comp.Name()] = &vzapi.ComponentStatusDetails{
				Name: comp.Name(),
				Conditions: []vzapi.Condition{
					{
						Type:   vzapi.CondInstallComplete,
						Status: corev1.ConditionTrue,
					},
				},
				State: vzapi.CompStateReady,
			}
		}
	}
	return statusMap
}

// return address of a bool var with true value
func True() *bool {
	x := true
	return &x
}
