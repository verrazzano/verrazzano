// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	"github.com/stretchr/testify/assert"
	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sort"
	"testing"
)

const (
	testConvertsASingleArg = `foo:
  bar: baz`
	testConvertsAListArg = `foo:
  - bar
  - baz`
	testConvertsMergedArgs = `foo:
  bar: biz`
	testConvertsMultipleNestedArgs = `a:
  b:
    c: hello
    d: world
    e:
      f: nested args`
)

func TestConvertInstallArgsToV1Beta1(t *testing.T) {
	var tests = []struct {
		name       string
		args       []InstallArgs
		mergedYaml string
	}{
		{
			name: "converts a single arg",
			args: []InstallArgs{
				{
					Name:  "foo.bar",
					Value: "baz",
				},
			},
			mergedYaml: testConvertsASingleArg,
		},
		{
			name: "converts a list arg",
			args: []InstallArgs{
				{
					Name: "foo",
					ValueList: []string{
						"bar",
						"baz",
					},
				},
			},
			mergedYaml: testConvertsAListArg,
		},
		{
			name: "converts merged args",
			args: []InstallArgs{
				{
					Name:  "foo.bar",
					Value: "baz",
				},
				{
					Name:  "foo.bar",
					Value: "biz",
				},
			},
			mergedYaml: testConvertsMergedArgs,
		},
		{
			name: "converts multiple nested args",
			args: []InstallArgs{
				{
					Name:  "a.b.c",
					Value: "hello",
				},
				{
					Name:  "a.b.d",
					Value: "world",
				},
				{
					Name:  "a.b.e.f",
					Value: "nested args",
				},
			},
			mergedYaml: testConvertsMultipleNestedArgs,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mergedYaml, err := convertInstallArgsToYaml(tt.args)
			assert.NoError(t, err)
			assert.Equal(t, tt.mergedYaml, mergedYaml)
		})
	}
}

func TestConvertInstallArgsToOSNodes(t *testing.T) {
	storage50Gi := "50Gi"
	storage250Gi := "250Gi"
	replicas3 := "3"
	q2GiString := "2Gi"
	q2Gi, err := resource.ParseQuantity(q2GiString)
	assert.NoError(t, err)
	resourceRequirements := &corev1.ResourceRequirements{
		Requests: map[corev1.ResourceName]resource.Quantity{
			corev1.ResourceMemory: q2Gi,
		},
	}

	var tests = []struct {
		name  string
		args  []InstallArgs
		nodes map[string]v1beta1.OpenSearchNode
	}{
		{
			"single node",
			[]InstallArgs{
				{
					Name:  masterNodeReplicas,
					Value: "1",
				},
			},
			map[string]v1beta1.OpenSearchNode{
				masterNodeName: {
					Replicas: 1,
					Name:     masterNodeName,
					Roles:    []vmov1.NodeRole{vmov1.MasterRole},
				},
				dataNodeName: {
					Replicas: 0,
					Name:     dataNodeName,
					Roles:    []vmov1.NodeRole{vmov1.DataRole},
				},
				ingestNodeName: {
					Replicas: 0,
					Name:     ingestNodeName,
					Roles:    []vmov1.NodeRole{vmov1.IngestRole},
				},
			},
		},
		{
			"multi node with storage",
			[]InstallArgs{
				{
					Name:  masterNodeReplicas,
					Value: replicas3,
				},
				{
					Name:  masterNodeMemory,
					Value: q2GiString,
				},
				{
					Name:  masterNodeStorage,
					Value: storage50Gi,
				},
				{
					Name:  dataNodeReplicas,
					Value: replicas3,
				},
				{
					Name:  dataNodeMemory,
					Value: q2GiString,
				},
				{
					Name:  dataNodeStorage,
					Value: storage250Gi,
				},
				{
					Name:  ingestNodeReplicas,
					Value: "2",
				},
				{
					Name:  ingestNodeMemory,
					Value: q2GiString,
				},
			},
			map[string]v1beta1.OpenSearchNode{
				masterNodeName: {
					Name:      masterNodeName,
					Replicas:  3,
					Resources: resourceRequirements,
					Roles:     []vmov1.NodeRole{vmov1.MasterRole},
					Storage: &v1beta1.OpenSearchNodeStorage{
						Size: storage50Gi,
					},
				},
				dataNodeName: {
					Name:      dataNodeName,
					Replicas:  3,
					Resources: resourceRequirements,
					Roles:     []vmov1.NodeRole{vmov1.DataRole},
					Storage: &v1beta1.OpenSearchNodeStorage{
						Size: storage250Gi,
					},
				},
				ingestNodeName: {
					Name:      ingestNodeName,
					Replicas:  2,
					Resources: resourceRequirements,
					Roles:     []vmov1.NodeRole{vmov1.IngestRole},
				},
			},
		},
		{
			"no replicas no nodes",
			[]InstallArgs{
				{
					Name:  masterNodeName,
					Value: "0",
				},
			},
			map[string]v1beta1.OpenSearchNode{
				masterNodeName: {
					Replicas: 0,
					Name:     masterNodeName,
					Roles:    []vmov1.NodeRole{vmov1.MasterRole},
				},
				dataNodeName: {
					Replicas: 0,
					Name:     dataNodeName,
					Roles:    []vmov1.NodeRole{vmov1.DataRole},
				},
				ingestNodeName: {
					Replicas: 0,
					Name:     ingestNodeName,
					Roles:    []vmov1.NodeRole{vmov1.IngestRole},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := convertInstallArgsToOSNodes(tt.args)
			assert.NoError(t, err)
			assert.EqualValues(t, tt.nodes, nodes)
		})
	}
}

func TestConvertCommonKubernetesToYamls(t *testing.T) {
	affinity := corev1.Affinity{
		PodAntiAffinity: &corev1.PodAntiAffinity{
			PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
				{
					Weight: 100,
					PodAffinityTerm: corev1.PodAffinityTerm{
						TopologyKey: "kubernetes.io/hostname",
						LabelSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"app": "foobar",
							},
						},
					},
				},
			},
		},
	}

	const (
		outputReplicas = `replicas: 1`
		outputAffinity = `affinity: |
  podAntiAffinity:
    preferredDuringSchedulingIgnoredDuringExecution:
    - podAffinityTerm:
        labelSelector:
          matchLabels:
            app: foobar
        topologyKey: kubernetes.io/hostname
      weight: 100
replicas: 1`
	)
	expandReplicas := expandInfo{
		0,
		"replicas",
	}
	expandAffinity := expandInfo{
		0,
		"affinity",
	}

	var tests = []struct {
		name         string
		spec         CommonKubernetesSpec
		replicasInfo expandInfo
		affinityInfo expandInfo
		output       string
	}{
		{
			"converts replicas and affinity",
			CommonKubernetesSpec{
				Replicas: 1,
				Affinity: &affinity,
			},
			expandReplicas,
			expandAffinity,
			outputAffinity,
		},
		{
			"converts only replicas when no affinity",
			CommonKubernetesSpec{
				Replicas: 1,
			},
			expandReplicas,
			expandAffinity,
			outputReplicas,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yamls, err := convertCommonKubernetesToYaml(tt.spec, tt.replicasInfo, tt.affinityInfo)
			assert.NoError(t, err)
			assert.EqualValues(t, tt.output, yamls)
		})
	}
}

