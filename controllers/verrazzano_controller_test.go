// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package controllers

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/api/v1alpha1"
	"github.com/verrazzano/verrazzano/internal/installjob"
	"github.com/verrazzano/verrazzano/internal/uninstalljob"
	"github.com/verrazzano/verrazzano/mocks"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Generate mocs for the Kerberos Client and StatusWriter interfaces for use in tests.
//go:generate mockgen -destination=../mocks/controller_mock.go -package=mocks -copyright_file=../hack/boilerplate.go.txt sigs.k8s.io/controller-runtime/pkg/client Client,StatusWriter

const installPrefix = "verrazzano-install-"
const uninstallPrefix = "verrazzano-uninstall-"

// TestGetConfigMapName tests generating a ConfigMap name
// GIVEN a name
// WHEN the method is called
// THEN return the generated ConfigMap name
func TestGetConfigMapName(t *testing.T) {
	name := "configMap"
	configMapName := getConfigMapName(name)
	assert.Equalf(t, installPrefix+name, configMapName, "Expected ConfigMap name did not match")
}

// TestGetClusterRoleBindingName tests generating a ClusterRoleBinding name
// GIVEN a name and namespace
// WHEN the method is called
// THEN return the generated ClusterRoleBinding name
func TestGetClusterRoleBindingName(t *testing.T) {
	name := "role"
	namespace := "verrazzano"
	roleBindingName := getClusterRoleBindingName(namespace, name)
	assert.Equalf(t, installPrefix+namespace+"-"+name, roleBindingName, "Expected ClusterRoleBinding name did not match")
}

// TestGetServiceAccountName tests generating a ServiceAccount name
// GIVEN a name
// WHEN the method is called
// THEN return the generated ServiceAccount name
func TestGetServiceAccountName(t *testing.T) {
	name := "sa"
	saName := getServiceAccountName(name)
	assert.Equalf(t, installPrefix+name, saName, "Expected ServiceAccount name did not match")
}

// TestGetUninstallJobName tests generating a Job name
// GIVEN a name
// WHEN the method is called
// THEN return the generated Job name
func TestGetUninstallJobName(t *testing.T) {
	name := "test"
	jobName := getUninstallJobName(name)
	assert.Equalf(t, uninstallPrefix+name, jobName, "Expected uninstall job name did not match")
}

// TestGetInstallJobName tests generating a Job name
// GIVEN a name
// WHEN the method is called
// THEN return the generated Job name
func TestGetInstallJobName(t *testing.T) {
	name := "test"
	jobName := getInstallJobName(name)
	assert.Equalf(t, installPrefix+name, jobName, "Expected install job name did not match")
}

// TestSuccessfulInstall tests the Reconcile method for the following use case
// GIVEN a request to reconcile an verrazzano resource
// WHEN a verrazzano resource has been applied
// THEN ensure all the objects are already created
func TestSuccessfulInstall(t *testing.T) {
	namespace := "verrazzano"
	name := "test"
	labels := map[string]string{"label1": "test"}
	var savedVerrazzano *vzapi.Verrazzano
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	// Expect a call to get the verrazzano resource.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: name}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, verrazzano *vzapi.Verrazzano) error {
			verrazzano.TypeMeta = metav1.TypeMeta{
				APIVersion: "install.verrazzano.io/v1alpha1",
				Kind:       "Verrazzano"}
			verrazzano.ObjectMeta = metav1.ObjectMeta{
				Namespace: name.Namespace,
				Name:      name.Name}
			savedVerrazzano = verrazzano
			return nil
		})

	// Expect a call to get the ServiceAccount - return that it exists
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: getServiceAccountName(name)}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, serviceAccount *corev1.ServiceAccount) error {
			newSA := installjob.NewServiceAccount(name.Namespace, name.Name, "", labels)
			serviceAccount.ObjectMeta = newSA.ObjectMeta
			return nil
		})

	// Expect a call to get the ClusterRoleBinding - return that it exists
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "", Name: getClusterRoleBindingName(namespace, name)}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, clusterRoleBinding *rbacv1.ClusterRoleBinding) error {
			crb := installjob.NewClusterRoleBinding(savedVerrazzano, name.Name, getServiceAccountName(name.Name))
			clusterRoleBinding.ObjectMeta = crb.ObjectMeta
			clusterRoleBinding.RoleRef = crb.RoleRef
			clusterRoleBinding.Subjects = crb.Subjects
			return nil
		})

	// Expect a call to get the ConfigMap - return that it exists
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: getConfigMapName(name)}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, configMap *corev1.ConfigMap) error {
			cm := installjob.NewConfigMap(name.Namespace, name.Name, labels)
			configMap.ObjectMeta = cm.ObjectMeta
			return nil
		})

	// Expect a call to get the Job - return that it exists
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: getInstallJobName(name)}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, job *batchv1.Job) error {
			newJob := installjob.NewJob(name.Namespace, name.Name, labels, getConfigMapName(name.Name), getServiceAccountName(name.Name), "image", vzapi.DNS{})
			job.ObjectMeta = newJob.ObjectMeta
			job.Spec = newJob.Spec
			job.Status = batchv1.JobStatus{
				Succeeded: 1,
			}
			return nil
		})

	// Expect a call to get the status writer and return a mock.
	mock.EXPECT().Status().Return(mockStatus).AnyTimes()

	// Expect a call to update the status of the Verrazzano resource
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, verrazzano *vzapi.Verrazzano, opts ...client.UpdateOption) error {
			asserts.Len(verrazzano.Status.Conditions, 1)
			return nil
		})

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(mock)
	result, err := reconciler.Reconcile(request)

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)
	asserts.Equal(false, result.Requeue)
	asserts.Equal(time.Duration(0), result.RequeueAfter)
}

