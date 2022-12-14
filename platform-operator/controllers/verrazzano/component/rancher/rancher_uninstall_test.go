// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"context"
	"fmt"
	"testing"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/stretchr/testify/assert"
	asserts "github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/constants"
	ctrlerrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	admv1 "k8s.io/api/admissionregistration/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	v12 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type fakeMonitor struct {
	result  bool
	err     error
	running bool
}

func (f *fakeMonitor) run(args postUninstallRoutineParams) {}
func (f *fakeMonitor) checkResult() (bool, error)          { return f.result, f.err }
func (f *fakeMonitor) reset()                              {}
func (f *fakeMonitor) init()                               {}
func (f *fakeMonitor) sendResult(r bool)                   {}
func (f *fakeMonitor) isRunning() bool                     { return f.running }
var _ postUninstallMonitor = &fakeMonitor{}

var nonRanNSName string = "local-not-rancher"
var rancherNSName string = "local"
var rancherNSName2 string = "fleet-rancher"
var rancherCrName string = "proxy-1234"
var mwcName string = "mutating-webhook-configuration"
var vwcName string = "validating-webhook-configuration"
var pvName string = "pvc-12345"
var pv2Name string = "ocid1.volume.oc1.ca-toronto-1.12345"
var rbName string = "rb-test"
var nonRancherRBName string = "testrb"
var randPV string = "randomPV"
var randCR string = "randomCR"
var randCRB string = "randomCRB"
var rancherCRDName string = "definitelyrancher.cattle.io"
var nonRancherCRDName string = "other.cattle"

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
var rancherNs2 v1.Namespace = v1.Namespace{
	ObjectMeta: metav1.ObjectMeta{
		Name: rancherNSName2,
	},
}
var mutWebhook admv1.MutatingWebhookConfiguration = admv1.MutatingWebhookConfiguration{
	ObjectMeta: metav1.ObjectMeta{
		Name: webhookName,
	},
}
var mutWebhook2 admv1.MutatingWebhookConfiguration = admv1.MutatingWebhookConfiguration{
	ObjectMeta: metav1.ObjectMeta{
		Name: mwcName,
	},
}
var valWebhook admv1.ValidatingWebhookConfiguration = admv1.ValidatingWebhookConfiguration{
	ObjectMeta: metav1.ObjectMeta{
		Name: webhookName,
	},
}
var valWebhook2 admv1.ValidatingWebhookConfiguration = admv1.ValidatingWebhookConfiguration{
	ObjectMeta: metav1.ObjectMeta{
		Name: vwcName,
	},
}
var crRancher rbacv1.ClusterRole = rbacv1.ClusterRole{
	ObjectMeta: metav1.ObjectMeta{
		Name: rancherCrName,
	},
}
var crbRancher rbacv1.ClusterRoleBinding = rbacv1.ClusterRoleBinding{
	ObjectMeta: metav1.ObjectMeta{
		Name: webhookName,
	},
}
var crNotRancher rbacv1.ClusterRole = rbacv1.ClusterRole{
	ObjectMeta: metav1.ObjectMeta{
		Name: randCR,
	},
}
var crbNotRancher rbacv1.ClusterRoleBinding = rbacv1.ClusterRoleBinding{
	ObjectMeta: metav1.ObjectMeta{
		Name: randCRB,
	},
}
var rbRancher rbacv1.RoleBinding = rbacv1.RoleBinding{
	ObjectMeta: metav1.ObjectMeta{
		Name: rbName,
	},
}
var rbNotRancher rbacv1.RoleBinding = rbacv1.RoleBinding{
	ObjectMeta: metav1.ObjectMeta{
		Name: nonRancherRBName,
	},
}
var controllerCM v1.ConfigMap = v1.ConfigMap{
	ObjectMeta: metav1.ObjectMeta{
		Name:      controllerCMName,
		Namespace: constants.KubeSystem,
	},
}
var lockCM v1.ConfigMap = v1.ConfigMap{
	ObjectMeta: metav1.ObjectMeta{
		Name:      lockCMName,
		Namespace: constants.KubeSystem,
	},
}
var rancherPV v1.PersistentVolume = v1.PersistentVolume{
	ObjectMeta: metav1.ObjectMeta{
		Name: pvName,
	},
}
var rancherPV2 v1.PersistentVolume = v1.PersistentVolume{
	ObjectMeta: metav1.ObjectMeta{
		Name: pv2Name,
	},
}
var nonRancherPV v1.PersistentVolume = v1.PersistentVolume{
	ObjectMeta: metav1.ObjectMeta{
		Name: randPV,
	},
}
var delTimestamp metav1.Time = metav1.NewTime(time.Now())
var crdAPIVersion string = "apiextensions.k8s.io/v1"
var crdKind string = "CustomResourceDefinition"
var rancherClusterCRD v12.CustomResourceDefinition = v12.CustomResourceDefinition{
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
var rancherNamespacedCRD v12.CustomResourceDefinition = v12.CustomResourceDefinition{
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
var nonRancherCRD v12.CustomResourceDefinition = v12.CustomResourceDefinition{
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
var settingCR unstructured.Unstructured = unstructured.Unstructured{
	Object: map[string]interface{}{
		"apiVersion": "management.cattle.io/v3",
		"kind":       "Setting",
		"metadata": map[string]interface{}{
			"name": "cr-name",
		},
	},
}
var projectCR unstructured.Unstructured = unstructured.Unstructured{
	Object: map[string]interface{}{
		"apiVersion": "management.cattle.io/v3",
		"kind":       "Project",
		"metadata": map[string]interface{}{
			"namespace": "cr-namespace",
			"name":      "cr-name",
		},
	},
}

type testStruct struct {
	name           string
	objects        []clipkg.Object
	nonRancherTest bool
}

var tests []testStruct = []testStruct{
	{
		name: "test empty cluster",
	},
	{
		name: "test non Rancher ns",
		objects: []clipkg.Object{
			&nonRancherNs,
		},
	},
	{
		name: "test Rancher ns",
		objects: []clipkg.Object{
			&nonRancherNs,
			&rancherNs,
		},
	},
	{
		name: "test multiple Rancher ns",
		objects: []clipkg.Object{
			&nonRancherNs,
			&rancherNs,
			&rancherNs2,
		},
	},
	{
		name: "test mutating webhook",
		objects: []clipkg.Object{
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
			&rancherClusterCRD,
		},
	},
	{
		name: "test Rancher CR cleanup",
		objects: []clipkg.Object{
			&rancherClusterCRD,
			&settingCR,
			&rancherNamespacedCRD,
			&projectCR,
		},
	},
}

// TODO: write description. Should cover the old TestPostUninstall
func TestPostUninstallAllObjectsDeleted(t *testing.T) {
	// TODO: modify to make sense with changes
	assert := assert.New(t)
	vz := v1alpha1.Verrazzano{}

	setRancherSystemTool("echo")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().WithScheme(getScheme()).WithObjects(tt.objects...).Build()
			ctx := spi.NewFakeContext(c, &vz, nil, false)

			crd1 := v12.CustomResourceDefinition{}
			c.Get(context.TODO(), types.NamespacedName{Name: rancherCRDName}, &crd1)

			m := &postUninstallMonitorType{}
			err := postUninstall(ctx, m)
			assert.NoError(err)

			// MutatingWebhookConfigurations should not exist
			err = c.Get(context.TODO(), types.NamespacedName{Name: webhookName}, &admv1.MutatingWebhookConfiguration{})
			assert.True(apierrors.IsNotFound(err))
			err = c.Get(context.TODO(), types.NamespacedName{Name: mwcName}, &admv1.MutatingWebhookConfiguration{})
			assert.True(apierrors.IsNotFound(err))
			// ValidatingWebhookConfigurations should not exist
			err = c.Get(context.TODO(), types.NamespacedName{Name: webhookName}, &admv1.ValidatingWebhookConfiguration{})
			assert.True(apierrors.IsNotFound(err))
			err = c.Get(context.TODO(), types.NamespacedName{Name: vwcName}, &admv1.ValidatingWebhookConfiguration{})
			assert.True(apierrors.IsNotFound(err))
			// ClusterRole should not exist
			err = c.Get(context.TODO(), types.NamespacedName{Name: rancherCrName}, &rbacv1.ClusterRole{})
			assert.True(apierrors.IsNotFound(err))
			// ClusterRoleBinding should not exist
			err = c.Get(context.TODO(), types.NamespacedName{Name: webhookName}, &rbacv1.ClusterRoleBinding{})
			assert.True(apierrors.IsNotFound(err))
			if tt.nonRancherTest {
				// Verify that non-Rancher components did not get cleaned up
				err = c.Get(context.TODO(), types.NamespacedName{Name: randCR}, &rbacv1.ClusterRole{})
				assert.Nil(err)
				err = c.Get(context.TODO(), types.NamespacedName{Name: randCRB}, &rbacv1.ClusterRoleBinding{})
				assert.Nil(err)
				err = c.Get(context.TODO(), types.NamespacedName{Name: randPV}, &v1.PersistentVolume{})
				assert.Nil(err)
				err = c.Get(context.TODO(), types.NamespacedName{Name: nonRancherRBName}, &rbacv1.RoleBinding{})
				assert.Nil(err)
				err = c.Get(context.TODO(), types.NamespacedName{Name: nonRancherCRDName}, &v12.CustomResourceDefinition{})
				assert.Nil(err)
			}
			// ConfigMaps should not exist
			err = c.Get(context.TODO(), types.NamespacedName{Name: controllerCMName}, &v1.ConfigMap{})
			assert.True(apierrors.IsNotFound(err))
			err = c.Get(context.TODO(), types.NamespacedName{Name: lockCMName}, &v1.ConfigMap{})
			assert.True(apierrors.IsNotFound(err))
			// Persistent volume should not exist
			err = c.Get(context.TODO(), types.NamespacedName{Name: pvName}, &v1.PersistentVolume{})
			assert.True(apierrors.IsNotFound(err))
			err = c.Get(context.TODO(), types.NamespacedName{Name: pv2Name}, &v1.PersistentVolume{})
			assert.True(apierrors.IsNotFound(err))
			// Role Binding should not exist
			err = c.Get(context.TODO(), types.NamespacedName{Name: rbName}, &rbacv1.RoleBinding{})
			assert.True(apierrors.IsNotFound(err))
			// Rancher CRD finalizer should have been deleted which should cause it to go away
			// since it had a deletion timestamp
			crd := v12.CustomResourceDefinition{}
			err = c.Get(context.TODO(), types.NamespacedName{Name: rancherCRDName}, &crd)
			assert.True(apierrors.IsNotFound(err))
		})
	}
}

