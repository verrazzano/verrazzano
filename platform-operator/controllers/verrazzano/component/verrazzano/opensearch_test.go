// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	k8sutilfake "github.com/verrazzano/verrazzano/pkg/k8sutil/fake"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
	"time"
)

const fakeCatIndicesOutput = `[
  {
    "index": "verrazzano-namespace-verrazzano-system"
  },
  {
    "index": "verrazzano-logstash-2022.01.23"
  },
  {
    "index": "verrazzano-namespace-kube-system"
  },
  {
    "index": "verrazzano-namespace-fleet-system"
  },
  {
    "index": "verrazzano-namespace-ingress-nginx"
  },
  {
    "index": "verrazzano-systemd-journal"
  },
  {
    "index": "verrazzano-namespace-todo-list"
  },
  {
    "index": "verrazzano-namespace-bobs-books"
  }
]`

const fakeGetTemplateOutput = `{
  "index_templates": [
    {
      "name":"verrazzano-data-stream",
      "index_template": {
        "index_patterns": [
          "verrazzano-system",
          "verrazzano-application*"
        ],
        "template": {
          "settings": {
            "index": {
              "mapping": {
                "total_fields": {
                  "limit": "2000"
                }
              },
              "refresh_interval": "5s",
              "number_of_shards": "1",
              "auto_expand_replicas": "0-1",
              "number_of_replicas": "0"
            }
          }
        }
      }
    }
  ]
}`

// TestGetSystemIndices tests the getSystemIndices function.
func TestGetSystemIndices(t *testing.T) {
	assert := assert.New(t)

	// GIVEN an Elasticsearch pod
	//  WHEN getSystemIndices is called
	//  THEN a command should be executed to get the indices information
	//   AND then Verrazzano system indices should be filtered
	//   AND no error should be returned
	k8sutil.NewPodExecutor = k8sutilfake.NewPodExecutor
	k8sutilfake.PodSTDOUT = fakeCatIndicesOutput
	k8sutil.ClientConfig = func() (*rest.Config, kubernetes.Interface, error) {
		config2, k := k8sutilfake.NewClientsetConfig()
		return config2, k, nil
	}
	pod := readyOpenSearchPods()
	client2 := fake.NewFakeClientWithScheme(testScheme, pod)
	ctx := spi.NewFakeContext(client2, &vzapi.Verrazzano{}, false)
	cfg, cli, err := k8sutil.ClientConfig()
	assert.NoError(err, "Error not expected")
	indices, err := getSystemIndices(ctx.Log(), cfg, cli, pod)
	assert.NoError(err, "Failed to get system indices")
	assert.Contains(indices, "verrazzano-systemd-journal")
	assert.Contains(indices, "verrazzano-namespace-kube-system")
	assert.NotContains(indices, "verrazzano-namespace-bobs-books")
}

// TestGetApplicationIndices tests the getApplicationIndices function.
func TestGetApplicationIndices(t *testing.T) {
	asrt := assert.New(t)

	// GIVEN an Elasticsearch pod
	//  WHEN getApplicationIndices is called
	//  THEN a command should be executed to get the indices information
	//   AND then Verrazzano application indices should be filtered
	//   AND no error should be returned
	k8sutil.NewPodExecutor = k8sutilfake.NewPodExecutor
	k8sutilfake.PodSTDOUT = fakeCatIndicesOutput
	k8sutil.ClientConfig = func() (*rest.Config, kubernetes.Interface, error) {
		config, k := k8sutilfake.NewClientsetConfig()
		return config, k, nil
	}
	pod := readyOpenSearchPods()
	client := fake.NewFakeClientWithScheme(testScheme, pod)
	ctx := spi.NewFakeContext(client, &vzapi.Verrazzano{}, false)
	cfg, cli, err := k8sutil.ClientConfig()
	asrt.NoError(err, "Error not expected")
	indices, err := getApplicationIndices(ctx.Log(), cfg, cli, pod)
	asrt.NoError(err, "Failed to get application indices")
	asrt.Contains(indices, "verrazzano-namespace-bobs-books")
	asrt.NotContains(indices, "verrazzano-systemd-journal")
	asrt.NotContains(indices, "verrazzano-namespace-kube-system")
}

// TestVerifyDataStreamTemplateExists tests if the template exists for data streams
func TestVerifyDataStreamTemplateExists(t *testing.T) {
	asrt := assert.New(t)

	// GIVEN an Elasticsearch pod
	//  WHEN verifyDataStreamTemplateExists is called
	//  THEN a command should be executed to get the specified template information
	//   AND no error should be returned
	k8sutil.NewPodExecutor = k8sutilfake.NewPodExecutor
	k8sutilfake.PodSTDOUT = fakeGetTemplateOutput
	k8sutil.ClientConfig = func() (*rest.Config, kubernetes.Interface, error) {
		config, k := k8sutilfake.NewClientsetConfig()
		return config, k, nil
	}
	pod := readyOpenSearchPods()
	client := fake.NewFakeClientWithScheme(testScheme, pod)
	ctx := spi.NewFakeContext(client, &vzapi.Verrazzano{}, false)
	cfg, cli, err := k8sutil.ClientConfig()
	asrt.NoError(err, "Error not expected")
	err = verifyDataStreamTemplateExists(ctx.Log(), cfg, cli, pod, dataStreamTemplateName, 2*time.Second, 1*time.Second)
	asrt.NoError(err, "Failed to verify data stream template existence")
	err = verifyDataStreamTemplateExists(ctx.Log(), cfg, cli, pod, "test", 2*time.Second, 1*time.Second)
	asrt.Error(err, "Error should be returned")
}

