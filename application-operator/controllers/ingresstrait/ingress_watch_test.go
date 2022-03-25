// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package ingresstrait

import (
	"context"
	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/app/v1alpha1"
	vzoam "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	k8net "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"testing"
)

// Test_isConsoleIngressUpdated tests the isConsoleIngressUpdated func for the following use case.
// GIVEN a request to isConsoleIngressUpdated
// WHEN the old and new ingress objects have been changed
// THEN true is returned only when the TLS fields differ, false otherwise
func Test_isConsoleIngressUpdated(t *testing.T) {

	oldIngress := &k8net.Ingress{
		Spec: k8net.IngressSpec{
			Rules: []k8net.IngressRule{
				{Host: "host1"},
				{Host: "host2"},
			},
			TLS: []k8net.IngressTLS{
				{Hosts: []string{"host1", "host2"}},
			},
		},
	}
	newIngress := oldIngress.DeepCopyObject().(*k8net.Ingress)

	assert.False(t, isConsoleIngressUpdated(event.UpdateEvent{
		ObjectOld: oldIngress,
		ObjectNew: newIngress,
	}))

	newIngress.Spec.Rules = []k8net.IngressRule{
		{Host: "host3"},
	}
	newIngress.Spec.TLS = []k8net.IngressTLS{
		{Hosts: []string{"host3"}},
	}
	assert.True(t, isConsoleIngressUpdated(event.UpdateEvent{
		ObjectOld: oldIngress,
		ObjectNew: newIngress,
	}))
}

// Test_createIngressTraitReconcileRequests tests the createIngressTraitReconcileRequests func for the following use case.
// GIVEN a request to createIngressTraitReconcileRequests
// THEN the correct set of reconcile requests is returned based on the number if IngressTraits across all namespaces
func Test_createIngressTraitReconcileRequests(t *testing.T) {

	asserts := assert.New(t)

	scheme := runtime.NewScheme()
	corev1.AddToScheme(scheme)
	vzapi.AddToScheme(scheme)
	vzoam.AddToScheme(scheme)
	client := fake.NewFakeClientWithScheme(scheme)

	reconciler := newIngressTraitReconciler(client)

	asserts.Len(reconciler.createIngressTraitReconcileRequests(), 0)

	client.Create(context.TODO(), &vzoam.IngressTrait{ObjectMeta: metav1.ObjectMeta{Name: "trait1", Namespace: "traitns1"}})
	client.Create(context.TODO(), &vzoam.IngressTrait{ObjectMeta: metav1.ObjectMeta{Name: "trait2", Namespace: "traitns1"}})
	client.Create(context.TODO(), &vzoam.IngressTrait{ObjectMeta: metav1.ObjectMeta{Name: "trait1", Namespace: "traitns2"}})
	client.Create(context.TODO(), &vzoam.IngressTrait{ObjectMeta: metav1.ObjectMeta{Name: "trait1", Namespace: "traitns3"}})
	client.Create(context.TODO(), &vzoam.IngressTrait{ObjectMeta: metav1.ObjectMeta{Name: "trait2", Namespace: "traitns3"}})

	expectedRequests := []reconcile.Request{
		{NamespacedName: types.NamespacedName{Name: "trait1", Namespace: "traitns1"}},
		{NamespacedName: types.NamespacedName{Name: "trait2", Namespace: "traitns1"}},
		{NamespacedName: types.NamespacedName{Name: "trait1", Namespace: "traitns2"}},
		{NamespacedName: types.NamespacedName{Name: "trait1", Namespace: "traitns3"}},
		{NamespacedName: types.NamespacedName{Name: "trait2", Namespace: "traitns3"}},
	}
	actualRequests := reconciler.createIngressTraitReconcileRequests()
	asserts.Len(actualRequests, 5)
	asserts.Equal(expectedRequests, actualRequests)
}
