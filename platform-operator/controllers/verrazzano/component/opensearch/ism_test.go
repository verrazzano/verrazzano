// Copyright (C) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	appv1 "k8s.io/api/apps/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	vzv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"

	"github.com/stretchr/testify/assert"
	vmcontrollerv1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"

	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
)

const (
	testPolicyNotFound = `{"error":{"root_cause":[{"type":"status_exception","reason":"Policy not found"}],"type":"status_exception","reason":"Policy not found"},"status":404}`
	testSystemPolicy   = `
{
    "_id" : "verrazzano-system",
    "_seq_no" : 0,
    "_primary_term" : 1,
    "policy" : {
    "policy_id" : "verrazzano-system",
    "description" : "Verrazzano-managed",
    "last_updated_time" : 1647551644420,
    "schema_version" : 12,
    "error_notification" : null,
    "default_state" : "ingest",
    "states" : [
        {
        "name" : "ingest",
        "actions" : [
            {
            "rollover" : {
                "min_index_age" : "1d"
            }
            }
        ],
        "transitions" : [
            {
            "state_name" : "delete",
            "conditions" : {
                "min_index_age" : "7d"
            }
            }
        ]
        },
        {
        "name" : "delete",
        "actions" : [
            {
            "delete" : { }
            }
        ],
        "transitions" : [ ]
        }
    ],
    "ism_template" : [
        {
        "index_patterns" : [
            "verrazzano-system"
        ],
        "priority" : 1,
        "last_updated_time" : 1647551644419
        }
    ]
    }
}
`
	dataStreamResponse = `{
  "data_streams" : [
    {
      "name" : "verrazzano-system",
      "timestamp_field" : {
        "name" : "@timestamp"
      },
      "indices" : [
        {
          "index_name" : ".ds-verrazzano-system-000001",
          "index_uuid" : "mcoToYr-TMKbcpHpuIhnCA"
        },
        {
          "index_name" : ".ds-verrazzano-system-000002",
          "index_uuid" : "N9CWCrVtTTWI7FgwqyrUVw"
        }
      ],
      "generation" : 2,
      "status" : "GREEN",
      "template" : "verrazzano-data-stream"
    }
  ]
}`
	ismExplainResponse1 = `{
  ".ds-verrazzano-system-000001" : {
    "index.plugins.index_state_management.policy_id" : null,
    "index.opendistro.index_state_management.policy_id" : null,
    "enabled" : null
  },
  "total_managed_indices" : 0
}`
	ismExplainResponse2 = `{
  ".ds-verrazzano-system-000001" : {
    "index.plugins.index_state_management.policy_id" : "%s",
    "index.opendistro.index_state_management.policy_id" : "%s",
    "index" : ".ds-verrazzano-system-000001",
    "index_uuid" : "ct0nKMwESQSaEV2eyIgAbA",
    "policy_id" : "%s",
    "enabled" : true
  },
  "total_managed_indices" : 1
}`
)

var testPolicyList = fmt.Sprintf(`{
	"policies": [
     %s
   ]
}`, testSystemPolicy)

func createTestPolicy(age, rolloverAge, indexPattern, minSize string, minDocCount int) *vmcontrollerv1.IndexManagementPolicy {
	return &vmcontrollerv1.IndexManagementPolicy{
		PolicyName:   "verrazzano-system",
		IndexPattern: indexPattern,
		MinIndexAge:  &age,
		Rollover: vmcontrollerv1.RolloverPolicy{
			MinIndexAge: &rolloverAge,
			MinSize:     &minSize,
			MinDocCount: &minDocCount,
		},
	}
}

