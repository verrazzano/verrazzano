// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestCreateOperatorNamespace(t *testing.T) {
	log := getTestLogger(t)

	var tests = []struct {
		testName string
		c        client.Client
	}{
		{
			"should create the rancher operator namespace",
			fake.NewFakeClientWithScheme(getScheme()),
		},
		{
			"should not fail if the rancher operator namespace already exists",
			fake.NewFakeClientWithScheme(getScheme(), &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: OperatorNamespace,
				},
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			assert.Nil(t, createRancherOperatorNamespace(log, tt.c))
		})
	}
}

func TestCreateCattleNamespace(t *testing.T) {
	log := getTestLogger(t)

	var tests = []struct {
		testName string
		c        client.Client
	}{
		{
			"should create the cattle namespace",
			fake.NewFakeClientWithScheme(getScheme()),
		},
		{
			"should edit the cattle namespace if already exists",
			fake.NewFakeClientWithScheme(getScheme(), &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: CattleSystem,
				},
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			assert.Nil(t, createCattleSystemNamespace(log, tt.c))
		})
	}
}

func TestCopyDefaultCACertificate(t *testing.T) {
	log := getTestLogger(t)
	secret := createCASecret()
	var tests = []struct {
		testName string
		c        client.Client
		vz       *vzapi.Verrazzano
		isErr    bool
	}{
		{
			"should not copy CA secret when not using the CA secret",
			fake.NewFakeClientWithScheme(getScheme()),
			&vzAcmeDev,
			false,
		},
		{
			"should fail to copy the CA secret when it does not exist",
			fake.NewFakeClientWithScheme(getScheme()),
			&vzDefaultCA,
			true,
		},
		{
			"should copy the CA secret when using the CA secret",
			fake.NewFakeClientWithScheme(getScheme(), &secret),
			&vzDefaultCA,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			err := copyDefaultCACertificate(log, tt.c, tt.vz)
			if tt.isErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestIsUsingDefaultCACertificate(t *testing.T) {
	var tests = []struct {
		testName string
		*vzapi.CertManagerComponent
		out bool
	}{
		{
			"no CA",
			nil,
			false,
		},
		{
			"acme CA",
			vzAcmeDev.Spec.Components.CertManager,
			false,
		},
		{
			"private CA",
			vzDefaultCA.Spec.Components.CertManager,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			assert.Equal(t, tt.out, isUsingDefaultCACertificate(tt.CertManagerComponent))
		})
	}
}
