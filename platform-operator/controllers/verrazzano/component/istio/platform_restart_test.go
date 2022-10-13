// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	"context"
	"testing"

	vzconst "github.com/verrazzano/verrazzano/pkg/constants"

	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/constants"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

const (
	unitTestBomFile = "../../../../verrazzano-bom.json"
	oldIstioImage   = "proxyv2:1.7.3"
)

// TestRestartAllWorkloadTypesWithOldProxy tests the RestartComponents method for the following use case
// GIVEN a request to RestartComponents passing DoesPodContainOldIstioSidecar
// WHEN where the fake client has deployments, statefulsets, and daemonsets that need to be restarted
// THEN the workloads have the restart annotation with the Verrazzano CR generation as the value
func TestRestartAllWorkloadTypesWithOldProxy(t *testing.T) {
	asserts := assert.New(t)
	config.SetDefaultBomFilePath(unitTestBomFile)

	defer config.Set(config.Get())
	config.Set(config.OperatorConfig{VersionCheckEnabled: false})

	// Setup fake client to provide workloads for restart platform testing
	clientSet := fake.NewSimpleClientset(initFakePod(oldIstioImage), initFakeDeployment(), initFakeStatefulSet(), initFakeDaemonSet())
	k8sutil.SetFakeClient(clientSet)

	namespaces := []string{constants.VerrazzanoSystemNamespace}
	err := RestartComponents(vzlog.DefaultLogger(), namespaces, 1, DoesPodContainOldIstioSidecar)

	// Validate the results
	asserts.NoError(err)
	dep, err := clientSet.AppsV1().Deployments("verrazzano-system").Get(context.TODO(), "test", metav1.GetOptions{})
	asserts.NoError(err)
	val := dep.Spec.Template.Annotations[vzconst.VerrazzanoRestartAnnotation]
	asserts.Equal("1", val, "Incorrect Deployment restart annotation")

	sts, err := clientSet.AppsV1().StatefulSets("verrazzano-system").Get(context.TODO(), "test", metav1.GetOptions{})
	asserts.NoError(err)
	val = sts.Spec.Template.Annotations[vzconst.VerrazzanoRestartAnnotation]
	asserts.Equal("1", val, "Incorrect StatefulSet restart annotation")

	daemonSet, err := clientSet.AppsV1().DaemonSets("verrazzano-system").Get(context.TODO(), "test", metav1.GetOptions{})
	asserts.NoError(err)
	val = daemonSet.Spec.Template.Annotations[vzconst.VerrazzanoRestartAnnotation]
	asserts.Equal("1", val, "Incorrect DaemonSet restart annotation")
}

// TestNoRestartAllWorkloadTypesWithNoProxy tests the RestartComponents method for the following use case
// GIVEN a request to RestartComponents passing DoesPodContainNoIstioSidecar
// WHEN where the fake client has deployments, statefulsets, and daemonsets that do not need to be restarted
// THEN the workloads should not have the restart annotation with the Verrazzano CR generation as the value
func TestNoRestartAllWorkloadTypesWithNoProxy(t *testing.T) {
	asserts := assert.New(t)
	config.SetDefaultBomFilePath(unitTestBomFile)

	defer config.Set(config.Get())
	config.Set(config.OperatorConfig{VersionCheckEnabled: false})

	// Setup fake client to provide workloads for restart platform testing
	clientSet := fake.NewSimpleClientset(initFakePod("someimage"), initFakeDeployment(), initFakeStatefulSet(), initFakeDaemonSet())
	k8sutil.SetFakeClient(clientSet)

	namespaces := []string{constants.VerrazzanoSystemNamespace}
	err := RestartComponents(vzlog.DefaultLogger(), namespaces, 1, DoesPodContainNoIstioSidecar)

	// Validate the results
	asserts.NoError(err)
	dep, err := clientSet.AppsV1().Deployments("verrazzano-system").Get(context.TODO(), "test", metav1.GetOptions{})
	asserts.NoError(err)
	val := dep.Spec.Template.Annotations[vzconst.VerrazzanoRestartAnnotation]
	asserts.Equal("1", val, "Incorrect Deployment restart annotation")

	sts, err := clientSet.AppsV1().StatefulSets("verrazzano-system").Get(context.TODO(), "test", metav1.GetOptions{})
	asserts.NoError(err)
	val = sts.Spec.Template.Annotations[vzconst.VerrazzanoRestartAnnotation]
	asserts.Equal("1", val, "Incorrect StatefulSet restart annotation")

	daemonSet, err := clientSet.AppsV1().DaemonSets("verrazzano-system").Get(context.TODO(), "test", metav1.GetOptions{})
	asserts.NoError(err)
	val = daemonSet.Spec.Template.Annotations[vzconst.VerrazzanoRestartAnnotation]
	asserts.Equal("1", val, "Incorrect DaemonSet restart annotation")
}

