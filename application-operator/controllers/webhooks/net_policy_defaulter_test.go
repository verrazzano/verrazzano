// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	"testing"

	oamv1 "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testNamespace = "unit-test-ns"
	appConfigName = "unit-test-app-config"
)

// erroringFakeClient is a client wrapper that allows us to simulate a conflict error on update
type erroringFakeClient struct {
	client.Client
	conflictReturned bool
}

// Update returns a conflict error one time, and then passes through to the wrapped client on subsequent calls.
// This allows us to test retries when updating the namespace with a label.
func (e *erroringFakeClient) Update(ctx context.Context, obj runtime.Object, opts ...client.UpdateOption) error {
	if !e.conflictReturned {
		e.conflictReturned = true
		return errors.NewConflict(schema.GroupResource{}, "", nil)
	}
	return e.Client.Update(ctx, obj, opts...)
}

// GIVEN an app config is being created
// WHEN the network policy defaulter Default function is called
// THEN the network policy defaulter labels the app namespace and creates a network policy in the Istio system namespace
func TestDefaultNetworkPolicy(t *testing.T) {
	appConfig := &oamv1.ApplicationConfiguration{ObjectMeta: metav1.ObjectMeta{Namespace: testNamespace, Name: appConfigName}}
	fakeClient := newFakeClient()
	defaulter := &NetPolicyDefaulter{Client: fakeClient}

	// create the test namespace so the defaulter can add a label to it
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace}}
	fakeClient.Create(context.TODO(), ns, &client.CreateOptions{})

	// this is the function under test, we expect this to label the namespace and create the network policy
	err := defaulter.Default(appConfig, false)
	assert.NoError(t, err, "Unexpected error creating network policy")

	// assert that the app namespace was labeled
	assertNamespaceLabeled(t, fakeClient)

	// assert that the network policy was created and the spec has the expected data
	assertNetworkPolicy(t, fakeClient)
}

// GIVEN an app config is being created
// WHEN the network policy defaulter Default function is called
// AND there is a conflict updating the app namespace
// THEN the network policy defaulter retries adding the label to the namespace, succeeds, and creates a network policy
func TestRetryLabelNamespace(t *testing.T) {
	appConfig := &oamv1.ApplicationConfiguration{ObjectMeta: metav1.ObjectMeta{Namespace: testNamespace, Name: appConfigName}}
	fakeClient := newFakeClient()
	errFakeClient := &erroringFakeClient{Client: fakeClient, conflictReturned: false}
	defaulter := &NetPolicyDefaulter{Client: errFakeClient}

	// create the test namespace so the defaulter can add a label to it
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace}}
	fakeClient.Create(context.TODO(), ns, &client.CreateOptions{})

	// this is the function under test, we expect this to label the namespace and create the network policy
	err := defaulter.Default(appConfig, false)
	assert.NoError(t, err, "Unexpected error creating network policy")

	// assert that the erroring fake client returned a conflict error
	assert.True(t, errFakeClient.conflictReturned)

	// assert that the app namespace was labeled
	assertNamespaceLabeled(t, fakeClient)

	// assert that the network policy was created and the spec has the expected data
	assertNetworkPolicy(t, fakeClient)
}

// GIVEN an app config is being deleted
// WHEN the network policy defaulter Cleanup function is called
// THEN the network policy defaulter deletes the network policy from the Istio system namespace
func TestDeleteNetworkPolicy(t *testing.T) {
	// create the app config with a non-nil deletion timestamp
	appConfig := &oamv1.ApplicationConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:         testNamespace,
			Name:              appConfigName,
			DeletionTimestamp: &metav1.Time{},
		},
	}
	fakeClient := newFakeClient()
	defaulter := &NetPolicyDefaulter{Client: fakeClient}

	// create a network policy so the defaulter can delete it
	netPol := newNetworkPolicy(appConfig)
	err := fakeClient.Create(context.TODO(), &netPol, &client.CreateOptions{})
	assert.NoError(t, err, "Unexpected error creating network policy")

	// this is the function under test, we expect this to delete the network policy
	err = defaulter.Cleanup(appConfig, false)
	assert.NoError(t, err, "Unexpected error deleting network policy")

	// assert that the network policy no longer exists
	assertNoNetworkPolicy(t, fakeClient)
}

// GIVEN an app config is being deleted
// WHEN the network policy defaulter Default function is called
// THEN the network policy defaulter does nothing
func TestAppConfigInDelete(t *testing.T) {
	// create the app config with a non-nil deletion timestamp
	appConfig := &oamv1.ApplicationConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:         testNamespace,
			Name:              appConfigName,
			DeletionTimestamp: &metav1.Time{},
		},
	}
	fakeClient := newFakeClient()
	defaulter := &NetPolicyDefaulter{Client: fakeClient}

	// this is the function under test, since the app config is being deleted, we expect
	// no resources to be created or updated
	err := defaulter.Default(appConfig, false)
	assert.NoError(t, err)

	// assert that the network policy did not get created
	assertNoNetworkPolicy(t, fakeClient)
}

// newFakeClient returns a new fake client
func newFakeClient() client.Client {
	scheme := runtime.NewScheme()
	corev1.AddToScheme(scheme)
	netv1.AddToScheme(scheme)
	return ctrlfake.NewFakeClientWithScheme(scheme)
}

// fetchNetworkPolicy fetches the network policy using the provided client
func fetchNetworkPolicy(t *testing.T, client client.Client) (*netv1.NetworkPolicy, error) {
	netPolicyName := testNamespace + "-" + appConfigName
	var netPolicy netv1.NetworkPolicy
	namespacedName := types.NamespacedName{Namespace: constants.IstioSystemNamespace, Name: netPolicyName}

	err := client.Get(context.TODO(), namespacedName, &netPolicy)

	return &netPolicy, err
}

// assertNetworkPolicy asserts that the network policy exists and that the spec contains the expected data.
func assertNetworkPolicy(t *testing.T, client client.Client) {
	netPolicy, err := fetchNetworkPolicy(t, client)

	assert.NoError(t, err, "Unexpected error fetching network policy")
	assert.Equal(t, newNetworkPolicySpec(testNamespace), netPolicy.Spec)
}

// assertNoNetworkPolicy asserts that the network policy does not exist
func assertNoNetworkPolicy(t *testing.T, client client.Client) {
	_, err := fetchNetworkPolicy(t, client)

	assert.True(t, errors.IsNotFound(err), "Expected to get NotFound error")
}

// assertNamespaceLabeled asserts that the namespace has been labeled with the Verrazzano namespace label.
func assertNamespaceLabeled(t *testing.T, client client.Client) {
	var ns corev1.Namespace
	namespacedName := types.NamespacedName{Name: testNamespace}

	err := client.Get(context.TODO(), namespacedName, &ns)

	assert.NoError(t, err, "Unexpected error fetching namespace")
	assert.Equal(t, testNamespace, ns.Labels[constants.LabelVerrazzanoNamespace])
}
