// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	globalconst "github.com/verrazzano/verrazzano/pkg/constants"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const testIngressName = "test-ingress"

// TestCreateOrUpdateSystemComponentIngress tests the CreateOrUpdateSystemComponentIngress function
func TestCreateOrUpdateSystemComponentIngress(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = vzapi.AddToScheme(scheme)

	// GIVEN a component context and a Verrazzano CR
	// WHEN  the CreateOrUpdateSystemComponentIngress function is called
	// THEN  the function call succeeds and the expected ingress has been created
	client := fake.NewClientBuilder().WithScheme(scheme).Build()

	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				DNS: &vzapi.DNSComponent{
					OCI: &vzapi.OCI{
						DNSZoneName: "mydomain.com",
					},
				},
			},
		},
	}

	ctx := spi.NewFakeContext(client, vz, nil, false)
	err := CreateOrUpdateSystemComponentIngress(ctx, IngressProperties{
		IngressName:   testIngressName,
		HostName:      "host",
		TLSSecretName: "tls-secret",
	})
	assert.NoError(t, err)

	ingress := &netv1.Ingress{}
	err = client.Get(context.TODO(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: testIngressName}, ingress)
	assert.NoError(t, err)
	assert.Equal(t, "host.default.mydomain.com", ingress.Spec.Rules[0].Host)
	assert.Equal(t, constants.VerrazzanoAuthProxyServiceName, ingress.Spec.Rules[0].IngressRuleValue.HTTP.Paths[0].Backend.Service.Name)
	assert.NotEmpty(t, ingress.Annotations["external-dns.alpha.kubernetes.io/target"])

	// GIVEN a component context and a Verrazzano CR configured with external DNS disabled
	// WHEN  the CreateOrUpdateSystemComponentIngress function is called
	// THEN  the function call succeeds and the expected ingress has been created
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      constants.NGINXControllerServiceName,
			Namespace: globalconst.IngressNamespace,
		},
		Spec: corev1.ServiceSpec{
			ExternalIPs: []string{"1.2.3.4"},
		},
	}
	client = fake.NewClientBuilder().WithScheme(scheme).WithObjects(service).Build()

	ctx = spi.NewFakeContext(client, &vzapi.Verrazzano{}, nil, false)
	err = CreateOrUpdateSystemComponentIngress(ctx, IngressProperties{
		IngressName:   testIngressName,
		HostName:      "host",
		TLSSecretName: "tls-secret",
	})
	assert.NoError(t, err)

	ingress = &netv1.Ingress{}
	err = client.Get(context.TODO(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: testIngressName}, ingress)
	assert.NoError(t, err)
	assert.Equal(t, "host.default.1.2.3.4.nip.io", ingress.Spec.Rules[0].Host)
	assert.Equal(t, constants.VerrazzanoAuthProxyServiceName, ingress.Spec.Rules[0].IngressRuleValue.HTTP.Paths[0].Backend.Service.Name)
	assert.Empty(t, ingress.Annotations["external-dns.alpha.kubernetes.io/target"])
}