// TestCreateVerrazzano tests the Reconcile method for the following use case
// GIVEN a request to reconcile an verrazzano resource
// WHEN a verrazzano resource has been created
// THEN ensure all the objects are created
func TestCreateVerrazzano(t *testing.T) {
	namespace := "verrazzano"
	name := "test"
	labels := map[string]string{"label1": "test1"}

	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	// Expect a call to get the verrazzano resource.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: name}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, verrazzano *vzapi.Verrazzano) error {
			verrazzano.TypeMeta = metav1.TypeMeta{
				APIVersion: "install.verrazzano.io/v1alpha1",
				Kind:       "Verrazzano"}
			verrazzano.ObjectMeta = metav1.ObjectMeta{
				Namespace: name.Namespace,
				Name:      name.Name,
				Labels:    labels}
			return nil
		})

	// Expect a call to get the ServiceAccount - return that it does not exist
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: getServiceAccountName(name)}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: namespace, Resource: "ServiceAccount"}, getServiceAccountName(name)))

	// Expect a call to create the ServiceAccount - return success
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, serviceAccount *corev1.ServiceAccount, opts ...client.CreateOption) error {
			asserts.Equalf(namespace, serviceAccount.Namespace, "ServiceAccount namespace did not match")
			asserts.Equalf(getServiceAccountName(name), serviceAccount.Name, "ServiceAccount name did not match")
			asserts.Equalf(labels, serviceAccount.Labels, "ServiceAccount labels did not match")
			return nil
		})

	// Expect a call to get the ClusterRoleBinding - return that it does not exist
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "", Name: getClusterRoleBindingName(namespace, name)}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: "", Resource: "ClusterRoleBinding"}, getClusterRoleBindingName(namespace, name)))

	// Expect a call to create the ClusterRoleBinding - return success
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, clusterRoleBinding *rbacv1.ClusterRoleBinding, opts ...client.CreateOption) error {
			asserts.Equalf("", clusterRoleBinding.Namespace, "ClusterRoleBinding namespace did not match")
			asserts.Equalf(getClusterRoleBindingName(namespace, name), clusterRoleBinding.Name, "ClusterRoleBinding name did not match")
			asserts.Equalf(labels, clusterRoleBinding.Labels, "ClusterRoleBinding labels did not match")
			asserts.Equalf(getServiceAccountName(name), clusterRoleBinding.Subjects[0].Name, "ClusterRoleBinding Subjects name did not match")
			asserts.Equalf(namespace, clusterRoleBinding.Subjects[0].Namespace, "ClusterRoleBinding Subjects namespace did not match")
			return nil
		})

	// Expect a call to get the ConfigMap - return that it does not exist
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: getConfigMapName(name)}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: namespace, Resource: "ConfigMap"}, getServiceAccountName(name)))

	// Expect a call to create the ConfigMap - return success
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, configMap *corev1.ConfigMap, opts ...client.CreateOption) error {
			asserts.Equalf(namespace, configMap.Namespace, "ConfigMap namespace did not match")
			asserts.Equalf(getConfigMapName(name), configMap.Name, "ConfigMap name did not match")
			asserts.Equalf(labels, configMap.Labels, "ConfigMap labels did not match")
			return nil
		})

	// Expect a call to get the Job - return that it does not exist
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: getInstallJobName(name)}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: namespace, Resource: "Job"}, getInstallJobName(name)))

	// Expect a call to create the Job - return success
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, job *batchv1.Job, opts ...client.CreateOption) error {
			asserts.Equalf(namespace, job.Namespace, "Job namespace did not match")
			asserts.Equalf(getInstallJobName(name), job.Name, "Job name did not match")
			asserts.Equalf(labels, job.Labels, "Job labels did not match")
			return nil
		})

	// Expect a call to update the Verrazzano resource
	mock.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)

	// Expect a call to get a stale uninstall job resource
	mock.EXPECT().Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: getUninstallJobName(name)}, gomock.Any()).Return(nil)

	// Expect a call to delete a stale uninstall job resource
	mock.EXPECT().Delete(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

	// Expect a call to get the status writer and return a mock.
	mock.EXPECT().Status().Return(mockStatus).AnyTimes()

	// Expect a call to update the status of the Verrazzano resource
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, verrazzano *vzapi.Verrazzano, opts ...client.UpdateOption) error {
			asserts.Len(verrazzano.Status.Conditions, 1)
			return nil
		})

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(mock)
	result, err := reconciler.Reconcile(request)

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)
	asserts.Equal(false, result.Requeue)
	asserts.Equal(time.Duration(0), result.RequeueAfter)
}

