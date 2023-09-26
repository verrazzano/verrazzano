// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhook

import (
	"context"
	"testing"

	"github.com/verrazzano/verrazzano/pkg/k8s/errors"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	adminv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
)

// TestDeleteValidating tests the deletion of a ValidatingWebhookConfiguration
// GIVEN a ValidatingWebhookConfiguration
// WHEN delete is called
// THEN the ValidatingWebhookConfiguration should get deleted
func TestDeleteValidating(t *testing.T) {
	asserts := assert.New(t)

	name := "test"
	_ = vzapi.AddToScheme(k8scheme.Scheme)
	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&adminv1.ValidatingWebhookConfiguration{
			ObjectMeta: metav1.ObjectMeta{Name: name},
		}).Build()

	// Validate webhook exists
	wh := adminv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}
	err := c.Get(context.TODO(), types.NamespacedName{Name: name}, &wh)
	asserts.NoError(err)

	// Delete the webhook
	err = DeleteValidatingWebhookConfiguration(vzlog.DefaultLogger(), c, name)

	// Validate that webhook is deleted
	asserts.NoError(err)
	err = c.Get(context.TODO(), types.NamespacedName{Name: name}, &wh)
	asserts.Error(err)
	asserts.True(errors.IsNotFound(err))
}

// TestDeleteValidatingNotExists tests the deletion of a ValidatingWebhookConfiguration
// GIVEN a ValidatingWebhookConfiguration that doesn't exist
// WHEN delete is called
// THEN the ValidatingWebhookConfiguration should get deleted
func TestDeleteValidatingNotExists(t *testing.T) {
	asserts := assert.New(t)

	name := "test"
	_ = vzapi.AddToScheme(k8scheme.Scheme)
	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()

	// Validate webhook exists
	wh := adminv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}
	err := c.Get(context.TODO(), types.NamespacedName{Name: name}, &wh)
	asserts.True(errors.IsNotFound(err))

	// Delete the webhook
	err = DeleteValidatingWebhookConfiguration(vzlog.DefaultLogger(), c, name)
	asserts.NoError(err)
}

// TestDeleteMutating tests the deletion of a MutatingWebhookConfiguration
// GIVEN a MutatingWebhookConfiguration
// WHEN delete is called
// THEN the MutatingWebhookConfiguration should not return an error
func TestDeleteMutating(t *testing.T) {
	asserts := assert.New(t)

	name := "test"
	_ = vzapi.AddToScheme(k8scheme.Scheme)
	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&adminv1.MutatingWebhookConfiguration{
			ObjectMeta: metav1.ObjectMeta{Name: name},
		}).Build()

	// Validate webhook exists
	wh := adminv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}
	err := c.Get(context.TODO(), types.NamespacedName{Name: name}, &wh)
	asserts.NoError(err)

	// Delete the webhook
	err = DeleteMutatingWebhookConfiguration(vzlog.DefaultLogger(), c, []string{name})

	// Validate that webhook is deleted
	asserts.NoError(err)
	err = c.Get(context.TODO(), types.NamespacedName{Name: name}, &wh)
	asserts.Error(err)
	asserts.True(errors.IsNotFound(err))
}

// TestDeleteMutatingNotExists tests the deletion of a MutatingWebhookConfiguration
// GIVEN a MutatingWebhookConfiguration that doesn't exist
// WHEN delete is called
// THEN the MutatingWebhookConfiguration should not return an error
func TestDeleteMutatingNotExists(t *testing.T) {
	asserts := assert.New(t)

	name := "test"
	_ = vzapi.AddToScheme(k8scheme.Scheme)
	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()

	// Validate webhook exists
	wh := adminv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}
	err := c.Get(context.TODO(), types.NamespacedName{Name: name}, &wh)
	asserts.True(errors.IsNotFound(err))

	// Delete the webhook
	err = DeleteMutatingWebhookConfiguration(vzlog.DefaultLogger(), c, []string{name})
	asserts.NoError(err)
}
