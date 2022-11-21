// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package helpers

import (
	"bytes"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	vpoconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	testhelpers "github.com/verrazzano/verrazzano/tools/vz/test/helpers"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	fakek8s "k8s.io/client-go/kubernetes/fake"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	podTemplateHashLabel         = "pod-template-hash"
	deploymentRevisionAnnotation = "deployment.kubernetes.io/revision"
	defaultTimeout               = time.Duration(1) * time.Second
	vpoPodName                   = "verrazzano-platform-wls-95d8c5d96-m6mbr"
)

var (
	vpoNamespacedName = types.NamespacedName{
		Name:      "myverrazzano",
		Namespace: "default",
	}
)

// TestGetVpoLogStream tests the functionality that returns the right log stream of the VPO pod.
func TestGetVpoLogStream(t *testing.T) {
	// GIVEN a k8s cluster with no VPO installed,
	// WHEN get log stream is called,
	// THEN no error is returned and a default no op log stream is returned.
	fakeClient := fakek8s.NewSimpleClientset()
	reader, err := GetVpoLogStream(fakeClient, "verrazzano-platform-wls-xyz")
	assert.NoError(t, err)
	assert.NotNil(t, reader)

}

// TestGetVerrazzanoPlatformOperatorPodName tests the functionality that returns the right VPO pod name.
func TestGetVerrazzanoPlatformOperatorPodName(t *testing.T) {
	// GIVEN a k8s cluster with no VPO installed,
	// WHEN GetVerrazzanoPlatformOperatorPodName is invoked,
	// THEN no error is returned and a default no op log stream is returned.
	fakeClient := fake.NewClientBuilder().Build()
	podName, err := GetVerrazzanoPlatformOperatorPodName(fakeClient)
	assert.Error(t, err)
	assert.Equal(t, "", podName)

	// GIVEN a k8s cluster with no VPO installed,
	// WHEN GetVerrazzanoPlatformOperatorPodName is invoked,
	// THEN no error is returned and a default no op log stream is returned.
	fakeClient = fake.NewClientBuilder().WithObjects(&corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: vpoconst.VerrazzanoInstallNamespace,
			Name:      vpoPodName,
			Labels: map[string]string{
				podTemplateHashLabel: "95d8c5d96",
				"app":                constants.VerrazzanoPlatformOperator,
			},
		},
	}).Build()
	podName, err = GetVerrazzanoPlatformOperatorPodName(fakeClient)
	assert.NoError(t, err)
	assert.Equal(t, vpoPodName, podName)
}

// TestGetVerrazzanoPlatformOperatorPodName tests the functionality that waits still the Verrazzano resource reaches the given state.
func TestWaitForPlatformOperator(t *testing.T) {
	// GIVEN a k8s cluster with VPO installed,
	// WHEN WaitForPlatformOperator is invoked,
	// THEN no error is returned and the expected pod name is returned.
	fakeClient := fake.NewClientBuilder().WithObjects(
		getAllVpoObjects()...).Build()
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	podName, err := WaitForPlatformOperator(fakeClient, rc, v1beta1.CondInstallComplete, time.Duration(1)*time.Second)
	assert.NoError(t, err)
	assert.Equal(t, vpoPodName, podName)

	// GIVEN a k8s cluster with no VPO installed,
	// WHEN WaitForPlatformOperator is invoked,
	// THEN an error is returned as there is no VPO pod object.
	fakeClient = fake.NewClientBuilder().Build()
	_, err = WaitForPlatformOperator(fakeClient, rc, v1beta1.CondInstallComplete, time.Duration(1)*time.Second)
	assert.Error(t, err)
}

