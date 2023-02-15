// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/constants"
	ctrlerrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	constants2 "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/monitor"
	fakemonitor "github.com/verrazzano/verrazzano/platform-operator/internal/monitor/fake"
	admv1 "k8s.io/api/admissionregistration/v1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	v12 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var nonRanNSName = "local-not-rancher"
var rancherNSName = "local"
var nonRancherNs v1.Namespace = v1.Namespace{
	ObjectMeta: metav1.ObjectMeta{
		Name: nonRanNSName,
	},
}
var rancherNs v1.Namespace = v1.Namespace{
	ObjectMeta: metav1.ObjectMeta{
		Name: rancherNSName,
	},
}

// TestPostUninstall tests the post uninstall process for Rancher
// GIVEN a call to postUninstall
// WHEN the background uninstall goroutine is not running yet
// THEN the post-uninstall starts a new attempt and returns a RetryableError to requeue
func TestPostUninstall(t *testing.T) {
	a := assert.New(t)
	vz := v1alpha1.Verrazzano{}
	testObjects := []clipkg.Object{
		&nonRancherNs,
		&rancherNs,
	}

	c := fake.NewClientBuilder().WithScheme(getScheme()).WithObjects(testObjects...).Build()
	ctx := spi.NewFakeContext(c, &vz, nil, false)

	expectedErr := ctrlerrors.RetryableError{Source: ComponentName}
	forkPostUninstallFunc = func(_ spi.ComponentContext, _ monitor.BackgroundProcessMonitor) error {
		return expectedErr
	}
	defer func() { forkPostUninstallFunc = forkPostUninstall }()

	monitor := &fakemonitor.BackgroundProcessMonitorType{Result: true, Running: false}
	err := postUninstall(ctx, monitor)
	a.Equal(expectedErr, err, "Uninstall returned an unexpected error")
}

// TestBackgroundPostUninstallCompletedSuccessfully tests the post uninstall process for Rancher
// GIVEN a call to postUninstall
// WHEN the goroutine is not finished running, but has a successful result in the monitor and no error
// THEN postUninstall returns nil without calling the forkPostUninstall function
func TestBackgroundPostUninstallCompletedSuccessfully(t *testing.T) {
	a := assert.New(t)
	vz := v1alpha1.Verrazzano{}
	testObjects := []clipkg.Object{
		&nonRancherNs,
		&rancherNs,
	}

	c := fake.NewClientBuilder().WithScheme(getScheme()).WithObjects(testObjects...).Build()
	ctx := spi.NewFakeContext(c, &vz, nil, false)

	forkPostUninstallFunc = func(_ spi.ComponentContext, _ monitor.BackgroundProcessMonitor) error {
		a.Fail("Unexpected call to forkPostUninstall() function")
		return nil
	}
	defer func() { forkPostUninstallFunc = forkPostUninstall }()

	monitor := &fakemonitor.BackgroundProcessMonitorType{Result: true, Running: true}
	err := postUninstall(ctx, monitor)
	a.NoError(err)
}

// TestPostUninstall tests the post uninstall process for Rancher
// GIVEN a call to postUninstall
// WHEN the the monitor goroutine failed to successfully complete
// THEN the postUninstall function calls the forkPostUninstall function and returns a RetryableError
func TestBackgroundPostUninstallRetryOnFailure(t *testing.T) {
	a := assert.New(t)
	vz := v1alpha1.Verrazzano{}
	testObjects := []clipkg.Object{
		&nonRancherNs,
		&rancherNs,
	}

	c := fake.NewClientBuilder().WithScheme(getScheme()).WithObjects(testObjects...).Build()
	ctx := spi.NewFakeContext(c, &vz, nil, false)

	forkFuncCalled := false
	expectedErr := ctrlerrors.RetryableError{Source: ComponentName}
	forkPostUninstallFunc = func(_ spi.ComponentContext, _ monitor.BackgroundProcessMonitor) error {
		forkFuncCalled = true
		return expectedErr
	}
	defer func() { forkPostUninstallFunc = forkPostUninstall }()

	monitor := &fakemonitor.BackgroundProcessMonitorType{Result: false, Running: true}
	err := postUninstall(ctx, monitor)
	a.True(forkFuncCalled)
	a.Equal(expectedErr, err, "Uninstall returned an unexpected error")
}