// TestUninstallComplete tests the Reconcile method for the following use case
// GIVEN a request to reconcile an verrazzano resource
// WHEN a verrazzano resource has been deleted
// THEN ensure all the objects are deleted
func TestUninstallComplete(t *testing.T) {
	namespace := "verrazzano"
	name := "test"
	labels := map[string]string{"label1": "test"}

	deleteTime := metav1.Time{
		Time: time.Now(),
	}

	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	// Expect a call to get the verrazzano resource.  Return resource with deleted timestamp.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: name}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, verrazzano *vzapi.Verrazzano) error {
			verrazzano.TypeMeta = metav1.TypeMeta{
				APIVersion: "install.verrazzano.io/v1alpha1",
				Kind:       "Verrazzano"}
			verrazzano.ObjectMeta = metav1.ObjectMeta{
				Namespace:         name.Namespace,
				Name:              name.Name,
				DeletionTimestamp: &deleteTime,
				Finalizers:        []string{finalizerName}}
			verrazzano.Status = vzapi.VerrazzanoStatus{
				Conditions: []vzapi.Condition{
					{
						Type: vzapi.UninstallComplete,
					},
				},
			}
			return nil
		})

	// Expect a call to get the uninstall Job - return that it exists
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: getUninstallJobName(name)}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, job *batchv1.Job) error {
			newJob := uninstalljob.NewJob(name.Namespace, name.Name, labels, getServiceAccountName(name.Name), "image")
			job.ObjectMeta = newJob.ObjectMeta
			job.Spec = newJob.Spec
			return nil
		})

	// Expect a call to update the finalizers - return success
	mock.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	// Expect a call to get the status writer and return a mock.
	mock.EXPECT().Status().Return(mockStatus).AnyTimes()

	// Expect a call to update the status of the Verrazzano resource
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, verrazzano *vzapi.Verrazzano, opts ...client.UpdateOption) error {
			asserts.Len(verrazzano.Status.Conditions, 2)
			return nil
		})

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(mock)
	result, err := reconciler.Reconcile(request)

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)
	asserts.Equal(false, result.Requeue)
	asserts.Equal(time.Duration(0), result.RequeueAfter)
}

// TestUninstallStarted tests the Reconcile method for the following use case
// GIVEN a request to reconcile an verrazzano resource
// WHEN a verrazzano resource has been deleted
// THEN ensure an unisntall job is started
func TestUninstallStarted(t *testing.T) {
	namespace := "verrazzano"
	name := "test"
	labels := map[string]string{"label1": "test"}

	deleteTime := metav1.Time{
		Time: time.Now(),
	}

	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	// Expect a call to get the verrazzano resource.  Return resource with deleted timestamp.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: name}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, verrazzano *vzapi.Verrazzano) error {
			verrazzano.TypeMeta = metav1.TypeMeta{
				APIVersion: "install.verrazzano.io/v1alpha1",
				Kind:       "Verrazzano"}
			verrazzano.ObjectMeta = metav1.ObjectMeta{
				Namespace:         name.Namespace,
				Name:              name.Name,
				Labels:            labels,
				DeletionTimestamp: &deleteTime,
				Finalizers:        []string{finalizerName}}
			verrazzano.Status = vzapi.VerrazzanoStatus{
				Conditions: []vzapi.Condition{
					{
						Type: vzapi.UninstallStarted,
					},
				},
			}
			return nil
		})

	// Expect a call to get the uninstall Job - return that it does not exist
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: getUninstallJobName(name)}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: namespace, Resource: "Job"}, getUninstallJobName(name)))

	// Expect a call to create the uninstall Job - return success
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, job *batchv1.Job, opts ...client.CreateOption) error {
			asserts.Equalf(namespace, job.Namespace, "Job namespace did not match")
			asserts.Equalf(getUninstallJobName(name), job.Name, "Job name did not match")
			asserts.Equalf(labels, job.Labels, "Job labels did not match")
			return nil
		})

	// Expect a call to update the job - return success
	mock.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	// Expect a call to get the status writer and return a mock.
	mock.EXPECT().Status().Return(mockStatus).AnyTimes()

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(mock)
	result, err := reconciler.Reconcile(request)

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)
	asserts.Equal(false, result.Requeue)
	asserts.Equal(time.Duration(0), result.RequeueAfter)
}

