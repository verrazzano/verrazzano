// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	clustersv1alpha1 "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
)

const (
	clusterName = "c-m-hcknpvs7"
	displayName = "unit-test-cluster"
)

// GIVEN a Rancher cluster resource is created
// WHEN  the reconciler runs
// THEN  a VMC is created and the cluster id is set in the status
func TestReconcileCreateVMC(t *testing.T) {
	asserts := assert.New(t)

	obj := &clustersv1alpha1.VerrazzanoManagedCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
	}
	fakeClient := fake.NewClientBuilder().WithScheme(newScheme()).WithStatusSubresource(obj).WithObjects(newCattleCluster(clusterName, displayName)).Build()
	reconciler := newRancherClusterReconciler(fakeClient)
	request := newRequest(clusterName)

	_, err := reconciler.Reconcile(context.TODO(), request)
	asserts.NoError(err)

	// expect that a VMC was created and that the cluster id is set in the status
	vmc := &clustersv1alpha1.VerrazzanoManagedCluster{}
	err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: displayName, Namespace: vzconst.VerrazzanoMultiClusterNamespace}, vmc)
	asserts.NoError(err)
	asserts.Equal(clusterName, vmc.Status.RancherRegistration.ClusterID)
}

// GIVEN a Rancher cluster resource is created with labels
// AND   the user has specified a cluster selector that matches the labels
// WHEN  the reconciler runs
// THEN  a VMC is created and the cluster id is set in the status
func TestReconcileWithClusterSelector(t *testing.T) {
	asserts := assert.New(t)

	const (
		labelName  = "test-label"
		labelValue = "test-label-value"
	)

	obj := &clustersv1alpha1.VerrazzanoManagedCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
	}
	cluster := newCattleCluster(clusterName, displayName)
	cluster.SetLabels(map[string]string{labelName: labelValue})
	fakeClient := fake.NewClientBuilder().WithScheme(newScheme()).WithStatusSubresource(obj).WithObjects(cluster).Build()

	reconciler := newRancherClusterReconciler(fakeClient)
	reconciler.ClusterSelector = &metav1.LabelSelector{
		MatchExpressions: []metav1.LabelSelectorRequirement{
			{
				Key:      labelName,
				Operator: metav1.LabelSelectorOpIn,
				Values:   []string{labelValue},
			},
		},
	}
	request := newRequest(clusterName)

	_, err := reconciler.Reconcile(context.TODO(), request)
	asserts.NoError(err)

	// expect that a VMC was created and that the cluster id is set in the status
	vmc := &clustersv1alpha1.VerrazzanoManagedCluster{}
	err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: displayName, Namespace: vzconst.VerrazzanoMultiClusterNamespace}, vmc)
	asserts.NoError(err)
	asserts.Equal(clusterName, vmc.Status.RancherRegistration.ClusterID)
}

// GIVEN a Rancher cluster resource is created with labels
// AND   the user has specified a cluster selector that DOES NOT match the labels
// WHEN  the reconciler runs
// THEN  a VMC is not created
func TestReconcileClusterSelectorMismatch(t *testing.T) {
	asserts := assert.New(t)

	const (
		labelName  = "test-label"
		labelValue = "test-label-value"
	)

	cluster := newCattleCluster(clusterName, displayName)
	cluster.SetLabels(map[string]string{"label": "does-not-match"})
	fakeClient := fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(cluster).Build()

	reconciler := newRancherClusterReconciler(fakeClient)
	reconciler.ClusterSelector = &metav1.LabelSelector{
		MatchExpressions: []metav1.LabelSelectorRequirement{
			{
				Key:      labelName,
				Operator: metav1.LabelSelectorOpIn,
				Values:   []string{labelValue},
			},
		},
	}
	request := newRequest(clusterName)

	_, err := reconciler.Reconcile(context.TODO(), request)
	asserts.NoError(err)

	// expect that a VMC was not created
	vmc := &clustersv1alpha1.VerrazzanoManagedCluster{}
	err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: displayName, Namespace: vzconst.VerrazzanoMultiClusterNamespace}, vmc)
	asserts.Error(err)
	asserts.True(errors.IsNotFound(err))
}

// GIVEN a Rancher cluster resource is created
// AND   Rancher cluster sync is disabled
// WHEN  the reconciler runs
// THEN  a VMC is not created
func TestReconcileClusterSyncDisabled(t *testing.T) {
	asserts := assert.New(t)

	fakeClient := fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(newCattleCluster(clusterName, displayName)).Build()
	reconciler := newRancherClusterReconciler(fakeClient)
	reconciler.ClusterSyncEnabled = false
	request := newRequest(clusterName)

	_, err := reconciler.Reconcile(context.TODO(), request)
	asserts.NoError(err)

	// expect that a VMC was not created
	vmc := &clustersv1alpha1.VerrazzanoManagedCluster{}
	err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: displayName, Namespace: vzconst.VerrazzanoMultiClusterNamespace}, vmc)
	asserts.Error(err)
	asserts.True(errors.IsNotFound(err))
}

