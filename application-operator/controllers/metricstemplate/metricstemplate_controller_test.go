// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricstemplate

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	asserts "github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/application-operator/mocks"
	k8sapps "k8s.io/api/apps/v1"
	k8score "k8s.io/api/core/v1"
	k8net "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// TestReconcilerSetupWithManager test the creation of the metrics trait reconciler.
// GIVEN a controller implementation
// WHEN the controller is created
// THEN verify no error is returned
func TestReconcilerSetupWithManager(t *testing.T) {
	assert := asserts.New(t)

	scheme := newScheme()
	client := fake.NewFakeClientWithScheme(scheme)
	reconciler := Reconciler{Client: client, Scheme: scheme, Scraper: "istio-system/prometheus"}

	mocker := gomock.NewController(t)
	manager := mocks.NewMockManager(mocker)
	manager.EXPECT().GetConfig().Return(&rest.Config{}).AnyTimes()
	manager.EXPECT().GetScheme().Return(scheme).AnyTimes()
	manager.EXPECT().GetLogger().Return(log.NullLogger{}).AnyTimes()
	manager.EXPECT().SetFields(gomock.Any()).Return(nil).AnyTimes()
	manager.EXPECT().Add(gomock.Any()).Return(nil).AnyTimes()

	err := reconciler.SetupWithManager(manager)
	assert.NoError(err, "Expected no error when setting up reconciler")
	mocker.Finish()
}

// TestUpdateDeployment tests ...
func TestUpdateDeployment(t *testing.T) {
	assert := asserts.New(t)

	// Create and make the request
	request := ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "test-namespace", Name: "test-deployment-name"}}
	client := fake.NewFakeClientWithScheme(newScheme())
	reconciler := newReconciler(client)
	result, err := reconciler.Reconcile(request)

	// Validate the results
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
	assert.Equal(time.Duration(0), result.RequeueAfter)
}

// newScheme creates a new scheme that includes this package's object to use for testing
func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	//_ = clientgoscheme.AddToScheme(scheme)
	k8sapps.AddToScheme(scheme)
	//	vzapi.AddToScheme(scheme)
	k8score.AddToScheme(scheme)
	//	certapiv1alpha2.AddToScheme(scheme)
	k8net.AddToScheme(scheme)
	return scheme
}

// newReconciler creates a new reconciler for testing
// c - The Kerberos client to inject into the reconciler
func newReconciler(c client.Client) Reconciler {
	log := ctrl.Log.WithName("test")
	scheme := newScheme()
	reconciler := Reconciler{
		Client: c,
		Log:    log,
		Scheme: scheme}
	return reconciler
}
