// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package resource

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	adminv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
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

	// Validate webhook exists
	wh := adminv1.ValidatingWebhookConfiguration{ObjectMeta: metav1.ObjectMeta{Name: name}}
	err := c.Get(context.TODO(), types.NamespacedName{Name: name}, &wh)
	asserts.NoError(err)

	// Delete the webhook
	err = Resource{
		Name:   name,
		Client: c,
		Object: &adminv1.ValidatingWebhookConfiguration{},
		Log:    vzlog.DefaultLogger(),
	}.Delete()

	// Validate that webhook is deleted
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

	// Validate webhook exists
	wh := adminv1.ValidatingWebhookConfiguration{ObjectMeta: metav1.ObjectMeta{Name: name}}
	err := c.Get(context.TODO(), types.NamespacedName{Name: name}, &wh)
	asserts.True(errors.IsNotFound(err))

	// Delete the webhook
	// Delete the webhook
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
			ObjectMeta: metav1.ObjectMeta{Name: name},
		}).Build()

	// Validate webhook exists
	pod := corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: name}}
	err := c.Get(context.TODO(), types.NamespacedName{Name: name}, &pod)
	asserts.NoError(err)

	// Delete the webhook
	err = Resource{
		Namespace: namespace,
		Name:      name,
		Client:    c,
		Object:    &corev1.Pod{},
		Log:       vzlog.DefaultLogger(),
	}.Delete()

	// Validate that webhook is deleted
	asserts.NoError(err)
	err = c.Get(context.TODO(), types.NamespacedName{Name: name}, &pod)
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

	// Validate webhook exists
	pod := corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: name}}
	err := c.Get(context.TODO(), types.NamespacedName{Name: name}, &pod)
	asserts.True(errors.IsNotFound(err))

	// Delete the webhook
	// Delete the webhook
	err = Resource{
		Namespace: namespace,
		Name:      name,
		Client:    c,
		Object:    &corev1.Pod{},
		Log:       vzlog.DefaultLogger(),
	}.Delete()
	asserts.NoError(err)
}
