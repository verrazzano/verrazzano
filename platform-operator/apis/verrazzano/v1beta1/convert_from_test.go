// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1beta1

import (
	"github.com/stretchr/testify/assert"
	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
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

func TestConvertInstallArgsFrom(t *testing.T) {
	var tests = []struct {
		name       string
		args       []v1alpha1.InstallArgs
		mergedYaml string
	}{
		{
			name: "converts a single arg",
			args: []v1alpha1.InstallArgs{
				{
					Name:  "foo.bar",
					Value: "baz",
				},
			},
			mergedYaml: testConvertsASingleArg,
		},
		{
			name: "converts a list arg",
			args: []v1alpha1.InstallArgs{
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
			args: []v1alpha1.InstallArgs{
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
			args: []v1alpha1.InstallArgs{
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
		args  []v1alpha1.InstallArgs
		nodes []OpenSearchNode
	}{
		{
			"single node",
			[]v1alpha1.InstallArgs{
				{
					Name:  masterNodeReplicas,
					Value: "1",
				},
			},
			[]OpenSearchNode{
				{
					Replicas: 1,
					Name:     masterNodeName,
					Roles:    []vmov1.NodeRole{vmov1.MasterRole},
				},
			},
		},
		{
			"multi node with storage",
			[]v1alpha1.InstallArgs{
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
			[]OpenSearchNode{
				{
					Name:      masterNodeName,
					Replicas:  3,
					Resources: resourceRequirements,
					Roles:     []vmov1.NodeRole{vmov1.MasterRole},
					Storage: &OpenSearchNodeStorage{
						Size: storage50Gi,
					},
				},
				{
					Name:      dataNodeName,
					Replicas:  3,
					Resources: resourceRequirements,
					Roles:     []vmov1.NodeRole{vmov1.DataRole},
					Storage: &OpenSearchNodeStorage{
						Size: storage250Gi,
					},
				},
				{
					Name:      ingestNodeName,
					Replicas:  2,
					Resources: resourceRequirements,
					Roles:     []vmov1.NodeRole{vmov1.IngestRole},
				},
			},
		},
		{
			"no replicas no nodes",
			[]v1alpha1.InstallArgs{
				{
					Name:  masterNodeName,
					Value: "0",
				},
			},
			nil,
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
