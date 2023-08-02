// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package secrets

import (
	"bytes"
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/healthcheck"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

type testUpdater struct{}

func (t *testUpdater) Update(event *healthcheck.UpdateEvent) {}

func TestVerrazzanoSecretsReconciler_reconcileVerrazzanoTLS(t *testing.T) {

	scheme := newScheme()
	log := vzlog.DefaultLogger()
	updater := &testUpdater{}

	updatedBundleData := []byte("bundleupdate")
	originalBundleData := []byte("original bundle data")

	vzTLSName := types.NamespacedName{Name: vzconst.VerrazzanoIngressTLSSecret, Namespace: vzconst.VerrazzanoSystemNamespace}
	privateCABundleName := types.NamespacedName{Name: vzconst.PrivateCABundle, Namespace: vzconst.VerrazzanoSystemNamespace}

	defaultReq := controllerruntime.Request{
		NamespacedName: vzTLSName,
	}

	defaultWantErr := assert.NoError
	defaultBundleWantErr := assert.NoError

	vzReady := vzapi.Verrazzano{
		Status: vzapi.VerrazzanoStatus{
			State: vzapi.VzStateReady,
		},
	}

	type args struct {
		req *controllerruntime.Request
		vz  *vzapi.Verrazzano
	}
	tests := []struct {
		name                string
		cli                 client.Client
		args                args
		requeueRequired     bool
		wantErr             assert.ErrorAssertionFunc
		expectedBundleData  []byte
		bundleSecretWantErr assert.ErrorAssertionFunc
	}{
		{
			name: "verrazzano-tls-update",
			cli: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				&corev1.Secret{
					ObjectMeta: v1.ObjectMeta{Name: vzconst.VerrazzanoIngressTLSSecret, Namespace: vzconst.VerrazzanoSystemNamespace},
					Data: map[string][]byte{
						vzconst.CACertKey: updatedBundleData,
					},
				},
				&corev1.Secret{
					ObjectMeta: v1.ObjectMeta{Name: vzconst.PrivateCABundle, Namespace: vzconst.VerrazzanoSystemNamespace},
					Data: map[string][]byte{
						vzconst.CABundleKey: originalBundleData,
					},
				},
			).Build(),
			expectedBundleData: updatedBundleData,
		},
		{
			name: "verrazzano-tls-does-not-exist",
			cli: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				&corev1.Secret{
					ObjectMeta: v1.ObjectMeta{Name: vzconst.PrivateCABundle, Namespace: vzconst.VerrazzanoSystemNamespace},
					Data: map[string][]byte{
						vzconst.CABundleKey: originalBundleData,
					},
				},
			).Build(),
			expectedBundleData: originalBundleData,
		},
		{
			name: "verrazzano-tls-ca-does-not-exist",
			cli: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				&corev1.Secret{
					ObjectMeta: v1.ObjectMeta{Name: vzconst.VerrazzanoIngressTLSSecret, Namespace: vzconst.VerrazzanoSystemNamespace},
					Data: map[string][]byte{
						vzconst.CACertKey: updatedBundleData,
					},
				},
			).Build(),
			bundleSecretWantErr: assert.Error,
			expectedBundleData:  []byte{},
		},
		{
			name: "verrazzano-tls-vz-not-ready",
			cli: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				&corev1.Secret{
					ObjectMeta: v1.ObjectMeta{Name: vzconst.VerrazzanoIngressTLSSecret, Namespace: vzconst.VerrazzanoSystemNamespace},
					Data: map[string][]byte{
						vzconst.CACertKey: updatedBundleData,
					},
				},
				&corev1.Secret{
					ObjectMeta: v1.ObjectMeta{Name: vzconst.PrivateCABundle, Namespace: vzconst.VerrazzanoSystemNamespace},
					Data: map[string][]byte{
						vzconst.CABundleKey: originalBundleData,
					},
				},
			).Build(),
			args: args{
				vz: &vzapi.Verrazzano{
					Status: vzapi.VerrazzanoStatus{
						State: vzapi.VzStateReconciling,
					},
				},
			},
			expectedBundleData: originalBundleData,
			requeueRequired:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &VerrazzanoSecretsReconciler{
				Client:        tt.cli,
				Scheme:        scheme,
				log:           log,
				StatusUpdater: updater,
			}
			wantErr := defaultWantErr
			if tt.wantErr != nil {
				wantErr = tt.wantErr
			}

			bundleSecretWantErr := defaultBundleWantErr
			if tt.wantErr != nil {
				bundleSecretWantErr = tt.bundleSecretWantErr
			}

			vz := &vzReady
			if tt.args.vz != nil {
				vz = tt.args.vz
			}

			req := defaultReq
			if tt.args.req != nil {
				req = *tt.args.req
			}

			ctx := context.TODO()
			got, err := r.reconcileVerrazzanoTLS(ctx, req, vz)
			if !wantErr(t, err, fmt.Sprintf("reconcileVerrazzanoTLS(%v, %v, %v)", ctx, req, vz)) {
				return
			}
			assert.Equal(t, got.Requeue, tt.requeueRequired, "Did not get expected result")

			// check that the VZ private CA bundle secret was updated if necessary
			bundleSecret := &corev1.Secret{}
			err = tt.cli.Get(ctx, privateCABundleName, bundleSecret)
			if err = client.IgnoreNotFound(err); err == nil {
				byteSlicesEqualTrimmedWhitespace(t, tt.expectedBundleData, bundleSecret.Data[vzconst.CABundleKey], "CA bundle copies did not match")
			}
			bundleSecretWantErr(t, err, fmt.Sprintf("Bundle secret get err %v", err))
		})
	}
}

