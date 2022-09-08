// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package fluentd

import (
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
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
		vz      *v1beta1.Verrazzano
		wantErr bool
	}{{
		name:    "default",
		vz:      &v1beta1.Verrazzano{},
		wantErr: false,
	}, {
		name: varlog,
		vz: &v1beta1.Verrazzano{
			Spec: v1beta1.VerrazzanoSpec{
				Components: v1beta1.ComponentSpec{
					Fluentd: &v1beta1.FluentdComponent{
						ExtraVolumeMounts: []v1beta1.VolumeMount{{Source: varlog}},
					},
				},
			},
		},
		wantErr: true,
	}, {
		name: homevar,
		vz: &v1beta1.Verrazzano{
			Spec: v1beta1.VerrazzanoSpec{
				Components: v1beta1.ComponentSpec{
					Fluentd: &v1beta1.FluentdComponent{
						ExtraVolumeMounts: []v1beta1.VolumeMount{{Source: varlog, Destination: homevar}},
					},
				},
			},
		},
		wantErr: false,
	}, {
		name: "oci and ext-es",
		vz: &v1beta1.Verrazzano{
			Spec: v1beta1.VerrazzanoSpec{
				Components: v1beta1.ComponentSpec{
					Fluentd: &v1beta1.FluentdComponent{
						OCI:           &v1beta1.OciLoggingConfiguration{},
						OpenSearchURL: "https://url",
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
		vz      *v1beta1.Verrazzano
		wantErr bool
	}{{
		name:    "default",
		vz:      &v1beta1.Verrazzano{},
		wantErr: false,
	}, {
		name: missing,
		vz: &v1beta1.Verrazzano{
			Spec: v1beta1.VerrazzanoSpec{
				Components: v1beta1.ComponentSpec{
					Fluentd: &v1beta1.FluentdComponent{
						OpenSearchSecret: missing,
					},
				},
			},
		},
		wantErr: true,
	}, {
		name: secName,
		vz: &v1beta1.Verrazzano{
			Spec: v1beta1.VerrazzanoSpec{
				Components: v1beta1.ComponentSpec{
					Fluentd: &v1beta1.FluentdComponent{
						OpenSearchSecret: secName,
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
