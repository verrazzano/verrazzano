// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certs

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

var testScheme = runtime.NewScheme()

func init() {
	_ = clientgoscheme.AddToScheme(testScheme)
}

func TestGetLocalClusterCABundleData(t *testing.T) {
	log := vzlog.DefaultLogger()
	privateCABundleData := []byte("verrazzano-tls-ca-bundle")
	tlsCAAdditionalBundleData := []byte("tls-ca-additional-bundle")
	vzTLSBundleData := []byte("verrazzano-tls-bundle")
	tests := []struct {
		name    string
		cli     client.Client
		want    []byte
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "verrazzano-tls-ca-only",
			cli: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      constants.PrivateCABundle,
						Namespace: constants.VerrazzanoSystemNamespace,
					},
					Data: map[string][]byte{
						constants.CABundleKey: privateCABundleData,
					},
				}).Build(),
			want:    privateCABundleData,
			wantErr: assert.NoError,
		},
		{
			name: "verrazzano-tls-only",
			cli: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      constants.VerrazzanoIngressTLSSecret,
						Namespace: constants.VerrazzanoSystemNamespace,
					},
					Data: map[string][]byte{
						constants.CACertKey: vzTLSBundleData,
					},
				}).Build(),
			want:    vzTLSBundleData,
			wantErr: assert.NoError,
		},
		{
			name: "verrazzano-tls-ca-and-verrazzano-tls",
			cli: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      constants.VerrazzanoIngressTLSSecret,
						Namespace: constants.VerrazzanoSystemNamespace,
					},
					Data: map[string][]byte{
						constants.CACertKey: vzTLSBundleData,
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      constants.PrivateCABundle,
						Namespace: constants.VerrazzanoSystemNamespace,
					},
					Data: map[string][]byte{
						constants.CABundleKey: privateCABundleData,
					},
				},
			).Build(),
			want:    privateCABundleData,
			wantErr: assert.NoError,
		},
		{
			name: "verrazzano-tls-ca-and-verrazzano-tls-and-tls-ca-additional",
			cli: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      constants.VerrazzanoIngressTLSSecret,
						Namespace: constants.VerrazzanoSystemNamespace,
					},
					Data: map[string][]byte{
						constants.CACertKey: vzTLSBundleData,
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      constants.AdditionalTLS,
						Namespace: constants.RancherSystemNamespace,
					},
					Data: map[string][]byte{
						constants.AdditionalTLSCAKey: tlsCAAdditionalBundleData,
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      constants.PrivateCABundle,
						Namespace: constants.VerrazzanoSystemNamespace,
					},
					Data: map[string][]byte{
						constants.CABundleKey: privateCABundleData,
					},
				},
			).Build(),
			want:    privateCABundleData,
			wantErr: assert.NoError,
		},
		{
			name: "verrazzano-tls-and-tls-ca-additional",
			cli: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      constants.AdditionalTLS,
						Namespace: constants.RancherSystemNamespace,
					},
					Data: map[string][]byte{
						constants.AdditionalTLSCAKey: tlsCAAdditionalBundleData,
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      constants.VerrazzanoIngressTLSSecret,
						Namespace: constants.VerrazzanoSystemNamespace,
					},
					Data: map[string][]byte{
						constants.CACertKey: vzTLSBundleData,
					},
				},
			).Build(),
			want:    tlsCAAdditionalBundleData,
			wantErr: assert.NoError,
		},
		{
			name: "tls-ca-additional-only",
			cli: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      constants.AdditionalTLS,
						Namespace: constants.RancherSystemNamespace,
					},
					Data: map[string][]byte{
						constants.AdditionalTLSCAKey: tlsCAAdditionalBundleData,
					},
				},
			).Build(),
			want:    tlsCAAdditionalBundleData,
			wantErr: assert.NoError,
		},
		{
			name: "verrazzano-tls-ca-and-tls-ca-additional",
			cli: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      constants.AdditionalTLS,
						Namespace: constants.RancherSystemNamespace,
					},
					Data: map[string][]byte{
						constants.AdditionalTLSCAKey: tlsCAAdditionalBundleData,
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      constants.PrivateCABundle,
						Namespace: constants.VerrazzanoSystemNamespace,
					},
					Data: map[string][]byte{
						constants.CABundleKey: privateCABundleData,
					},
				},
			).Build(),
			want:    privateCABundleData,
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.TODO()
			got, err := GetLocalClusterCABundleData(log.GetZapLogger(), tt.cli, ctx)
			if !tt.wantErr(t, err, fmt.Sprintf("GetLocalClusterCABundleData(%v, %v, %v)", log, tt.cli, ctx)) {
				return
			}
			assert.Equalf(t, tt.want, got, "GetLocalClusterCABundleData(%v, %v, %v)", log, tt.cli, ctx)
		})
	}
}
