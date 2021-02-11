// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vzinstance

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestGetInstanceInfo tests buildDomain method
// GIVEN a request to GetInstanceInfo
// WHEN with a proper domain
// THEN the an instance info struct is returned with the expected URLs
func TestGetInstanceInfo(t *testing.T) {
	const dnsDomain = "myenv.testverrazzano.com"
	instanceInfo := GetInstanceInfo(dnsDomain)
	assert.NotNil(t, instanceInfo)
	assert.Equal(t, fmt.Sprintf("https://%s.%s", "verrazzano", dnsDomain), *instanceInfo.Console)
	assert.Equal(t, fmt.Sprintf("https://%s.%s", "rancher", dnsDomain), *instanceInfo.RancherURL)
	assert.Equal(t, fmt.Sprintf("https://%s.%s", "keycloak", dnsDomain), *instanceInfo.KeyCloakURL)
	assert.Equal(t, fmt.Sprintf("https://%s.vmi.system.%s", "elasticsearch", dnsDomain), *instanceInfo.ElasticURL)
	assert.Equal(t, fmt.Sprintf("https://%s.vmi.system.%s", "grafana", dnsDomain), *instanceInfo.GrafanaURL)
	assert.Equal(t, fmt.Sprintf("https://%s.vmi.system.%s", "kibana", dnsDomain), *instanceInfo.KibanaURL)
	assert.Equal(t, fmt.Sprintf("https://%s.vmi.system.%s", "prometheus", dnsDomain), *instanceInfo.PrometheusURL)
}

// TestDeriveNegative tests buildDomain method
// GIVEN a request to deriveURL
// WHEN with an empty domain
// THEN nil is returned
func TestDeriveNegative(t *testing.T) {
	url := deriveURL("", "foo")
	assert.Nil(t, url)
}
