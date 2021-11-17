// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

const (
	dnsSuffix = "DNS"
	name      = "NAME"
)

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
				"cert-manager.io/issuer":                  "null",
				"cert-manager.io/issuer-kind":             "null",
				"external-dns.alpha.kubernetes.io/ttl":    "60",
			},
		},
	}

	addAcmeIngressAnnotations(name, dnsSuffix, &in)
	assert.Equal(t, out, in)
}

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
				"cert-manager.io/cluster-issuer":         "verrazzano-cluster-issuer",
			},
		},
	}

	addCAIngressAnnotations(name, dnsSuffix, &in)
	assert.Equal(t, out, in)
}

func TestGetRancherContainer(t *testing.T) {
	var tests = []struct {
		in  []v1.Container
		out bool
	}{
		{[]v1.Container{{Name: "foo"}}, false},
		{[]v1.Container{{Name: "rancher"}}, true},
		{[]v1.Container{{Name: "baz"}, {Name: "rancher"}}, true},
		{[]v1.Container{{Name: "bar"}, {Name: "foo"}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.in[0].Name, func(t *testing.T) {
			_, res := getRancherContainer(tt.in)
			assert.Equal(t, tt.out, res)
		})
	}
}

func TestPatchRancherIngress(t *testing.T) {
	ingress := networking.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   ComponentNamespace,
			Name:        ComponentName,
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
		c := fake.NewFakeClientWithScheme(getScheme(), &tt.in)
		t.Run(tt.vzapi.Spec.EnvironmentName, func(t *testing.T) {
			assert.Nil(t, patchRancherIngress(c, &tt.vzapi))
		})
	}
}

func TestPatchRancherIngressNotFound(t *testing.T) {
	c := fake.NewFakeClientWithScheme(getScheme())
	err := patchRancherIngress(c, &vzAcmeDev)
	assert.NotNil(t, err)
	assert.True(t, apierrors.IsNotFound(err))
}

func TestPatchRancherDeploymentNotFound(t *testing.T) {
	c := fake.NewFakeClientWithScheme(getScheme())
	err := patchRancherDeployment(c)
	assert.NotNil(t, err)
	assert.True(t, apierrors.IsNotFound(err))
}

func TestPatchRancherDeployment(t *testing.T) {
	var tests = []struct {
		testName   string
		deployment *appsv1.Deployment
		isError    bool
	}{
		{
			"rancherContainer",
			&appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: ComponentNamespace,
					Name:      ComponentName,
				},
				Spec: appsv1.DeploymentSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								{Name: "foobar"},
								{Name: ComponentName},
							},
						},
					},
				},
			},
			false,
		},
		{
			"noRancher",
			&appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: ComponentNamespace,
					Name:      ComponentName,
				},
				Spec: appsv1.DeploymentSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								{Name: "foobar"},
								{Name: "barfoo"},
							},
						},
					},
				},
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			c := fake.NewFakeClientWithScheme(getScheme(), tt.deployment)
			err := patchRancherDeployment(c)
			if err == nil && tt.isError {
				assert.Fail(t, "there should be an error")
			} else if err != nil && !tt.isError {
				assert.Fail(t, "there should NOT be an error")
			}
		})
	}
}