// GIVEN a Rancher cluster resource is created and a VMC already exists for the cluster
// WHEN  the reconciler runs
// THEN  the VMC is updated and the cluster id is set in the status
func TestReconcileCreateVMCAlreadyExists(t *testing.T) {
	asserts := assert.New(t)

	vmc := newVMC(displayName)
	fakeClient := fake.NewClientBuilder().WithScheme(newScheme()).WithStatusSubresource(vmc).WithObjects(newCattleCluster(clusterName, displayName), vmc).Build()
	reconciler := newRancherClusterReconciler(fakeClient)
	request := newRequest(clusterName)

	_, err := reconciler.Reconcile(context.TODO(), request)
	asserts.NoError(err)

	// expect the VMC still exists and that the cluster id is set in the status
	err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: displayName, Namespace: vzconst.VerrazzanoMultiClusterNamespace}, vmc)
	asserts.NoError(err)
	asserts.Equal(clusterName, vmc.Status.RancherRegistration.ClusterID)
}

// TestReconcileDeleteVMC tests reconciling and deleting VMCs.
func TestReconcileDeleteVMC(t *testing.T) {
	asserts := assert.New(t)

	// GIVEN a Rancher cluster resource is being deleted
	// AND   cluster sync is enabled
	// WHEN  the reconciler runs
	// THEN  the corresponding VMC is deleted
	cluster := newCattleCluster(clusterName, displayName)
	now := metav1.Now()
	cluster.SetDeletionTimestamp(&now)
	cluster.SetFinalizers([]string{finalizerName})
	vmc := newVMC(displayName)
	vmc.Status.RancherRegistration.ClusterID = clusterName
	fakeClient := fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(cluster, vmc).Build()

	reconciler := newRancherClusterReconciler(fakeClient)
	request := newRequest(clusterName)

	_, err := reconciler.Reconcile(context.TODO(), request)
	asserts.NoError(err)

	// expect that the VMC was deleted
	err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: displayName, Namespace: vzconst.VerrazzanoMultiClusterNamespace}, vmc)
	asserts.Error(err)
	asserts.True(errors.IsNotFound(err))

	// since the last finalizer was removed from the Rancher cluster, the cluster should be gone as well
	cluster = &unstructured.Unstructured{}
	cluster.SetGroupVersionKind(gvk)
	err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: clusterName}, cluster)
	asserts.Error(err)
	asserts.True(errors.IsNotFound(err))

	// GIVEN a Rancher cluster resource is being deleted
	// AND   cluster sync is disabled
	// WHEN  the reconciler runs
	// THEN  the corresponding VMC is deleted
	cluster = newCattleCluster(clusterName, displayName)
	cluster.SetDeletionTimestamp(&now)
	cluster.SetFinalizers([]string{finalizerName})
	vmc = newVMC(displayName)
	vmc.Status.RancherRegistration.ClusterID = clusterName
	fakeClient = fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(cluster, vmc).Build()

	reconciler = newRancherClusterReconciler(fakeClient)
	// disable cluster sync, the VMC (if it exists) should be deleted even if this flag is false
	reconciler.ClusterSyncEnabled = false
	request = newRequest(clusterName)

	_, err = reconciler.Reconcile(context.TODO(), request)
	asserts.NoError(err)

	// expect that the VMC was deleted
	err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: displayName, Namespace: vzconst.VerrazzanoMultiClusterNamespace}, vmc)
	asserts.Error(err)
	asserts.True(errors.IsNotFound(err))

	// since the last finalizer was removed from the Rancher cluster, the cluster should be gone as well
	cluster = &unstructured.Unstructured{}
	cluster.SetGroupVersionKind(gvk)
	err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: clusterName}, cluster)
	asserts.Error(err)
	asserts.True(errors.IsNotFound(err))
}

// GIVEN a Rancher cluster resource is being deleted and the VMC does not exist
// WHEN  the reconciler runs
// THEN  no error is returned
func TestReconcileDeleteVMCNotFound(t *testing.T) {
	asserts := assert.New(t)

	cluster := newCattleCluster(clusterName, displayName)
	now := metav1.Now()
	cluster.SetDeletionTimestamp(&now)
	cluster.SetFinalizers([]string{finalizerName})
	fakeClient := fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(cluster).Build()
	reconciler := newRancherClusterReconciler(fakeClient)
	request := newRequest(clusterName)

	_, err := reconciler.Reconcile(context.TODO(), request)
	asserts.NoError(err)
}

