// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package console

import (
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/bom"
	globalconst "github.com/verrazzano/verrazzano/pkg/constants"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vpoconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestPreHook(t *testing.T) {
	c := fake.NewClientBuilder().
		WithScheme(k8scheme.Scheme).
		WithObjects(&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      ComponentName,
			},
		}).
		Build()
	ctx := spi.NewFakeContext(c, &vzapi.Verrazzano{}, nil, false)
	assert.NoError(t, preHook(ctx))
}

func TestAppendOverrides(t *testing.T) {
	config.SetDefaultBomFilePath("../../testdata/test_bom.json")
	defer func() { config.SetDefaultBomFilePath("") }()
	c := fake.NewClientBuilder().
		WithScheme(k8scheme.Scheme).
		WithObjects(&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{Name: vpoconst.NGINXControllerServiceName, Namespace: globalconst.IngressNamespace},
			Spec: corev1.ServiceSpec{
				Type: corev1.ServiceTypeLoadBalancer,
			},
			Status: corev1.ServiceStatus{
				LoadBalancer: corev1.LoadBalancerStatus{
					Ingress: []corev1.LoadBalancerIngress{
						{IP: "11.22.33.44"},
					},
				},
			},
		}).
		Build()
	ctx := spi.NewFakeContext(c, &vzapi.Verrazzano{}, nil, false)
	overrides, err := AppendOverrides(ctx, "", "", "", []bom.KeyValue{})
	assert.NoError(t, err)
	assert.Len(t, overrides, 4)
}
