// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package namespace

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	globalconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/platform-operator/mocks"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	testScheme = runtime.NewScheme()

	istioLabels = map[string]string{
		globalconst.LabelIstioInjection: "enabled",
	}
)

// createVZAndIstioLabels - create an expected map with both the VZ and Istio labels
func createVZAndIstioLabels(ns string) map[string]string {
	return map[string]string{
		globalconst.LabelVerrazzanoNamespace: ns,
		globalconst.LabelIstioInjection:      "enabled",
	}
}

// createVzLabels - create a map with only the VZ-managed label
func createVzLabels(ns string) map[string]string {
	return map[string]string{
		globalconst.LabelVerrazzanoNamespace: ns,
	}
}

func init() {
	_ = clientgoscheme.AddToScheme(testScheme)
	// +kubebuilder:scaffold:testScheme
}

// TestCreateAndLabelNamespace tests the CreateAndLabelNamespace method for the following use case
// GIVEN a call to CreateAndLabelNamespace
// WHEN both the Verrazzano-managed and Istio injection label is specified
// THEN no error is returned and the labels are added to the namepace
func TestCreateAndLabelNamespace(t *testing.T) {
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Name: "testns"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, ns *corev1.Namespace) error {
			return errors.NewNotFound(schema.ParseGroupResource("Namespace"), "testns")
		})

	mock.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, ns *corev1.Namespace, opts ...client.CreateOption) error {
			asserts.Equal("testns", ns.Name)
			asserts.Equal(createVZAndIstioLabels("testns"), ns.Labels)
			return nil
		})

	asserts.NoError(CreateAndLabelNamespace(mock, "testns", true, true))
}

// TestCreateAndLabelNamespace tests the CreateAndLabelNamespace method for the following use case
// GIVEN a call to CreateAndLabelNamespace
// WHEN only the Verrazzano-managed injection label is specified
// THEN no error is returned and only the requested the labels are added to the namepace
func TestCreateAndLabelNamespaceIstioInjection(t *testing.T) {
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Name: "testns"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, ns *corev1.Namespace) error {
			return errors.NewNotFound(schema.ParseGroupResource("Namespace"), "testns")
		})

	mock.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, ns *corev1.Namespace, opts ...client.CreateOption) error {
			asserts.Equal("testns", ns.Name)
			asserts.Equal(istioLabels, ns.Labels)
			return nil
		})

	asserts.NoError(CreateAndLabelNamespace(mock, "testns", false, true))
}

// TestCreateAndLabelNamespace tests the CreateAndLabelNamespace method for the following use case
// GIVEN a call to CreateAndLabelNamespace
// WHEN only the Istio injection labels are specified
// THEN no error is returned and only the requested the labels are added to the namepace
func TestCreateAndLabelNamespaceVzManaged(t *testing.T) {
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Name: "testns"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, ns *corev1.Namespace) error {
			return errors.NewNotFound(schema.ParseGroupResource("Namespace"), "testns")
		})

	mock.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, ns *corev1.Namespace, opts ...client.CreateOption) error {
			asserts.Equal("testns", ns.Name)
			asserts.Equal(createVzLabels("testns"), ns.Labels)
			return nil
		})

	asserts.NoError(CreateAndLabelNamespace(mock, "testns", true, false))
}

// TestCreateAndLabelNamespace tests the CreateAndLabelNamespace method for the following use case
// GIVEN a call to CreateAndLabelNamespace
// WHEN an unexpected error occurs during the Create call
// THEN an error is returned
func TestCreateAndLabelNamespaceReturnsError(t *testing.T) {
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Name: "testns"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, ns *corev1.Namespace) error {
			return errors.NewNotFound(schema.ParseGroupResource("Namespace"), "testns")
		})

	mock.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, ns *corev1.Namespace, opts ...client.CreateOption) error {
			return fmt.Errorf("UnexpectedError")
		})

	asserts.Error(CreateAndLabelNamespace(mock, "testns", true, true))
}

// TestCreateVerrazzanoSystemNamespace tests the CreateVerrazzanoSystemNamespace function
// GIVEN a call to CreateVerrazzanoSystemNamespace
// WHEN no error occurs
// THEN no error is returned, the namespace is created, and the proper labels have been added
func TestCreateVerrazzanoSystemNamespace(t *testing.T) {
	runNamespaceTestWithIstioFlag(t, globalconst.VerrazzanoSystemNamespace,
		createVZAndIstioLabels(globalconst.VerrazzanoSystemNamespace),
		CreateVerrazzanoSystemNamespace)
}

// TestCreateKeycloakNamespace tests the CreateKeycloakNamespace function
// GIVEN a call to CreateKeycloakNamespace
// WHEN no error occurs
// THEN no error is returned, the namespace is created, and the proper labels have been added
func TestCreateKeycloakNamespace(t *testing.T) {
	runNamespaceTestWithIstioFlag(t, globalconst.KeycloakNamespace,
		createVZAndIstioLabels(globalconst.KeycloakNamespace),
		CreateKeycloakNamespace)
}

// TestCreateRancherNamespace tests the CreateRancherNamespace function
// GIVEN a call to CreateRancherNamespace
// WHEN no error occurs
// THEN no error is returned, the namespace is created, and the proper labels have been added
func TestCreateRancherNamespace(t *testing.T) {
	runNamespaceTest(t, globalconst.RancherSystemNamespace,
		createVzLabels(globalconst.RancherSystemNamespace),
		CreateRancherNamespace)
}

// TestCreateArgoCDNamespace tests the CreateArgoCDNamespace function
// GIVEN a call to CreateArgoCDNamespace
// WHEN no error occurs
// THEN no error is returned, the namespace is created, and the proper labels have been added
func TestCreateArgoCDNamespace(t *testing.T) {
	runNamespaceTestWithIstioFlag(t, constants.ArgoCDNamespace,
		createVZAndIstioLabels(constants.ArgoCDNamespace),
		CreateArgoCDNamespace)
}

// TestCreateVerrazzanoMultiClusterNamespace tests the CreateVerrazzanoMultiClusterNamespace function
// GIVEN a call to CreateVerrazzanoMultiClusterNamespace
// WHEN no error occurs
// THEN no error is returned, the namespace is created, and the proper labels have been added
func TestCreateVerrazzanoMultiClusterNamespace(t *testing.T) {
	runNamespaceTest(t, globalconst.VerrazzanoMultiClusterNamespace,
		map[string]string{},
		CreateVerrazzanoMultiClusterNamespace)
}

func runNamespaceTest(t *testing.T, namespace string, expectedLabels map[string]string, namespaceFunc func(client client.Client) error) {
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Name: namespace}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, ns *corev1.Namespace) error {
			return errors.NewNotFound(schema.ParseGroupResource("Namespace"), namespace)
		})

	mock.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, ns *corev1.Namespace, opts ...client.CreateOption) error {
			asserts.Equal(namespace, ns.Name)
			asserts.Equal(expectedLabels, ns.Labels)
			return nil
		})

	asserts.NoError(namespaceFunc(mock))
}

func runNamespaceTestWithIstioFlag(t *testing.T, namespace string, expectedLabels map[string]string, namespaceFunc func(client client.Client, istioInjectionEnabled bool) error) {
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Name: namespace}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, ns *corev1.Namespace) error {
			return errors.NewNotFound(schema.ParseGroupResource("Namespace"), namespace)
		})

	mock.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, ns *corev1.Namespace, opts ...client.CreateOption) error {
			asserts.Equal(namespace, ns.Name)
			asserts.Equal(expectedLabels, ns.Labels)
			return nil
		})

	asserts.NoError(namespaceFunc(mock, true))
}
