// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package operator

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/mcconstants"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

const (
	testOSURL       = "http://opensearch:9200"
	testOSTLSKey    = "key"
	testClusterName = "cluster"
)

func createTestRegistrationSecret(params map[string]string) *corev1.Secret {
	s := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vzconst.MCRegistrationSecret,
			Namespace: vzconst.VerrazzanoSystemNamespace,
		},
		Data: map[string][]byte{},
	}

	for k, v := range params {
		s.Data[k] = []byte(v)
	}
	return s
}

func TestCreateOrUpdateMCJaeger(t *testing.T) {
	var tests = []struct {
		name    string
		client  clipkg.Client
		created bool
	}{
		{
			"Not created when MC secret is not present",
			fake.NewClientBuilder().Build(),
			false,
		},
		{
			"Created when MC secret is present",
			fake.NewClientBuilder().WithObjects(createTestRegistrationSecret(map[string]string{
				mcconstants.JaegerOSURLKey: testOSURL,
			})).Build(),
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			created, err := createOrUpdateMCJaeger(tt.client)
			assert.NoError(t, err)
			assert.Equal(t, tt.created, created)
		})
	}
}

func TestBuildJaegerMCData(t *testing.T) {
	var tests = []struct {
		name     string
		secret   *corev1.Secret
		hasError bool
		data     *jaegerMCData
	}{
		{
			"error when no OS URL",
			createTestRegistrationSecret(map[string]string{}),
			true,
			nil,
		},
		{
			"uses mutual TLS when TLS Key present",
			createTestRegistrationSecret(map[string]string{
				mcconstants.JaegerOSTLSKey: testOSTLSKey,
				mcconstants.JaegerOSURLKey: testOSURL,
			}),
			false,
			&jaegerMCData{
				MutualTLS:       true,
				IsForceRecreate: true,
				OpenSearchURL:   testOSURL,
				SecretName:      mcconstants.JaegerManagedClusterSecretName,
			},
		},
		{
			"does not use mutual TLS when TLS Key absent",
			createTestRegistrationSecret(map[string]string{
				vzconst.ClusterNameData:    testClusterName,
				mcconstants.JaegerOSURLKey: testOSURL,
			}),
			false,
			&jaegerMCData{
				ClusterName:     testClusterName,
				MutualTLS:       false,
				IsForceRecreate: true,
				OpenSearchURL:   testOSURL,
				SecretName:      mcconstants.JaegerManagedClusterSecretName,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := buildJaegerMCData(tt.secret)
			assert.Equal(t, tt.hasError, err != nil)
			if data != nil {
				assert.Equal(t, *tt.data, *data)
			}
		})
	}
}

func TestCreateOrUpdateMCSecret(t *testing.T) {
	s := createTestRegistrationSecret(map[string]string{
		mcconstants.JaegerOSUsernameKey: mcconstants.JaegerOSUsernameKey,
		mcconstants.JaegerOSPasswordKey: mcconstants.JaegerOSPasswordKey,
		mcconstants.JaegerOSTLSCAKey:    mcconstants.JaegerOSTLSCAKey,
		mcconstants.JaegerOSTLSKey:      mcconstants.JaegerOSTLSKey,
		mcconstants.JaegerOSTLSCertKey:  mcconstants.JaegerOSTLSCertKey,
	})
	client := fake.NewClientBuilder().Build()
	err := createOrUpdateMCSecret(client, s)
	assert.NoError(t, err)

	jaegerSecret := &corev1.Secret{}
	err = client.Get(context.TODO(), types.NamespacedName{
		Name:      mcconstants.JaegerManagedClusterSecretName,
		Namespace: ComponentNamespace,
	}, jaegerSecret)
	assert.NoError(t, err)
	assertJaegerSecretHasKey := func(key string) {
		_, ok := jaegerSecret.Data[key]
		assert.True(t, ok)
	}
	assertJaegerSecretHasKey(mcconstants.JaegerOSUsernameKey)
	assertJaegerSecretHasKey(mcconstants.JaegerOSPasswordKey)
	assertJaegerSecretHasKey(jaegerOSTLSCAKey)
	assertJaegerSecretHasKey(jaegerOSTLSKey)
	assertJaegerSecretHasKey(jaegerOSTLSCertKey)
}
