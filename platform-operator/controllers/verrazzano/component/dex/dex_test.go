// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package dex

import (
	"context"
	"strings"
	"testing"

	"github.com/verrazzano/verrazzano/pkg/test/ip"

	networkv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/types"
	k8scheme "k8s.io/client-go/kubernetes/scheme"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/bom"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testBomFilePath      = "../../testdata/test_bom.json"
	profilesRelativePath = "../../../../manifests/profiles"
	ingressClass         = "verrazzano-nginx"
	testEnv              = "test-env"
)

var testVZ = &vzapi.Verrazzano{
	Spec: vzapi.VerrazzanoSpec{
		Profile: "dev",
		Components: vzapi.ComponentSpec{
			Dex: &vzapi.DexComponent{},
		},
	},
}

func createTestNginxService() *v1.Service {
	return &v1.Service{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ingress-controller-ingress-nginx-controller",
			Namespace: constants.IngressNginxNamespace,
		},
		Spec: v1.ServiceSpec{},
		Status: v1.ServiceStatus{
			LoadBalancer: v1.LoadBalancerStatus{
				Ingress: []v1.LoadBalancerIngress{
					{IP: ip.RandomIP(),
						Hostname: ""},
				},
			},
		},
	}
}

// TestAppendDexOverrides tests that the Dex overrides are generated correctly.
// GIVEN a Verrazzano BOM
// WHEN I call AppendDexOverrides
// THEN the Dex overrides Key:Value array has the expected content.
func TestAppendDexOverrides(t *testing.T) {
	a := assert.New(t)

	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			EnvironmentName: testEnv,
		},
	}

	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&v1.Secret{ObjectMeta: metav1.ObjectMeta{Name: constants.Verrazzano,
			Namespace: constants.VerrazzanoSystemNamespace}}, createTestNginxService(),
	).Build()
	config.SetDefaultBomFilePath(testBomFilePath)
	kvs, err := AppendDexOverrides(spi.NewFakeContext(c, vz, nil, false), "", "", "", nil)

	a.NoError(err, "AppendDexOverrides returned an error")
	a.Contains(kvs, bom.KeyValue{
		Key:   ingressClassKey,
		Value: ingressClass,
	})

	dnsDomain, err := getDNSDomain(c, vz)
	assert.NoError(t, err)

	testDexIngressHost := "auth." + dnsDomain
	a.Contains(kvs, bom.KeyValue{
		Key:   tlsHosts,
		Value: testDexIngressHost,
	})

	a.Contains(kvs, bom.KeyValue{
		Key:       configIssuer,
		Value:     httpsPrefix + testDexIngressHost,
		SetString: true,
	})
}

// TestPreInstallUpgrade tests the preInstallUpgrade function.
// GIVEN a Verrazzano CR
// WHEN I call preInstallUpgrade
// THEN the namespace of Dex component is created.
func TestPreInstallUpgrade(t *testing.T) {
	config.TestHelmConfigDir = "../../../../../thirdparty"
	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
	ctx := spi.NewFakeContext(client, &vzapi.Verrazzano{}, nil, false)

	err := preInstallUpgrade(ctx)
	assert.NoError(t, err)

	ns := v1.Namespace{}
	err = client.Get(context.TODO(), types.NamespacedName{Name: ComponentNamespace}, &ns)
	assert.NoError(t, err)
}

// TestUpdateDexIngress tests the updateDexIngress function.
// GIVEN a Verrazzano CR
// WHEN I call updateDexIngress
// THEN there is no error returned.
func TestUpdateDexIngress(t *testing.T) {
	ingress := &networkv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: constants.DexIngress, Namespace: constants.DexNamespace},
	}
	annotations := make(map[string]string)
	annotations["abc"] = "foo"
	annotations["def"] = "bar"
	ingress.SetAnnotations(annotations)
	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(ingress, createTestNginxService()).Build()
	ctx := spi.NewFakeContext(c, testVZ, nil, false)
	err := updateDexIngress(ctx)
	assert.NoError(t, err)
}

// TestPopulateAdminUserData tests the populateAdminUser function.
// GIVEN a Verrazzano CR
// WHEN I call populateAdminUser
// THEN it populates the data for the admin user, created as static password  in Dex
func TestPopulateAdminUserData(t *testing.T) {
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			EnvironmentName: testEnv,
		},
	}
	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&v1.Secret{ObjectMeta: metav1.ObjectMeta{Name: constants.Verrazzano,
			Namespace: constants.VerrazzanoSystemNamespace}}, createTestNginxService(),
	).Build()
	ctx := spi.NewFakeContext(c, vz, nil, false)

	staticUserData, err := populateStaticPasswordsTemplate()
	assert.NoError(t, err)
	err = populateAdminUser(ctx, &staticUserData)
	assert.NoError(t, err)
	assert.True(t, strings.Contains(staticUserData.String(), "username"))
}

// TestPopulateClients tests the functions to populate client data.
// GIVEN a Verrazzano CR
// WHEN I call populatePKCEClient and populatePGClient
// THEN it populates the respective client data
func TestPopulateClients(t *testing.T) {
	assert.True(t, true)
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			EnvironmentName: testEnv,
		},
	}
	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&v1.Secret{ObjectMeta: metav1.ObjectMeta{Name: constants.Verrazzano,
			Namespace: constants.VerrazzanoSystemNamespace}}, createTestNginxService(),
	).Build()
	ctx := spi.NewFakeContext(c, vz, nil, false)
	dnsHost := "test.dns.io"

	staticClientData, err := populateStaticClientsTemplate()
	assert.NoError(t, err)
	err = populatePKCEClient(ctx, dnsHost, &staticClientData)
	assert.NoError(t, err)
	err = populatePGClient(ctx, &staticClientData)
	assert.NoError(t, err)
	clientDataStr := staticClientData.String()
	assert.True(t, strings.Contains(clientDataStr, pkceClient))
	assert.True(t, strings.Contains(clientDataStr, pgClient))
	assert.True(t, strings.Contains(clientDataStr, "https://verrazzano."+dnsHost+"/*"))
}
