// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package appconfig

import (
	"context"
	"testing"

	oamcore "github.com/crossplane/oam-kubernetes-runtime/apis/core"
	oamv1 "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	asserts "github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testNamespace     = "test-ns"
	testAppConfigName = "test-appconfig"
)

func TestReconcileApplicationConfigurationNotFound(t *testing.T) {
	assert := asserts.New(t)
	oamcore.AddToScheme(k8scheme.Scheme)
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme)

	reconciler := newReconciler(client)
	request := newRequest(testNamespace, testAppConfigName)

	_, err := reconciler.Reconcile(request)
	assert.NoError(err)
}

func TestReconcileNoResetVersion(t *testing.T) {
	assert := asserts.New(t)
	oamcore.AddToScheme(k8scheme.Scheme)
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme)

	reconciler := newReconciler(client)
	request := newRequest(testNamespace, testAppConfigName)

	err := client.Create(context.TODO(), newAppConfig())
	assert.NoError(err)

	_, err = reconciler.Reconcile(request)
	assert.NoError(err)
}

func TestReconcileNoPreviousResetVersion(t *testing.T) {
	assert := asserts.New(t)
	oamcore.AddToScheme(k8scheme.Scheme)
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme)

	reconciler := newReconciler(client)
	request := newRequest(testNamespace, testAppConfigName)

	appConfig := newAppConfig()
	appConfig.Annotations[restartVersionAnnotation] = "1"
	err := client.Create(context.TODO(), appConfig)
	assert.NoError(err)

	_, err = reconciler.Reconcile(request)
	assert.NoError(err)

	err = client.Get(context.TODO(), request.NamespacedName, appConfig)
	assert.NoError(err)
	assert.Equal("1", appConfig.Annotations[previousRestartVersionAnnotation])
}

func TestReconcileVersionsMismatch(t *testing.T) {
	assert := asserts.New(t)
	oamcore.AddToScheme(k8scheme.Scheme)
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme)

	reconciler := newReconciler(client)
	request := newRequest(testNamespace, testAppConfigName)

	appConfig := newAppConfig()
	appConfig.Annotations[restartVersionAnnotation] = "2"
	appConfig.Annotations[previousRestartVersionAnnotation] = "1"
	err := client.Create(context.TODO(), appConfig)
	assert.NoError(err)

	_, err = reconciler.Reconcile(request)
	assert.NoError(err)

	err = client.Get(context.TODO(), request.NamespacedName, appConfig)
	assert.NoError(err)
	assert.Equal("2", appConfig.Annotations[previousRestartVersionAnnotation])
}

// newScheme creates a new scheme that includes this package's object to use for testing
func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	oamcore.AddToScheme(scheme)
	return scheme
}

// newReconciler creates a new reconciler for testing
func newReconciler(c client.Client) Reconciler {
	return Reconciler{
		Client: c,
		Log:    ctrl.Log.WithName("test"),
		Scheme: newScheme(),
	}
}

// newRequest creates a new reconciler request for testing
func newRequest(namespace string, name string) ctrl.Request {
	return ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		},
	}
}

// newAppConfig creates a minimal ApplicationConfiguration struct
func newAppConfig() *oamv1.ApplicationConfiguration {
	return &oamv1.ApplicationConfiguration{
		ObjectMeta: v1.ObjectMeta{
			Name:        testAppConfigName,
			Namespace:   testNamespace,
			Annotations: make(map[string]string),
		},
	}
}
