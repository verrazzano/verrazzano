// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package instance

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

const urlTemplate = "https://%v.%v"

const vzURI = "abc.v8o.example.com"

type uriTest struct {
	name           string
	expectedPrefix string
	testChangedURI bool
	verrazzanoURI  string
}

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

// Test KeyCloak Url
func TestGetKeyCloakUrl(t *testing.T) {
	var tests = [...]uriTest{
		{"With Verrazzano URI set", "keycloak", true, vzURI},
	}
	for _, tt := range tests {
		runURLTestWithExpectedPrefix(t, tt, getKeyCloakURL, tt.expectedPrefix)
	}
}

// Test Kibana Url
func TestGetKibanaUrl(t *testing.T) {
	var tests = [...]uriTest{
		{"With Verrazzano URI set", "kibana.vmi.system", true, vzURI},
	}
	for _, tt := range tests {
		runURLTestWithExpectedPrefix(t, tt, getKibanaURL, tt.expectedPrefix)
	}
}

// Test Grafana Url
func TestGetGrafanaUrl(t *testing.T) {
	var tests = [...]uriTest{
		{"With Verrazzano URI set", "grafana.vmi.system", true, vzURI},
	}
	for _, tt := range tests {
		runURLTestWithExpectedPrefix(t, tt, getGrafanaURL, tt.expectedPrefix)
	}
}

// Test Prometheus Url
func TestGetPrometheusUrl(t *testing.T) {
	var tests = [...]uriTest{
		{"With Verrazzano URI set", "prometheus.vmi.system", true, vzURI},
	}
	for _, tt := range tests {
		runURLTestWithExpectedPrefix(t, tt, getPrometheusURL, tt.expectedPrefix)
	}
}

// Test Elastic Search Url
func TestGetElasticUrl(t *testing.T) {
	var tests = [...]uriTest{
		{"With Verrazzano URI set", "elasticsearch.vmi.system", true, vzURI},
	}
	for _, tt := range tests {
		runURLTestWithExpectedPrefix(t, tt, getElasticURL, tt.expectedPrefix)
	}
}

// Test Console Url
func TestGetConsoleURL(t *testing.T) {
	var tests = [...]uriTest{
		{"With Verrazzano URI set", "verrazzano", true, vzURI},
	}
	for _, tt := range tests {
		runURLTestWithExpectedPrefix(t, tt, getConsoleURL, tt.expectedPrefix)
	}
}

func runURLTestWithExpectedPrefix(t *testing.T, tt uriTest, methodUnderTest func(string) *string, expectedURLPrefix string) {
	//GIVEN the verrazzano URI is set
	//SetVerrazzanoURI(tt.verrazzanoURI)
	expectedURL := fmt.Sprintf(urlTemplate, expectedURLPrefix, vzURI)

	//WHEN methodUnderTest is called, THEN assert the URL value is as expected
	assert.Equal(t, expectedURL, *methodUnderTest(vzURI), "URL not as expected")

	if tt.testChangedURI {
		vzURI2 := fmt.Sprintf("changed.%v", tt.verrazzanoURI)
		//GIVEN the verrazzano URI is changed
		//SetVerrazzanoURI(vzURI2)
		expectedURL = fmt.Sprintf(urlTemplate, expectedURLPrefix, vzURI2)

		//WHEN methodUnderTest is called, THEN assert the value changes as expected
		assert.Equal(t, expectedURL, *methodUnderTest(vzURI2), "URL not as expected after changing Verrazzano URI")
	}
}