// Test_forkPostUninstallSuccess tests the forkPostUninstall function
// GIVEN a call to rancher.forkPostUninstall()
// WHEN when the monitor install successfully runs the rancher uninstall tool
// THEN retryerrors are returned until the goroutine completes and sends a success message
func Test_forkPostUninstallSuccess(t *testing.T) {
	a := assert.New(t)
	vz := v1alpha1.Verrazzano{}
	testObjects := []clipkg.Object{
		&nonRancherNs,
		&rancherNs,
	}

	c := fake.NewClientBuilder().WithScheme(getScheme()).WithObjects(testObjects...).Build()
	ctx := spi.NewFakeContext(c, &vz, nil, false)

	postUninstallFunc = func(ctx spi.ComponentContext) error {
		return nil
	}
	defer func() { postUninstallFunc = invokeRancherSystemToolAndCleanup }()

	var monitor = &monitor.BackgroundProcessMonitorType{ComponentName: ComponentName}
	err := forkPostUninstall(ctx, monitor)
	a.Equal(ctrlerrors.RetryableError{Source: ComponentName}, err)
	for i := 0; i < 100; i++ {
		result, retryError := monitor.CheckResult()
		if retryError != nil {
			t.Log("Waiting for result...")
			time.Sleep(100 * time.Millisecond)
			continue
		}
		a.True(result)
		a.Nil(retryError)
		return
	}
	a.Fail("Did not detect completion in time")
}

// Test_forkPostUninstallFailure tests the forkPostUninstall function
// GIVEN a call to rancher.forkPostUninstall()
// WHEN when the monitor install unsuccessfully runs rancher post-uninstall
// THEN retryerrors are returned until the goroutine completes and sends a failure message when the rancher uninstall tool fails
func Test_forkPostUninstallFailure(t *testing.T) {
	a := assert.New(t)
	vz := v1alpha1.Verrazzano{}
	testObjects := []clipkg.Object{
		&nonRancherNs,
		&rancherNs,
	}

	c := fake.NewClientBuilder().WithScheme(getScheme()).WithObjects(testObjects...).Build()
	ctx := spi.NewFakeContext(c, &vz, nil, false)

	postUninstallFunc = func(ctx spi.ComponentContext) error {
		return fmt.Errorf("Unexpected error on uninstall")
	}
	defer func() { postUninstallFunc = invokeRancherSystemToolAndCleanup }()

	var monitor = &monitor.BackgroundProcessMonitorType{ComponentName: ComponentName}
	err := forkPostUninstall(ctx, monitor)
	a.Equal(ctrlerrors.RetryableError{Source: ComponentName}, err)
	for i := 0; i < 100; i++ {
		result, retryError := monitor.CheckResult()
		if retryError != nil {
			t.Log("Waiting for result...")
			time.Sleep(100 * time.Millisecond)
			continue
		}
		a.False(result)
		a.Nil(retryError)
		return
	}
	a.Fail("Did not detect completion in time")
}

