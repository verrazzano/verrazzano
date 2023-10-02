// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package capi

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/cluster-operator/internal/capi"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	clusterName = "capi1"
)

// GIVEN a CAPI cluster resource is created
// WHEN  the reconciler runs
// THEN  a rancher registration and associated artifacts are created
func TestClusterRancherRegistration(t *testing.T) {
	asserts := assert.New(t)

	rancherDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      common.RancherName,
			Namespace: common.CattleSystem,
		},
		Status: appsv1.DeploymentStatus{
			Replicas:      1,
			ReadyReplicas: 1,
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(newCAPICluster(clusterName), rancherDeployment).Build()
	reconciler := newCAPIClusterReconciler(fakeClient)
	request := newRequest(clusterName)

	SetClusterRancherRegistrationFunction(func(ctx context.Context, r *RancherRegistration, cluster *unstructured.Unstructured) (ctrl.Result, error) {
		persistClusterStatus(ctx, reconciler.Client, cluster, reconciler.Log, "capi1Id", registrationInitiated)
		return ctrl.Result{}, nil
	})
	defer SetDefaultClusterRancherRegistrationFunction()

	_, err := reconciler.Reconcile(context.TODO(), request)
	asserts.NoError(err)

	clusterRegistrationSecret := &v1.Secret{}
	err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: clusterName + clusterStatusSuffix, Namespace: constants.VerrazzanoCAPINamespace}, clusterRegistrationSecret)
	asserts.NoError(err)
	asserts.Equal(registrationInitiated, string(clusterRegistrationSecret.Data[clusterRegistrationStatusKey]))
	cluster := &unstructured.Unstructured{}
	cluster.SetGroupVersionKind(capi.GVKCAPICluster)
	err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: clusterName}, cluster)
	asserts.NoError(err)
	asserts.Equal(finalizerName, cluster.GetFinalizers()[0])
}

// GIVEN a CAPI cluster resource is deleted
// WHEN  the reconciler runs
// THEN  a rancher registration and associated artifacts are removed
func TestClusterUnregistration(t *testing.T) {
	asserts := assert.New(t)

	rancherDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      common.RancherName,
			Namespace: common.CattleSystem,
		},
		Status: appsv1.DeploymentStatus{
			Replicas:      1,
			ReadyReplicas: 1,
		},
	}

	clusterRegistrationSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterName + clusterStatusSuffix,
			Namespace: constants.VerrazzanoCAPINamespace,
		},
		Data: map[string][]byte{clusterIDKey: []byte("capi1Id"), clusterRegistrationStatusKey: []byte(registrationInitiated)},
	}

	cluster := newCAPICluster(clusterName)
	now := metav1.Now()
	cluster.SetDeletionTimestamp(&now)
	cluster.SetFinalizers([]string{finalizerName})

	fakeClient := fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(cluster, rancherDeployment, clusterRegistrationSecret).Build()
	reconciler := newCAPIClusterReconciler(fakeClient)
	request := newRequest(clusterName)

	SetClusterRancherUnregistrationFunction(func(ctx context.Context, r *RancherRegistration, cluster *unstructured.Unstructured) error {
		return nil
	})
	defer SetDefaultClusterRancherUnregistrationFunction()

	_, err := reconciler.Reconcile(context.TODO(), request)
	asserts.NoError(err)

	remainingSecret := &v1.Secret{}
	asserts.Error(fakeClient.Get(context.TODO(), types.NamespacedName{Name: clusterName + clusterStatusSuffix, Namespace: constants.VerrazzanoCAPINamespace}, remainingSecret))
	deletedCluster := &unstructured.Unstructured{}
	deletedCluster.SetGroupVersionKind(capi.GVKCAPICluster)
	asserts.Error(fakeClient.Get(context.TODO(), types.NamespacedName{Name: clusterName}, deletedCluster))
}

func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	clientgoscheme.AddToScheme(scheme)

	return scheme
}

func newCAPICluster(name string) *unstructured.Unstructured {
	cluster := &unstructured.Unstructured{}
	cluster.SetGroupVersionKind(capi.GVKCAPICluster)
	cluster.SetName(name)
	return cluster
}

func newCAPIClusterReconciler(c client.Client) CAPIClusterReconciler {
	rancherRegistrar := &RancherRegistration{
		Client: c,
		Log:    zap.S(),
	}
	return CAPIClusterReconciler{
		Client:           c,
		Scheme:           newScheme(),
		Log:              zap.S(),
		RancherEnabled:   true,
		RancherRegistrar: rancherRegistrar,
	}
}

func newRequest(name string) ctrl.Request {
	return ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name: name,
		},
	}
}
