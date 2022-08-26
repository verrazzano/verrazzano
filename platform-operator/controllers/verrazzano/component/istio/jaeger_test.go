// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

var testZipkinNamespace = "foo"
var testZipkinService = corev1.Service{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: testZipkinNamespace,
		Name:      "jaeger-collector",
		Labels: map[string]string{
			constants.KubernetesAppLabel: constants.JaegerCollectorService,
		},
	},
	Spec: corev1.ServiceSpec{
		Ports: []corev1.ServicePort{
			{
				Name: "foo",
				Port: 1,
			},
			{
				Name: "http-zipkin",
				Port: 5555,
			},
		},
	},
}

func TestConfigureJaeger(t *testing.T) {
	ctxNoService := spi.NewFakeContext(fake.NewClientBuilder().WithScheme(testScheme).Build(), jaegerEnabledCR, nil, false)
	ctxWithServiceAndUnmanagedNamespace := spi.NewFakeContext(fake.NewClientBuilder().
		WithObjects(&testZipkinService).
		WithScheme(testScheme).
		Build(), jaegerEnabledCR, nil, false)

	var tests = []struct {
		name    string
		ctx     spi.ComponentContext
		numArgs int
	}{
		{
			"2 args (tls mode and zipkin address) returned when Jaeger operator is disabled",
			spi.NewFakeContext(fake.NewClientBuilder().Build(), &vzapi.Verrazzano{}, nil, false),
			2,
		},
		{
			"2 args (tls mode and zipkin address) returned when service is not present",
			ctxNoService,
			2,
		},
		{
			"2 args (tls mode and zipkin address) returned when service is present",
			ctxWithServiceAndUnmanagedNamespace,
			2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, err := configureJaeger(tt.ctx)
			assert.NoError(t, err)
			assert.Len(t, args, tt.numArgs)
		})
	}
}
