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

var testZipkinService = corev1.Service{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: "foo",
		Name:      "foo",
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
				Port: 2,
			},
		},
	},
}

func TestZipkinPort(t *testing.T) {
	var tests = []struct {
		name    string
		service corev1.Service
		port    int32
	}{
		{
			"9411 when no named port",
			corev1.Service{},
			9411,
		},
		{
			"service port when named port",
			testZipkinService,
			2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.port, zipkinPort(tt.service))
		})
	}
}

func TestConfigureJaeger(t *testing.T) {
	ctxNoService := spi.NewFakeContext(fake.NewClientBuilder().WithScheme(testScheme).Build(), jaegerEnabledCR, false)
	ctxWithService := spi.NewFakeContext(fake.NewClientBuilder().
		WithObjects(&testZipkinService).
		WithScheme(testScheme).
		Build(),
		jaegerEnabledCR,
		false,
	)
	var tests = []struct {
		name    string
		ctx     spi.ComponentContext
		numArgs int
	}{
		{
			"no args when jaeger disabled",
			spi.NewFakeContext(fake.NewClientBuilder().Build(), &vzapi.Verrazzano{}, false),
			0,
		},
		{
			"no args when service not present",
			ctxNoService,
			0,
		},
		{
			"two args when service present",
			ctxWithService,
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
