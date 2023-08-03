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

// Update test impl for reconciler unit tests
func (t *testUpdater) Update(_ *healthcheck.UpdateEvent) {}

// TestReconcileVerrazzanoTLS tests the reconcileVerrazzanoTLS method keeping the VZ CA secret in sync when rotated
// GIVEN a call to reconcileVerrazzanoTLS to reconcile the verrazzano-tls secret update
// WHEN the Verrazzano ingress secret CA bundle exists and has been updated
// THEN the reconcileVerrazzanoTLS can extract the request and call the function to update the copies, unless a VZ reconcile is in progress
func TestReconcileVerrazzanoTLS(t *testing.T) {

	scheme := newScheme()
	log := vzlog.DefaultLogger()
	updater := &testUpdater{}

	vzTLSName := types.NamespacedName{Name: vzconst.VerrazzanoIngressTLSSecret, Namespace: vzconst.VerrazzanoSystemNamespace}
	privateCABundleName := types.NamespacedName{Name: vzconst.PrivateCABundle, Namespace: vzconst.VerrazzanoSystemNamespace}
	rancherTLSCATestSecret := types.NamespacedName{Namespace: vzconst.RancherSystemNamespace, Name: vzconst.RancherTLSCA}
	multiclusterCASecret := types.NamespacedName{Namespace: constants.VerrazzanoMultiClusterNamespace, Name: constants.VerrazzanoLocalCABundleSecret}

	defaultReq := controllerruntime.Request{
		NamespacedName: vzTLSName,
	}

	defaultWantErr := assert.NoError

	ingressTLSSecret := &corev1.Secret{
		ObjectMeta: v1.ObjectMeta{Name: vzTLSSecret.Name, Namespace: vzTLSSecret.Namespace},
	}
	privateCASecret := &corev1.Secret{
		ObjectMeta: v1.ObjectMeta{Name: privateCABundleName.Name, Namespace: privateCABundleName.Namespace},
	}
	defaultObjsList := []runtime.Object{
		&corev1.Secret{
			ObjectMeta: v1.ObjectMeta{Namespace: rancherTLSCATestSecret.Namespace, Name: rancherTLSCATestSecret.Name},
		},
		&corev1.Secret{
			ObjectMeta: v1.ObjectMeta{Namespace: multiclusterCASecret.Namespace, Name: multiclusterCASecret.Name},
		},
		privateCASecret,
		&corev1.Namespace{
			ObjectMeta: v1.ObjectMeta{Name: constants.VerrazzanoMultiClusterNamespace},
		},
		&appsv1.Deployment{
			ObjectMeta: v1.ObjectMeta{Name: rancherDeploymentName, Namespace: vzconst.RancherSystemNamespace},
		},
	}

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
		name            string
		cli             client.Client
		args            args
		requeueRequired bool
		wantErr         assert.ErrorAssertionFunc
	}{
		{
			name: "verrazzano-tls-update",
			cli: fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(
				append(defaultObjsList, ingressTLSSecret)...,
			).Build(),
		},
		{
			name: "verrazzano-tls-does-not-exist",
			cli: fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(
				defaultObjsList...,
			).Build(),
		},
		{
			name: "verrazzano-tls-vz-not-ready",
			cli: fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(
				append(defaultObjsList, ingressTLSSecret)...,
			).Build(),
			args: args{
				vz: &vzapi.Verrazzano{
					Status: vzapi.VerrazzanoStatus{
						State: vzapi.VzStateReconciling,
					},
				},
			},
			requeueRequired: true,
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
		})
	}
}

