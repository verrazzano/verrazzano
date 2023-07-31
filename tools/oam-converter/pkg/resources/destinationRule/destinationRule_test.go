// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package destinationRule

import (
	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	istio "istio.io/api/networking/v1beta1"
	istioclient "istio.io/client-go/pkg/apis/networking/v1alpha3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestCreateDestinationFromRule(t *testing.T) {
	testCases := []struct {
		name          string
		rule          vzapi.IngressRule
		expectedDest  *istio.HTTPRouteDestination
		expectedError error
	}{
		// Test case 1: Rule with host and port
		{
			name: "Test case 1",
			rule: vzapi.IngressRule{
				Destination: vzapi.IngressDestination{
					Host: "example.com",
					Port: 8080,
				},
			},
			expectedDest: &istio.HTTPRouteDestination{
				Destination: &istio.Destination{
					Host: "example.com",
					Port: &istio.PortSelector{Number: 8080},
				},
			},
			expectedError: nil,
		},
		// Test case 2: Rule with host only
		{
			name: "Test case 2",
			rule: vzapi.IngressRule{
				Destination: vzapi.IngressDestination{
					Host: "example2.com",
				},
			},
			expectedDest: &istio.HTTPRouteDestination{
				Destination: &istio.Destination{
					Host: "example2.com",
				},
			},
			expectedError: nil,
		},
		// Test case 3: Rule without host or port
		{
			name:          "Test case 3",
			rule:          vzapi.IngressRule{},
			expectedDest:  &istio.HTTPRouteDestination{Destination: &istio.Destination{}},
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dest, err := CreateDestinationFromRule(tc.rule)
			assert.Equal(t, tc.expectedDest, dest, "Unexpected destination")
			assert.Equal(t, tc.expectedError, err, "Unexpected error")
		})
	}
}

func TestCreateDestinationRule(t *testing.T) {

	// Test case 1: Rule with HTTPCookie
	trait := &vzapi.IngressTrait{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "my-namespace",
		},
	}
	rule1 := vzapi.IngressRule{
		Destination: vzapi.IngressDestination{
			HTTPCookie: &vzapi.IngressDestinationHTTPCookie{
				Name: "my-cookie",
				Path: "/",
				TTL:  60,
			},
		},
	}
	name := "my-destination-rule"
	destRule1, err1 := CreateDestinationRule(trait, rule1, name)
	assert.NoError(t, err1, "Error was not expected for test case 1")
	assert.NotNil(t, destRule1, "DestinationRule should not be nil for test case 1")

	// Test case 2: Rule without HTTPCookie
	rule2 := vzapi.IngressRule{}
	destRule2, err2 := CreateDestinationRule(trait, rule2, name)
	assert.NoError(t, err2, "Error was not expected for test case 2")
	assert.Nil(t, destRule2, "DestinationRule should be nil for test case 2")
}

func TestMutateDestinationRule(t *testing.T) {
	// Test case 1: Namespace with istio-injection enabled
	namespace1 := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{"istio-injection": "enabled"},
		},
	}
	rule1 := vzapi.IngressRule{
		Destination: vzapi.IngressDestination{

			Host: "example.com",
			HTTPCookie: &vzapi.IngressDestinationHTTPCookie{
				Name: "my-cookie",
				Path: "/",
				TTL:  60,
			},
		},
	}
	destinationRule1 := &istioclient.DestinationRule{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "my-namespace",
			Name:      "my-destination-rule",
		},
	}
	destRule1, err1 := mutateDestinationRule(destinationRule1, rule1, namespace1)
	assert.NoError(t, err1, "Error was not expected for test case 1")
	assert.NotNil(t, destRule1, "DestinationRule should not be nil for test case 1")

}
