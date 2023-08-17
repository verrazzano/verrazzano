// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vmc

import (
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

func TestVerrazzanoManagedClusterReconcilerGetAdminCaBundle(t *testing.T) {
	testScheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(testScheme)
	log := vzlog.DefaultLogger()

	tlsCABundleData := []byte("tls-ca-bundle")
	vzTLSBundleData := []byte("verrazzano-tls-bundle")

	tests := []struct {
		name    string
		cli     client.Client
		want    []byte
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "tls-ca-and-verrazzano-tls-different-data",
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
						constants.CABundleKey: tlsCABundleData,
					},
				},
			).Build(),
			want:    append(append([]byte{}, tlsCABundleData...), vzTLSBundleData...),
			wantErr: assert.NoError,
		},
		{
			name: "tls-ca-and-verrazzano-tls-same-data",
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
						constants.CABundleKey: vzTLSBundleData,
					},
				},
			).Build(),
			want:    vzTLSBundleData,
			wantErr: assert.NoError,
		},
		{
			name: "tls-ca-only",
			cli: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      constants.RancherTLSCA,
						Namespace: constants.RancherSystemNamespace,
					},
					Data: map[string][]byte{
						constants.CABundleKey: tlsCABundleData,
					},
				},
			).Build(),
			want:    nil,
			wantErr: assert.Error,
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
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      constants.PrivateCABundle,
						Namespace: constants.VerrazzanoSystemNamespace,
					},
					Data: map[string][]byte{
						constants.CABundleKey: vzTLSBundleData,
					},
				},
			).Build(),
			want:    vzTLSBundleData,
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &VerrazzanoManagedClusterReconciler{
				Client: tt.cli,
				log:    log,
			}
			got, err := r.getAdminCaBundle()
			if !tt.wantErr(t, err, fmt.Sprintf("getAdminCaBundle(): %v", err)) {
				return
			}
			assert.Equalf(t, tt.want, got, "getAdminCaBundle()")
		})
	}
}
