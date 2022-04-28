// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package ingresstrait

import (
	"context"
	"testing"

	"github.com/verrazzano/verrazzano/application-operator/constants"
	k8net "k8s.io/api/networking/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"

	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/app/v1alpha1"
	vzoam "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Test_isConsoleIngressUpdated tests the isConsoleIngressUpdated func for the following use case.
// GIVEN a request to isConsoleIngressUpdated
// WHEN only the Verrazzano Console ingress has changed
// THEN true is returned only when the Verrazzano Console ingress has changed, false otherwise
func Test_isConsoleIngressUpdated(t *testing.T) {

	asserts := assert.New(t)

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = vzapi.AddToScheme(scheme)
	_ = vzoam.AddToScheme(scheme)
	client := fake.NewClientBuilder().WithScheme(scheme).Build()

	r := newIngressTraitReconciler(client)

	oldIngress := &k8net.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: constants.VzConsoleIngress, Namespace: constants.VerrazzanoSystemNamespace},
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

	asserts.False(r.isConsoleIngressUpdated(event.UpdateEvent{
		ObjectOld: oldIngress,
		ObjectNew: newIngress,
	}))

	newIngress.Spec.Rules = []k8net.IngressRule{
		{Host: "host3"},
	}
	newIngress.Spec.TLS = []k8net.IngressTLS{
		{Hosts: []string{"host3"}},
	}
	asserts.True(r.isConsoleIngressUpdated(event.UpdateEvent{
		ObjectOld: oldIngress,
		ObjectNew: newIngress,
	}))

	oldOtherIngress := &k8net.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: "someingress", Namespace: constants.VerrazzanoSystemNamespace},
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
	newOtherIngress := oldIngress.DeepCopyObject().(*k8net.Ingress)
	asserts.False(r.isConsoleIngressUpdated(event.UpdateEvent{
		ObjectOld: oldOtherIngress,
		ObjectNew: newOtherIngress,
	}))
}

// Test_isIstioIngressGatewayUpdated tests the isIstioIngressGatewayUpdated func for the following use case.
// GIVEN a request to isIstioIngressGatewayUpdated
// WHEN only the IstioIngressGateway has changed
// THEN true is returned only when the IstioIngressGateway has changed, false otherwise
func Test_isIstioIngressGatewayUpdated(t *testing.T) {

	asserts := assert.New(t)

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = vzapi.AddToScheme(scheme)
	_ = vzoam.AddToScheme(scheme)
	client := fake.NewClientBuilder().WithScheme(scheme).Build()

	r := newIngressTraitReconciler(client)

	oldSvc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: istioIngressGateway, Namespace: constants.IstioSystemNamespace},
		Spec: corev1.ServiceSpec{
			Type: "LoadBalancer",
		},
	}
	newSvc := oldSvc.DeepCopyObject().(*corev1.Service)

	asserts.False(r.isIstioIngressGatewayUpdated(event.UpdateEvent{
		ObjectOld: oldSvc,
		ObjectNew: newSvc,
	}))

	newSvc.Spec.Type = "NodePort"
	asserts.True(r.isIstioIngressGatewayUpdated(event.UpdateEvent{
		ObjectOld: oldSvc,
		ObjectNew: newSvc,
	}))

	oldOtherIngress := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "somesvc", Namespace: constants.VerrazzanoSystemNamespace},
		Spec: corev1.ServiceSpec{
			Type: "LoadBalancer",
		},
	}
	newOtherIngress := oldSvc.DeepCopyObject().(*corev1.Service)
	asserts.False(r.isIstioIngressGatewayUpdated(event.UpdateEvent{
		ObjectOld: oldOtherIngress,
		ObjectNew: newOtherIngress,
	}))
}

// Test_createIngressTraitReconcileRequests tests the createIngressTraitReconcileRequests func for the following use case.
// GIVEN a request to createIngressTraitReconcileRequests
// THEN the correct set of reconcile requests is returned based on the number if IngressTraits across all namespaces
func Test_createIngressTraitReconcileRequests(t *testing.T) {

	asserts := assert.New(t)

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = vzapi.AddToScheme(scheme)
	_ = vzoam.AddToScheme(scheme)
	client := fake.NewClientBuilder().WithScheme(scheme).Build()

	reconciler := newIngressTraitReconciler(client)

	asserts.Len(reconciler.createIngressTraitReconcileRequests(), 0)

	_ = client.Create(context.TODO(), &vzoam.IngressTrait{ObjectMeta: metav1.ObjectMeta{Name: "trait1", Namespace: "traitns1"}})
	_ = client.Create(context.TODO(), &vzoam.IngressTrait{ObjectMeta: metav1.ObjectMeta{Name: "trait2", Namespace: "traitns1"}})
	_ = client.Create(context.TODO(), &vzoam.IngressTrait{ObjectMeta: metav1.ObjectMeta{Name: "trait1", Namespace: "traitns2"}})
	_ = client.Create(context.TODO(), &vzoam.IngressTrait{ObjectMeta: metav1.ObjectMeta{Name: "trait1", Namespace: "traitns3"}})
	_ = client.Create(context.TODO(), &vzoam.IngressTrait{ObjectMeta: metav1.ObjectMeta{Name: "trait2", Namespace: "traitns3"}})

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
