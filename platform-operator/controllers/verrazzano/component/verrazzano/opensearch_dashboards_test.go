// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"github.com/stretchr/testify/assert"
	globalconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	k8sutilfake "github.com/verrazzano/verrazzano/pkg/k8sutil/fake"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

const fakeGetPatternOutput = `{
  "page": 1,
  "per_page": 20,
  "total": 2,
  "saved_objects": [
    {
      "type": "index-pattern",
      "id": "0f2ede70-8e15-11ec-abc1-6bc5e972b077",
      "attributes": {
        "title": "verrazzano-namespace-bobs-books"
      },
      "references": [],
      "migrationVersion": {
        "index-pattern": "7.6.0"
      },
      "updated_at": "2022-02-15T04:09:24.182Z",
      "version": "WzQsMV0=",
      "namespaces": [
        "default"
      ],
      "score": 0
    },
    {
      "type": "index-pattern",
      "id": "1cb7fcc0-8e15-11ec-abc1-6bc5e972b077",
      "attributes": {
        "title": "verrazzano-namespace-todo*"
      },
      "references": [],
      "migrationVersion": {
        "index-pattern": "7.6.0"
      },
      "updated_at": "2022-02-15T04:09:46.892Z",
      "version": "WzksMV0=",
      "namespaces": [
        "default"
      ],
      "score": 0
    }
  ]
}`

func readyOpenSearchDashboardsPods() *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod",
			Namespace: globalconst.VerrazzanoSystemNamespace,
			Labels: map[string]string{
				"app": OSDashboardSystemName,
			},
		},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:  OSDashboardContainerName,
					Ready: true,
				},
			},
		},
	}
}

// TestGetPatterns tests the getPatterns function.
func TestGetPatterns(t *testing.T) {
	asrt := assert.New(t)

	// GIVEN an OpenSearch Dashboards pod
	//  WHEN getPatterns is called
	//  THEN a command should be executed to get the index pattern information
	//   AND then a map of index pattern id and title should be returned
	k8sutil.NewPodExecutor = k8sutilfake.NewPodExecutor
	k8sutilfake.PodSTDOUT = fakeGetPatternOutput
	k8sutil.ClientConfig = func() (*rest.Config, kubernetes.Interface, error) {
		config2, k := k8sutilfake.NewClientsetConfig()
		return config2, k, nil
	}
	pod := readyOpenSearchDashboardsPods()
	client2 := fake.NewFakeClientWithScheme(testScheme, pod)
	ctx := spi.NewFakeContext(client2, &vzapi.Verrazzano{}, false)
	cfg, cli, err := k8sutil.ClientConfig()
	asrt.NoError(err, "Error not expected")
	patterns, err := getPatterns(ctx.Log(), cfg, cli, pod)
	asrt.NoError(err, "Failed to get patterns from OpenSearch Dashboards")
	expectedValue := map[string]string{"0f2ede70-8e15-11ec-abc1-6bc5e972b077": "verrazzano-namespace-bobs-books",
		"1cb7fcc0-8e15-11ec-abc1-6bc5e972b077": "verrazzano-namespace-todo*"}
	asrt.Equal(patterns, expectedValue)
}

// TestConstructUpdatedPattern tests the constructUpdatedPattern function.
func TestConstructUpdatedPattern(t *testing.T) {
	asrt := assert.New(t)
	pattern := constructUpdatedPattern("verrazzano-*")
	asrt.Equal("verrazzano-*", pattern)
	pattern = constructUpdatedPattern("verrazzano-namespace-bobs-books")
	asrt.Equal("verrazzano-application-bobs-books", pattern)
	pattern = constructUpdatedPattern("verrazzano-systemd-journal")
	asrt.Equal("verrazzano-system", pattern)
	pattern = constructUpdatedPattern("verrazzano-namespace-kube-system")
	asrt.Equal("verrazzano-system", pattern)
	pattern = constructUpdatedPattern("verrazzano-namespace-todo-*")
	asrt.Equal("verrazzano-application-todo-*", pattern)
	pattern = constructUpdatedPattern("verrazzano-namespace-s*,verrazzano-namespace-bobs-books")
	asrt.Equal("verrazzano-application-s*,verrazzano-application-bobs-books", pattern)
	pattern = constructUpdatedPattern("verrazzano-namespace-k*,verrazzano-namespace-sock-shop")
	// As verrazzano-namespace-k* matches system index verrazzano-namespace-kube-system,
	// system data stream name should also be added
	asrt.Equal("verrazzano-system,verrazzano-application-k*,verrazzano-application-sock-shop", pattern)
}
