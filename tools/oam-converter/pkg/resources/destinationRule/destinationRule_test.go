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
	// Test case 1: Rule with host and port
	rule1 := vzapi.IngressRule{
		Destination: vzapi.IngressDestination{
			Host: "example.com",
			Port: 8080,
		},
	}
	dest1, err1 := CreateDestinationFromRule(rule1)
	assert.NoError(t, err1, "Error was not expected for test case 1")
	expectedDest1 := &istio.HTTPRouteDestination{
		Destination: &istio.Destination{
			Host: "example.com",
			Port: &istio.PortSelector{Number: 8080},
		},
	}
	assert.Equal(t, expectedDest1, dest1, "Unexpected destination for test case 1")

	// Test case 2: Rule with host only
	rule2 := vzapi.IngressRule{
		Destination: vzapi.IngressDestination{
			Host: "example2.com",
		},
	}
	dest2, err2 := CreateDestinationFromRule(rule2)
	assert.NoError(t, err2, "Error was not expected for test case 2")
	expectedDest2 := &istio.HTTPRouteDestination{
		Destination: &istio.Destination{
			Host: "example2.com",
		},
	}
	assert.Equal(t, expectedDest2, dest2, "Unexpected destination for test case 2")

	// Test case 3: Rule without host or port
	rule3 := vzapi.IngressRule{}
	dest3, err3 := CreateDestinationFromRule(rule3)
	assert.NoError(t, err3, "Error was not expected for test case 3")
	expectedDest3 := &istio.HTTPRouteDestination{
		Destination: &istio.Destination{},
	}
	assert.Equal(t, expectedDest3, dest3, "Unexpected destination for test case 3")
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
	// Add more assertions as needed for other fields in destRule1

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
