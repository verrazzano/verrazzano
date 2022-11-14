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

// TestRetainPersistentVolume tests the RetainPersistentVolume function
// GIVEN a component context, pvc to retain and component name
// WHEN  the RetainPersistentVolume function is called
// THEN  the function call succeeds with nil error
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

// TestDeleteExistingVolumeClaim tests the DeleteExistingVolumeClaim function
// GIVEN a component context and a pvc to delete
// WHEN  the DeleteExistingVolumeClaim function is called
// THEN  the function call succeeds with nil error
func TestDeleteExistingVolumeClaim(t *testing.T) {
	mock := gomock.NewController(t)
	client := mocks.NewMockClient(mock)
	ctx := spi.NewFakeContext(client, &v1alpha1.Verrazzano{ObjectMeta: v12.ObjectMeta{Namespace: "foo"}}, nil, false)
	client.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).Return(nil).AnyTimes()
	client.EXPECT().Delete(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).Return(nil).AnyTimes()

	pvc := types.NamespacedName{Namespace: "foo", Name: "foo1"}
	err := DeleteExistingVolumeClaim(ctx, pvc)
	assert.Nil(t, err)
}

// TestUpdateExistingVolumeClaims tests the UpdateExistingVolumeClaims function
// GIVEN a component context, existing pvc, new claim name and component name
// WHEN  the UpdateExistingVolumeClaims function is called
// THEN  the function call succeeds with a nil error
func TestUpdateExistingVolumeClaims(t *testing.T) {
	mock := gomock.NewController(t)
	client := mocks.NewMockClient(mock)
	ctx := spi.NewFakeContext(client, &v1alpha1.Verrazzano{ObjectMeta: v12.ObjectMeta{Namespace: "foo"}}, nil, false)
	client.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).Return(nil).AnyTimes()
	client.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	client.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).DoAndReturn(func(ctx context.Context, list *v1.PersistentVolumeList, opts ...cli.ListOption) error {
		pv := v1.PersistentVolume{
			ObjectMeta: v12.ObjectMeta{
				Name:   "testVol",
				Labels: map[string]string{vzconst.OldReclaimPolicyLabel: string(v1.PersistentVolumeReclaimDelete)},
			},
			Spec: v1.PersistentVolumeSpec{
				ClaimRef: &v1.ObjectReference{
					Namespace: "foo",
					Name:      "bar",
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

	pvc := types.NamespacedName{Namespace: "foo", Name: "bar"}
	err := UpdateExistingVolumeClaims(ctx, pvc, "testclaim", "testcomp")
	assert.Nil(t, err)
}

// TestResetVolumeReclaimPolicy tests the ResetVolumeReclaimPolicy function
// GIVEN a component context and component name
// WHEN  the ResetVolumeReclaimPolicy function is called
// THEN  the function call succeeds with nil error and VolumeReclaimPolicy is reset successfully.
func TestResetVolumeReclaimPolicy(t *testing.T) {
	mock := gomock.NewController(t)
	client := mocks.NewMockClient(mock)
	ctx := spi.NewFakeContext(client, &v1alpha1.Verrazzano{ObjectMeta: v12.ObjectMeta{Namespace: "foo"}}, nil, false)
	client.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	client.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).DoAndReturn(func(ctx context.Context, list *v1.PersistentVolumeList, opts ...cli.ListOption) error {
		pv := v1.PersistentVolume{
			ObjectMeta: v12.ObjectMeta{
				Name:   "testVol",
				Labels: map[string]string{vzconst.OldReclaimPolicyLabel: string(v1.PersistentVolumeReclaimDelete)},
			},
			Spec: v1.PersistentVolumeSpec{
				ClaimRef: &v1.ObjectReference{
					Namespace: "foo",
					Name:      "bar",
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
	err := ResetVolumeReclaimPolicy(ctx, "testcomp")
	assert.Nil(t, err)
}