// TestPostUninstall tests the post uninstall process for Rancher
// GIVEN a call to postUninstall
// WHEN the objects exist in the cluster
// THEN the post-uninstall starts a new attempt and returns a RetryableError to requeue
func TestPostUninstall(t *testing.T) {
	a := assert.New(t)
	vz := v1alpha1.Verrazzano{}

	tt := tests[2]
	c := fake.NewClientBuilder().WithScheme(getScheme()).WithObjects(tt.objects...).Build()
	ctx := spi.NewFakeContext(c, &vz, nil, false)

	crd1 := v12.CustomResourceDefinition{}
	c.Get(context.TODO(), types.NamespacedName{Name: rancherCRDName}, &crd1)

	expectedErr := ctrlerrors.RetryableError{Source: ComponentName}
	forkPostUninstallFunc = func(_ spi.ComponentContext, _ postUninstallMonitor) error {
		return expectedErr
	}
	defer func() { forkPostUninstallFunc = forkPostUninstall }()

	monitor := &fakeMonitor{result: true, running: false}
	err := postUninstall(ctx, monitor)
	a.Equal(expectedErr, err, "Uninstall returned an unexpected error")
}

// TestBackgroundPostUninstallCompletedSuccessfully tests the post uninstall process for Rancher
// GIVEN a call to postUninstall
// WHEN the monitor goroutine fails to successfully complete
// THEN the post-uninstall returns nil without calling the forkPostUninstall function
func TestBackgroundPostUninstallCompletedSuccessfully(t *testing.T) {
	a := assert.New(t)
	vz := v1alpha1.Verrazzano{}

	tt := tests[2]
	c := fake.NewClientBuilder().WithScheme(getScheme()).WithObjects(tt.objects...).Build()
	ctx := spi.NewFakeContext(c, &vz, nil, false)

	crd1 := v12.CustomResourceDefinition{}
	c.Get(context.TODO(), types.NamespacedName{Name: rancherCRDName}, &crd1)

	forkPostUninstallFunc = func(_ spi.ComponentContext, _ postUninstallMonitor) error {
		a.Fail("Unexpected call to forkPostUninstall() function")
		return nil
	}
	defer func() { forkPostUninstallFunc = forkPostUninstall }()

	monitor := &fakeMonitor{result: true, running: true}
	err := postUninstall(ctx, monitor)
	a.NoError(err)
}

