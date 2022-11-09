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
	k8sAppNameLabel              = "app.kubernetes.io/name"
	k8sInstanceNameLabel         = "app.kubernetes.io/instance"
	deploymentRevisionAnnotation = "deployment.kubernetes.io/revision"
	defaultTimeout               = time.Duration(1) * time.Second
)

var (
	vpoNamespacedName = types.NamespacedName{
		Name:      "myverrazzano",
		Namespace: "default",
	}
)

func TestGetVpoLogStream(t *testing.T) {
	// GIVEN a k8s cluster with no VPO installed,
	// WHEN get log stream is called,
	// THEN no error is returned and a default no op log stream is returned.
	fakeClient := fakek8s.NewSimpleClientset()
	reader, err := GetVpoLogStream(fakeClient, "verrazzano-platform-operator-xyz")
	assert.NoError(t, err)
	assert.NotNil(t, reader)

}

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
			Name:      "verrazzano-platform-operator-95d8c5d96-m6mbr",
			Labels: map[string]string{
				podTemplateHashLabel: "95d8c5d96",
				"app":                constants.VerrazzanoPlatformOperator,
			},
		},
	}).Build()
	podName, err = GetVerrazzanoPlatformOperatorPodName(fakeClient)
	assert.NoError(t, err)
	assert.Equal(t, "verrazzano-platform-operator-95d8c5d96-m6mbr", podName)
}

func TestWaitForPlatformOperator(t *testing.T) {
	// GIVEN a k8s cluster with no VPO installed,
	// WHEN GetVerrazzanoPlatformOperatorPodName is invoked,
	// THEN no error is returned and a default no op log stream is returned.
	fakeClient := fake.NewClientBuilder().WithObjects(
		getAllVpoObjects()...).Build()
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	podName, err := WaitForPlatformOperator(fakeClient, rc, v1beta1.CondInstallComplete, time.Duration(1)*time.Second)
	assert.NoError(t, err)
	assert.Equal(t, "verrazzano-platform-operator-95d8c5d96-m6mbr", podName)

	fakeClient = fake.NewClientBuilder().Build()
	_, err = WaitForPlatformOperator(fakeClient, rc, v1beta1.CondInstallComplete, time.Duration(1)*time.Second)
	assert.Error(t, err)
}

func TestWaitForOperationToComplete(t *testing.T) {
	scheme := k8scheme.Scheme
	v1beta1.AddToScheme(scheme)
	v1alpha1.AddToScheme(scheme)
	k8sClient := fakek8s.NewSimpleClientset()
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(append(getAllVpoObjects(), getVZObject())...).Build()
	err := WaitForOperationToComplete(fakeClient, k8sClient, rc, vpoNamespacedName, defaultTimeout, defaultTimeout, LogFormatSimple, v1beta1.CondInstallComplete)
	assert.Error(t, err)
}

func TestApplyPlatformOperatorYaml(t *testing.T) {
	fakeClient := fake.NewClientBuilder().Build()
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	err := ApplyPlatformOperatorYaml(getCommandWithoutFlags(), fakeClient, rc, "1.5.0")
	assert.Error(t, err)

	cmdWithOperatorYaml := getCommandWithoutFlags()
	cmdWithOperatorYaml.PersistentFlags().String(constants.OperatorFileFlag, "operator.yaml", "")
	err = ApplyPlatformOperatorYaml(cmdWithOperatorYaml, fakeClient, rc, "1.5.0")
	assert.Error(t, err)
}

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

	fakeClient = fake.NewClientBuilder().WithObjects(getVpoDeployment("1.f", 1, 1)).Build()
	upgradeFlag, err = UsePlatformOperatorUninstallJob(fakeClient)
	assert.Error(t, err)
	assert.False(t, upgradeFlag)

	fakeClient = fake.NewClientBuilder().WithObjects(getVpoDeployment("1.4.0", 1, 1)).Build()
	upgradeFlag, err = UsePlatformOperatorUninstallJob(fakeClient)
	assert.NoError(t, err)
	assert.False(t, upgradeFlag)

	fakeClient = fake.NewClientBuilder().WithObjects(getVpoDeployment("1.3.0", 1, 1)).Build()
	upgradeFlag, err = UsePlatformOperatorUninstallJob(fakeClient)
	assert.NoError(t, err)
	assert.True(t, upgradeFlag)

	fakeClient = fake.NewClientBuilder().WithObjects(getVpoDeployment("1.5.0", 1, 1)).Build()
	upgradeFlag, err = UsePlatformOperatorUninstallJob(fakeClient)
	assert.NoError(t, err)
	assert.False(t, upgradeFlag)
}

func TestVpoIsReady(t *testing.T) {
	fakeClient := fake.NewClientBuilder().WithObjects(getVpoDeployment("1.4.0", 1, 1)).Build()
	isReady, err := vpoIsReady(fakeClient)
	assert.NoError(t, err)
	assert.False(t, isReady)

	fakeClient = fake.NewClientBuilder().WithObjects(getVpoDeployment("1.4.0", 0, 0)).Build()
	isReady, err = vpoIsReady(fakeClient)
	assert.NoError(t, err)
	assert.False(t, isReady)

	fakeClient = fake.NewClientBuilder().WithObjects(getVpoDeployment("1.4.0", 0, 1)).Build()
	isReady, err = vpoIsReady(fakeClient)
	assert.NoError(t, err)
	assert.False(t, isReady)

	fakeClient = fake.NewClientBuilder().WithObjects(getVpoDeployment("1.4.0", 1, 0)).Build()
	isReady, err = vpoIsReady(fakeClient)
	assert.NoError(t, err)
	assert.False(t, isReady)

}

func TestGetScanner(t *testing.T) {
	fakeClient := fake.NewClientBuilder().WithObjects(getAllVpoObjects()...).Build()
	k8sClient := fakek8s.NewSimpleClientset()
	scanner, err := getScanner(fakeClient, k8sClient)
	assert.NoError(t, err)
	assert.NotNil(t, scanner)
}

func TestDeleteLeftoverPlatformOperator(t *testing.T) {
	err := deleteLeftoverPlatformOperator(fake.NewClientBuilder().Build())
	assert.NoError(t, err)
}

func TestSetDeleteFunc(t *testing.T) {
	SetDeleteFunc(func(client client.Client) error {
		return fmt.Errorf("dummy error")
	})
	err := DeleteFunc(fake.NewClientBuilder().Build())
	assert.Error(t, err)
	assert.Equal(t, "dummy error", err.Error())
}

func TestSetDefaultDeleteFunc(t *testing.T) {
	SetDefaultDeleteFunc()
	err := DeleteFunc(fake.NewClientBuilder().Build())
	assert.NoError(t, err)
}

func TestFakeDeleteFunc(t *testing.T) {
	err := FakeDeleteFunc(fake.NewClientBuilder().Build())
	assert.NoError(t, err)
}

func TestGetOperationString(t *testing.T) {
	operation := getOperationString(v1beta1.CondInstallComplete)
	assert.Equal(t, "install", operation)
	operation = getOperationString(v1beta1.CondUpgradeComplete)
	assert.Equal(t, "upgrade", operation)
}
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
