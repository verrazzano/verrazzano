// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package oci

import (
	"context"
	capociv1beta2 "github.com/oracle/cluster-api-provider-oci/api/v1beta2"
	"github.com/stretchr/testify/assert"
	vmcv1alpha1 "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

var testRef = vmcv1alpha1.NamespacedRef{
	Name:      "test",
	Namespace: "test",
}

func TestLoadCredentials(t *testing.T) {
	emptyClient := fake.NewClientBuilder().WithScheme().Build()
	var tests = []struct {
		name             string
		cli              clipkg.Client
		hasError         bool
		clusterNamespace string
	}{
		{
			"errors when identity not found",
			fake.
		},
	}

	for _, tt := range tests {
		c, err := LoadCredentials(context.TODO(), tt.cli, testRef, tt.clusterNamespace)
		if tt.hasError {
			assert.Error(t, err)
			assert.Nil(t, c)
		} else {
			assert.NoError(t, err)
			assert.NotNil(t, c)
		}
	}
}

func testPrincipalSecret(creds *Credentials) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testRef.Name,
			Namespace: testRef.Namespace,
		},
		Data: map[string][]byte{
			ociUseInstancePrincipalField: []byte(creds.UseInstancePrincipal),
			ociPassphraseField:           []byte(creds.Passphrase),
			ociFingerprintField:          []byte(creds.Fingerprint),
			ociKeyField:                  []byte(creds.PrivateKey),
			ociUserField:                 []byte(creds.User),
			ociTenancyField:              []byte(creds.Tenancy),
			ociRegionField:               []byte(creds.Region),
		},
	}
}

func testIdentity(namespaces *capociv1beta2.AllowedNamespaces) *capociv1beta2.OCIClusterIdentity {
	return &capociv1beta2.OCIClusterIdentity{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testRef.Name,
			Namespace: testRef.Namespace,
		},
		Spec: capociv1beta2.OCIClusterIdentitySpec{
			AllowedNamespaces: namespaces,
			PrincipalSecret: corev1.SecretReference{
				Name:      testRef.Name,
				Namespace: testRef.Namespace,
			},
		},
	}
}