// TestWaitForOperationToComplete tests the functionality to wait till the given operation completes
func TestWaitForOperationToComplete(t *testing.T) {
	scheme := k8scheme.Scheme
	_ = v1beta1.AddToScheme(scheme)
	_ = v1alpha1.AddToScheme(scheme)
	k8sClient := fakek8s.NewSimpleClientset()
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})

	// GIVEN a k8s cluster with VPO installed,
	// WHEN WaitForOperationToComplete is invoked,
	// THEN an error is returned as the VZ resource is not in InstallComplete state.
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(append(getAllVpoObjects(), getVZObject())...).Build()
	err := WaitForOperationToComplete(fakeClient, k8sClient, rc, vpoNamespacedName, defaultTimeout, defaultTimeout, LogFormatSimple, v1beta1.CondInstallComplete)
	assert.Error(t, err)
}

// TestApplyPlatformOperatorYaml tests the functionality to apply VPO wls yaml
func TestApplyPlatformOperatorYaml(t *testing.T) {
	fakeClient := fake.NewClientBuilder().Build()
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	// GIVEN a k8s cluster with VPO installed,
	// WHEN ApplyPlatformOperatorYaml is invoked,
	// THEN an error is returned as the VZ resource is not in InstallComplete state.
	rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	err := ApplyPlatformOperatorYaml(getCommandWithoutFlags(), fakeClient, rc, "1.5.0")
	assert.Error(t, err)

	// GIVEN a k8s cluster with VPO installed,
	// WHEN ApplyPlatformOperatorYaml is invoked,
	// THEN an error is returned as the VZ resource is not in InstallComplete state.
	cmdWithOperatorYaml := getCommandWithoutFlags()
	cmdWithOperatorYaml.PersistentFlags().String(constants.OperatorFileFlag, "wls.yaml", "")
	err = ApplyPlatformOperatorYaml(cmdWithOperatorYaml, fakeClient, rc, "1.5.0")
	assert.Error(t, err)
}

// TestUsePlatformOperatorUninstallJob tests the functionality of VPO Uninstall job for different versions.
func TestUsePlatformOperatorUninstallJob(t *testing.T) {
	upgradeFlag, err := UsePlatformOperatorUninstallJob(fake.NewClientBuilder().Build())
	assert.Error(t, err)
	assert.False(t, upgradeFlag)

	deploy := getVpoDeployment("", 1, 1)
	deploy.SetLabels(map[string]string{})
	fakeClient := fake.NewClientBuilder().WithObjects(deploy).Build()
	upgradeFlag, err = UsePlatformOperatorUninstallJob(fakeClient)
	assert.NoError(t, err)
	assert.True(t, upgradeFlag)

	// GIVEN a k8s cluster with VPO installed,
	// WHEN UsePlatformOperatorUninstallJob is invoked for an invalid vpo version which does not match the semversion pattern,
	// THEN an error is returned.
	fakeClient = fake.NewClientBuilder().WithObjects(getVpoDeployment("1.f", 1, 1)).Build()
	upgradeFlag, err = UsePlatformOperatorUninstallJob(fakeClient)
	assert.Error(t, err)
	assert.False(t, upgradeFlag)

	// GIVEN a k8s cluster with VPO installed,
	// WHEN UsePlatformOperatorUninstallJob is invoked for an invalid vpo version which does not match the semversion pattern,
	// THEN an error is returned.
	fakeClient = fake.NewClientBuilder().WithObjects(getVpoDeployment("1.4.0", 1, 1)).Build()
	upgradeFlag, err = UsePlatformOperatorUninstallJob(fakeClient)
	assert.NoError(t, err)
	assert.False(t, upgradeFlag)

	// GIVEN a k8s cluster with VPO installed,
	// WHEN UsePlatformOperatorUninstallJob is invoked for a valid sem version lesser than 1.4.0,
	// THEN no error is returned and the upgrade flag is set to true.
	fakeClient = fake.NewClientBuilder().WithObjects(getVpoDeployment("1.3.0", 1, 1)).Build()
	upgradeFlag, err = UsePlatformOperatorUninstallJob(fakeClient)
	assert.NoError(t, err)
	assert.True(t, upgradeFlag)

	// GIVEN a k8s cluster with VPO installed,
	// WHEN UsePlatformOperatorUninstallJob is invoked for a valid sem version greater than 1.4.0,
	// THEN no error is returned and upgrade flag is set to false.
	fakeClient = fake.NewClientBuilder().WithObjects(getVpoDeployment("1.5.0", 1, 1)).Build()
	upgradeFlag, err = UsePlatformOperatorUninstallJob(fakeClient)
	assert.NoError(t, err)
	assert.False(t, upgradeFlag)
}