// TestUninstallFailed tests the Reconcile method for the following use case
// GIVEN an uninstall job has failed
// WHEN a verrazzano resource has been deleted
// THEN ensure the error is handled
func TestUninstallFailed(t *testing.T) {
	namespace := "verrazzano"
	name := "test"
	labels := map[string]string{"label1": "test"}

	deleteTime := metav1.Time{
		Time: time.Now(),
	}

	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	// Expect a call to get the verrazzano resource.  Return resource with deleted timestamp.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: name}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, verrazzano *vzapi.Verrazzano) error {
			verrazzano.TypeMeta = metav1.TypeMeta{
				APIVersion: "install.verrazzano.io/v1alpha1",
				Kind:       "Verrazzano"}
			verrazzano.ObjectMeta = metav1.ObjectMeta{
				Namespace:         name.Namespace,
				Name:              name.Name,
				DeletionTimestamp: &deleteTime,
				Finalizers:        []string{finalizerName}}
			return nil
		})

	// Expect a call to get the uninstall Job - return that it exists and the job failed
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: getUninstallJobName(name)}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, job *batchv1.Job) error {
			newJob := uninstalljob.NewJob(name.Namespace, name.Name, labels, getServiceAccountName(name.Name), "image")
			job.ObjectMeta = newJob.ObjectMeta
			job.Spec = newJob.Spec
			job.Status = batchv1.JobStatus{
				Failed: 1,
			}
			return nil
		})

	// Expect a status update on the job
	mockStatus.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)

	// Expect a call to update the finalizers - return success
	mock.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)

	// Expect a call to get the status writer and return a mock.
	mock.EXPECT().Status().Return(mockStatus).AnyTimes()

	// Expect a call to update the status of the Verrazzano resource
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, verrazzano *vzapi.Verrazzano, opts ...client.UpdateOption) error {
			return nil
		})

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(mock)
	result, err := reconciler.Reconcile(request)

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)
	asserts.Equal(false, result.Requeue)
	asserts.Equal(time.Duration(0), result.RequeueAfter)
}

// TestUninstallSucceeded tests the Reconcile method for the following use case
// GIVEN an uninstall job has succeeded
// WHEN a verrazzano resource has been deleted
// THEN ensure all the objects are deleted
func TestUninstallSucceeded(t *testing.T) {
	namespace := "verrazzano"
	name := "test"
	labels := map[string]string{"label1": "test"}

	deleteTime := metav1.Time{
		Time: time.Now(),
	}

	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	// Expect a call to get the verrazzano resource.  Return resource with deleted timestamp.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: name}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, verrazzano *vzapi.Verrazzano) error {
			verrazzano.TypeMeta = metav1.TypeMeta{
				APIVersion: "install.verrazzano.io/v1alpha1",
				Kind:       "Verrazzano"}
			verrazzano.ObjectMeta = metav1.ObjectMeta{
				Namespace:         name.Namespace,
				Name:              name.Name,
				DeletionTimestamp: &deleteTime,
				Finalizers:        []string{finalizerName}}
			return nil
		})

	// Expect a call to get the uninstall Job - return that it exists and the job succeeded
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: getUninstallJobName(name)}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, job *batchv1.Job) error {
			newJob := uninstalljob.NewJob(name.Namespace, name.Name, labels, getServiceAccountName(name.Name), "image")
			job.ObjectMeta = newJob.ObjectMeta
			job.Spec = newJob.Spec
			job.Status = batchv1.JobStatus{
				Succeeded: 1,
			}
			return nil
		})

	// Expect a status update on the job
	mockStatus.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)

	// Expect a call to update the finalizers - return success
	mock.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)

	// Expect a call to get the status writer and return a mock.
	mock.EXPECT().Status().Return(mockStatus).AnyTimes()

	// Expect a call to update the status of the Verrazzano resource
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, verrazzano *vzapi.Verrazzano, opts ...client.UpdateOption) error {
			return nil
		})

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(mock)
	result, err := reconciler.Reconcile(request)

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)
	asserts.Equal(false, result.Requeue)
	asserts.Equal(time.Duration(0), result.RequeueAfter)
}

