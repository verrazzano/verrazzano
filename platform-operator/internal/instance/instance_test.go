// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package instance

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/verrazzano/verrazzano-operator/pkg/util"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano-crd-generator/pkg/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano-operator/pkg/api/clusters"
	"github.com/verrazzano/verrazzano-operator/pkg/testutil"
	"github.com/verrazzano/verrazzano-operator/pkg/testutilcontroller"
	"github.com/verrazzano/verrazzano-operator/pkg/types"
	"k8s.io/client-go/kubernetes"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

const urlTemplate = "https://%v.%v"

const vzURI = "abc.v8o.example.com"

type uriTest struct {
	name           string
	expectedPrefix string
	testChangedURI bool
	verrazzanoURI  string
}

var origLookupEnvFunc = util.LookupEnvFunc

// Test KeyCloak Url
func TestGetKeyCloakUrl(t *testing.T) {
	var tests = [...]uriTest{
		{"With Verrazzano URI set", "keycloak", true, vzURI},
		{"Without Verrazzano URI set", "", false, ""},
	}
	for _, tt := range tests {
		runURLTestWithExpectedPrefix(t, tt, GetKeyCloakURL, tt.expectedPrefix)
	}
}

// Test Kibana Url
func TestGetKibanaUrl(t *testing.T) {
	var tests = [...]uriTest{
		{"With Verrazzano URI set", "kibana.vmi.system", true, vzURI},
		{"Without Verrazzano URI set", "", false, ""},
	}
	for _, tt := range tests {
		runURLTestWithExpectedPrefix(t, tt, GetKibanaURL, tt.expectedPrefix)
	}
}

// Test Grafana Url
func TestGetGrafanaUrl(t *testing.T) {
	var tests = [...]uriTest{
		{"With Verrazzano URI set", "grafana.vmi.system", true, vzURI},
		{"Without Verrazzano URI set", "", false, ""},
	}
	for _, tt := range tests {
		runURLTestWithExpectedPrefix(t, tt, GetGrafanaURL, tt.expectedPrefix)
	}
}

// Test Prometheus Url
func TestGetPrometheusUrl(t *testing.T) {
	var tests = [...]uriTest{
		{"With Verrazzano URI set", "prometheus.vmi.system", true, vzURI},
		{"Without Verrazzano URI set", "", false, ""},
	}
	for _, tt := range tests {
		runURLTestWithExpectedPrefix(t, tt, GetPrometheusURL, tt.expectedPrefix)
	}
}

// Test Elastic Search Url
func TestGetElasticUrl(t *testing.T) {
	var tests = [...]uriTest{
		{"With Verrazzano URI set", "elasticsearch.vmi.system", true, vzURI},
		{"Without Verrazzano URI set", "", false, ""},
	}
	for _, tt := range tests {
		runURLTestWithExpectedPrefix(t, tt, GetElasticURL, tt.expectedPrefix)
	}
}

// Test Console Url
func TestGetConsoleURL(t *testing.T) {
	var tests = [...]uriTest{
		{"With Verrazzano URI set", "console", true, vzURI},
		{"Without Verrazzano URI set", "", false, ""},
	}
	for _, tt := range tests {
		runURLTestWithExpectedPrefix(t, tt, GetConsoleURL, tt.expectedPrefix)
	}
}

func runURLTestWithExpectedPrefix(t *testing.T, tt uriTest, methodUnderTest func() string, expectedURLPrefix string) {
	//GIVEN the verrazzano URI is set
	SetVerrazzanoURI(tt.verrazzanoURI)
	expectedURL := fmt.Sprintf(urlTemplate, expectedURLPrefix, vzURI)
	if expectedURLPrefix == "" {
		expectedURL = ""
	}

	//WHEN methodUnderTest is called, THEN assert the URL value is as expected
	assert.Equal(t, expectedURL, methodUnderTest(), "URL not as expected")

	if tt.testChangedURI {
		vzURI2 := fmt.Sprintf("changed.%v", tt.verrazzanoURI)
		//GIVEN the verrazzano URI is changed
		SetVerrazzanoURI(vzURI2)
		expectedURL = fmt.Sprintf(urlTemplate, expectedURLPrefix, vzURI2)

		//WHEN methodUnderTest is called, THEN assert the value changes as expected
		assert.Equal(t, expectedURL, methodUnderTest(), "URL not as expected after changing Verrazzano URI")
	}
}

func TestIsUsingSharedVMI(t *testing.T) {
	var modelBindingPairs = map[string]*types.ModelBindingPair{
		"test-pair-1": testutil.ReadModelBindingPair(
			"../../testutil/testdata/test_model.yaml",
			"../../testutil/testdata/test_binding.yaml",
			"../../testutil/testdata/test_managed_cluster_1.yaml", "../../testutil/testdata/test_managed_cluster_2.yaml"),
	}
	// GIVEN empty model binding pairs, managed clusters and fake kubernetes clients.
	var clients kubernetes.Interface = k8sfake.NewSimpleClientset()
	managedClusters := []v1beta1.VerrazzanoManagedCluster{}
	clusters.Init(testutilcontroller.NewControllerListers(&clients, managedClusters, &modelBindingPairs))
	SetVerrazzanoURI(vzURI)

	type args struct {
		w http.ResponseWriter
		r *http.Request
	}
	tests := []struct {
		name             string
		isUsingSharedVMI bool
	}{
		{
			name:             "verifyIsUsingSharedVMIUnset",
			isUsingSharedVMI: false,
		},
		{
			name:             "verifyIsUsingSharedVMISet",
			isUsingSharedVMI: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			var myInstance InstanceInfo
			assert := assert.New(t)
			resp := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/instance/0", nil)
			util.LookupEnvFunc = func(key string) (string, bool) {
				if key == "USE_SYSTEM_VMI" {
					return strconv.FormatBool(tt.isUsingSharedVMI), true
				}
				return origLookupEnvFunc(key)
			}
			ReturnSingleInstance(resp, req)
			assert.Equal(http.StatusOK, resp.Code, "expect the http return code to be http.StatusOk")
			json.NewDecoder(resp.Body).Decode(&myInstance)
			assert.Equal(tt.isUsingSharedVMI, myInstance.IsUsingSharedVMI, fmt.Sprintf("expect IsUsingSharedVMI in instance api response to be %v.", tt.isUsingSharedVMI))
		})
	}
	defer func() { util.LookupEnvFunc = origLookupEnvFunc }()
}