// TestConfigureIndexManagementPluginHappyPath Tests configuration of the ISM plugin
// WHEN I call Configure
// THEN the ISM configuration is created in OpenSearch
func TestConfigureIndexManagementPluginHappyPath(t *testing.T) {
	o := NewOSClient("abc")
	o.DoHTTP = func(request *http.Request) (*http.Response, error) {
		switch request.Method {
		case "GET":
			if strings.Contains(request.URL.Path, "verrazzano-system") {
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(testPolicyNotFound)),
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(testPolicyList)),
			}, nil

		case "PUT":
			return &http.Response{
				StatusCode: http.StatusCreated,
				Body:       io.NopCloser(strings.NewReader(testSystemPolicy)),
			}, nil
		default:
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("")),
			}, nil
		}
	}
	sts := &appv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "opensearch",
			Namespace: constants.VerrazzanoLoggingNamespace,
		},
		Spec: appv1.StatefulSetSpec{
			MinReadySeconds: 5,
		},
	}
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "verrazzano-system",
			Name:      "opensearch",
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					Host: "test",
				},
			},
		},
	}
	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(sts, ingress).Build()
	testLogger := vzlog.DefaultLogger()
	enabled := true
	vz := &vzv1alpha1.Verrazzano{
		Spec: vzv1alpha1.VerrazzanoSpec{
			Components: vzv1alpha1.ComponentSpec{Elasticsearch: &vzv1alpha1.ElasticsearchComponent{
				Enabled: &enabled,
				Policies: []vmcontrollerv1.IndexManagementPolicy{
					*createTestPolicy("1d", "1d", "*", "1gb", 1),
				}}},
		},
	}
	err := o.ConfigureISM(testLogger, client, vz)
	assert.NoError(t, err)
}

// TestGetPolicyByName Tests retrieving ISM policies by name
// GIVEN an OpenSearch instance
// WHEN I call getPolicyByName
// THEN the specified policy should be returned, if it exists
func TestGetPolicyByName(t *testing.T) {
	var tests = []struct {
		name       string
		policyName string
		status     int
	}{
		{
			"policy is fetched when it exists",
			"verrazzano-system",
			200,
		},
	}

	o := NewOSClient("abc")
	o.DoHTTP = func(request *http.Request) (*http.Response, error) {
		if strings.Contains(request.URL.Path, "verrazzano-system") {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(testSystemPolicy)),
			}, nil
		}
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       io.NopCloser(strings.NewReader(testPolicyNotFound)),
		}, nil
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy, err := o.getPolicyByName("http://localhost:9200/" + tt.policyName)
			assert.NoError(t, err)
			assert.Equal(t, tt.status, *policy.Status)
			if tt.status == http.StatusOK {
				assert.Equal(t, 0, *policy.SequenceNumber)
				assert.Equal(t, 1, *policy.PrimaryTerm)
			}
		})
	}
}

// TestPutUpdatedPolicy_PolicyExists Tests updating a policy in place
// GIVEN a policy that already exists in the server
// WHEN I call putUpdatedPolicy
// THEN the ISM policy should be updated in place IFF there are changes to the policy
func TestPutUpdatedPolicy_PolicyExists(t *testing.T) {
	httpFunc := func(request *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(testSystemPolicy)),
		}, nil
	}

	var tests = []struct {
		name          string
		age           string
		httpFunc      func(request *http.Request) (*http.Response, error)
		policyUpdated bool
		hasError      bool
	}{
		{
			"Policy should be updated when it already exists and the index lifecycle has changed",
			"1d",
			httpFunc,
			true,
			false,
		},
		{
			"Policy should not be updated when the index lifecycle has not changed",
			"7d",
			httpFunc,
			false,
			false,
		},
		{
			"Policy should not be updated when the update call fails",
			"1d",
			func(request *http.Request) (*http.Response, error) {
				return nil, errors.New("boom")
			},
			false,
			true,
		},
	}

	for _, tt := range tests {
		existingPolicy := &ISMPolicy{}
		err := json.NewDecoder(strings.NewReader(testSystemPolicy)).Decode(existingPolicy)
		assert.NoError(t, err)
		status := http.StatusOK
		existingPolicy.Status = &status
		o := NewOSClient("abc")
		o.DoHTTP = tt.httpFunc
		t.Run(tt.name, func(t *testing.T) {
			newPolicy := &vmcontrollerv1.IndexManagementPolicy{
				PolicyName:   "verrazzano-system",
				IndexPattern: "verrazzano-system",
				MinIndexAge:  &tt.age,
			}
			updatedPolicy, err := o.putUpdatedPolicy("http://localhost:9200", newPolicy.PolicyName, toISMPolicy(newPolicy), existingPolicy)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			if tt.policyUpdated {
				assert.NotNil(t, updatedPolicy)
			} else {
				assert.Nil(t, updatedPolicy)
			}
		})
	}
}