func TestVerrazzanoSecretsReconciler_reconcileVerrazzanoCABundleCopies(t *testing.T) {
	scheme := newScheme()
	log := vzlog.DefaultLogger()
	updater := &testUpdater{}

	updatedBundleData := []byte("bundleupdate")
	originalBundleData := []byte("original bundle data")

	privateCABundleName := types.NamespacedName{Name: vzconst.PrivateCABundle, Namespace: vzconst.VerrazzanoSystemNamespace}

	rancherTLSCASecret := types.NamespacedName{Namespace: vzconst.RancherSystemNamespace, Name: vzconst.RancherTLSCA}
	multiclusterCASecret := types.NamespacedName{Namespace: constants.VerrazzanoMultiClusterNamespace, Name: constants.VerrazzanoLocalCABundleSecret}

	defaultReq := controllerruntime.Request{
		NamespacedName: privateCABundleName,
	}

	defaultWantErr := assert.NoError
	defaultBundleWantErr := assert.NoError

	vzReady := vzapi.Verrazzano{
		Status: vzapi.VerrazzanoStatus{
			State: vzapi.VzStateReady,
		},
	}

	defaultObjsList := []runtime.Object{
		&corev1.Secret{
			ObjectMeta: v1.ObjectMeta{Namespace: rancherTLSCASecret.Namespace, Name: rancherTLSCASecret.Name},
			Data: map[string][]byte{
				vzconst.RancherTLSCAKey: originalBundleData,
			},
		},
		&corev1.Secret{
			ObjectMeta: v1.ObjectMeta{Namespace: multiclusterCASecret.Namespace, Name: multiclusterCASecret.Name},
			Data: map[string][]byte{
				mcCABundleKey: originalBundleData,
			},
		},
		&corev1.Secret{
			ObjectMeta: v1.ObjectMeta{Name: privateCABundleName.Name, Namespace: privateCABundleName.Namespace},
			Data: map[string][]byte{
				vzconst.CABundleKey: updatedBundleData,
			},
		},
		&corev1.Namespace{
			ObjectMeta: v1.ObjectMeta{Name: constants.VerrazzanoMultiClusterNamespace},
		},
		&appsv1.Deployment{
			ObjectMeta: v1.ObjectMeta{Name: "rancher", Namespace: "cattle-system"},
		},
	}

	type args struct {
		req *controllerruntime.Request
		vz  *vzapi.Verrazzano
	}
	tests := []struct {
		name                       string
		description                string
		cli                        client.Client
		args                       args
		requeueRequired            bool
		wantErr                    assert.ErrorAssertionFunc
		mcExpectedBundleData       []byte
		mcBundleSecretWantErr      assert.ErrorAssertionFunc
		rancherExpectedBundleData  []byte
		rancherBundleSecretWantErr assert.ErrorAssertionFunc
	}{
		{
			name:                      "verrazzano-tls-ca-update",
			description:               `Basic case where the CA bundle has been updated and the MC and Rancher copies exist `,
			cli:                       fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(defaultObjsList...).Build(),
			mcExpectedBundleData:      updatedBundleData,
			rancherExpectedBundleData: updatedBundleData,
		},
		{
			name:        "verrazzano-tls-ca-does-not-exist",
			description: `TLS CA bundle does not exist, the target copies should not be updated; likely a case where the secret was deleted, but should not happen`,
			cli: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				&corev1.Secret{
					ObjectMeta: v1.ObjectMeta{Namespace: rancherTLSCASecret.Namespace, Name: rancherTLSCASecret.Name},
					Data: map[string][]byte{
						vzconst.RancherTLSCAKey: originalBundleData,
					},
				},
				&corev1.Secret{
					ObjectMeta: v1.ObjectMeta{Namespace: multiclusterCASecret.Namespace, Name: multiclusterCASecret.Name},
					Data: map[string][]byte{
						mcCABundleKey: originalBundleData,
					},
				},
				&corev1.Namespace{
					ObjectMeta: v1.ObjectMeta{Name: constants.VerrazzanoMultiClusterNamespace},
				},
			).Build(),
			mcExpectedBundleData:      originalBundleData,
			rancherExpectedBundleData: originalBundleData,
		},
		{
			name:        "verrazzano-tls-vz-not-ready",
			description: `TLS CA bundle updated, but VZ is reconciling; requeue until VZ is in Ready state`,
			cli:         fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(defaultObjsList...).Build(),
			args: args{
				vz: &vzapi.Verrazzano{
					Status: vzapi.VerrazzanoStatus{
						State: vzapi.VzStateReconciling,
					},
				},
			},
			mcExpectedBundleData:      originalBundleData,
			rancherExpectedBundleData: originalBundleData,
			requeueRequired:           true,
		},
		{
			name: "rancher-tls-ca-does-not-exist",
			description: `TLS CA bundle updated, Rancher TLS CA does not exist; the Rancher secret should not be 
							created and only the MC bundle should be updated`,
			cli: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				&corev1.Secret{
					ObjectMeta: v1.ObjectMeta{Namespace: multiclusterCASecret.Namespace, Name: multiclusterCASecret.Name},
					Data: map[string][]byte{
						mcCABundleKey: originalBundleData,
					},
				},
				&corev1.Namespace{
					ObjectMeta: v1.ObjectMeta{Name: constants.VerrazzanoMultiClusterNamespace},
				},
			).Build(),
			mcBundleSecretWantErr:      assert.NoError,
			rancherBundleSecretWantErr: assert.Error,
			mcExpectedBundleData:       originalBundleData,
		},
		{
			name:        "mc-namespace-does-not-exist",
			description: `TLS CA bundle updated, but MC namespace does not exist;  Rancher secret should be updated but we should requeue until the verrazzano-mc namespace exists`,
			cli: fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(
				&corev1.Secret{
					ObjectMeta: v1.ObjectMeta{Namespace: rancherTLSCASecret.Namespace, Name: rancherTLSCASecret.Name},
					Data: map[string][]byte{
						vzconst.RancherTLSCAKey: originalBundleData,
					},
				},
				&corev1.Secret{
					ObjectMeta: v1.ObjectMeta{Namespace: multiclusterCASecret.Namespace, Name: multiclusterCASecret.Name},
					Data: map[string][]byte{
						mcCABundleKey: originalBundleData,
					},
				},
				&corev1.Secret{
					ObjectMeta: v1.ObjectMeta{Name: privateCABundleName.Name, Namespace: privateCABundleName.Namespace},
					Data: map[string][]byte{
						vzconst.CABundleKey: updatedBundleData,
					},
				},
			).Build(),
			requeueRequired:           true,
			mcExpectedBundleData:      originalBundleData,
			rancherExpectedBundleData: updatedBundleData,
		},
		{
			name:        "mc-bundle-secret-does-not-exist",
			description: `TLS CA bundle updated, but MC bundle secret does not initially exist;  MC bundle secret should be created`,
			cli: fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(
				&corev1.Secret{
					ObjectMeta: v1.ObjectMeta{Namespace: rancherTLSCASecret.Namespace, Name: rancherTLSCASecret.Name},
					Data: map[string][]byte{
						vzconst.RancherTLSCAKey: originalBundleData,
					},
				},
				&corev1.Secret{
					ObjectMeta: v1.ObjectMeta{Name: privateCABundleName.Name, Namespace: privateCABundleName.Namespace},
					Data: map[string][]byte{
						vzconst.CABundleKey: updatedBundleData,
					},
				},
				&corev1.Namespace{
					ObjectMeta: v1.ObjectMeta{Name: constants.VerrazzanoMultiClusterNamespace},
				},
			).Build(),
			mcExpectedBundleData:      updatedBundleData,
			rancherExpectedBundleData: updatedBundleData,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &VerrazzanoSecretsReconciler{
				Client:        tt.cli,
				Scheme:        scheme,
				log:           log,
				StatusUpdater: updater,
			}
			wantErr := defaultWantErr
			if tt.wantErr != nil {
				wantErr = tt.wantErr
			}

			mcBundleSecretWantErr := defaultBundleWantErr
			if tt.mcBundleSecretWantErr != nil {
				mcBundleSecretWantErr = tt.mcBundleSecretWantErr
			}

			rancherBundleSecretWantErr := defaultBundleWantErr
			if tt.rancherBundleSecretWantErr != nil {
				rancherBundleSecretWantErr = tt.rancherBundleSecretWantErr
			}

			vz := &vzReady
			if tt.args.vz != nil {
				vz = tt.args.vz
			}

			req := defaultReq
			if tt.args.req != nil {
				req = *tt.args.req
			}

			ctx := context.TODO()

			got, err := r.reconcileVerrazzanoCABundleCopies(ctx, req, vz)
			if !wantErr(t, err, fmt.Sprintf("reconcileVerrazzanoCABundleCopies(%v, %v, %v)", ctx, req, vz)) {
				return
			}
			assert.Equal(t, tt.requeueRequired, got.Requeue, "Did not get expected result")

			// check that the VZ private CA bundle secret was updated if necessary
			assertTargetCopy(t, tt.cli, multiclusterCASecret, mcCABundleKey, tt.mcExpectedBundleData, mcBundleSecretWantErr)
			assertTargetCopy(t, tt.cli, rancherTLSCASecret, vzconst.RancherTLSCAKey, tt.rancherExpectedBundleData, rancherBundleSecretWantErr)
		})
	}
}

func assertTargetCopy(t *testing.T, cli client.Client, targetSecretName types.NamespacedName, key string, expectedBundleData []byte, bundleSecretWantErr assert.ErrorAssertionFunc) {
	bundleSecret := &corev1.Secret{}
	err := cli.Get(context.TODO(), targetSecretName, bundleSecret)
	if !bundleSecretWantErr(t, err, fmt.Sprintf("Bundle secret get err %v", err)) {
		return
	}
	byteSlicesEqualTrimmedWhitespace(t, expectedBundleData, bundleSecret.Data[key], fmt.Sprintf("CA bundle copy for %s did not match", targetSecretName))
}

func byteSlicesEqualTrimmedWhitespace(t *testing.T, byteSlice1, byteSlice2 []byte, msg string) bool {
	a := bytes.Trim(byteSlice1, " \t\n\r")
	b := bytes.Trim(byteSlice2, " \t\n\r")
	return assert.Equal(t, a, b, msg)
}
