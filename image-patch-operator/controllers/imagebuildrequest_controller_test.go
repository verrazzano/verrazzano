// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package controllers

import (
	"testing"

	"github.com/golang/mock/gomock"
	asserts "github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/image-patch-operator/mocks"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"k8s.io/apimachinery/pkg/runtime"

	ipoapi "github.com/verrazzano/verrazzano/image-patch-operator/api/images/v1alpha1"
)

func TestReconcilerSetupWithManager(t *testing.T) {
	assert := asserts.New(t)

	var mocker *gomock.Controller
	var mgr *mocks.MockManager
	var cli *mocks.MockClient
	var scheme *runtime.Scheme
	var reconciler ImageBuildRequestReconciler
	var err error

	mocker = gomock.NewController(t)
	mgr = mocks.NewMockManager(mocker)
	cli = mocks.NewMockClient(mocker)
	scheme = runtime.NewScheme()
	ipoapi.AddToScheme(scheme)
	reconciler = ImageBuildRequestReconciler{Client: cli, Scheme: scheme}
	mgr.EXPECT().GetConfig().Return(&rest.Config{})
	mgr.EXPECT().GetScheme().Return(scheme)
	mgr.EXPECT().GetLogger().Return(log.NullLogger{})
	mgr.EXPECT().SetFields(gomock.Any()).Return(nil).AnyTimes()
	mgr.EXPECT().Add(gomock.Any()).Return(nil).AnyTimes()
	err = reconciler.SetupWithManager(mgr)
	mocker.Finish()
	assert.NoError(err)
}