// TestInvokeRancherSystemToolAndCleanup tests the deletion of objects in the post-uninstall process for Rancher
// GIVEN a call to invokeRancherSystemToolAndCleanup
// WHEN the objects exist in the cluster
// THEN all objects are deleted
func TestInvokeRancherSystemToolAndCleanup(t *testing.T) {
	a := assert.New(t)
	vz := v1alpha1.Verrazzano{}

	rancherNSName2 := "fleet-rancher"
	rancherCrName := "proxy-1234"
	mwcName := "mutating-webhook-configuration"
	vwcName := "validating-webhook-configuration"
	pvName := "pvc-12345"
	pv2Name := "ocid1.volume.oc1.ca-toronto-1.12345"
	rbName := "rb-test"
	nonRancherRBName := "testrb"
	randPV := "randomPV"
	randCR := "randomCR"
	randCRB := "randomCRB"
	rancherCRDName := "definitelyrancher.cattle.io"
	nonRancherCRDName := "other.cattle"

	rancherNs2 := v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: rancherNSName2,
		},
	}
	mutWebhook := admv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: webhookName,
		},
	}
	mutWebhook2 := admv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: mwcName,
		},
	}
	valWebhook := admv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: webhookName,
		},
	}
	valWebhook2 := admv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: vwcName,
		},
	}
	crRancher := rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: rancherCrName,
		},
	}
	crbRancher := rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: webhookName,
		},
	}
	crNotRancher := rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: randCR,
		},
	}
	crbNotRancher := rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: randCRB,
		},
	}
	rbRancher := rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: rbName,
		},
	}
	rbNotRancher := rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: nonRancherRBName,
		},
	}
	controllerCM := v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      controllerCMName,
			Namespace: constants.KubeSystem,
		},
	}
	lockCM := v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      lockCMName,
			Namespace: constants.KubeSystem,
		},
	}
	rancherPV := v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: pvName,
		},
	}
	rancherPV2 := v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: pv2Name,
		},
	}
	nonRancherPV := v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: randPV,
		},
	}

	delTimestamp := metav1.NewTime(time.Now())
	crdAPIVersion := "apiextensions.k8s.io/v1"
	crdKind := "CustomResourceDefinition"
	rancherClusterCRD := v12.CustomResourceDefinition{
		TypeMeta: metav1.TypeMeta{
			Kind:       crdKind,
			APIVersion: crdAPIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:              rancherCRDName,
			Finalizers:        []string{"somefinalizer"},
			DeletionTimestamp: &delTimestamp,
		},
		Spec: v12.CustomResourceDefinitionSpec{
			Group: "management.cattle.io",
			Names: v12.CustomResourceDefinitionNames{
				Kind: "Setting",
			},
			Scope: "Cluster",
			Versions: []v12.CustomResourceDefinitionVersion{
				{
					Name: "v3",
				},
			},
		},
	}
	rancherNamespacedCRD := v12.CustomResourceDefinition{
		TypeMeta: metav1.TypeMeta{
			Kind:       crdKind,
			APIVersion: crdAPIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "projects.management.cattle.io",
		},
		Spec: v12.CustomResourceDefinitionSpec{
			Group: "management.cattle.io",
			Names: v12.CustomResourceDefinitionNames{
				Kind: "Project",
			},
			Scope: "Namespaced",
			Versions: []v12.CustomResourceDefinitionVersion{
				{
					Name: "v3",
				},
			},
		},
	}
	nonRancherCRD := v12.CustomResourceDefinition{
		TypeMeta: metav1.TypeMeta{
			Kind:       crdKind,
			APIVersion: crdAPIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: nonRancherCRDName,
		},
		Spec: v12.CustomResourceDefinitionSpec{
			Group:                 "cattle.io",
			Names:                 v12.CustomResourceDefinitionNames{},
			Scope:                 "",
			Versions:              nil,
			Conversion:            nil,
			PreserveUnknownFields: false,
		},
	}
	settingCR := unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "management.cattle.io/v3",
			"kind":       "Setting",
			"metadata": map[string]interface{}{
				"name": "cr-name",
			},
		},
	}
	projectCR := unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "management.cattle.io/v3",
			"kind":       "Project",
			"metadata": map[string]interface{}{
				"namespace": "cr-namespace",
				"name":      "cr-name",
			},
		},
	}
	rancherCleanupJob := batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: rancherCleanupJobNamespace,
			Name:      rancherCleanupJobName,
		},
		Status: batchv1.JobStatus{
			Conditions: []batchv1.JobCondition{
				{
					Type: batchv1.JobComplete,
				},
			},
		},
	}
	tests := []struct {
		name           string
		objects        []clipkg.Object
		nonRancherTest bool
	}{
		{
			name: "test empty cluster",
			objects: []clipkg.Object{
				&rancherCleanupJob,
			},
		},
		{
			name: "test non Rancher ns",
			objects: []clipkg.Object{
				&rancherCleanupJob,
				&nonRancherNs,
			},
		},
		{
			name: "test Rancher ns",
			objects: []clipkg.Object{
				&rancherCleanupJob,
				&nonRancherNs,
				&rancherNs,
			},
		},
		{
			name: "test multiple Rancher ns",
			objects: []clipkg.Object{
				&rancherCleanupJob,
				&nonRancherNs,
				&rancherNs,
				&rancherNs2,
			},
		},
		{
			name: "test mutating webhook",
			objects: []clipkg.Object{
				&rancherCleanupJob,
				&nonRancherNs,
				&rancherNs,
				&rancherNs2,
				&mutWebhook,
				&mutWebhook2,
			},
		},
		{
			name: "test validating webhook",
			objects: []clipkg.Object{
				&rancherCleanupJob,
				&nonRancherNs,
				&rancherNs,
				&rancherNs2,
				&mutWebhook,
				&valWebhook,
				&valWebhook2,
			},
		},
		{
			name: "test CR and CRB",
			objects: []clipkg.Object{
				&rancherCleanupJob,
				&nonRancherNs,
				&rancherNs,
				&rancherNs2,
				&mutWebhook,
				&valWebhook,
				&crRancher,
				&crbRancher,
			},
		},
		{
			name: "test non Rancher components",
			objects: []clipkg.Object{
				&rancherCleanupJob,
				&nonRancherNs,
				&crNotRancher,
				&crbNotRancher,
				&nonRancherPV,
				&rbNotRancher,
				&nonRancherCRD,
			},
			nonRancherTest: true,
		},
		{
			name: "test config maps",
			objects: []clipkg.Object{
				&rancherCleanupJob,
				&nonRancherNs,
				&rancherNs,
				&rancherNs2,
				&mutWebhook,
				&valWebhook,
				&crRancher,
				&crbRancher,
				&crNotRancher,
				&crbNotRancher,
				&controllerCM,
				&lockCM,
			},
		},
		{
			name: "test persistent volume",
			objects: []clipkg.Object{
				&rancherCleanupJob,
				&nonRancherNs,
				&rancherNs,
				&rancherNs2,
				&mutWebhook,
				&valWebhook,
				&crRancher,
				&crbRancher,
				&crNotRancher,
				&crbNotRancher,
				&controllerCM,
				&lockCM,
				&rancherPV,
				&rancherPV2,
			},
		},
		{
			name: "test clusterRole binding",
			objects: []clipkg.Object{
				&rancherCleanupJob,
				&nonRancherNs,
				&rancherNs,
				&rancherNs2,
				&mutWebhook,
				&valWebhook,
				&crRancher,
				&crbRancher,
				&crNotRancher,
				&crbNotRancher,
				&controllerCM,
				&lockCM,
				&rancherPV,
				&rancherPV2,
				&rbRancher,
			},
		},
		{
			name: "test CRD finalizer cleanup",
			objects: []clipkg.Object{
				&rancherCleanupJob,
				&rancherClusterCRD,
			},
		},
		{
			name: "test Rancher CR cleanup",
			objects: []clipkg.Object{
				&rancherCleanupJob,
				&rancherClusterCRD,
				&settingCR,
				&rancherNamespacedCRD,
				&projectCR,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().WithScheme(getScheme()).WithObjects(tt.objects...).Build()
			ctx := spi.NewFakeContext(c, &vz, nil, false)

			crd1 := v12.CustomResourceDefinition{}
			c.Get(context.TODO(), types.NamespacedName{Name: rancherCRDName}, &crd1)

			err := invokeRancherSystemToolAndCleanup(ctx)
			a.NoError(err)

			// MutatingWebhookConfigurations should not exist
			err = c.Get(context.TODO(), types.NamespacedName{Name: webhookName}, &admv1.MutatingWebhookConfiguration{})
			a.True(apierrors.IsNotFound(err))
			err = c.Get(context.TODO(), types.NamespacedName{Name: mwcName}, &admv1.MutatingWebhookConfiguration{})
			a.True(apierrors.IsNotFound(err))
			// ValidatingWebhookConfigurations should not exist
			err = c.Get(context.TODO(), types.NamespacedName{Name: webhookName}, &admv1.ValidatingWebhookConfiguration{})
			a.True(apierrors.IsNotFound(err))
			err = c.Get(context.TODO(), types.NamespacedName{Name: vwcName}, &admv1.ValidatingWebhookConfiguration{})
			a.True(apierrors.IsNotFound(err))
			// ClusterRole should not exist
			err = c.Get(context.TODO(), types.NamespacedName{Name: rancherCrName}, &rbacv1.ClusterRole{})
			a.True(apierrors.IsNotFound(err))
			// ClusterRoleBinding should not exist
			err = c.Get(context.TODO(), types.NamespacedName{Name: webhookName}, &rbacv1.ClusterRoleBinding{})
			a.True(apierrors.IsNotFound(err))
			if tt.nonRancherTest {
				// Verify that non-Rancher components did not get cleaned up
				err = c.Get(context.TODO(), types.NamespacedName{Name: randCR}, &rbacv1.ClusterRole{})
				a.Nil(err)
				err = c.Get(context.TODO(), types.NamespacedName{Name: randCRB}, &rbacv1.ClusterRoleBinding{})
				a.Nil(err)
				err = c.Get(context.TODO(), types.NamespacedName{Name: randPV}, &v1.PersistentVolume{})
				a.Nil(err)
				err = c.Get(context.TODO(), types.NamespacedName{Name: nonRancherRBName}, &rbacv1.RoleBinding{})
				a.Nil(err)
				err = c.Get(context.TODO(), types.NamespacedName{Name: nonRancherCRDName}, &v12.CustomResourceDefinition{})
				a.Nil(err)
			}
			// ConfigMaps should not exist
			err = c.Get(context.TODO(), types.NamespacedName{Name: controllerCMName}, &v1.ConfigMap{})
			a.True(apierrors.IsNotFound(err))
			err = c.Get(context.TODO(), types.NamespacedName{Name: lockCMName}, &v1.ConfigMap{})
			a.True(apierrors.IsNotFound(err))
			// Persistent volume should not exist
			err = c.Get(context.TODO(), types.NamespacedName{Name: pvName}, &v1.PersistentVolume{})
			a.True(apierrors.IsNotFound(err))
			err = c.Get(context.TODO(), types.NamespacedName{Name: pv2Name}, &v1.PersistentVolume{})
			a.True(apierrors.IsNotFound(err))
			// Role Binding should not exist
			err = c.Get(context.TODO(), types.NamespacedName{Name: rbName}, &rbacv1.RoleBinding{})
			a.True(apierrors.IsNotFound(err))
			// Rancher CRD finalizer should have been deleted which should cause it to go away
			// since it had a deletion timestamp
			crd := v12.CustomResourceDefinition{}
			err = c.Get(context.TODO(), types.NamespacedName{Name: rancherCRDName}, &crd)
			a.True(apierrors.IsNotFound(err))
		})
	}
}

