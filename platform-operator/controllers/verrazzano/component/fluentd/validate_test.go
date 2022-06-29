// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package fluentd

import (
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestValidateFluentd(t *testing.T) {
	varlog := "/var/log"
	homevar := "/home/var_log"
	tests := []struct {
		name    string
		vz      *vzapi.Verrazzano
		wantErr bool
	}{{
		name:    "default",
		vz:      &vzapi.Verrazzano{},
		wantErr: false,
	}, {
		name: varlog,
		vz: &vzapi.Verrazzano{
			Spec: vzapi.VerrazzanoSpec{
				Components: vzapi.ComponentSpec{
					Fluentd: &vzapi.FluentdComponent{
						ExtraVolumeMounts: []vzapi.VolumeMount{{Source: varlog}},
					},
				},
			},
		},
		wantErr: true,
	}, {
		name: homevar,
		vz: &vzapi.Verrazzano{
			Spec: vzapi.VerrazzanoSpec{
				Components: vzapi.ComponentSpec{
					Fluentd: &vzapi.FluentdComponent{
						ExtraVolumeMounts: []vzapi.VolumeMount{{Source: varlog, Destination: homevar}},
					},
				},
			},
		},
		wantErr: false,
	}, {
		name: "oci and ext-es",
		vz: &vzapi.Verrazzano{
			Spec: vzapi.VerrazzanoSpec{
				Components: vzapi.ComponentSpec{
					Fluentd: &vzapi.FluentdComponent{
						OCI:              &vzapi.OciLoggingConfiguration{},
						ElasticsearchURL: "https://url",
					},
				},
			},
		},
		wantErr: true,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateFluentd(tt.vz); (err != nil) != tt.wantErr {
				t.Errorf("validateFluentd() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateExternalES(t *testing.T) {
	secName := "TestValidateExternalES-sec"
	getFakeSecret(secName)
	missing := "missing"
	defer func() { getControllerRuntimeClient = getClient }()
	tests := []struct {
		name    string
		vz      *vzapi.Verrazzano
		wantErr bool
	}{{
		name:    "default",
		vz:      &vzapi.Verrazzano{},
		wantErr: false,
	}, {
		name: missing,
		vz: &vzapi.Verrazzano{
			Spec: vzapi.VerrazzanoSpec{
				Components: vzapi.ComponentSpec{
					Fluentd: &vzapi.FluentdComponent{
						ElasticsearchSecret: missing,
					},
				},
			},
		},
		wantErr: true,
	}, {
		name: secName,
		vz: &vzapi.Verrazzano{
			Spec: vzapi.VerrazzanoSpec{
				Components: vzapi.ComponentSpec{
					Fluentd: &vzapi.FluentdComponent{
						ElasticsearchSecret: secName,
					},
				},
			},
		},
		wantErr: false,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateFluentd(tt.vz); (err != nil) != tt.wantErr {
				t.Errorf("validateFluentd() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func getFakeSecret(secName string) corev1.Secret {
	sec := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secName,
			Namespace: constants.VerrazzanoInstallNamespace,
		},
		Data: map[string][]byte{
			esUsernameKey: []byte(secName),
			esPasswordKey: []byte(secName),
		},
	}
	getControllerRuntimeClient = func() (client.Client, error) {
		return fake.NewClientBuilder().WithScheme(newScheme()).WithRuntimeObjects(&sec).Build(), nil
	}
	return sec
}
