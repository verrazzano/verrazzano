// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package kiali

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/bom"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"testing"

	"github.com/stretchr/testify/assert"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestAppendOverrides tests the AppendOverrides function
// GIVEN a call to AppendOverrides
//  WHEN there is a valid DNS configuration
//  THEN the correct Helm overrides are returned
func TestAppendOverrides(t *testing.T) {
	fakeClient := fake.NewFakeClientWithScheme(k8scheme.Scheme)
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				DNS: &vzapi.DNSComponent{
					OCI: &vzapi.OCI{
						DNSZoneName: "mydomain.com",
					},
				},
			},
		},
	}
	kvs, err := AppendOverrides(spi.NewFakeContext(fakeClient, vz, false), "", "", "", []bom.KeyValue{{Key: "key1", Value: "value1"}})
	assert.Nil(t, err)
	assert.Len(t, kvs, 2)
	assert.Equal(t, bom.KeyValue{Key: "key1", Value: "value1"}, kvs[0])
	assert.Equal(t, bom.KeyValue{Key: webFQDNKey, Value: fmt.Sprintf("%s.default.mydomain.com", kialiHostName)}, kvs[1])
}
