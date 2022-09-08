// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	globalconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/mcconstants"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var deployment = &appsv1.Deployment{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: jaegerNamespace,
		Name:      jaegerOperatorName,
	},
}

func TestConfigureJaegerCR(t *testing.T) {
	type fields struct {
		mcRegSecret  *corev1.Secret
		joDeployment *appsv1.Deployment
		jaegerCreate bool
		mutualTLS    bool
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{"Default Jaeger instance", fields{mcRegSecret: createMCRegSecretWithoutMutualTLS(),
			joDeployment: deployment, jaegerCreate: true}},
		{"Jaeger instance with external OpenSearch mutual TLS", fields{mcRegSecret: createMCRegSecretWithMutualTLS(),
			joDeployment: deployment, jaegerCreate: true, mutualTLS: true}},
		{"OpenSearch URL not present", fields{mcRegSecret: &corev1.Secret{},
			joDeployment: deployment, jaegerCreate: false}},
		{"MC registration secret not exists", fields{mcRegSecret: &corev1.Secret{}, joDeployment: deployment,
			jaegerCreate: false}},
		{"Jaeger Operator not installed", fields{mcRegSecret: &corev1.Secret{},
			joDeployment: &appsv1.Deployment{}, jaegerCreate: false}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			_ = corev1.AddToScheme(scheme)
			_ = appsv1.AddToScheme(scheme)
			mgdClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				tt.fields.mcRegSecret, tt.fields.joDeployment).Build()
			adminClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()

			s := &Syncer{
				AdminClient: adminClient,
				LocalClient: mgdClient,
				Log:         zap.S().With(tt.name),
				Context:     context.TODO(),
			}
			s.configureJaegerCR(false)
			if tt.fields.jaegerCreate {
				assertJaegerSecret(t, mgdClient, tt.fields.mcRegSecret, tt.fields.mutualTLS)
				assertJaegerCR(t, mgdClient)
			} else {
				assertNoJaegerSecret(t, mgdClient)
				assertNoJaegerCR(t, mgdClient)
			}
		})
	}
}

func assertJaegerSecret(t *testing.T, client client.WithWatch, mcRegSec *corev1.Secret, mutualTLS bool) {
	secret := corev1.Secret{}
	err := client.Get(context.TODO(), types.NamespacedName{Namespace: jaegerNamespace,
		Name: mcconstants.JaegerManagedClusterSecretName}, &secret)
	assert.NoError(t, err)
	assert.Equal(t, mcRegSec.Data[mcconstants.JaegerOSUsernameKey], secret.Data[mcconstants.JaegerOSUsernameKey])
	assert.Equal(t, mcRegSec.Data[mcconstants.JaegerOSUsernameKey], secret.Data[mcconstants.JaegerOSUsernameKey])
	assert.Equal(t, mcRegSec.Data[mcconstants.JaegerOSTLSCAKey], secret.Data[jaegerOSTLSCAKey])
	if mutualTLS {
		assert.Equal(t, mcRegSec.Data[mcconstants.JaegerOSTLSKey], secret.Data[jaegerOSTLSKey])
		assert.Equal(t, mcRegSec.Data[mcconstants.JaegerOSTLSCertKey], secret.Data[jaegerOSTLSCertKey])
	}
}

func assertJaegerCR(t *testing.T, c client.WithWatch) {
	var uns unstructured.Unstructured
	uns.SetAPIVersion(jaegerAPIVersion)
	uns.SetKind(jaegerKind)
	uns.SetName(jaegerName)
	uns.SetNamespace(jaegerNamespace)
	key := client.ObjectKeyFromObject(&uns)
	err := c.Get(context.TODO(), key, &uns)
	assert.NoError(t, err)
	tags, _, err := unstructured.NestedString(uns.Object, "spec", "collector", "options", "collector", "tags")
	assert.NoError(t, err)
	assert.Equal(t, "verrazzano_cluster=managed1", tags)
	url, _, err := unstructured.NestedString(uns.Object, "spec", "storage", "options", "es", "server-urls")
	assert.NoError(t, err)
	assert.Equal(t, "https://opensearch-url", url)
	secName, _, err := unstructured.NestedString(uns.Object, "spec", "storage", "secretName")
	assert.NoError(t, err)
	assert.Equal(t, mcconstants.JaegerManagedClusterSecretName, secName)
}

func assertNoJaegerSecret(t *testing.T, client client.WithWatch) {
	secret := corev1.Secret{}
	err := client.Get(context.TODO(), types.NamespacedName{Namespace: jaegerNamespace,
		Name: globalconst.DefaultJaegerSecretName}, &secret)
	assert.Error(t, err)
}

func assertNoJaegerCR(t *testing.T, c client.WithWatch) {
	var uns unstructured.Unstructured
	uns.SetAPIVersion(jaegerAPIVersion)
	uns.SetKind(jaegerKind)
	uns.SetName(jaegerName)
	uns.SetNamespace(jaegerNamespace)
	key := client.ObjectKeyFromObject(&uns)
	err := c.Get(context.TODO(), key, &uns)
	assert.Error(t, err)
}

func createMCRegSecretWithoutMutualTLS() *corev1.Secret {
	return createMCRegSecret(false)
}

func createMCRegSecretWithMutualTLS() *corev1.Secret {
	return createMCRegSecret(true)
}

func createMCRegSecret(mutualTLS bool) *corev1.Secret {
	data := map[string][]byte{
		constants.ClusterNameData:       []byte("managed1"),
		mcconstants.JaegerOSUsernameKey: []byte("username"),
		mcconstants.JaegerOSPasswordKey: []byte("password"),
		mcconstants.JaegerOSURLKey:      []byte("https://opensearch-url"),
		mcconstants.JaegerOSTLSCAKey:    []byte("jaegeropensearchtlscakey"),
	}
	if mutualTLS {
		data[mcconstants.JaegerOSTLSKey] = []byte("jaegeropensearchtlskey")
		data[mcconstants.JaegerOSTLSCertKey] = []byte("jaegeropensearchtlscertkey")
	}
	return &corev1.Secret{
		ObjectMeta: v12.ObjectMeta{Name: constants.MCRegistrationSecret,
			Namespace: constants.VerrazzanoSystemNamespace},
		Data: data,
	}
}
