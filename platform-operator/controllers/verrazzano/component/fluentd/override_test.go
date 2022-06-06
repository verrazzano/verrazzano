// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package fluentd

import (
	"context"
	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vpoconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

// Test_loggingPreInstall tests the Fluentd loggingPreInstall call
func Test_loggingPreInstall(t *testing.T) {
	// GIVEN a Fluentd component
	//  WHEN I call loggingPreInstall with fluentd overrides for ES and a custom ES secret
	//  THEN no error is returned and the secret has been copied
	trueValue := true
	secretName := "my-es-secret" //nolint:gosec //#gosec G101
	c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Namespace: vpoconst.VerrazzanoInstallNamespace, Name: secretName},
	}).Build()

	ctx := spi.NewFakeContext(c,
		&vzapi.Verrazzano{
			Spec: vzapi.VerrazzanoSpec{
				Components: vzapi.ComponentSpec{
					Fluentd: &vzapi.FluentdComponent{
						Enabled:             &trueValue,
						ElasticsearchURL:    "https://myes.mydomain.com:9200",
						ElasticsearchSecret: secretName,
					},
				},
			},
		},
		false)
	err := loggingPreInstall(ctx)
	assert.NoError(t, err)

	secret := &corev1.Secret{}
	err = c.Get(context.TODO(), types.NamespacedName{Name: secretName, Namespace: ComponentNamespace}, secret)
	assert.NoError(t, err)

	// GIVEN a Verrazzano component
	//  WHEN I call loggingPreInstall with fluentd overrides for OCI logging, including an OCI API secret name
	//  THEN no error is returned and the secret has been copied
	secretName = "my-oci-api-secret" //nolint:gosec //#gosec G101
	cs := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Namespace: vpoconst.VerrazzanoInstallNamespace, Name: secretName},
		},
	).Build()
	ctx = spi.NewFakeContext(cs,
		&vzapi.Verrazzano{
			Spec: vzapi.VerrazzanoSpec{
				Components: vzapi.ComponentSpec{
					Fluentd: &vzapi.FluentdComponent{
						Enabled: &trueValue,
						OCI: &vzapi.OciLoggingConfiguration{
							APISecret: secretName,
						},
					},
				},
			},
		},
		false)
	err = loggingPreInstall(ctx)
	assert.NoError(t, err)

	err = cs.Get(context.TODO(), types.NamespacedName{Name: secretName, Namespace: ComponentNamespace}, secret)
	assert.NoError(t, err)
}

// Test_loggingPreInstallSecretNotFound tests the Verrazzano loggingPreInstall call
// GIVEN a Verrazzano component
//  WHEN I call loggingPreInstall with fluentd overrides for ES and a custom ES secret and the secret does not exist
//  THEN an error is returned
func Test_loggingPreInstallSecretNotFound(t *testing.T) {
	trueValue := true
	c := fake.NewClientBuilder().WithScheme(testScheme).Build()
	ctx := spi.NewFakeContext(c,
		&vzapi.Verrazzano{
			Spec: vzapi.VerrazzanoSpec{
				Components: vzapi.ComponentSpec{
					Fluentd: &vzapi.FluentdComponent{
						Enabled:             &trueValue,
						ElasticsearchURL:    "https://myes.mydomain.com:9200",
						ElasticsearchSecret: "my-es-secret",
					},
				},
			},
		},
		false)
	err := loggingPreInstall(ctx)
	assert.Error(t, err)
}

// Test_loggingPreInstallFluentdNotEnabled tests the Verrazzano loggingPreInstall call
// GIVEN a Verrazzano component
//  WHEN I call loggingPreInstall and fluentd is disabled
//  THEN no error is returned
func Test_loggingPreInstallFluentdNotEnabled(t *testing.T) {
	falseValue := false
	c := fake.NewClientBuilder().WithScheme(testScheme).Build()
	ctx := spi.NewFakeContext(c,
		&vzapi.Verrazzano{
			Spec: vzapi.VerrazzanoSpec{
				Components: vzapi.ComponentSpec{
					Fluentd: &vzapi.FluentdComponent{
						Enabled: &falseValue,
					},
				},
			},
		},
		false)
	err := loggingPreInstall(ctx)
	assert.NoError(t, err)
}
