// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"context"
	"github.com/golang/mock/gomock"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/mocks"
	"k8s.io/apimachinery/pkg/types"
	cli "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRetainPersistentVolume(t *testing.T) {
	mock := gomock.NewController(t)

	client := mocks.NewMockClient(mock)
	pvc := v1.PersistentVolumeClaim{}
	ctx := spi.NewFakeContext(client, &v1alpha1.Verrazzano{ObjectMeta: v12.ObjectMeta{Namespace: "foo"}}, nil, false)
	client.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).Return(nil).AnyTimes()
	client.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	err := RetainPersistentVolume(ctx, &pvc, "pvc")
	assert.Nil(t, err)

}
func TestDeleteExistingVolumeClaim(t *testing.T) {
	mock := gomock.NewController(t)

	client := mocks.NewMockClient(mock)
	ctx := spi.NewFakeContext(client, &v1alpha1.Verrazzano{ObjectMeta: v12.ObjectMeta{Namespace: "foo"}}, nil, false)
	client.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).Return(nil).AnyTimes()
	client.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	client.EXPECT().Delete(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).Return(nil).AnyTimes()

	pvc := types.NamespacedName{"pvc", "pvc1"}
	err := DeleteExistingVolumeClaim(ctx, pvc)
	assert.Nil(t, err)

}
func TestUpdateExistingVolumeClaims(t *testing.T) {
	mock := gomock.NewController(t)

	client := mocks.NewMockClient(mock)
	ctx := spi.NewFakeContext(client, &v1alpha1.Verrazzano{ObjectMeta: v12.ObjectMeta{Namespace: "foo"}}, nil, false)
	client.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).Return(nil).AnyTimes()
	client.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	client.EXPECT().Delete(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).Return(nil).AnyTimes()
	client.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).DoAndReturn(func(ctx context.Context, list *v1.PersistentVolumeList, opts ...cli.ListOption) error {
		pv := v1.PersistentVolume{
			ObjectMeta: v12.ObjectMeta{
				Name:   "volumeName",
				Labels: map[string]string{vzconst.OldReclaimPolicyLabel: string(v1.PersistentVolumeReclaimDelete)},
			},
			Spec: v1.PersistentVolumeSpec{
				ClaimRef: &v1.ObjectReference{
					Namespace: "pvc",
					Name:      "pvc1",
				},
				PersistentVolumeReclaimPolicy: v1.PersistentVolumeReclaimRetain,
			},
			Status: v1.PersistentVolumeStatus{
				Phase: v1.VolumeReleased,
			},
		}
		list.Items = []v1.PersistentVolume{pv}
		return nil
	})

	pvc := types.NamespacedName{"pvc", "pvc1"}
	err := UpdateExistingVolumeClaims(ctx, pvc, "test", "test1")
	assert.Nil(t, err)

}

func TestResetVolumeReclaimPolicy(t *testing.T) {
	mock := gomock.NewController(t)

	client := mocks.NewMockClient(mock)

	ctx := spi.NewFakeContext(client, &v1alpha1.Verrazzano{ObjectMeta: v12.ObjectMeta{Namespace: "foo"}}, nil, false)
	client.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).Return(nil).AnyTimes()
	client.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	client.EXPECT().Delete(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).Return(nil).AnyTimes()
	client.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).DoAndReturn(func(ctx context.Context, list *v1.PersistentVolumeList, opts ...cli.ListOption) error {
		pv := v1.PersistentVolume{
			ObjectMeta: v12.ObjectMeta{
				Name:   "volumeName",
				Labels: map[string]string{vzconst.OldReclaimPolicyLabel: string(v1.PersistentVolumeReclaimDelete)},
			},
			Spec: v1.PersistentVolumeSpec{
				ClaimRef: &v1.ObjectReference{
					Namespace: "ComponentNamespace",
					Name:      "DeploymentPersistentVolumeClaim",
				},
				PersistentVolumeReclaimPolicy: v1.PersistentVolumeReclaimRetain,
			},
			Status: v1.PersistentVolumeStatus{
				Phase: v1.VolumeBound,
			},
		}
		list.Items = []v1.PersistentVolume{pv}
		return nil
	})
	err := ResetVolumeReclaimPolicy(ctx, "pvc")
	assert.Nil(t, err)

}
