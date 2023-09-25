// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package reconcile

import (
	"context"
	"testing"

	asserts "github.com/stretchr/testify/assert"
	constants2 "github.com/verrazzano/verrazzano/pkg/constants"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const host1 = "rancher1.somedomain.com"
const host2 = "rancher2.somedomain.com"
const host3 = "rancher3.otherdomain.com"
const secret1 = "secret1"
const secret2 = "secret2"

// GIVEN The Rancher ingress and cert copies do not already exist
// WHEN createRancherIngressAndCertCopies is called
// THEN the copies of the ingress and cert(s) are created and there is no error
func TestCreate_createRancherIngressAndCertCopies(t *testing.T) {
	rancherSecret1 := createRancherSecret(secret1)
	rancherSecret2 := rancherSecret1.DeepCopy()
	rancherSecret2.Name = secret2
	rancherSecret2.Annotations["extraAnnotation"] = "extraAnnotationValue"

	rancherIngress := netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: common.CattleSystem,
			Name:      constants.RancherIngress,
		},
		Spec: netv1.IngressSpec{
			Rules: []netv1.IngressRule{
				{Host: host1},
				{Host: host2},
				{Host: host3},
			},
			TLS: []netv1.IngressTLS{
				{SecretName: secret1, Hosts: []string{host1, host2}},
				{SecretName: secret2, Hosts: []string{host3}},
			},
		},
	}

	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).
		WithObjects(rancherSecret1, rancherSecret2, &rancherIngress).Build()

	ctx := spi.NewFakeContext(c, &vzapi.Verrazzano{}, nil, false)
	reconciler := newVerrazzanoReconciler(ctx.Client())
	reconciler.createRancherIngressAndCertCopies(ctx)

	ingressList := netv1.IngressList{}
	err := c.List(context.TODO(), &ingressList)
	asserts.NoError(t, err)
	asserts.Equal(t, 2, len(ingressList.Items))

	expectedLabels := rancherIngress.Labels
	if expectedLabels == nil {
		expectedLabels = map[string]string{}
	}
	expectedLabels[constants2.VerrazzanoManagedLabelKey] = "true"
	verifyIngressCopyEqualToOriginal(t, ingressList, &rancherIngress)

	secretList := corev1.SecretList{}
	err = c.List(context.TODO(), &secretList)
	asserts.NoError(t, err)
	asserts.Equal(t, 4, len(secretList.Items))
}

func createRancherSecret(secretName string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: common.CattleSystem,
			Name:      secret1,
			Annotations: map[string]string{
				"cert-manager.io/certificate-name": "verrazzano-ca-certificate",
				"cert-manager.io/common-name":      "verrazzano-root-ca-abcdef",
			},
		},
		Data: map[string][]byte{
			"ca.crt":  []byte("cacert"),
			"tls.crt": []byte("tls cert"),
		},
	}
}

// GIVEN The Rancher ingress and cert copies already exist
// WHEN createRancherIngressAndCertCopies is called
// THEN the existing copies are updated and there is no error
func TestUpdate_createRancherIngressAndCertCopies(t *testing.T) {
	rancherSecret1 := createRancherSecret(secret1)
	rancherIngress := &netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: common.CattleSystem,
			Name:      constants.RancherIngress,
		},
		Spec: netv1.IngressSpec{
			Rules: []netv1.IngressRule{
				{Host: host1},
			},
			TLS: []netv1.IngressTLS{
				{SecretName: secret1, Hosts: []string{host1, host2}},
			},
		},
	}

	// if the existing copy of the secret has different content, it should be updated
	existingCopyOfRancherSecret := rancherSecret1.DeepCopy()
	existingCopyOfRancherSecret.Name = "vz-" + rancherSecret1.Name
	existingCopyOfRancherSecret.Data["tls.crt"] = []byte("someothervalue")

	// if the existing copy of the ingress has a different hostname, it should be updated
	existingCopyOfRancherIngress := rancherIngress.DeepCopy()
	existingCopyOfRancherIngress.Name = "vz-" + rancherIngress.Name
	existingCopyOfRancherIngress.Spec.Rules[0].Host = host2
	existingCopyOfRancherIngress.Spec.TLS[0].SecretName = existingCopyOfRancherSecret.Name

	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).
		WithObjects(rancherSecret1, existingCopyOfRancherSecret, rancherIngress, existingCopyOfRancherIngress).Build()

	ctx := spi.NewFakeContext(c, &vzapi.Verrazzano{}, nil, false)
	reconciler := newVerrazzanoReconciler(ctx.Client())
	reconciler.createRancherIngressAndCertCopies(ctx)

	ingressList := netv1.IngressList{}
	err := c.List(context.TODO(), &ingressList)
	asserts.NoError(t, err)
	asserts.Equal(t, 2, len(ingressList.Items))

	verifyIngressCopyEqualToOriginal(t, ingressList, rancherIngress)

	secretList := corev1.SecretList{}
	err = c.List(context.TODO(), &secretList)
	asserts.NoError(t, err)
	asserts.Equal(t, 2, len(secretList.Items))
	// the data in both secrets should be the same as in the rancherSecret1 (existing secret copy
	// should have been updated to match rancherSecret1)
	asserts.Equal(t, rancherSecret1.Data, secretList.Items[0].Data)
	asserts.Equal(t, rancherSecret1.Data, secretList.Items[1].Data)
}

func verifyIngressCopyEqualToOriginal(t *testing.T, ingressList netv1.IngressList, original *netv1.Ingress) {
	expectedLabels := original.Labels
	if expectedLabels == nil {
		expectedLabels = map[string]string{}
	}
	expectedLabels[constants2.VerrazzanoManagedLabelKey] = "true"
	for _, ing := range ingressList.Items {
		if ing.Name == original.Name {
			asserts.Equal(t, *original, ing)
		} else {
			asserts.Equal(t, original.Namespace, ing.Namespace)
			asserts.Equal(t, original.Annotations, ing.Annotations)
			asserts.Equal(t, expectedLabels, ing.Labels)
			asserts.Equal(t, original.Spec.Rules, ing.Spec.Rules)

			// should have same number of TLS entries but with different content
			asserts.Equal(t, len(original.Spec.TLS), len(ing.Spec.TLS))

			for i, tls := range original.Spec.TLS {
				// host name should have been updated since it is different from the existing copy
				asserts.Equal(t, tls.Hosts, ing.Spec.TLS[i].Hosts)
				asserts.Equal(t, ing.Spec.TLS[i].SecretName, "vz-"+tls.SecretName)
			}
		}
	}

}