// GIVEN a Rancher cluster resource has been deleted
// WHEN  the reconciler runs
// THEN  no error is returned
func TestReconcileClusterGone(t *testing.T) {
	asserts := assert.New(t)

	fakeClient := fake.NewClientBuilder().WithScheme(newScheme()).Build()
	reconciler := newRancherClusterReconciler(fakeClient)
	request := newRequest(clusterName)

	_, err := reconciler.Reconcile(context.TODO(), request)
	asserts.NoError(err)
}

// GIVEN the local Rancher cluster resource is being reconciled
// WHEN  the reconciler runs
// THEN  no VMC is created
func TestReconcileLocalCluster(t *testing.T) {
	asserts := assert.New(t)

	fakeClient := fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(newCattleCluster(localClusterName, localClusterName)).Build()
	reconciler := newRancherClusterReconciler(fakeClient)
	request := newRequest(clusterName)

	_, err := reconciler.Reconcile(context.TODO(), request)
	asserts.NoError(err)

	// expect no VMC created for the local cluster
	vmc := &clustersv1alpha1.VerrazzanoManagedCluster{}
	err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: localClusterName, Namespace: vzconst.VerrazzanoMultiClusterNamespace}, vmc)
	asserts.True(errors.IsNotFound(err))
}

// GIVEN a GVK exists for cattle clusters resources
// WHEN  we fetch the client object for the cattle clusters GVK
// THEN  the returned client object is not nil
func TestCattleClusterClientObject(t *testing.T) {
	asserts := assert.New(t)
	obj := CattleClusterClientObject()
	asserts.NotNil(obj)
}

func TestParseClusterErrorCases(t *testing.T) {
	// GIVEN a cluster resource with no spec displayName field set
	// WHEN  we attempt to get the displayName from the cluster object
	// THEN  the expected error is returned
	asserts := assert.New(t)
	cluster := &unstructured.Unstructured{}
	cluster.SetGroupVersionKind(gvk)
	reconciler := newRancherClusterReconciler(nil)
	_, err := reconciler.getClusterDisplayName(cluster)
	asserts.ErrorContains(err, "Could not find spec displayName field")

	// GIVEN a cluster resource with a bad type for the spec field
	// WHEN  we attempt to get the displayName from the cluster object
	// THEN  the expected error is returned
	unstructured.SetNestedField(cluster.Object, true, "spec")
	_, err = reconciler.getClusterDisplayName(cluster)
	asserts.ErrorContains(err, ".spec.displayName accessor error")
}

// TestDeleteVMC tests the DeleteVMC function
func TestDeleteVMC(t *testing.T) {
	asserts := assert.New(t)
	cluster := newCattleCluster(clusterName, displayName)
	now := metav1.Now()
	cluster.SetDeletionTimestamp(&now)
	cluster.SetFinalizers([]string{finalizerName})
	vmc := newVMC(displayName)
	vmc.Status.RancherRegistration.ClusterID = clusterName
	fakeClient := fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(cluster, vmc).Build()
	reconciler := newRancherClusterReconciler(fakeClient)

	// GIVEN a VMC exists
	// WHEN DeleteVMC is called
	// THEN the VMC is deleted
	err := reconciler.DeleteVMC(cluster)
	asserts.NoError(err)

	// expect that the VMC was deleted
	err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: displayName, Namespace: vzconst.VerrazzanoMultiClusterNamespace}, vmc)
	asserts.True(errors.IsNotFound(err))
	// GIVEN no VMC exists
	// WHEN DeleteVMC is called
	// THEN no error is returned
	err = reconciler.DeleteVMC(cluster)
	asserts.NoError(err)
}

func newRequest(name string) ctrl.Request {
	return ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name: name,
		},
	}
}

func newRancherClusterReconciler(c client.Client) RancherClusterReconciler {
	return RancherClusterReconciler{
		Client:             c,
		Scheme:             newScheme(),
		ClusterSyncEnabled: true,
		Log:                zap.S(),
	}
}

func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	clientgoscheme.AddToScheme(scheme)
	clustersv1alpha1.AddToScheme(scheme)
	return scheme
}

func newCattleCluster(name, displayName string) *unstructured.Unstructured {
	cluster := &unstructured.Unstructured{}
	cluster.SetGroupVersionKind(gvk)
	cluster.SetName(name)
	unstructured.SetNestedField(cluster.Object, displayName, "spec", "displayName")
	return cluster
}