// TestNoRestartAllWorkloadTypesNoOldProxy tests the RestartComponents method for the following use case
// GIVEN a request to RestartComponents a component passing DoesPodContainOldIstioSidecar
// WHEN where the fake client has deployments, statefulsets, and daemonsets that do not need to be restarted
// THEN the workloads should not have the restart annotation with the Verrazzano CR generation as the value
func TestNoRestartAllWorkloadTypesNoOldProxy(t *testing.T) {
	asserts := assert.New(t)
	config.SetDefaultBomFilePath(unitTestBomFile)

	defer config.Set(config.Get())
	config.Set(config.OperatorConfig{VersionCheckEnabled: false})

	// Setup fake client to provide workloads for restart platform testing
	clientSet := fake.NewSimpleClientset(initFakePod("someimage"), initFakeDeployment(), initFakeStatefulSet(), initFakeDaemonSet())
	k8sutil.SetFakeClient(clientSet)

	namespaces := []string{constants.VerrazzanoSystemNamespace}
	err := RestartComponents(vzlog.DefaultLogger(), namespaces, 1, DoesPodContainOldIstioSidecar)

	// Validate the results
	asserts.NoError(err)
	dep, err := clientSet.AppsV1().Deployments("verrazzano-system").Get(context.TODO(), "test", metav1.GetOptions{})
	asserts.NoError(err)
	asserts.Nil(dep.Spec.Template.Annotations, "Incorrect Deployment restart annotation")

	sts, err := clientSet.AppsV1().StatefulSets("verrazzano-system").Get(context.TODO(), "test", metav1.GetOptions{})
	asserts.NoError(err)
	asserts.Nil(sts.Spec.Template.Annotations, "Incorrect StatefulSet restart annotation")

	daemonSet, err := clientSet.AppsV1().DaemonSets("verrazzano-system").Get(context.TODO(), "test", metav1.GetOptions{})
	asserts.NoError(err)
	asserts.Nil(daemonSet.Spec.Template.Annotations, "Incorrect DaemonSet restart annotation")
}

// TestNoRestartAllWorkloadTypesWithProxy tests the RestartComponents method for the following use case
// GIVEN a request to RestartComponents a component passing DoesPodContainNoIstioSidecar
// WHEN where the fake client has deployments, statefulsets, and daemonsets that do not need to be restarted
// THEN the workloads should not have the restart annotation with the Verrazzano CR generation as the value
func TestNoRestartAllWorkloadTypesWithProxy(t *testing.T) {
	asserts := assert.New(t)
	config.SetDefaultBomFilePath(unitTestBomFile)

	defer config.Set(config.Get())
	config.Set(config.OperatorConfig{VersionCheckEnabled: false})

	// Setup fake client to provide workloads for restart platform testing
	clientSet := fake.NewSimpleClientset(initFakePod("proxyv2"), initFakeDeployment(), initFakeStatefulSet(), initFakeDaemonSet())
	k8sutil.SetFakeClient(clientSet)

	namespaces := []string{constants.VerrazzanoSystemNamespace}
	err := RestartComponents(vzlog.DefaultLogger(), namespaces, 1, DoesPodContainNoIstioSidecar)

	// Validate the results
	asserts.NoError(err)
	dep, err := clientSet.AppsV1().Deployments("verrazzano-system").Get(context.TODO(), "test", metav1.GetOptions{})
	asserts.NoError(err)
	asserts.Nil(dep.Spec.Template.Annotations, "Incorrect Deployment restart annotation")

	sts, err := clientSet.AppsV1().StatefulSets("verrazzano-system").Get(context.TODO(), "test", metav1.GetOptions{})
	asserts.NoError(err)
	asserts.Nil(sts.Spec.Template.Annotations, "Incorrect StatefulSet restart annotation")

	daemonSet, err := clientSet.AppsV1().DaemonSets("verrazzano-system").Get(context.TODO(), "test", metav1.GetOptions{})
	asserts.NoError(err)
	asserts.Nil(daemonSet.Spec.Template.Annotations, "Incorrect DaemonSet restart annotation")
}

