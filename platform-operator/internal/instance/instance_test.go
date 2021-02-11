// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package instance

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetInstanceInfo(t *testing.T) {
	const dnsDomain = "myenv.testverrazzano.com"
	instanceInfo := GetInstanceInfo(dnsDomain)
	assert.NotNil(t, instanceInfo)
	assert.Equal(t, fmt.Sprintf("https://%s.%s", "verrazzano", dnsDomain), *instanceInfo.Console)
	assert.Equal(t, fmt.Sprintf("https://%s.%s", "rancher", dnsDomain), *instanceInfo.RancherURL)
	assert.Equal(t, fmt.Sprintf("https://%s.%s", "api", dnsDomain), *instanceInfo.VzAPIURL)
	assert.Equal(t, fmt.Sprintf("https://%s.%s", "keycloak", dnsDomain), *instanceInfo.KeyCloakURL)
	assert.Equal(t, fmt.Sprintf("https://%s.vmi.system.%s", "elasticsearch", dnsDomain), *instanceInfo.ElasticURL)
	assert.Equal(t, fmt.Sprintf("https://%s.vmi.system.%s", "grafana", dnsDomain), *instanceInfo.GrafanaURL)
	assert.Equal(t, fmt.Sprintf("https://%s.vmi.system.%s", "kibana", dnsDomain), *instanceInfo.KibanaURL)
	assert.Equal(t, fmt.Sprintf("https://%s.vmi.system.%s", "prometheus", dnsDomain), *instanceInfo.PrometheusURL)
}