// TestVpoIsReady tests the functionality to check if VPO deployment is ready based on the VPO pods that are available
func TestVpoIsReady(t *testing.T) {
	// GIVEN a k8s cluster with all VPO deployment with 1 updated replicas, and 1 available replicas
	// WHEN vpoIsReady is invoked,
	// THEN it returns false .
	fakeClient := fake.NewClientBuilder().WithObjects(getVpoDeployment("1.4.0", 1, 1)).Build()
	isReady, err := vpoIsReady(fakeClient)
	assert.NoError(t, err)
	assert.False(t, isReady)

	// GIVEN a k8s cluster with all VPO deployment with 0 updated replicas, and 0 available replicas
	// WHEN vpoIsReady is invoked,
	// THEN it returns false .
	fakeClient = fake.NewClientBuilder().WithObjects(getVpoDeployment("1.4.0", 0, 0)).Build()
	isReady, err = vpoIsReady(fakeClient)
	assert.NoError(t, err)
	assert.False(t, isReady)

	// GIVEN a k8s cluster with all VPO deployment with 0 updated replicas, and 1 available replicas
	// WHEN vpoIsReady is invoked,
	// THEN it returns false .
	fakeClient = fake.NewClientBuilder().WithObjects(getVpoDeployment("1.4.0", 0, 1)).Build()
	isReady, err = vpoIsReady(fakeClient)
	assert.NoError(t, err)
	assert.False(t, isReady)

	// GIVEN a k8s cluster with all VPO deployment with no available replicas
	// WHEN vpoIsReady is invoked,
	// THEN it returns false .
	fakeClient = fake.NewClientBuilder().WithObjects(getVpoDeployment("1.4.0", 1, 0)).Build()
	isReady, err = vpoIsReady(fakeClient)
	assert.NoError(t, err)
	assert.False(t, isReady)

}

// TestGetScanner tests the functionality of returning the right scanner object
func TestGetScanner(t *testing.T) {
	// GIVEN a k8s cluster with all VPO specific objects,
	// WHEN getScanner is invoked,
	// THEN no error is returned and the scanner returned is the default no-op scanner .
	fakeClient := fake.NewClientBuilder().WithObjects(getAllVpoObjects()...).Build()
	k8sClient := fakek8s.NewSimpleClientset()
	scanner, err := getScanner(fakeClient, k8sClient)
	assert.NoError(t, err)
	assert.NotNil(t, scanner)
}

// TestDeleteLeftoverPlatformOperator tests the functionality of deleting the left over VPO pod.
func TestDeleteLeftoverPlatformOperator(t *testing.T) {
	// GIVEN a k8s cluster,
	// WHEN deleteLeftoverPlatformOperator is invoked,
	// THEN no error is returned.
	err := deleteLeftoverPlatformOperator(fake.NewClientBuilder().Build())
	assert.NoError(t, err)
}

// TestSetDeleteFunc tests the functionality where user can provide a custom delete function, and it gets invoked when
// calling the delete operation.
func TestSetDeleteFunc(t *testing.T) {
	// GIVEN a k8s cluster,
	// WHEN delete function is overridden to a custom function using SetDeleteFunc
	// THEN the expected error is returned with the string 'dummy error'.
	SetDeleteFunc(func(client client.Client) error {
		return fmt.Errorf("dummy error")
	})
	err := DeleteFunc(fake.NewClientBuilder().Build())
	assert.Error(t, err)
	assert.Equal(t, "dummy error", err.Error())
}