// TestIsRancherNamespace tests the namespace belongs to Rancher
// GIVEN a call to isRancherNamespace
// WHEN the namespace belings to Rancher or not
// THEN we see true if it is and false if not
func TestIsRancherNamespace(t *testing.T) {
	a := assert.New(t)

	a.True(isRancherNamespace(&v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cattle-system",
		},
	}))
	a.True(isRancherNamespace(&v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "p-12345",
			Annotations: map[string]string{
				rancherSysNS: "true",
			},
		},
	}))
	a.True(isRancherNamespace(&v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "local",
			Annotations: map[string]string{
				rancherSysNS: "false",
			},
		},
	}))
	a.False(isRancherNamespace(&v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "p-12345",
			Annotations: map[string]string{
				rancherSysNS: "false",
			},
		},
	}))
	a.False(isRancherNamespace(&v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "p-12345",
		},
	}))
}

// TestCleanupJob tests the creation and deletion of the rancher-cleanup job
// GIVEN a call to runCleanupJob
// WHEN no cleanup job exists
// THEN expect a cleanup job to be created
// GIVEN a call to deleteCleanupJob
// WHEN a job already exists
// THEN expect the job to be deleted
func TestCleanupJob(t *testing.T) {
	a := assert.New(t)
	vz := v1alpha1.Verrazzano{}

	c := fake.NewClientBuilder().WithScheme(getScheme()).Build()
	ctx := spi.NewFakeContext(c, &vz, nil, false)
	config.SetDefaultBomFilePath("../../../../verrazzano-bom.json")
	setCleanupJobYamlPath("../../../../thirdparty/manifests/rancher-cleanup/rancher-cleanup.yaml")

	// Expect the job to get created
	err := runCleanupJob(ctx)
	a.Error(err)
	a.Contains(err.Error(), "waiting for job")
	job := batchv1.Job{}
	err = ctx.Client().Get(context.TODO(), types.NamespacedName{Namespace: rancherCleanupJobNamespace, Name: rancherCleanupJobName}, &job)
	a.NoError(err)

	// Expect the job to get deleted
	deleteCleanupJob(ctx)
	err = ctx.Client().Get(context.TODO(), types.NamespacedName{Namespace: rancherCleanupJobNamespace, Name: rancherCleanupJobName}, &job)
	a.Error(err)
	a.True(apierrors.IsNotFound(err))
}