// TestPolicyNeedsUpdate Tests that the ISM policy will only be updated when it changes
// GIVEN a new ISM policy and the existing ISM policy
// WHEN I call policyNeedsUpdate
// THEN true is only returned if the new ISM policy has changed
func TestPolicyNeedsUpdate(t *testing.T) {
	basePolicy := createTestPolicy("7d", "1d", "verrazzano-system", "10gb", 1000)
	policyExtraState := toISMPolicy(basePolicy)
	policyExtraState.Policy.States = append(policyExtraState.Policy.States, PolicyState{
		Name:        "warm",
		Actions:     []map[string]interface{}{},
		Transitions: []PolicyTransition{},
	})
	var tests = []struct {
		name        string
		p1          *vmcontrollerv1.IndexManagementPolicy
		p2          *ISMPolicy
		needsUpdate bool
	}{
		{
			"no update when equal",
			basePolicy,
			toISMPolicy(basePolicy),
			false,
		},
		{
			"needs update when age changed",
			basePolicy,
			toISMPolicy(createTestPolicy("14d", "1d", "verrazzano-system", "10gb", 1000)),
			true,
		},
		{
			"needs update when rollover age changed",
			basePolicy,
			toISMPolicy(createTestPolicy("7d", "2d", "verrazzano-system", "10gb", 1000)),
			true,
		},
		{
			"needs update when index pattern changed",
			basePolicy,
			toISMPolicy(createTestPolicy("7d", "1d", "verrazzano-system-*", "10gb", 1000)),
			true,
		},
		{
			"needs update when min size changed",
			basePolicy,
			toISMPolicy(createTestPolicy("7d", "1d", "verrazzano-system", "20gb", 1000)),
			true,
		},
		{
			"needs update when min doc count changed",
			basePolicy,
			toISMPolicy(createTestPolicy("7d", "1d", "verrazzano-system", "10gb", 5000)),
			true,
		},
		{
			"needs update when states changed",
			basePolicy,
			policyExtraState,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			needsUpdate := policyNeedsUpdate(toISMPolicy(tt.p1), tt.p2)
			assert.Equal(t, tt.needsUpdate, needsUpdate)
		})
	}
}

// TestCleanupPolicies Tests cleaning up policies no longer managed by the VMI
// GIVEN a list of expected policies
// WHEN I call cleanupPolicies
// THEN then the existing policies should be queried and any non-matching members removed
func TestCleanupPolicies(t *testing.T) {
	o := NewOSClient("abc")

	id1 := "myapp"
	id2 := "anotherapp"

	p1 := createTestPolicy("1d", "1d", id1, "1d", 1)
	p1.PolicyName = id1
	p2 := createTestPolicy("1d", "1d", id2, "1d", 1)
	p2.PolicyName = id2
	expectedPolicies := []vmcontrollerv1.IndexManagementPolicy{
		*p1,
	}

	p1ISM := toISMPolicy(p1)
	p1ISM.ID = &id1
	p2ISM := toISMPolicy(p2)
	p2ISM.ID = &id2
	existingPolicies := &PolicyList{
		Policies: []ISMPolicy{
			*p1ISM,
			*p2ISM,
		},
	}
	existingPolicyJSON, err := json.Marshal(existingPolicies)
	assert.NoError(t, err)

	var getCalls, deleteCalls int
	o.DoHTTP = func(request *http.Request) (*http.Response, error) {
		switch request.Method {
		case "GET":
			getCalls++
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(existingPolicyJSON)),
			}, nil
		default:
			deleteCalls++
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("")),
			}, nil
		}
	}

	err = o.cleanupPolicies("http://localhost:9200", expectedPolicies)
	assert.NoError(t, err)
	assert.Equal(t, 1, getCalls)
	assert.Equal(t, 1, deleteCalls)
}