// TestSetDefaultDeleteFunc tests the functionality where a custom delete function can be provided by the user and invoked.
func TestSetDefaultDeleteFunc(t *testing.T) {
	// GIVEN a k8s cluster,
	// WHEN SetDefaultDeleteFunc is set and DeleteFunc invoked,
	// THEN no error is returned.
	SetDefaultDeleteFunc()
	err := DeleteFunc(fake.NewClientBuilder().Build())
	assert.NoError(t, err)
}

// TestFakeDeleteFunc tests the fake delete function.
func TestFakeDeleteFunc(t *testing.T) {
	// When FakeDeleteFunc it should return nil
	err := FakeDeleteFunc(fake.NewClientBuilder().Build())
	assert.NoError(t, err)
}

// TestGetOperationString tests the functionality that returns the right operation string - install or upgrade
func TestGetOperationString(t *testing.T) {
	// GIVEN a k8s cluster with VPO installed,
	// WHEN getOperationString is invoked for install complete state,
	// THEN it returns a string install.
	operation := getOperationString(v1beta1.CondInstallComplete)
	assert.Equal(t, "install", operation)

	// GIVEN a k8s cluster with VPO installed,
	// WHEN getOperationString is invoked for upgrade complete state,
	// THEN it returns a string upgrade.
	operation = getOperationString(v1beta1.CondUpgradeComplete)
	assert.Equal(t, "upgrade", operation)
}

// getVpoDeployment returns just the deployment object simulating a Verrazzano Platform Operator deployment.
func getVpoDeployment(vpoVersion string, updatedReplicas, availableReplicas int32) client.Object {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: vpoconst.VerrazzanoInstallNamespace,
			Name:      constants.VerrazzanoPlatformOperator,
			Labels: map[string]string{
				"app":                       constants.VerrazzanoPlatformOperator,
				"app.kubernetes.io/version": vpoVersion},
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": constants.VerrazzanoPlatformOperator},
			},
		},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: availableReplicas,
			ReadyReplicas:     1,
			Replicas:          1,
			UpdatedReplicas:   updatedReplicas,
		},
	}
}

// getAllVpoObjects returns the deployment, pod and replica set objects simulating a Verrazzano Platform Operator deployment.
func getAllVpoObjects() []client.Object {
	return []client.Object{
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: vpoconst.VerrazzanoInstallNamespace,
				Name:      constants.VerrazzanoPlatformOperator,
				Labels:    map[string]string{"app": constants.VerrazzanoPlatformOperator},
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": constants.VerrazzanoPlatformOperator},
				},
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 1,
				ReadyReplicas:     1,
				Replicas:          1,
				UpdatedReplicas:   1,
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: vpoconst.VerrazzanoInstallNamespace,
				Name:      constants.VerrazzanoPlatformOperator + "-95d8c5d96-m6mbr",
				Labels: map[string]string{
					podTemplateHashLabel: "95d8c5d96",
					"app":                constants.VerrazzanoPlatformOperator,
				},
			},
		},
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   vpoconst.VerrazzanoInstallNamespace,
				Name:        constants.VerrazzanoPlatformOperator + "-95d8c5d96",
				Annotations: map[string]string{deploymentRevisionAnnotation: "1"},
			},
		},
	}
}

// getVZObject returns the Verrazzano CR object configured by the user.
func getVZObject() client.Object {
	return &v1beta1.Verrazzano{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: vpoNamespacedName.Namespace,
			Name:      vpoNamespacedName.Name,
		},
		Spec: v1beta1.VerrazzanoSpec{
			Profile: "dev",
		},
		Status: v1beta1.VerrazzanoStatus{
			Conditions: []v1beta1.Condition{
				{
					LastTransitionTime: time.Now().Add(time.Duration(-1) * time.Hour).Format(time.RFC3339),
					Type:               "InstallComplete",
					Message:            "Verrazzano install completed successfully",
					Status:             corev1.ConditionTrue,
				},
			},
		},
	}
}