// TestDeleteRancherFinalizers
// GIVEN cluster resources with rancher finalizers
//
//	WHEN deleteRancherFinalizers is called
//	THEN expect those resources to have their finalizers removed
func TestDeleteRancherFinalizers(t *testing.T) {
	a := assert.New(t)
	vz := v1alpha1.Verrazzano{}

	// Namespaces
	ns1 := newNamespace(constants2.KeycloakNamespace)
	ns2 := newNamespace(constants2.VerrazzanoSystemNamespace)

	// ClusterRole that contains Rancher finalizers
	cr1 := newClusterRole("cr1", "default", []string{"test", "test.cattle.io"})
	crb1 := newClusterRoleBinding("crb1", "default", []string{"test", "test.cattle.io"})

	// Role and RoleBinding that does not contain any Rancher finalizers
	r1 := newRole("rb1", constants2.VerrazzanoSystemNamespace, []string{"test"})
	rb1 := newRoleBinding("rb1", constants2.VerrazzanoSystemNamespace, []string{"test"})

	// Role and RoleBinding that does contain Rancher finalizers
	r2 := newRole("rb2", constants2.VerrazzanoSystemNamespace, []string{"test", "test.cattle.io"})
	rb2 := newRoleBinding("rb2", constants2.VerrazzanoSystemNamespace, []string{"test", "test.cattle.io"})

	// RoleBinding that does contain Rancher finalizers
	r3 := newRole("rb3", constants2.KeycloakNamespace, []string{"test", "test.cattle.io"})
	rb3 := newRoleBinding("rb3", constants2.KeycloakNamespace, []string{"test", "test.cattle.io"})

	c := fake.NewClientBuilder().WithScheme(getScheme()).WithObjects(ns1, ns2, cr1, crb1, r1, r2, r3, rb1, rb2, rb3).Build()
	ctx := spi.NewFakeContext(c, &vz, nil, false)
	deleteRancherFinalizers(ctx)

	var clusterRole = &rbacv1.ClusterRole{}
	var clusterRoleBinding = &rbacv1.ClusterRoleBinding{}
	var role = &rbacv1.Role{}
	var roleBinding = &rbacv1.RoleBinding{}

	type testCase struct {
		clipkg.Object
		expectedFinalizers int
		destObject         clipkg.Object
	}
	testCases := []testCase{
		{Object: cr1, expectedFinalizers: 0, destObject: clusterRole},
		{Object: crb1, expectedFinalizers: 0, destObject: clusterRoleBinding},
		{Object: rb1, expectedFinalizers: 1, destObject: roleBinding},
		{Object: rb2, expectedFinalizers: 0, destObject: roleBinding},
		{Object: rb3, expectedFinalizers: 0, destObject: roleBinding},
		{Object: r1, expectedFinalizers: 1, destObject: role},
		{Object: r2, expectedFinalizers: 0, destObject: role},
		{Object: r3, expectedFinalizers: 0, destObject: role},
	}

	for _, test := range testCases {
		err := ctx.Client().Get(context.TODO(), types.NamespacedName{Namespace: test.GetNamespace(), Name: test.GetName()}, test.destObject)
		a.NoError(err)
		a.Equal(test.expectedFinalizers, len(test.destObject.GetFinalizers()))
	}
}

func newNamespace(name string) *v1.Namespace {
	return &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}
func newClusterRole(name string, namespace string, finalizers []string) *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:  namespace,
			Name:       name,
			Finalizers: finalizers,
		},
	}
}

func newClusterRoleBinding(name string, namespace string, finalizers []string) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:  namespace,
			Name:       name,
			Finalizers: finalizers,
		},
	}
}

func newRole(name string, namespace string, finalizers []string) *rbacv1.Role {
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:  namespace,
			Name:       name,
			Finalizers: finalizers,
		},
	}
}

func newRoleBinding(name string, namespace string, finalizers []string) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:  namespace,
			Name:       name,
			Finalizers: finalizers,
		},
	}
}