// TestVerrazzanoNotFound tests the Reconcile method for the following use case
// GIVEN an reqyest for a verrazzano custom resource
// WHEN it does not exist
// THEN ensure the error not found is handled
func TestVerrazzanoNotFound(t *testing.T) {
	namespace := "verrazzano"
	name := "test"

	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	// Expect a call to get the verrazzano resource - return that it does not exist
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: name}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: namespace, Resource: "Verrazzano"}, name))

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(mock)
	result, err := reconciler.Reconcile(request)

	// Validate the results
	mocker.Finish()
	asserts.NoError(err)
	asserts.Equal(false, result.Requeue)
	asserts.Equal(time.Duration(0), result.RequeueAfter)
}

// TestVerrazzanoGetError tests the Reconcile method for the following use case
// GIVEN an reqyest for a verrazzano custom resource
// WHEN there is a failure getting it
// THEN ensure the error is handled
func TestVerrazzanoGetError(t *testing.T) {
	namespace := "verrazzano"
	name := "test"

	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	// Expect a call to get the verrazzano resource - return that it does not exist
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: name}, gomock.Not(gomock.Nil())).
		Return(errors.NewBadRequest("failed to get Verrazzano custom resource"))

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(mock)
	result, err := reconciler.Reconcile(request)

	// Validate the results
	mocker.Finish()
	asserts.EqualError(err, "failed to get Verrazzano custom resource")
	asserts.Equal(false, result.Requeue)
	asserts.Equal(time.Duration(0), result.RequeueAfter)
}

// TestServiceAccountGetError tests the Reconcile method for the following use case
// GIVEN a request to reconcile an verrazzano resource
// WHEN a verrazzano resource has been applied
// THEN return error if failure getting ServiceAccount
func TestServiceAccountGetError(t *testing.T) {
	namespace := "verrazzano"
	name := "test"
	labels := map[string]string{"label1": "test"}
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	// Expect a call to get the verrazzano resource.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: name}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, verrazzano *vzapi.Verrazzano) error {
			verrazzano.TypeMeta = metav1.TypeMeta{
				APIVersion: "install.verrazzano.io/v1alpha1",
				Kind:       "Verrazzano"}
			verrazzano.ObjectMeta = metav1.ObjectMeta{
				Namespace: name.Namespace,
				Name:      name.Name,
				Labels:    labels}
			return nil
		})

	// Expect a call to get the ServiceAccount - return a failure error
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: getServiceAccountName(name)}, gomock.Not(gomock.Nil())).
		Return(errors.NewBadRequest("failed to get ServiceAccount"))

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(mock)
	result, err := reconciler.Reconcile(request)

	// Validate the results
	mocker.Finish()
	asserts.EqualError(err, "failed to get ServiceAccount")
	asserts.Equal(false, result.Requeue)
	asserts.Equal(time.Duration(0), result.RequeueAfter)
}

// TestServiceAccountCreateError tests the Reconcile method for the following use case
// GIVEN a request to reconcile an verrazzano resource
// WHEN a there is a failure creating a ServiceAccount
// THEN return error
func TestServiceAccountCreateError(t *testing.T) {
	namespace := "verrazzano"
	name := "test"
	labels := map[string]string{"label1": "test"}
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	// Expect a call to get the verrazzano resource.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: name}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, verrazzano *vzapi.Verrazzano) error {
			verrazzano.TypeMeta = metav1.TypeMeta{
				APIVersion: "install.verrazzano.io/v1alpha1",
				Kind:       "Verrazzano"}
			verrazzano.ObjectMeta = metav1.ObjectMeta{
				Namespace: name.Namespace,
				Name:      name.Name,
				Labels:    labels}
			return nil
		})

	// Expect a call to get the ServiceAccount - return not found
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: getServiceAccountName(name)}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: namespace, Resource: "ServiceAccount"}, name))

	// Expect a call to create the ServiceAccount - return failure
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		Return(errors.NewBadRequest("failed to create ServiceAccount"))

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(mock)
	result, err := reconciler.Reconcile(request)

	// Validate the results
	mocker.Finish()
	asserts.EqualError(err, "failed to create ServiceAccount")
	asserts.Equal(false, result.Requeue)
	asserts.Equal(time.Duration(0), result.RequeueAfter)
}