// TestNoRestartAllWorkloadTypesWithOAMPod tests the RestartComponents method for the following use case
// GIVEN a request to RestartComponents a component passing DoesPodContainNoIstioSidecar
// WHEN where the fake client has deployments, statefulsets, and daemonsets that do not need to be restarted
// THEN the workloads should not have the restart annotation with the Verrazzano CR generation as the value
func TestNoRestartAllWorkloadTypesWithOAMPod(t *testing.T) {
	asserts := assert.New(t)
	config.SetDefaultBomFilePath(unitTestBomFile)

	defer config.Set(config.Get())
	config.Set(config.OperatorConfig{VersionCheckEnabled: false})

	// Setup fake client to provide workloads for restart platform testing
	clientSet := fake.NewSimpleClientset(&v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "oam-kubernetes-runtime-85b66fcf68-94zl9",
			Namespace: "verrazzano-system",
			Labels:    map[string]string{"app": "foo"},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{{
				Name:  "c1",
				Image: "someimage",
			}},
		},
	}, initFakeDeployment(), initFakeStatefulSet(), initFakeDaemonSet())
	k8sutil.SetFakeClient(clientSet)

	namespaces := []string{constants.VerrazzanoSystemNamespace}
	err := RestartComponents(vzlog.DefaultLogger(), namespaces, 1, DoesPodContainNoIstioSidecar)

	// Validate the results
	asserts.NoError(err)
	dep, err := clientSet.AppsV1().Deployments("verrazzano-system").Get(context.TODO(), "test", metav1.GetOptions{})
	asserts.NoError(err)
	asserts.Nil(dep.Spec.Template.Annotations, "Incorrect Deployment restart annotation")

	sts, err := clientSet.AppsV1().StatefulSets("verrazzano-system").Get(context.TODO(), "test", metav1.GetOptions{})
	asserts.NoError(err)
	asserts.Nil(sts.Spec.Template.Annotations, "Incorrect StatefulSet restart annotation")

	daemonSet, err := clientSet.AppsV1().DaemonSets("verrazzano-system").Get(context.TODO(), "test", metav1.GetOptions{})
	asserts.NoError(err)
	asserts.Nil(daemonSet.Spec.Template.Annotations, "Incorrect DaemonSet restart annotation")
}

// initFakeDeployment inits a fake Deployment
func initFakeDeployment() *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "verrazzano-system",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: nil,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "foo"},
			},
		},
	}
}

// initFakeStatefulSet inits a fake StatefulSet
func initFakeStatefulSet() *appsv1.StatefulSet {
	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "verrazzano-system",
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: nil,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "foo"},
			},
		},
	}
}

// initFakeDaemonSet inits a fake DaemonSet
func initFakeDaemonSet() *appsv1.DaemonSet {
	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "verrazzano-system",
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "foo"},
			},
		},
	}
}

// initFakePod inits a fake Pod with specified image
func initFakePod(image string) *v1.Pod {
	return initFakePodWithLabels(image, map[string]string{"app": "foo"})
}

// initFakePodWithLabels inits a fake Pod with specified image and labels
func initFakePodWithLabels(image string, labels map[string]string) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testPod",
			Namespace: "verrazzano-system",
			Labels:    labels,
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{{
				Name:  "c1",
				Image: image,
			}},
		},
	}
}
