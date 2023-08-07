// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package services

import (
	"github.com/stretchr/testify/assert"
	istio "istio.io/api/networking/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"testing"
)

func TestCreateDestinationFromService(t *testing.T) {
	testCases := []struct {
		name          string
		service       *corev1.Service
		expectedDest  *istio.HTTPRouteDestination
		expectedError error
	}{
		// Test case 1: Service with ports
		{
			name: "Test case 1",
			service: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{Name: "my-service"},
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{
						{
							Name:       "http",
							Port:       8080,
							TargetPort: intstr.FromInt(8080),
						},
					},
				},
			},
			expectedDest: &istio.HTTPRouteDestination{
				Destination: &istio.Destination{Host: "my-service",
					Port: &istio.PortSelector{Number: 8080},
				},
			},
			expectedError: nil,
		},
		// Test case 2: Service without ports
		{
			name: "Test case 2",
			service: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{Name: "my-service"},
				Spec:       corev1.ServiceSpec{},
			},
			expectedDest:  &istio.HTTPRouteDestination{Destination: &istio.Destination{Host: "my-service"}},
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dest, err := CreateDestinationFromService(tc.service)
			assert.Equal(t, tc.expectedDest, dest, "Unexpected destination")
			assert.Equal(t, tc.expectedError, err, "Unexpected error")
		})
	}
}

func TestSelectPortForDestination(t *testing.T) {
	testCases := []struct {
		name          string
		service       *corev1.Service
		expectedPort  corev1.ServicePort
		expectedError error
	}{
		// Test case 1: Service with a single port
		{
			name: "Test case 1",
			service: &corev1.Service{
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{
						{
							Name:     "http",
							Port:     8080,
							Protocol: corev1.ProtocolTCP,
						},
					},
				},
			},
			expectedPort: corev1.ServicePort{
				Name:     "http",
				Port:     8080,
				Protocol: corev1.ProtocolTCP,
			},
			expectedError: nil,
		},
		// Test case 2: Service with multiple ports and one http/WebLogic port
		{
			name: "Test case 2",
			service: &corev1.Service{
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{
						{
							Name:     "http",
							Port:     8080,
							Protocol: corev1.ProtocolTCP,
						},
						{
							Name:     "not-http",
							Port:     9090,
							Protocol: corev1.ProtocolTCP,
						},
					},
				},
			},
			expectedPort: corev1.ServicePort{
				Name:     "http",
				Port:     8080,
				Protocol: corev1.ProtocolTCP,
			},
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			port, err := selectPortForDestination(tc.service)
			assert.Equal(t, tc.expectedPort, port, "Unexpected service port")
			assert.Equal(t, tc.expectedError, err, "Unexpected error")
		})
	}
}