// TestIsEligibleForDeletion Tests whether a policy is eligible for deletion or not
// GIVEN a policy and the expected policy set
// WHEN I call isEligibleForDeletion
// THEN only managed policies that are not expected are eligible for deletion
func TestIsEligibleForDeletion(t *testing.T) {
	id1 := "id1"
	p1 := ISMPolicy{ID: &id1, Policy: InlinePolicy{Description: operatorManagedPolicy}}
	id2 := "id2"
	p2 := ISMPolicy{ID: &id2}
	var tests = []struct {
		name     string
		p        ISMPolicy
		e        map[string]bool
		eligible bool
	}{
		{
			"eligible when policy is managed and policy isn't expected",
			p1,
			map[string]bool{},
			true,
		},
		{
			"ineligible when policy is not managed",
			p2,
			map[string]bool{
				id1: true,
			},
			false,
		},
		{
			"ineligible when policy is managed and policy is expected",
			p1,
			map[string]bool{
				id1: true,
			},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := isEligibleForDeletion(tt.p, tt.e)
			assert.Equal(t, tt.eligible, res)
		})
	}
}

// TestGetISMPolicyFromFile tests getISMPolicyFromFile to check ISM policy object is created or not from a given json file.
// WHEN I call getISMPolicyFromFile with policyFilePath, policyFileName
// THEN the ISM policy object is created if given json file contains the policy. .
func TestGetISMPolicyFromFile(t *testing.T) {
	type args struct {
		policyFileName string
	}
	validArgs := args{
		policyFileName: systemDefaultPolicyFileName,
	}
	invalidArgs := args{
		policyFileName: "invalidFile",
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"TestGetISMPolicyFromFile when file is valid",
			validArgs,
			false,
		},
		{
			"TestGetISMPolicyFromFile when file doesn't exist",
			invalidArgs,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.TestThirdPartyManifestDir = "../../../../thirdparty/manifests"
			_, err := getISMPolicyFromFile(tt.args.policyFileName)
			if (err != nil) != tt.wantErr {
				t.Errorf("getISMPolicyFromFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

// TestUpdateISMPolicyFromFile tests updateISMPolicyFromFile to check default policies are getting created or not.
// GIVEN a default VMI instance with OS client.
// WHEN I call updateISMPolicyFromFile with policyFilePath, policyFileName and policyName
// THEN the default ISM policies are created.
func TestUpdateISMPolicyFromFile(t *testing.T) {
	type fields struct {
		httpClient *http.Client
		DoHTTP     func(request *http.Request) (*http.Response, error)
	}
	policy, err := getISMPolicyFromFile(systemDefaultPolicyFileName)
	if err != nil {
		t.Errorf("Error while creating test policy")
	}
	existingPolicyJSON, err := json.Marshal(policy)
	assert.NoError(t, err)
	field1 := fields{
		nil,
		func(request *http.Request) (*http.Response, error) {
			switch request.Method {
			case "GET":
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader("")),
				}, nil
			default:
				return &http.Response{
					StatusCode: http.StatusCreated,
					Body:       io.NopCloser(bytes.NewReader(existingPolicyJSON)),
				}, nil
			}
		},
	}
	fieldWithError := fields{
		nil,
		func(request *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(strings.NewReader("")),
			}, fmt.Errorf("internal server error")
		},
	}
	type args struct {
		openSearchEndpoint string
		policyFileName     string
		policyName         string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *ISMPolicy
		wantErr bool
	}{
		{
			"TestUpdateISMPolicyFromFile",
			field1,
			args{
				"localhost:9090",
				systemDefaultPolicyFileName,
				systemDefaultPolicy,
			},
			policy,
			false,
		},
		{
			"TestUpdateISMPolicyFromFile",
			fieldWithError,
			args{
				"localhost:9090",
				systemDefaultPolicyFileName,
				systemDefaultPolicy,
			},
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &OSClient{
				httpClient: tt.fields.httpClient,
				DoHTTP:     tt.fields.DoHTTP,
			}
			policyObject, err := getISMPolicyFromFile(tt.args.policyFileName)
			if err != nil {
				return
			}
			got, err := o.updateISMPolicy(tt.args.openSearchEndpoint, tt.args.policyFileName, policyObject)
			if (err != nil) != tt.wantErr {
				t.Errorf("updateISMPolicyFromFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("updateISMPolicyFromFile() got = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGetWriteIndexForDataStreams tests whether the correct write index for a data stream is being returned or not
// GIVEN a pattern and OS client
// WHEN I call getWriteIndexForDataStream with pattern
// THEN the correct write index should be returned
func TestGetWriteIndexForDataStreams(t *testing.T) {
	log := vzlog.DefaultLogger()
	openSearchEndpoint := "localhost:9090"
	pattern := "verrazzano-system"
	tests := []struct {
		DoHTTP             func(request *http.Request) (*http.Response, error)
		expectErr          bool
		expectedWriteIndex []string
	}{
		{
			DoHTTP: func(request *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(dataStreamResponse)),
				}, nil
			},
			expectErr:          false,
			expectedWriteIndex: []string{".ds-verrazzano-system-000002"},
		},
		{
			DoHTTP: func(request *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader("{reason: \"no such index [verrazzano-system]\"}")),
				}, nil
			},
			expectErr:          false,
			expectedWriteIndex: nil,
		},
		{
			DoHTTP: func(request *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader("")),
				}, fmt.Errorf("no healthy upstream")
			},
			expectErr:          true,
			expectedWriteIndex: nil,
		},
	}
	for _, tt := range tests {
		o := &OSClient{
			httpClient: nil,
			DoHTTP:     tt.DoHTTP,
		}
		writeIndex, err := o.getWriteIndexForDataStream(log, openSearchEndpoint, pattern)
		if tt.expectErr {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
		}
		assert.Equal(t, writeIndex, tt.expectedWriteIndex)
	}
}

