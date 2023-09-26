// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package resource

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/k8s/errors"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	adminv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestDeleteClusterResource tests the deletion of a cluster resource
// GIVEN a Kubernetes resource
// WHEN delete is called
// THEN the resource should get deleted
func TestDeleteClusterResource(t *testing.T) {
	asserts := assert.New(t)

	name := "test"
	_ = vzapi.AddToScheme(k8scheme.Scheme)
	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&adminv1.ValidatingWebhookConfiguration{
			ObjectMeta: metav1.ObjectMeta{Name: name},
		}).Build()

	// Validate resource exists
	wh := adminv1.ValidatingWebhookConfiguration{ObjectMeta: metav1.ObjectMeta{Name: name}}
	err := c.Get(context.TODO(), types.NamespacedName{Name: name}, &wh)
	asserts.NoError(err)

	// Delete the resource
	err = Resource{
		Name:   name,
		Client: c,
		Object: &adminv1.ValidatingWebhookConfiguration{},
		Log:    vzlog.DefaultLogger(),
	}.Delete()

	// Validate that resource is deleted
	asserts.NoError(err)
	err = c.Get(context.TODO(), types.NamespacedName{Name: name}, &wh)
	asserts.Error(err)
	asserts.True(errors.IsNotFound(err))
}

// TestDeleteClusterResourceNotExists tests the deletion of a cluster resource
// GIVEN a resource that doesn't exist
// WHEN delete is called
// THEN the delete function should not return an error
func TestDeleteClusterResourceNotExists(t *testing.T) {
	asserts := assert.New(t)

	name := "test"
	_ = vzapi.AddToScheme(k8scheme.Scheme)
	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()

	// Validate resource does not exist
	wh := adminv1.ValidatingWebhookConfiguration{ObjectMeta: metav1.ObjectMeta{Name: name}}
	err := c.Get(context.TODO(), types.NamespacedName{Name: name}, &wh)
	asserts.True(errors.IsNotFound(err))

	// Delete the resource
	err = Resource{
		Name:   name,
		Client: c,
		Object: &adminv1.ValidatingWebhookConfiguration{},
		Log:    vzlog.DefaultLogger(),
	}.Delete()
	asserts.NoError(err)
}

// TestDelete tests the deletion of a resource
// GIVEN a Kubernetes resource
// WHEN delete is called
// THEN the resource should get deleted
func TestDelete(t *testing.T) {
	asserts := assert.New(t)

	name := "test"
	namespace := "testns"

	_ = vzapi.AddToScheme(k8scheme.Scheme)
	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name},
		}).Build()

	// Validate resource exists
	pod := corev1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name}}
	err := c.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, &pod)
	asserts.NoError(err)

	// Delete the resource
	err = Resource{
		Namespace: namespace,
		Name:      name,
		Client:    c,
		Object:    &corev1.Pod{},
		Log:       vzlog.DefaultLogger(),
	}.Delete()

	// Validate that resource is deleted
	asserts.NoError(err)
	err = c.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, &pod)
	asserts.Error(err)
	asserts.True(errors.IsNotFound(err))
}

// TestDeleteNotExists tests the deletion of a resource
// GIVEN a resource that doesn't exist
// WHEN delete is called
// THEN the delete function should not return an error
func TestDeleteNotExists(t *testing.T) {
	asserts := assert.New(t)

	name := "test"
	namespace := "testns"

	_ = vzapi.AddToScheme(k8scheme.Scheme)
	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()

	// Validate resource does not exist
	pod := corev1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name}}
	err := c.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, &pod)
	asserts.True(errors.IsNotFound(err))

	// Delete the resource
	err = Resource{
		Namespace: namespace,
		Name:      name,
		Client:    c,
		Object:    &corev1.Pod{},
		Log:       vzlog.DefaultLogger(),
	}.Delete()
	asserts.NoError(err)
}

// TestRemoveFinalizers tests the removing a finalizers from a resource
// GIVEN a Kubernetes resource
// WHEN RemoveFinalizers is called
// THEN the resource will have finalizers removed
func TestRemoveFinalizers(t *testing.T) {
	asserts := assert.New(t)

	name := "test"
	_ = vzapi.AddToScheme(k8scheme.Scheme)
	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
				Finalizers: []string{
					"fake-finalizer",
				},
			},
		}).Build()

	// Verify the resource exists with the finalizer
	ns := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}
	err := c.Get(context.TODO(), types.NamespacedName{Name: name}, &ns)
	asserts.NoError(err)
	asserts.Equal(1, len(ns.Finalizers))

	// Remove the finalizer from the resource
	err = Resource{
		Name:   name,
		Client: c,
		Object: &corev1.Namespace{},
		Log:    vzlog.DefaultLogger(),
	}.RemoveFinalizers()

	// Validate that resource is updated with no finalizer
	asserts.NoError(err)
	err = c.Get(context.TODO(), types.NamespacedName{Name: name}, &ns)
	asserts.NoError(err)
	asserts.Equal(0, len(ns.Finalizers))
}

// TestRemoveFinalizersNotExists tests the removal of finalizer from a resource
// GIVEN a resource that doesn't exist
// WHEN RemoveFinalizers is called
// THEN the RemoveFinalizers function should not return an error
func TestRemoveFinalizersNotExists(t *testing.T) {
	asserts := assert.New(t)

	name := "test"

	_ = vzapi.AddToScheme(k8scheme.Scheme)
	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()

	// Validate resource does not exist
	ns := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}
	err := c.Get(context.TODO(), types.NamespacedName{Name: name}, &ns)
	asserts.True(errors.IsNotFound(err))

	// Remove the finalizer from the resource
	err = Resource{
		Name:   name,
		Client: c,
		Object: &corev1.Pod{},
		Log:    vzlog.DefaultLogger(),
	}.RemoveFinalizers()
	asserts.NoError(err)
}

// TestRemoveFinalizersAndDelete tests the removing a finalizers from a resource and deleting the resource
// GIVEN a Kubernetes resource
// WHEN RemoveFinalizersAndDelete is called
// THEN the resource will have finalizers removed and be deleted
func TestRemoveFinalizersAndDelete(t *testing.T) {
	asserts := assert.New(t)

	name := "test"
	_ = vzapi.AddToScheme(k8scheme.Scheme)
	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
				Finalizers: []string{
					"fake-finalizer",
				},
			},
		}).Build()

	// Verify the resource exists with the finalizer
	ns := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}
	err := c.Get(context.TODO(), types.NamespacedName{Name: name}, &ns)
	asserts.NoError(err)
	asserts.Equal(1, len(ns.Finalizers))

	// Remove the finalizer from the resource and delete the resource
	err = Resource{
		Name:   name,
		Client: c,
		Object: &corev1.Namespace{},
		Log:    vzlog.DefaultLogger(),
	}.RemoveFinalizersAndDelete()

	// Validate that resource is deleted
	asserts.NoError(err)
	err = c.Get(context.TODO(), types.NamespacedName{Name: name}, &ns)
	asserts.True(errors.IsNotFound(err))
}
