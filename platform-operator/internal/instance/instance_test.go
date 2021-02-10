// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package instance

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

	func TestGetInstanceInfo(t *testing.T) {
	const dnsDomain = "testverrazzano.com"
	const envName = "myenv"
	instanceInfo := GetInstanceInfo(envName, dnsDomain)
	assert.NotNil(t, instanceInfo)
	assert.Equal(t, fmt.Sprintf("https://%s.%s.%s", "verrazzano", envName, dnsDomain), *instanceInfo.Console)
	assert.Equal(t, fmt.Sprintf("https://%s.%s.%s", "rancher", envName, dnsDomain), *instanceInfo.RancherURL)
	assert.Equal(t, fmt.Sprintf("https://%s.%s.%s", "api", envName, dnsDomain), *instanceInfo.VzAPIURL)
	assert.Equal(t, fmt.Sprintf("https://%s.%s.%s", "keycloak", envName, dnsDomain), *instanceInfo.KeyCloakURL)
	assert.Equal(t, fmt.Sprintf("https://%s.vmi.system.%s.%s", "elasticsearch", envName, dnsDomain), *instanceInfo.ElasticURL)
	assert.Equal(t, fmt.Sprintf("https://%s.vmi.system.%s.%s", "grafana", envName, dnsDomain), *instanceInfo.GrafanaURL)
	assert.Equal(t, fmt.Sprintf("https://%s.vmi.system.%s.%s", "kibana", envName, dnsDomain), *instanceInfo.KibanaURL)
	assert.Equal(t, fmt.Sprintf("https://%s.vmi.system.%s.%s", "prometheus", envName, dnsDomain), *instanceInfo.PrometheusURL)
}