// TestShouldAddOrRemoveDefaultPolicy tests whether we should add or remove policy from the index or not
// GIVEN an index and policy name and OS client
// WHEN I call shouldAddOrRemoveDefaultPolicy with an index and policy name
// THEN a boolean is returned indicating whether we should add or remove policy from the index or not
func TestShouldAddOrRemoveDefaultPolicy(t *testing.T) {
	openSearchEndpoint := "localhost:9090"
	indexName := ".ds-verrazzano-system-000001"
	defaultPolicyName := "vz-system"
	otherPolicyName := "other-policy"
	tests := []struct {
		DoHTTP         func(request *http.Request) (*http.Response, error)
		expectErr      bool
		expectedResult bool
	}{
		{
			DoHTTP: func(request *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(ismExplainResponse1)),
				}, nil
			},
			expectErr:      false,
			expectedResult: true,
		},
		{
			DoHTTP: func(request *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(fmt.Sprintf(ismExplainResponse2, defaultPolicyName, defaultPolicyName, defaultPolicyName))),
				}, nil
			},
			expectErr:      false,
			expectedResult: true,
		},
		{
			DoHTTP: func(request *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(fmt.Sprintf(ismExplainResponse2, otherPolicyName, otherPolicyName, otherPolicyName))),
				}, nil
			},
			expectErr:      false,
			expectedResult: false,
		},
	}
	for _, tt := range tests {
		o := &OSClient{
			httpClient: nil,
			DoHTTP:     tt.DoHTTP,
		}
		ok, err := o.shouldAddOrRemoveDefaultPolicy(openSearchEndpoint, indexName, defaultPolicyName)
		if tt.expectErr {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
		}
		assert.Equal(t, ok, tt.expectedResult)
	}
}
