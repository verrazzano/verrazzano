// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package secretgenerator

import (
	"context"
	promoperapi "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/verrazzano/verrazzano/pkg/constants"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"testing"

	asserts "github.com/stretchr/testify/assert"
)

// GIVEN a request to the controller
// WHEN the Prometheus resource is retrieved
// THEN if it is the Verrazzano Prometheus, create the Istio TLS secret
func TestReconcile(t *testing.T) {
	assert := asserts.New(t)

	testPromName := "test-prom-name"
	fakeDir := "/fake/dir"

	promNoLabel := &promoperapi.Prometheus{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testPromName,
			Namespace: constants.PrometheusOperatorNamespace,
		},
	}

	promLabel := promNoLabel.DeepCopy()
	promLabel.Labels = map[string]string{constants.VerrazzanoComponentLabelKey: constants.PromOperatorComponentName}

	tests := []struct {
		name        string
		req         reconcile.Request
		prometheus  *promoperapi.Prometheus
		expectError bool
		expectSkip  bool
		certDir     *string
	}{
		{
			name:       "test invalid namespace",
			req:        reconcile.Request{NamespacedName: types.NamespacedName{Namespace: "incorrect-ns", Name: testPromName}},
			expectSkip: true,
		},
		{
			name:        "test no prom",
			req:         reconcile.Request{NamespacedName: types.NamespacedName{Namespace: constants.PrometheusOperatorNamespace, Name: testPromName}},
			expectError: true,
		},
		{
			name:       "test no prom label",
			req:        reconcile.Request{NamespacedName: types.NamespacedName{Namespace: constants.PrometheusOperatorNamespace, Name: testPromName}},
			prometheus: promNoLabel,
			expectSkip: true,
		},
		{
			name:       "test prom label",
			req:        reconcile.Request{NamespacedName: types.NamespacedName{Namespace: constants.PrometheusOperatorNamespace, Name: testPromName}},
			prometheus: promLabel,
		},
		{
			name:        "test no files",
			req:         reconcile.Request{NamespacedName: types.NamespacedName{Namespace: constants.PrometheusOperatorNamespace, Name: testPromName}},
			prometheus:  promLabel,
			certDir:     &fakeDir,
			expectError: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			_ = k8scheme.AddToScheme(scheme)
			_ = promoperapi.AddToScheme(scheme)
			c := fake.NewClientBuilder().WithScheme(scheme)
			if tt.prometheus != nil {
				c.WithObjects(tt.prometheus)
			}

			cli := c.Build()

			if tt.certDir == nil {
				setCertDir("./testdata")
			} else {
				setCertDir(*tt.certDir)
			}

			r := newReconciler(cli)
			result, err := r.Reconcile(context.TODO(), tt.req)

			if tt.expectError {
				assert.Error(err)
				return
			}
			if tt.expectSkip {
				assert.NoError(err)
				assert.Equal(result, reconcile.Result{})
				return
			}

			assert.NoError(err)
			requeueTime := result.RequeueAfter.Seconds()
			assert.True(requeueTime <= 80)
			assert.True(requeueTime >= 40)

			secret := corev1.Secret{}
			err = r.Client.Get(context.TODO(), types.NamespacedName{Namespace: constants.PrometheusOperatorNamespace, Name: istioTLSSecret}, &secret)
			assert.NoError(err)

			// Verify the certificate files
			fileData, ok := secret.Data[certKeyFile]
			assert.True(ok)
			assert.Contains(string(fileData), "test-key")
			fileData, ok = secret.Data[rootCertFile]
			assert.True(ok)
			assert.Contains(string(fileData), "test-root-cert")
			fileData, ok = secret.Data[certChainFile]
			assert.True(ok)
			assert.Contains(string(fileData), "test-cert-chain")
		})
	}
}

func newReconciler(cli client.Client) Reconciler {
	scheme := runtime.NewScheme()
	_ = vzapi.AddToScheme(scheme)
	reconciler := Reconciler{
		Client:  cli,
		Log:     zap.S(),
		Scheme:  scheme,
		Scraper: "secretgenerator",
	}
	return reconciler
}
