// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package keycloak

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/verrazzano/verrazzano/pkg/bom"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
)

const (
	testBomFilePath = "../../testdata/test_bom.json"

	testConsoleIngressHost  = "console-ingress-host"
	testKeycloakIngressHost = "keycloak-ingress-host"
)

// TestAppendKeycloakOverrides tests that the Keycloak overrides are generated correctly.
// GIVEN a Verrazzano BOM
// WHEN I call AppendKeycloakOverrides
// THEN the Keycloak overrides Key:Value array has the expected content.
func TestAppendKeycloakOverrides(t *testing.T) {
	assert := assert.New(t)

	const env = "test-env"
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			EnvironmentName: env,
		},
	}

	client := fake.NewFakeClientWithScheme(k8scheme.Scheme)
	err := createIngresses(client)
	assert.NoError(err, "Error creating test ingress resources")

	config.SetDefaultBomFilePath(testBomFilePath)
	kvs, err := AppendKeycloakOverrides(spi.NewContext(zap.S(), client, vz, false), "", "", "", nil)

	assert.NoError(err, "AppendKeycloakOverrides returned an error")
	assert.Len(kvs, 5, "AppendKeycloakOverrides returned wrong number of Key:Value pairs")

	assert.Contains(kvs, bom.KeyValue{
		Key:       dnsTarget,
		Value:     testConsoleIngressHost,
		SetString: true,
	})
	assert.Contains(kvs, bom.KeyValue{
		Key:   rulesHost,
		Value: testKeycloakIngressHost,
	})
	assert.Contains(kvs, bom.KeyValue{
		Key:   tlsHosts,
		Value: testKeycloakIngressHost,
	})
	assert.Contains(kvs, bom.KeyValue{
		Key:   tlsSecret,
		Value: env + "-secret",
	})
}

// createIngresses creates the k8s ingress resources that AppendKeycloakOverrides will fetch
func createIngresses(cli client.Client) error {
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: constants.VerrazzanoSystemNamespace,
			Name:      constants.VzConsoleIngress,
		},
		Spec: networkingv1.IngressSpec{
			TLS: []networkingv1.IngressTLS{
				{
					Hosts: []string{
						testConsoleIngressHost,
					},
				},
			},
		},
	}
	if err := cli.Create(context.TODO(), ingress); err != nil {
		return err
	}

	ingress = &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: constants.KeycloakNamespace,
			Name:      constants.KeycloakIngress,
		},
		Spec: networkingv1.IngressSpec{
			TLS: []networkingv1.IngressTLS{
				{
					Hosts: []string{
						testKeycloakIngressHost,
					},
				},
			},
		},
	}
	return cli.Create(context.TODO(), ingress)
}