// TestClusterRoleBindingGetError tests the Reconcile method for the following use case
// GIVEN a request to reconcile an verrazzano resource
// WHEN a there is an error getting the ClusterRoleBinding
// THEN return error
func TestClusterRoleBindingGetError(t *testing.T) {
	namespace := "verrazzano"
	name := "test"
	labels := map[string]string{"label1": "test"}
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	// Expect a call to get the verrazzano resource.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: name}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, verrazzano *vzapi.Verrazzano) error {
			verrazzano.TypeMeta = metav1.TypeMeta{
				APIVersion: "install.verrazzano.io/v1alpha1",
				Kind:       "Verrazzano"}
			verrazzano.ObjectMeta = metav1.ObjectMeta{
				Namespace: name.Namespace,
				Name:      name.Name,
				Labels:    labels}
			return nil
		})

	// Expect a call to get the ServiceAccount - return that it exists
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: getServiceAccountName(name)}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, serviceAccount *corev1.ServiceAccount) error {
			newSA := installjob.NewServiceAccount(name.Namespace, name.Name, "", labels)
			serviceAccount.ObjectMeta = newSA.ObjectMeta
			return nil
		})

	// Expect a call to get the ClusterRoleBinding - return a failure error
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "", Name: getClusterRoleBindingName(namespace, name)}, gomock.Not(gomock.Nil())).
		Return(errors.NewBadRequest("failed to get ClusterRoleBinding"))

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(mock)
	result, err := reconciler.Reconcile(request)

	// Validate the results
	mocker.Finish()
	asserts.EqualError(err, "failed to get ClusterRoleBinding")
	asserts.Equal(false, result.Requeue)
	asserts.Equal(time.Duration(0), result.RequeueAfter)
}

// TestClusterRoleBindingCreateError tests the Reconcile method for the following use case
// GIVEN a request to reconcile an verrazzano resource
// WHEN a there is a failure creating a ClusterRoleBinding
// THEN return error
func TestClusterRoleBindingCreateError(t *testing.T) {
	namespace := "verrazzano"
	name := "test"
	labels := map[string]string{"label1": "test"}
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	// Expect a call to get the verrazzano resource.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: name}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, verrazzano *vzapi.Verrazzano) error {
			verrazzano.TypeMeta = metav1.TypeMeta{
				APIVersion: "install.verrazzano.io/v1alpha1",
				Kind:       "Verrazzano"}
			verrazzano.ObjectMeta = metav1.ObjectMeta{
				Namespace: name.Namespace,
				Name:      name.Name,
				Labels:    labels}
			return nil
		})

	// Expect a call to get the ServiceAccount - return that it exists
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: getServiceAccountName(name)}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, serviceAccount *corev1.ServiceAccount) error {
			newSA := installjob.NewServiceAccount(name.Namespace, name.Name, "", labels)
			serviceAccount.ObjectMeta = newSA.ObjectMeta
			return nil
		})

	// Expect a call to get the ClusterRoleBinding - return not found
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "", Name: getClusterRoleBindingName(namespace, name)}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: namespace, Resource: "ClusterRoleBinding"}, name))

	// Expect a call to create the ClusterRoleBinding - return failure
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		Return(errors.NewBadRequest("failed to create ClusterRoleBinding"))

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(mock)
	result, err := reconciler.Reconcile(request)

	// Validate the results
	mocker.Finish()
	asserts.EqualError(err, "failed to create ClusterRoleBinding")
	asserts.Equal(false, result.Requeue)
	asserts.Equal(time.Duration(0), result.RequeueAfter)
}

// TestConfigMapGetError tests the Reconcile method for the following use case
// GIVEN a request to reconcile an verrazzano resource
// WHEN a there is an error getting the ConfigMap
// THEN return error
func TestConfigMapGetError(t *testing.T) {
	namespace := "verrazzano"
	name := "test"
	labels := map[string]string{"label1": "test"}
	var savedVerrazzano *vzapi.Verrazzano
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	// Expect a call to get the verrazzano resource.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: name}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, verrazzano *vzapi.Verrazzano) error {
			verrazzano.TypeMeta = metav1.TypeMeta{
				APIVersion: "install.verrazzano.io/v1alpha1",
				Kind:       "Verrazzano"}
			verrazzano.ObjectMeta = metav1.ObjectMeta{
				Namespace: name.Namespace,
				Name:      name.Name,
				Labels:    labels}
			savedVerrazzano = verrazzano
			return nil
		})

	// Expect a call to get the ServiceAccount - return that it exists
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: getServiceAccountName(name)}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, serviceAccount *corev1.ServiceAccount) error {
			newSA := installjob.NewServiceAccount(name.Namespace, name.Name, "", labels)
			serviceAccount.ObjectMeta = newSA.ObjectMeta
			return nil
		})

	// Expect a call to get the ClusterRoleBinding - return that it exists
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "", Name: getClusterRoleBindingName(namespace, name)}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, clusterRoleBinding *rbacv1.ClusterRoleBinding) error {
			crb := installjob.NewClusterRoleBinding(savedVerrazzano, name.Name, getServiceAccountName(name.Name))
			clusterRoleBinding.ObjectMeta = crb.ObjectMeta
			clusterRoleBinding.RoleRef = crb.RoleRef
			clusterRoleBinding.Subjects = crb.Subjects
			return nil
		})

	// Expect a call to get the ConfigMap - return a failure error
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: getConfigMapName(name)}, gomock.Not(gomock.Nil())).
		Return(errors.NewBadRequest("failed to get ConfigMap"))

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(mock)
	result, err := reconciler.Reconcile(request)

	// Validate the results
	mocker.Finish()
	asserts.EqualError(err, "failed to get ConfigMap")
	asserts.Equal(false, result.Requeue)
	asserts.Equal(time.Duration(0), result.RequeueAfter)
}