func Test_formatISMPayload(t *testing.T) {
	age := "12d"

	var tests = []struct {
		name        string
		policy      vzapi.RetentionPolicy
		containsStr string
	}{
		{
			"Should format with default values",
			vzapi.RetentionPolicy{},
			defaultMinIndexAge,
		},
		{
			"Should format with custom values",
			vzapi.RetentionPolicy{
				MinAge: &age,
			},
			age,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, err := formatISMPayload(tt.policy, systemISMPayloadTemplate)
			assert.NoError(t, err)
			assert.Contains(t, payload, tt.containsStr)
		})
	}
}

// Test_fixupOpenSearchReplicaCount tests the fixupOpenSearchReplicaCount function.
func TestFixupOpenSearchReplicaCount(t *testing.T) {
	assert := assert.New(t)

	// GIVEN an OpenSearch pod with a http port
	//  WHEN fixupOpenSearchReplicaCount is called
	//  THEN a command should be executed to get the cluster health information
	//   AND a command should be executed to update the cluster index settings
	//   AND no error should be returned
	context, err := createFakeComponentContext()
	assert.NoError(err, "Failed to create fake component context.")
	createOpenSearchPod(context.Client(), "http")
	execCommand = fakeExecCommand
	fakeExecScenarioNames = []string{"fixupOpenSearchReplicaCount/get", "fixupOpenSearchReplicaCount/put"} //nolint,ineffassign
	fakeExecScenarioIndex = 0                                                                              //nolint,ineffassign
	err = fixupOpenSearchReplicaCount(context, "verrazzano-system")
	assert.NoError(err, "Failed to fixup Elasticsearch index template")

	// GIVEN an OpenSearch pod with no http port
	//  WHEN fixupOpenSearchReplicaCount is called
	//  THEN an error should be returned
	//   AND no commands should be invoked
	fakeExecScenarioNames = []string{} //nolint,ineffassign
	fakeExecScenarioIndex = 0          //nolint,ineffassign
	context, err = createFakeComponentContext()
	assert.NoError(err, "Failed to create fake component context.")
	createOpenSearchPod(context.Client(), "tcp")
	err = fixupOpenSearchReplicaCount(context, "verrazzano-system")
	assert.Error(err, "Error should be returned if there is no http port for elasticsearch pods")

	// GIVEN a Verrazzano resource with version 1.1.0 in the status
	//  WHEN fixupOpenSearchReplicaCount is called
	//  THEN no error should be returned
	//   AND no commands should be invoked
	fakeExecScenarioNames = []string{} //nolint,ineffassign
	fakeExecScenarioIndex = 0          //nolint,ineffassign
	context, err = createFakeComponentContext()
	assert.NoError(err, "Unexpected error")
	context.ActualCR().Status.Version = "1.1.0"
	err = fixupOpenSearchReplicaCount(context, "verrazzano-system")
	assert.NoError(err, "No error should be returned if the source version is 1.1.0 or later")

	// GIVEN a Verrazzano resource with OpenSearch disabled
	//  WHEN fixupOpenSearchReplicaCount is called
	//  THEN no error should be returned
	//   AND no commands should be invoked
	fakeExecScenarioNames = []string{}
	fakeExecScenarioIndex = 0
	falseValue := false
	context, err = createFakeComponentContext()
	assert.NoError(err, "Unexpected error")
	context.EffectiveCR().Spec.Components.Elasticsearch.Enabled = &falseValue
	err = fixupOpenSearchReplicaCount(context, "verrazzano-system")
	assert.NoError(err, "No error should be returned if the elasticsearch is not enabled")
}

func TestFormatReindexPayload(t *testing.T) {
	asrt := assert.New(t)
	input := ReindexInput{SourceName: "verrazzano-namespace-bobs-books",
		DestinationName: "verrazzano-application-bobs-books",
		NumberOfSeconds: "60s"}
	payload, err := formatReindexPayload(input, reindexPayload)
	asrt.NoError(err, "Error not expected")
	asrt.Contains(payload, "verrazzano-namespace-bobs-books")
	asrt.Contains(payload, "verrazzano-application-bobs-books")
	asrt.Contains(payload, "60s")
}

func TestCalculateSeconds(t *testing.T) {
	asrt := assert.New(t)
	_, err := calculateSeconds("ww5s")
	asrt.Error(err, "Error should be returned from exec")
	_, err = calculateSeconds("12y")
	asrt.Error(err, "should fail for 'years'")
	_, err = calculateSeconds("10M")
	asrt.Error(err, "should fail for 'months'")
	seconds, err := calculateSeconds("6d")
	asrt.NoError(err, "Should not fail for valid day unit")
	asrt.Equal(uint64(518400), seconds)
	seconds, err = calculateSeconds("120m")
	asrt.NoError(err, "Should not fail for valid minute unit")
	asrt.Equal(uint64(7200), seconds)
	seconds, err = calculateSeconds("5h")
	asrt.NoError(err, "Should not fail for valid hour unit")
	asrt.Equal(uint64(18000), seconds)
	seconds, err = calculateSeconds("20s")
	asrt.NoError(err, "Should not fail for valid second unit")
	asrt.Equal(uint64(20), seconds)
	seconds, err = calculateSeconds("1w")
	asrt.NoError(err, "Should not fail for valid week unit")
	asrt.Equal(uint64(604800), seconds)
}