// TestPostUninstall tests the post uninstall process for Rancher
// GIVEN a call to postUninstall
// WHEN the the monitor goroutine failed to successfully complete
// THEN the postUninstall function calls the forkPostUninstall function and returns a retry error
func TestBackgroundPostUninstallRetryOnFailure(t *testing.T) {
	a := assert.New(t)
	vz := v1alpha1.Verrazzano{}

	tt := tests[2]
	c := fake.NewClientBuilder().WithScheme(getScheme()).WithObjects(tt.objects...).Build()
	ctx := spi.NewFakeContext(c, &vz, nil, false)

	crd1 := v12.CustomResourceDefinition{}
	c.Get(context.TODO(), types.NamespacedName{Name: rancherCRDName}, &crd1)

	forkFuncCalled := false
	expectedErr := ctrlerrors.RetryableError{Source: ComponentName}
	forkPostUninstallFunc = func(_ spi.ComponentContext, _ postUninstallMonitor) error {
		forkFuncCalled = true
		return expectedErr
	}
	defer func() { forkPostUninstallFunc = forkPostUninstall }()

	monitor := &fakeMonitor{result: false, running: true}
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

	tt := tests[2]
	c := fake.NewClientBuilder().WithScheme(getScheme()).WithObjects(tt.objects...).Build()
	ctx := spi.NewFakeContext(c, &vz, nil, false)

	crd1 := v12.CustomResourceDefinition{}
	c.Get(context.TODO(), types.NamespacedName{Name: rancherCRDName}, &crd1)

	postUninstallFunc = func(ctx spi.ComponentContext) error {
		return nil
	}
	defer func() { postUninstallFunc = invokeRancherSystemToolAndCleanup }()

	var monitor = &postUninstallMonitorType{}
	err := forkPostUninstall(ctx, monitor)
	a.Equal(ctrlerrors.RetryableError{Source: ComponentName}, err)
	for i := 0; i < 100; i++ {
		result, retryError := monitor.checkResult()
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

	tt := tests[2]
	c := fake.NewClientBuilder().WithScheme(getScheme()).WithObjects(tt.objects...).Build()
	ctx := spi.NewFakeContext(c, &vz, nil, false)

	crd1 := v12.CustomResourceDefinition{}
	c.Get(context.TODO(), types.NamespacedName{Name: rancherCRDName}, &crd1)

	postUninstallFunc = func(ctx spi.ComponentContext) error {
		return fmt.Errorf("Unexpected error on uninstall")
	}
	defer func() { postUninstallFunc = invokeRancherSystemToolAndCleanup }()

	var monitor = &postUninstallMonitorType{}
	err := forkPostUninstall(ctx, monitor)
	a.Equal(ctrlerrors.RetryableError{Source: ComponentName}, err)
	for i := 0; i < 100; i++ {
		result, retryError := monitor.checkResult()
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

// TestIsRancherNamespace tests the namespace belongs to Rancher
// GIVEN a call to isRancherNamespace
// WHEN the namespace belings to Rancher or not
// THEN we see true if it is and false if not
func TestIsRancherNamespace(t *testing.T) {
	// FIXME: perhaps need to change this?
	assert := asserts.New(t)

	assert.True(isRancherNamespace(&v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cattle-system",
		},
	}))
	assert.True(isRancherNamespace(&v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "p-12345",
			Annotations: map[string]string{
				rancherSysNS: "true",
			},
		},
	}))
	assert.True(isRancherNamespace(&v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "local",
			Annotations: map[string]string{
				rancherSysNS: "false",
			},
		},
	}))
	assert.False(isRancherNamespace(&v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "p-12345",
			Annotations: map[string]string{
				rancherSysNS: "false",
			},
		},
	}))
	assert.False(isRancherNamespace(&v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "p-12345",
		},
	}))
}
