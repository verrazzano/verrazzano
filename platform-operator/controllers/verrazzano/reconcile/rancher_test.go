// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package reconcile

import (
	"context"
	"testing"

	asserts "github.com/stretchr/testify/assert"
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

func TestReconciler_createRancherIngressAndCertCopies(t *testing.T) {
	host1 := "rancher1.somedomain.com"
	host2 := "rancher2.somedomain.com"
	host3 := "rancher3.otherdomain.com"
	secret1 := "secret1"
	secret2 := "secret2"
	rancherSecret1 := corev1.Secret{
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
				{SecretName: "secret1", Hosts: []string{host1, host2}},
				{SecretName: "secret2", Hosts: []string{host3}},
			},
		},
	}

	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).
		WithObjects(&rancherSecret1, rancherSecret2, &rancherIngress).Build()

	ctx := spi.NewFakeContext(c, &vzapi.Verrazzano{}, nil, false)
	reconciler := newVerrazzanoReconciler(ctx.Client())
	reconciler.createRancherIngressAndCertCopies(ctx)

	ingressList := netv1.IngressList{}
	err := c.List(context.TODO(), &ingressList)
	asserts.NoError(t, err)
	asserts.Equal(t, 2, len(ingressList.Items))

	for _, ing := range ingressList.Items {
		if ing.Name == rancherIngress.Name {
			asserts.Equal(t, rancherIngress, ing)
		} else {
			asserts.Equal(t, rancherIngress.Namespace, ing.Namespace)
			asserts.Equal(t, rancherIngress.Annotations, ing.Annotations)
			asserts.Equal(t, rancherIngress.Labels, ing.Labels)
			asserts.Equal(t, rancherIngress.Spec.Rules, ing.Spec.Rules)

			// should have same number of TLS entries but with different content
			asserts.Equal(t, len(rancherIngress.Spec.TLS), len(ing.Spec.TLS))
			for i, tls := range rancherIngress.Spec.TLS {
				asserts.Equal(t, tls.Hosts, ing.Spec.TLS[i].Hosts)
				asserts.Equal(t, ing.Spec.TLS[i].SecretName, "vz-"+tls.SecretName)
			}
		}
	}

	secretList := corev1.SecretList{}
	err = c.List(context.TODO(), &secretList)
	asserts.NoError(t, err)
	asserts.Equal(t, 4, len(secretList.Items))
}