func TestConvertToV1Beta1(t *testing.T) {
	var tests = []converisonTestCase{
		{
			"converts from v1alpha1 in the basic case",
			testCaseBasic,
			false,
		},
		{
			"converts from v1alpha1 status",
			testCaseStatus,
			false,
		},
		{
			"converts components that use install args and install overrides",
			testCaseInstallArgs,
			false,
		},
		{
			"convert istio args from v1alpha1",
			testCaseIstioInstallArgs,
			false,
		},
		{
			"convert istio affinity args from v1alpha1",
			testCaseIstioAffinityArgs,
			false,
		},
		{
			"convert all components from 1alpha1",
			testCaseFromAllComps,
			false,
		},
		{
			"convert opensearch from v1alpha1",
			testCaseOpensearch,
			false,
		},
		{
			"convert rancher keycloak auth from v1alpha1",
			testCaseRancherKeycloak,
			false,
		},
		{
			"convert volume claim templates from v1alpha1",
			testCaseVolumeOverrides,
			false,
		},
		{
			"convert istio overrides",
			testCaseIstioOverrides,
			false,
		},
		{
			"convert istio node port",
			testCaseIstioNodePort,
			false,
		},
		{
			"convert err on keycloak install args",
			testCaseInstallArgsErr,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// load the expected v1alpha1 CR for conversion
			v1alpha1CR, err := loadV1Alpha1CR(tt.testCase)
			assert.NoError(t, err)

			// compute the actual v1beta1 CR from the v1alpha1 CR
			v1beta1Actual := &v1beta1.Verrazzano{}
			err = v1alpha1CR.ConvertTo(v1beta1Actual)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// load the expected v1beta1 CR
				v1beta1Expected, err := loadV1Beta1(tt.testCase)
				assert.NoError(t, err)
				// Arrays may not be in order, so we have to sort them by name for reflect.DeepEqual to be true
				if v1beta1Expected.Spec.Components.OpenSearch != nil && v1beta1Actual.Spec.Components.OpenSearch != nil {
					sortNodes(v1beta1Expected.Spec.Components.OpenSearch.Nodes)
					sortNodes(v1beta1Actual.Spec.Components.OpenSearch.Nodes)
				}
				// expected and actual v1beta1 CRs must be equal
				assert.EqualValues(t, v1beta1Expected.ObjectMeta, v1beta1Actual.ObjectMeta)
				assert.EqualValues(t, v1beta1Expected.Spec, v1beta1Actual.Spec)
				assert.EqualValues(t, v1beta1Expected.Status, v1beta1Actual.Status)
			}
		})
	}
}

func sortNodes(nodes []v1beta1.OpenSearchNode) {
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Name < nodes[j].Name
	})
}