// TestConfigMapCreateError tests the Reconcile method for the following use case
// GIVEN a request to reconcile an verrazzano resource
// WHEN a there is a failure creating a ConfigMap
// THEN return error
func TestConfigMapCreateError(t *testing.T) {
	namespace := "verrazzano"
	name := "test"
	labels := map[string]string{"label1": "test"}
	var savedVerrazzano *vzapi.Verrazzano
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	// Expect a call to get the verrazzano resource.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: name}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, verrazzano *vzapi.Verrazzano) error {
			verrazzano.TypeMeta = metav1.TypeMeta{
				APIVersion: "install.verrazzano.io/v1alpha1",
				Kind:       "Verrazzano"}
			verrazzano.ObjectMeta = metav1.ObjectMeta{
				Namespace: name.Namespace,
				Name:      name.Name,
				Labels:    labels}
			savedVerrazzano = verrazzano
			return nil
		})

	// Expect a call to get the ServiceAccount - return that it exists
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: getServiceAccountName(name)}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, serviceAccount *corev1.ServiceAccount) error {
			newSA := installjob.NewServiceAccount(name.Namespace, name.Name, "", labels)
			serviceAccount.ObjectMeta = newSA.ObjectMeta
			return nil
		})

	// Expect a call to get the ClusterRoleBinding - return that it exists
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "", Name: getClusterRoleBindingName(namespace, name)}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, clusterRoleBinding *rbacv1.ClusterRoleBinding) error {
			crb := installjob.NewClusterRoleBinding(savedVerrazzano, name.Name, getServiceAccountName(name.Name))
			clusterRoleBinding.ObjectMeta = crb.ObjectMeta
			clusterRoleBinding.RoleRef = crb.RoleRef
			clusterRoleBinding.Subjects = crb.Subjects
			return nil
		})

	// Expect a call to get the ConfigMap - return not found
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: getConfigMapName(name)}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: namespace, Resource: "ConfigMap"}, name))

	// Expect a call to create the ConfigMap - return failure
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		Return(errors.NewBadRequest("failed to create ConfigMap"))

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(mock)
	result, err := reconciler.Reconcile(request)

	// Validate the results
	mocker.Finish()
	asserts.EqualError(err, "failed to create ConfigMap")
	asserts.Equal(false, result.Requeue)
	asserts.Equal(time.Duration(0), result.RequeueAfter)
}

// TestJobGetError tests the Reconcile method for the following use case
// GIVEN a request to reconcile an verrazzano resource
// WHEN a there is an error getting the Job
// THEN return error
func TestJobGetError(t *testing.T) {
	namespace := "verrazzano"
	name := "test"
	labels := map[string]string{"label1": "test"}
	var savedVerrazzano *vzapi.Verrazzano
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	// Expect a call to get the verrazzano resource.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: name}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, verrazzano *vzapi.Verrazzano) error {
			verrazzano.TypeMeta = metav1.TypeMeta{
				APIVersion: "install.verrazzano.io/v1alpha1",
				Kind:       "Verrazzano"}
			verrazzano.ObjectMeta = metav1.ObjectMeta{
				Namespace: name.Namespace,
				Name:      name.Name,
				Labels:    labels}
			savedVerrazzano = verrazzano
			return nil
		})

	// Expect a call to get the ServiceAccount - return that it exists
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: getServiceAccountName(name)}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, serviceAccount *corev1.ServiceAccount) error {
			newSA := installjob.NewServiceAccount(name.Namespace, name.Name, "", labels)
			serviceAccount.ObjectMeta = newSA.ObjectMeta
			return nil
		})

	// Expect a call to get the ClusterRoleBinding - return that it exists
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "", Name: getClusterRoleBindingName(namespace, name)}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, clusterRoleBinding *rbacv1.ClusterRoleBinding) error {
			crb := installjob.NewClusterRoleBinding(savedVerrazzano, name.Name, getServiceAccountName(name.Name))
			clusterRoleBinding.ObjectMeta = crb.ObjectMeta
			clusterRoleBinding.RoleRef = crb.RoleRef
			clusterRoleBinding.Subjects = crb.Subjects
			return nil
		})

	// Expect a call to get the ConfigMap - return that it exists
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: getConfigMapName(name)}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, configMap *corev1.ConfigMap) error {
			cm := installjob.NewConfigMap(name.Namespace, name.Name, labels)
			configMap.ObjectMeta = cm.ObjectMeta
			return nil
		})

	// Expect a call to get the Job - return a failure error
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: getInstallJobName(name)}, gomock.Not(gomock.Nil())).
		Return(errors.NewBadRequest("failed to get Job"))

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(mock)
	result, err := reconciler.Reconcile(request)

	// Validate the results
	mocker.Finish()
	asserts.EqualError(err, "failed to get Job")
	asserts.Equal(false, result.Requeue)
	asserts.Equal(time.Duration(0), result.RequeueAfter)
}

