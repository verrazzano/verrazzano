// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricstrait

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/mocks"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestOperatorReconcile(t *testing.T) {
	tests := []struct {
		name        string
		trait       *vzapi.MetricsTrait
		expectError bool
		requeue     bool
	}{
		{
			name:        "Test reconcile empty trait",
			trait:       &vzapi.MetricsTrait{},
			expectError: false,
			requeue:     false,
		},
	}
	var r Reconciler
	mock := getNewMock(t)
	mock.EXPECT().Update(gomock.Any(), gomock.Not(gomock.Nil()))
	mock.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).Times(2)
	r.Client = mock
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := r.doOperatorReconcile(context.TODO(), tt.trait, vzlog.DefaultLogger())
			if tt.expectError {
				assert.Error(t, err, "Expected an error from the reconcile")
			} else {
				assert.NoError(t, err, err, "Expected no error from the reconcile")
			}
			assert.Equal(t, reconcile.Result{Requeue: tt.requeue}, result)
		})
	}
}

func getNewMock(t *testing.T) *mocks.MockClient {
	mocker := gomock.NewController(t)
	return mocks.NewMockClient(mocker)
}
