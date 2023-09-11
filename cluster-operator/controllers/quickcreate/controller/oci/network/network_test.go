// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package network_test

import (
	"context"
	_ "embed"
	"github.com/stretchr/testify/assert"
	vmcv1alpha1 "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/cluster-operator/controllers/quickcreate/controller/oci/network"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	apiyaml "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

var (
	scheme *runtime.Scheme
	//go:embed testdata/ocicluster.yaml
	ocicluster []byte
	testName   = "test"
)

func init() {
	scheme = runtime.NewScheme()
}

func NewOCICluster() runtime.Object {
	j, _ := apiyaml.ToJSON(ocicluster)
	obj, _ := runtime.Decode(unstructured.UnstructuredJSONScheme, j)
	return obj
}

func TestLoadNetwork(t *testing.T) {
	cluster := &v1beta1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testName,
		},
	}
	obj := NewOCICluster()

	cli := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(obj).Build()
	net, err := network.GetNetwork(context.TODO(), cli, cluster, network.GVKOCICluster)
	assert.NoError(t, err)
	assert.NotNil(t, net)
	assert.Len(t, net.Subnets, 3)
	assert.Equal(t, net.VCN, testName)
}

func TestGetLoadBalancerSubnet(t *testing.T) {
	var tests = []struct {
		n   *vmcv1alpha1.Network
		res string
	}{
		{
			nil,
			"",
		},
		{
			&vmcv1alpha1.Network{
				Subnets: []vmcv1alpha1.Subnet{
					{
						Role: vmcv1alpha1.SubnetRoleServiceLB,
						ID:   "foo",
					},
					{
						Role: vmcv1alpha1.SubnetRoleWorker,
						ID:   "foo",
					},
				},
			},
			"foo",
		},
		{
			&vmcv1alpha1.Network{
				Subnets: []vmcv1alpha1.Subnet{
					{
						Role: vmcv1alpha1.SubnetRoleWorker,
						ID:   "foo",
					},
				},
			},
			"",
		},
	}

	for _, tt := range tests {
		res := network.GetLoadBalancerSubnet(tt.n)
		assert.Equal(t, tt.res, res)
	}
}
