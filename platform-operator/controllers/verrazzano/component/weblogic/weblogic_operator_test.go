// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package weblogic

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/bom"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// Test_appendWeblogicOperatorOverridesExtraKVs tests the AppendWeblogicOperatorOverrides fn
// GIVEN a call to AppendWeblogicOperatorOverrides
//
//	WHEN I call with no extra kvs
//	THEN the correct number of KeyValue objects are returned and no errors occur
func Test_appendWeblogicOperatorOverrides(t *testing.T) {
	kvs, err := AppendWeblogicOperatorOverrides(spi.NewFakeContext(nil, nil, nil, false), "weblogic-operator", "verrazzano-system", "", []bom.KeyValue{})
	assert.NoError(t, err)
	assert.Len(t, kvs, 5)
}

// Test_appendWeblogicOperatorOverridesExtraKVs tests the AppendWeblogicOperatorOverrides fn
// GIVEN a call to AppendWeblogicOperatorOverrides
//
//	WHEN I pass in a KeyValue list
//	THEN the values passed in are preserved and no errors occur
func Test_appendWeblogicOperatorOverridesExtraKVs(t *testing.T) {
	kvs := []bom.KeyValue{
		{Key: "Key", Value: "Value"},
	}
	var err error
	kvs, err = AppendWeblogicOperatorOverrides(spi.NewFakeContext(nil, nil, nil, false), "weblogic-operator", "verrazzano-system", "", kvs)
	assert.NoError(t, err)
	assert.Len(t, kvs, 6)
}

// Test_weblogicOperatorPreInstall tests the WeblogicOperatorPreInstall fn
// GIVEN a call to this fn
//
//	WHEN I call WeblogicOperatorPreInstall
//	THEN no errors are returned
func Test_weblogicOperatorPreInstall(t *testing.T) {
	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
	err := WeblogicOperatorPreInstall(spi.NewFakeContext(client, &vzapi.Verrazzano{}, nil, false), "weblogic-operator", "verrazzano-system", "")
	assert.NoError(t, err)
}

// TestIsWeblogicOperatorReady tests the isWeblogicOperatorReady function
// GIVEN a call to isWeblogicOperatorReady
//
//	WHEN the deployment object has enough replicas available
//	THEN true is returned
func TestIsWeblogicOperatorReady(t *testing.T) {

	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      ComponentName,
				Labels:    map[string]string{"app": ComponentName},
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": ComponentName},
				},
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 1,
				Replicas:          1,
				UpdatedReplicas:   1,
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      ComponentName + "-95d8c5d96-m6mbr",
				Labels: map[string]string{
					"pod-template-hash": "95d8c5d96",
					"app":               ComponentName,
				},
			},
		},
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   ComponentNamespace,
				Name:        ComponentName + "-95d8c5d96",
				Annotations: map[string]string{"deployment.kubernetes.io/revision": "1"},
			},
		},
	).Build()
	weblogic := NewComponent().(weblogicComponent)
	assert.True(t, weblogic.isWeblogicOperatorReady(spi.NewFakeContext(fakeClient, nil, nil, false)))
}

// TestIsWeblogicOperatorNotReady tests the isWeblogicOperatorReady function
// GIVEN a call to isWeblogicOperatorReady
//
//	WHEN the deployment object does NOT have enough replicas available
//	THEN false is returned
func TestIsWeblogicOperatorNotReady(t *testing.T) {

	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ComponentNamespace,
			Name:      ComponentName,
		},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: 1,
			Replicas:          1,
			UpdatedReplicas:   0,
		},
	}).Build()
	weblogic := NewComponent().(weblogicComponent)
	assert.False(t, weblogic.isWeblogicOperatorReady(spi.NewFakeContext(fakeClient, nil, nil, false)))
}
