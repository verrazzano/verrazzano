// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	networking "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	dnsSuffix = "DNS"
	name      = "NAME"
)

// TestAddAcmeIngressAnnotations verifies if ACME Annotations are added to the Ingress
// GIVEN a Rancher Ingress
//
//	WHEN addAcmeIngressAnnotations is called
//	THEN addAcmeIngressAnnotations should annotate the ingress
func TestAddAcmeIngressAnnotations(t *testing.T) {
	in := networking.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{},
		},
	}
	out := networking.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"nginx.ingress.kubernetes.io/auth-realm":  fmt.Sprintf("%s auth", dnsSuffix),
				"external-dns.alpha.kubernetes.io/target": fmt.Sprintf("verrazzano-ingress.%s.%s", name, dnsSuffix),
				"external-dns.alpha.kubernetes.io/ttl":    "60",
			},
		},
	}

	addAcmeIngressAnnotations(name, dnsSuffix, &in)
	assert.Equal(t, out, in)
}

// TestAddCAIngressAnnotations verifies if CA Annotations are added to the Ingress
// GIVEN a Rancher Ingress
//
//	WHEN addCAIngressAnnotations is called
//	THEN addCAIngressAnnotations should annotate the ingress
func TestAddCAIngressAnnotations(t *testing.T) {
	in := networking.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{},
		},
	}
	out := networking.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"nginx.ingress.kubernetes.io/auth-realm": fmt.Sprintf("%s.%s auth", name, dnsSuffix),
			},
		},
	}

	addCAIngressAnnotations(name, dnsSuffix, &in)
	assert.Equal(t, out, in)
}

// TestPatchRancherIngress should annotate the Rancher ingress with Acme/Private CA values
// GIVEN a Rancher Ingress and a Verrazzano CR
//
//	WHEN patchRancherIngress is called
//	THEN patchRancherIngress should annotate the ingress according to the Verrazzano CR
func TestPatchRancherIngress(t *testing.T) {
	ingress := networking.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   common.CattleSystem,
			Name:        common.RancherName,
			Annotations: map[string]string{"test": "data"},
		},
	}
	var tests = []struct {
		in    networking.Ingress
		vzapi vzapi.Verrazzano
	}{
		{ingress, vzAcmeDev},
		{ingress, vzDefaultCA},
	}

	for _, tt := range tests {
		c := fake.NewClientBuilder().WithScheme(getScheme()).WithObjects(&tt.in).Build()
		t.Run(tt.vzapi.Spec.EnvironmentName, func(t *testing.T) {
			assert.Nil(t, patchRancherIngress(c, &tt.vzapi))
		})
	}
}

// TestPatchRancherIngressNotFound should fail to find the ingress
// GIVEN no Rancher Ingress and a Verrazzano CR
//
//	WHEN patchRancherIngress is called
//	THEN patchRancherIngress should fail to annotate the Ingress
func TestPatchRancherIngressNotFound(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(getScheme()).Build()
	err := patchRancherIngress(c, &vzAcmeDev)
	assert.NotNil(t, err)
	assert.True(t, apierrors.IsNotFound(err))
}