// TestJobCreateError tests the Reconcile method for the following use case
// GIVEN a request to reconcile an verrazzano resource
// WHEN a there is a failure creating a Job
// THEN return error
func TestJobCreateError(t *testing.T) {
	namespace := "verrazzano"
	name := "test"
	labels := map[string]string{"label1": "test"}
	var savedVerrazzano *vzapi.Verrazzano
	asserts := assert.New(t)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	asserts.NotNil(mockStatus)

	// Expect a call to get the verrazzano resource.
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: name}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, verrazzano *vzapi.Verrazzano) error {
			verrazzano.TypeMeta = metav1.TypeMeta{
				APIVersion: "install.verrazzano.io/v1alpha1",
				Kind:       "Verrazzano"}
			verrazzano.ObjectMeta = metav1.ObjectMeta{
				Namespace: name.Namespace,
				Name:      name.Name,
				Labels:    labels}
			savedVerrazzano = verrazzano
			return nil
		})

	// Expect a call to get the ServiceAccount - return that it exists
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: getServiceAccountName(name)}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, serviceAccount *corev1.ServiceAccount) error {
			newSA := installjob.NewServiceAccount(name.Namespace, name.Name, "", labels)
			serviceAccount.ObjectMeta = newSA.ObjectMeta
			return nil
		})

	// Expect a call to get the ClusterRoleBinding - return that it exists
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "", Name: getClusterRoleBindingName(namespace, name)}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, clusterRoleBinding *rbacv1.ClusterRoleBinding) error {
			crb := installjob.NewClusterRoleBinding(savedVerrazzano, name.Name, getServiceAccountName(name.Name))
			clusterRoleBinding.ObjectMeta = crb.ObjectMeta
			clusterRoleBinding.RoleRef = crb.RoleRef
			clusterRoleBinding.Subjects = crb.Subjects
			return nil
		})

	// Expect a call to get the ConfigMap - return that it exists
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: getConfigMapName(name)}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, configMap *corev1.ConfigMap) error {
			cm := installjob.NewConfigMap(name.Namespace, name.Name, labels)
			configMap.ObjectMeta = cm.ObjectMeta
			return nil
		})

	// Expect a call to get the Job - return not found
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: getInstallJobName(name)}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: namespace, Resource: "Job"}, name))

	// Expect a call to create the Job - return failure
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		Return(errors.NewBadRequest("failed to create Job"))

	// Create and make the request
	request := newRequest(namespace, name)
	reconciler := newVerrazzanoReconciler(mock)
	result, err := reconciler.Reconcile(request)

	// Validate the results
	mocker.Finish()
	asserts.EqualError(err, "failed to create Job")
	asserts.Equal(false, result.Requeue)
	asserts.Equal(time.Duration(0), result.RequeueAfter)
}

// newScheme creates a new scheme that includes this package's object to use for testing
func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	//_ = clientgoscheme.AddToScheme(scheme)
	//_ = core.AddToScheme(scheme)
	vzapi.AddToScheme(scheme)
	return scheme
}

// newRequest creates a new reconciler request for testing
// namespace - The namespace to use in the request
// name - The name to use in the request
func newRequest(namespace string, name string) ctrl.Request {
	return ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: namespace,
			Name:      name}}
}

// newVerrazzanoReconciler creates a new reconciler for testing
// c - The Kerberos client to inject into the reconciler
func newVerrazzanoReconciler(c client.Client) VerrazzanoReconciler {
	log := ctrl.Log.WithName("test")
	scheme := newScheme()
	reconciler := VerrazzanoReconciler{
		Client: c,
		Log:    log,
		Scheme: scheme}
	return reconciler
}
