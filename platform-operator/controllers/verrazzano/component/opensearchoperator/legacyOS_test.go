// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearchoperator

import (
	"context"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/mocks"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
)

// TestGetNodeNameFromClaimName tests the getNodeNameFromClaimName function
// GIVEN a list of OpenSearch nodes and claim names
// WHEN getNodeNameFromClaimName is called
// THEN expected node name is returned for each claim name
func TestGetNodeNameFromClaimName(t *testing.T) {
	var tests = []struct {
		nodes            []vzapi.OpenSearchNode
		claimNames       []string
		expectedNodeName []string
	}{
		{
			[]vzapi.OpenSearchNode{{Name: "es-data"}, {Name: "es-data1"}},
			[]string{"vmi-system-es-data", "vmi-system-es-data1-1", "vmi-system-es-data-tqxkq", "vmi-system-es-data1-1-8m66v"},
			[]string{"es-data", "es-data1", "es-data", "es-data1"},
		},
		{
			[]vzapi.OpenSearchNode{{Name: "es-data"}, {Name: "es-data1"}},
			[]string{"vmi-system-es-data1", "vmi-system-es-data-1", "vmi-system-es-data1-tqxkq", "vmi-system-es-data-1-8m66v"},
			[]string{"es-data1", "es-data", "es-data1", "es-data"},
		},
		{
			[]vzapi.OpenSearchNode{{Name: "es-master"}},
			[]string{"elasticsearch-master-vmi-system-es-master-0"},
			[]string{"es-master"},
		},
	}

	for _, tt := range tests {
		for i := range tt.claimNames {
			nodePool := getNodeNameFromClaimName(tt.claimNames[i], tt.nodes)
			assert.Equal(t, tt.expectedNodeName[i], nodePool)
		}
	}
}

func TestHandleLegacyOpenSearch(t *testing.T) {
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	pvList := getFakePersistentVolumeList()

	// set PVs to retain
	mock.EXPECT().
		List(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *v1.PersistentVolumeList, opts ...client.ListOption) error {
			list.Items = pvList.Items
			return nil
		})
	mock.EXPECT().
		Update(gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, pv *v1.PersistentVolume, opts ...client.UpdateOption) error {
			assert.Equal(t, 3, len(pv.Labels))
			assert.Equal(t, v1.PersistentVolumeReclaimRetain, pv.Spec.PersistentVolumeReclaimPolicy)
			return nil
		}).Times(4)

	// set OS and OSD to disabled in VMI
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: "system"}, gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, vmi *vmov1.VerrazzanoMonitoringInstance, opts ...client.GetOption) error {
			vmi.Name = name.Name
			vmi.Namespace = name.Namespace
			return nil
		})
	mock.EXPECT().
		Update(gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, vmi *vmov1.VerrazzanoMonitoringInstance, opts ...client.UpdateOption) error {
			assert.False(t, vmi.Spec.Opensearch.Enabled)
			assert.False(t, vmi.Spec.OpensearchDashboards.Enabled)
			return nil
		})

	// delete master node PVC
	mock.EXPECT().
		List(gomock.Any(), gomock.Any()).
		Return(nil)

	// are PVs released check
	mock.EXPECT().
		List(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *v1.PersistentVolumeList, opts ...client.ListOption) error {
			list.Items = pvList.Items
			return nil
		})

	// create new PVCs
	mock.EXPECT().
		List(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *v1.PersistentVolumeList, opts ...client.ListOption) error {
			list.Items = pvList.Items
			return nil
		})
	mock.EXPECT().
		List(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)
	mock.EXPECT().
		Update(gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, pv *v1.PersistentVolume, opts ...client.UpdateOption) error {
			assert.Nil(t, pv.Spec.ClaimRef)
			return nil
		}).Times(4)
	mock.EXPECT().
		Get(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		Return(errors.NewNotFound(schema.GroupResource{}, "Unable to fetch resource")).Times(4)
	mock.EXPECT().Create(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).Return(nil).Times(4)

	mock.EXPECT().
		List(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).Times(4)

	// check is PVs and PVCs are bound
	mock.EXPECT().
		List(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).Times(2)

	// reset reclaim policy
	mock.EXPECT().
		List(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).Times(3)

	fakeCtx := spi.NewFakeContext(mock, getVZ(), nil, false)
	err := handleLegacyOpenSearch(fakeCtx)
	assert.NoError(t, err)
}

func getVZ() *vzapi.Verrazzano {
	falseValue := false
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Ingress: &vzapi.IngressNginxComponent{Enabled: &falseValue},
				Elasticsearch: &vzapi.ElasticsearchComponent{
					Nodes: []vzapi.OpenSearchNode{{Name: "es-master"}, {Name: "es-data"}, {Name: "data-ingest"}},
				},
			},
		},
	}
	return vz
}