// TestReconcileVerrazzanoCABundleCopies tests the reconcileVerrazzanoCABundleCopies method keeping upstream copies in sync
// GIVEN a call to reconcileVerrazzanoCABundleCopies to reconcile the verrazzano-tls secret
// WHEN the verrazzano-tls secret CA bundle exists and has been updated
// THEN the upstream copies are updated and any required actions are taken
func TestReconcileVerrazzanoCABundleCopies(t *testing.T) {
	scheme := newScheme()
	log := vzlog.DefaultLogger()
	updater := &testUpdater{}

	updatedBundleData := []byte("bundleupdate")
	originalBundleData := []byte("original bundle data")
	letsEncryptStagingBundleData := []byte("letsencrypt staging bundle data")

	privateCABundleName := types.NamespacedName{Name: vzconst.PrivateCABundle, Namespace: vzconst.VerrazzanoSystemNamespace}
	rancherTLSCATestSecret := types.NamespacedName{Namespace: vzconst.RancherSystemNamespace, Name: vzconst.RancherTLSCA}
	multiclusterCASecret := types.NamespacedName{Namespace: constants.VerrazzanoMultiClusterNamespace, Name: constants.VerrazzanoLocalCABundleSecret}
	rancherDeploymentNSName := types.NamespacedName{Namespace: vzconst.RancherSystemNamespace, Name: rancherDeploymentName}

	defaultWantErr := assert.NoError
	defaultBundleWantErr := assert.NoError

	ingressLeafCertOnly := &corev1.Secret{
		ObjectMeta: v1.ObjectMeta{Name: vzTLSSecret.Name, Namespace: vzTLSSecret.Namespace},
		Data: map[string][]byte{
			"tls.crt": []byte("leaf-cert"),
			"tls.key": []byte("leaf-cert-key"),
		},
	}
	ingressTLSSecretPrivateCA := &corev1.Secret{
		ObjectMeta: v1.ObjectMeta{Name: vzTLSSecret.Name, Namespace: vzTLSSecret.Namespace},
		Data: map[string][]byte{
			vzconst.CACertKey: updatedBundleData,
			"tls.crt":         []byte("leaf-cert"),
			"tls.key":         []byte("leaf-cert-key"),
		},
	}
	ingressTLSSecretPrivateCANotUpdated := &corev1.Secret{
		ObjectMeta: v1.ObjectMeta{Name: vzTLSSecret.Name, Namespace: vzTLSSecret.Namespace},
		Data: map[string][]byte{
			vzconst.CACertKey: originalBundleData,
			"tls.crt":         []byte("leaf-cert"),
			"tls.key":         []byte("leaf-cert-key"),
		},
	}

	privateCASecret := &corev1.Secret{
		ObjectMeta: v1.ObjectMeta{Name: privateCABundleName.Name, Namespace: privateCABundleName.Namespace},
		Data: map[string][]byte{
			vzconst.CABundleKey: originalBundleData,
		},
	}
	defaultObjsList := []runtime.Object{
		&corev1.Secret{
			ObjectMeta: v1.ObjectMeta{Namespace: rancherTLSCATestSecret.Namespace, Name: rancherTLSCATestSecret.Name},
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
		privateCASecret,
		ingressTLSSecretPrivateCA,
		// verrazzano-mc namespace exists
		&corev1.Namespace{
			ObjectMeta: v1.ObjectMeta{Name: constants.VerrazzanoMultiClusterNamespace},
		},
		// Rancher deployment to detect when we have/haven't issued a restart
		&appsv1.Deployment{
			ObjectMeta: v1.ObjectMeta{Name: rancherDeploymentName, Namespace: vzconst.RancherSystemNamespace},
		},
	}

	tests := []struct {
		name                         string
		description                  string
		cli                          client.Client
		requeueRequired              bool
		wantErr                      assert.ErrorAssertionFunc
		sourceSecret                 *corev1.Secret
		privateCAExpectedBundleData  []byte
		privateCABundleSecretWantErr assert.ErrorAssertionFunc
		mcExpectedBundleData         []byte
		mcBundleSecretWantErr        assert.ErrorAssertionFunc
		rancherExpectedBundleData    []byte
		rancherBundleSecretWantErr   assert.ErrorAssertionFunc
		rancherRestartRequired       bool
	}{
		{
			name:                        "self-signed-ca",
			description:                 `Basic case where the verrazzano-tls CA bundle has been updated and the MC and Rancher copies exist `,
			cli:                         fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(defaultObjsList...).Build(),
			privateCAExpectedBundleData: updatedBundleData,
			mcExpectedBundleData:        updatedBundleData,
			rancherExpectedBundleData:   updatedBundleData,
			rancherRestartRequired:      true,
		},
		{
			name:        "verrazzano-tls-ca-does-not-exist",
			description: `TLS CA bundle does not exist, the target copies should not be updated; likely a case where the secret was deleted, but should not happen`,
			cli: fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				&corev1.Secret{
					ObjectMeta: v1.ObjectMeta{Namespace: multiclusterCASecret.Namespace, Name: multiclusterCASecret.Name},
					Data: map[string][]byte{
						mcCABundleKey: originalBundleData,
					},
				},
				ingressLeafCertOnly,
				&corev1.Namespace{
					ObjectMeta: v1.ObjectMeta{Name: constants.VerrazzanoMultiClusterNamespace},
				},
				// Rancher deployment to detect when we have/haven't issued a restart
				&appsv1.Deployment{
					ObjectMeta: v1.ObjectMeta{Name: rancherDeploymentName, Namespace: vzconst.RancherSystemNamespace},
				},
			).Build(),
			sourceSecret:                 ingressLeafCertOnly,
			privateCABundleSecretWantErr: assert.Error,
			rancherBundleSecretWantErr:   assert.Error,
			mcExpectedBundleData:         []byte(nil),
		},
		{
			name: "lets-encrypt-staging-update-scenario",
			description: "ACME/Let's encrypt staging case, TLS CA bundle-key does not exist in ingress secret but the leaf cert has been rotated.  " +
				"The target copies should not be updated, preserving the staging CA root bundle",
			sourceSecret: ingressLeafCertOnly,
			cli: fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(
				&corev1.Secret{
					ObjectMeta: v1.ObjectMeta{Namespace: rancherTLSCATestSecret.Namespace, Name: rancherTLSCATestSecret.Name},
					Data: map[string][]byte{
						vzconst.RancherTLSCAKey: letsEncryptStagingBundleData,
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
						vzconst.CABundleKey: letsEncryptStagingBundleData,
					},
				},
				ingressLeafCertOnly,
				&corev1.Namespace{
					ObjectMeta: v1.ObjectMeta{Name: constants.VerrazzanoMultiClusterNamespace},
				},
				&appsv1.Deployment{
					ObjectMeta: v1.ObjectMeta{Name: rancherDeploymentName, Namespace: vzconst.RancherSystemNamespace},
				},
			).Build(),
			privateCAExpectedBundleData: letsEncryptStagingBundleData,
			mcExpectedBundleData:        letsEncryptStagingBundleData,
			rancherExpectedBundleData:   letsEncryptStagingBundleData,
		},
		{
			name: "lets-encrypt-staging-to-production",
			description: "ACME/Let's encrypt staging-to-production case; VZ and Rancher TLS CA bundle-key do not exist," +
				"leaf cert has no CA bundle, but the ingress secret but the leaf cert has been rotated.  Only the" +
				"multi-cluster copy should be updated with an empty value, and the other copies should not exist",
			sourceSecret: ingressLeafCertOnly,
			cli: fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(
				&corev1.Secret{
					ObjectMeta: v1.ObjectMeta{Namespace: multiclusterCASecret.Namespace, Name: multiclusterCASecret.Name},
					Data: map[string][]byte{
						mcCABundleKey: letsEncryptStagingBundleData,
					},
				},
				ingressLeafCertOnly,
				&corev1.Namespace{
					ObjectMeta: v1.ObjectMeta{Name: constants.VerrazzanoMultiClusterNamespace},
				},
				&appsv1.Deployment{
					ObjectMeta: v1.ObjectMeta{Name: rancherDeploymentName, Namespace: vzconst.RancherSystemNamespace},
				},
			).Build(),
			privateCABundleSecretWantErr: assert.Error,
			rancherBundleSecretWantErr:   assert.Error,
			mcExpectedBundleData:         []byte(nil),
		},
		{
			name: "lets-encrypt-staging-to-self-signed-rotated",
			description: "System has been updated from LE staging to self-signed; verrazzano-tls-ca and tls-ca are " +
				"using private CA with old data, verrazzano-local-ca-bundle has LE staging data.  verrazzano-tls updated" +
				"with new bundle data.  All secrets should be updated with new bundle data.",
			sourceSecret: ingressTLSSecretPrivateCA,
			cli: fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(
				&corev1.Secret{
					ObjectMeta: v1.ObjectMeta{Namespace: multiclusterCASecret.Namespace, Name: multiclusterCASecret.Name},
					Data: map[string][]byte{
						mcCABundleKey: letsEncryptStagingBundleData,
					},
				},
				&corev1.Secret{
					ObjectMeta: v1.ObjectMeta{Namespace: rancherTLSCATestSecret.Namespace, Name: rancherTLSCATestSecret.Name},
					Data: map[string][]byte{
						vzconst.RancherTLSCAKey: originalBundleData,
					},
				},
				privateCASecret,
				ingressTLSSecretPrivateCA,
				&corev1.Namespace{
					ObjectMeta: v1.ObjectMeta{Name: constants.VerrazzanoMultiClusterNamespace},
				},
				&appsv1.Deployment{
					ObjectMeta: v1.ObjectMeta{Name: rancherDeploymentName, Namespace: vzconst.RancherSystemNamespace},
				},
			).Build(),
			rancherRestartRequired:      true,
			privateCAExpectedBundleData: updatedBundleData,
			rancherExpectedBundleData:   updatedBundleData,
			mcExpectedBundleData:        updatedBundleData,
		},
		{
			name: "lets-encrypt-staging-to-self-signed-not-rotated",
			description: "System has been updated from LE staging to self-signed; verrazzano-tls-ca and tls-ca are " +
				"using up-to-date private CA data, verrazzano-local-ca-bundle has stale LE staging data.  verrazzano-tls updated" +
				"with new bundle data.  All secrets should be updated with new bundle data.",
			sourceSecret: ingressTLSSecretPrivateCANotUpdated,
			cli: fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(
				&corev1.Secret{
					ObjectMeta: v1.ObjectMeta{Namespace: multiclusterCASecret.Namespace, Name: multiclusterCASecret.Name},
					Data: map[string][]byte{
						mcCABundleKey: letsEncryptStagingBundleData,
					},
				},
				&corev1.Secret{
					ObjectMeta: v1.ObjectMeta{Namespace: rancherTLSCATestSecret.Namespace, Name: rancherTLSCATestSecret.Name},
					Data: map[string][]byte{
						vzconst.RancherTLSCAKey: originalBundleData,
					},
				},
				privateCASecret,
				ingressTLSSecretPrivateCANotUpdated,
				&corev1.Namespace{
					ObjectMeta: v1.ObjectMeta{Name: constants.VerrazzanoMultiClusterNamespace},
				},
				&appsv1.Deployment{
					ObjectMeta: v1.ObjectMeta{Name: rancherDeploymentName, Namespace: vzconst.RancherSystemNamespace},
				},
			).Build(),
			privateCAExpectedBundleData: originalBundleData,
			rancherExpectedBundleData:   originalBundleData,
			mcExpectedBundleData:        originalBundleData,
		},
		{
			name:        "mc-namespace-does-not-exist",
			description: `TLS CA bundle updated, but MC namespace does not exist;  Rancher secret should be updated but we should requeue until the verrazzano-mc namespace exists`,
			cli: fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(
				&corev1.Secret{
					ObjectMeta: v1.ObjectMeta{Namespace: rancherTLSCATestSecret.Namespace, Name: rancherTLSCATestSecret.Name},
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
				ingressTLSSecretPrivateCA,
				&appsv1.Deployment{
					ObjectMeta: v1.ObjectMeta{Name: rancherDeploymentName, Namespace: vzconst.RancherSystemNamespace},
				},
			).Build(),
			requeueRequired:             true,
			privateCAExpectedBundleData: updatedBundleData,
			mcExpectedBundleData:        originalBundleData,
			rancherExpectedBundleData:   updatedBundleData,
			rancherRestartRequired:      true,
		},
		{
			name:        "mc-bundle-secret-does-not-exist",
			description: `TLS CA bundle updated, but MC bundle secret does not initially exist;  MC bundle secret should be created`,
			cli: fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(
				&corev1.Secret{
					ObjectMeta: v1.ObjectMeta{Namespace: rancherTLSCATestSecret.Namespace, Name: rancherTLSCATestSecret.Name},
					Data: map[string][]byte{
						vzconst.RancherTLSCAKey: originalBundleData,
					},
				},
				&corev1.Secret{
					ObjectMeta: v1.ObjectMeta{Name: privateCABundleName.Name, Namespace: privateCABundleName.Namespace},
					Data: map[string][]byte{
						vzconst.CABundleKey: originalBundleData,
					},
				},
				ingressTLSSecretPrivateCA,
				&corev1.Namespace{
					ObjectMeta: v1.ObjectMeta{Name: constants.VerrazzanoMultiClusterNamespace},
				},
				&appsv1.Deployment{
					ObjectMeta: v1.ObjectMeta{Name: rancherDeploymentName, Namespace: vzconst.RancherSystemNamespace},
				},
			).Build(),
			privateCAExpectedBundleData: updatedBundleData,
			mcExpectedBundleData:        updatedBundleData,
			rancherExpectedBundleData:   updatedBundleData,
			rancherRestartRequired:      true,
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

			privateCABundleSecretWantErr := defaultBundleWantErr
			if tt.privateCABundleSecretWantErr != nil {
				privateCABundleSecretWantErr = tt.privateCABundleSecretWantErr
			}

			mcBundleSecretWantErr := defaultBundleWantErr
			if tt.mcBundleSecretWantErr != nil {
				mcBundleSecretWantErr = tt.mcBundleSecretWantErr
			}

			rancherBundleSecretWantErr := defaultBundleWantErr
			if tt.rancherBundleSecretWantErr != nil {
				rancherBundleSecretWantErr = tt.rancherBundleSecretWantErr
			}

			sourceSecret := ingressTLSSecretPrivateCA
			if tt.sourceSecret != nil {
				sourceSecret = tt.sourceSecret
			}

			got, err := r.reconcileVerrazzanoCABundleCopies(sourceSecret)
			if !wantErr(t, err, "reconcileVerrazzanoCABundleCopies did not get expected error result") {
				return
			}
			assert.Equal(t, tt.requeueRequired, got.Requeue, "Did not get expected result")

			//if tt.rancherRestartRequired {
			deployment := &appsv1.Deployment{}
			assert.NoError(t, tt.cli.Get(context.TODO(), rancherDeploymentNSName, deployment))
			_, foundRestartAnnotation := deployment.Spec.Template.ObjectMeta.Annotations[vzconst.VerrazzanoRestartAnnotation]
			assert.Equal(t, tt.rancherRestartRequired, foundRestartAnnotation, "Rancher restart check failed, expected %v", tt.rancherRestartRequired)
			//}
			// check that the VZ private CA bundle secret was updated if necessary
			assertTargetCopy(t, tt.cli, privateCABundleName, vzconst.CABundleKey, tt.privateCAExpectedBundleData, privateCABundleSecretWantErr)
			assertTargetCopy(t, tt.cli, multiclusterCASecret, mcCABundleKey, tt.mcExpectedBundleData, mcBundleSecretWantErr)
			assertTargetCopy(t, tt.cli, rancherTLSCATestSecret, vzconst.RancherTLSCAKey, tt.rancherExpectedBundleData, rancherBundleSecretWantErr)
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
