// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package restart

import (
	"context"
	"fmt"
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
	unitTestBomFile           = "../../../../verrazzano-bom.json"
	appRestartUnitTestBomFile = "../../../../../pkg/bom/testdata/verrazzano-bom.json"
	oldIstioImage             = "proxyv2:1.4.3"
)

// TestRestartPodWithIstioProxy tests the RestartComponents method for the following use case
// GIVEN a request to RestartComponents
// WHEN a component is supposed to have an Istio proxy sidecar, and the proxy is current, old, or missing
// THEN ensure the workloads have the restart annotation if needed
func TestRestartPodWithIstioProxy(t *testing.T) {
	asserts := assert.New(t)
	config.SetDefaultBomFilePath(unitTestBomFile)

	defer config.Set(config.Get())
	config.Set(config.OperatorConfig{VersionCheckEnabled: false})

	const (
		someImage     = "someimage"
		proxyImage    = "proxyv2:1.4.5"
		oldProxyImage = "proxyv2:0.1.1"
	)

	tests := []struct {
		name          string
		namespace     *v1.Namespace
		pods          []*v1.Pod
		expectRestart bool
	}{
		// No restart, NS does NOT have injection enabled
		{
			name:          "norestart-ns-not-labeled",
			namespace:     initNamespace(constants.VerrazzanoSystemNamespace, false),
			pods:          []*v1.Pod{initFakePodWithIstioInject("pod1", []string{oldProxyImage}, "")},
			expectRestart: false,
		},
		// No restart, NS has injection enabled, proxy image current
		{
			name:          "norestart-proxy-current",
			namespace:     initNamespace(constants.VerrazzanoSystemNamespace, true),
			pods:          []*v1.Pod{initFakePodWithIstioInject("pod1", []string{proxyImage}, "")},
			expectRestart: false,
		},
		// No restart, NS has injection enabled, proxy image current, pod injection true
		{
			name:          "norestart-proxy-current-pod-inject-true",
			namespace:     initNamespace(constants.VerrazzanoSystemNamespace, true),
			pods:          []*v1.Pod{initFakePodWithIstioInject("pod1", []string{proxyImage}, "true")},
			expectRestart: false,
		},
		// No restart, NS has injection enabled, proxy image current, pod injection missing
		{
			name:          "norestart-proxy-current-pod-inject-missing",
			namespace:     initNamespace(constants.VerrazzanoSystemNamespace, true),
			pods:          []*v1.Pod{initFakePodWithIstioInject("pod1", []string{proxyImage}, "true")},
			expectRestart: false,
		},
		// No restart, NS has injection enabled, proxy image old, pod injection false
		{
			name:          "norestart-proxy-old-pod-inject-false",
			namespace:     initNamespace(constants.VerrazzanoSystemNamespace, true),
			pods:          []*v1.Pod{initFakePodWithIstioInject("pod1", []string{oldProxyImage}, "false")},
			expectRestart: false,
		},
		// No restart, NS has injection enabled, proxy image missing, pod injection false
		{
			name:          "norestart-proxy-missing-pod-inject-false",
			namespace:     initNamespace(constants.VerrazzanoSystemNamespace, true),
			pods:          []*v1.Pod{initFakePodWithIstioInject("pod1", []string{someImage}, "false")},
			expectRestart: false,
		},
		// No restart, NS has injection enabled, proxy image current, second container first, pod injection true
		{
			name:          "norestart-proxy-missing-pod-inject-2a-containers-false",
			namespace:     initNamespace(constants.VerrazzanoSystemNamespace, true),
			pods:          []*v1.Pod{initFakePodWithIstioInject("pod1", []string{someImage, proxyImage}, "true")},
			expectRestart: false,
		},
		// No restart, NS has injection enabled, proxy image current, second container last, pod injection true
		{
			name:          "norestart-proxy-missing-pod-inject-2b-containers-false",
			namespace:     initNamespace(constants.VerrazzanoSystemNamespace, true),
			pods:          []*v1.Pod{initFakePodWithIstioInject("pod1", []string{proxyImage, someImage}, "true")},
			expectRestart: false,
		},
		// Restart, NS has injection enabled, proxy image old, pod injection true
		{
			name:          "restart-proxy-old-pod-inject-true",
			namespace:     initNamespace(constants.VerrazzanoSystemNamespace, true),
			pods:          []*v1.Pod{initFakePodWithIstioInject("pod1", []string{oldProxyImage}, "true")},
			expectRestart: true,
		},
		// Restart, NS has injection enabled, proxy image old, pod injection missing
		{
			name:          "restart-proxy-old-pod-inject-missing",
			namespace:     initNamespace(constants.VerrazzanoSystemNamespace, true),
			pods:          []*v1.Pod{initFakePodWithIstioInject("pod1", []string{oldProxyImage}, "")},
			expectRestart: true,
		},
		// Restart, NS has injection enabled, proxy image missing, pod injection missing
		{
			name:          "restart-proxy-missing-pod-inject-missing",
			namespace:     initNamespace(constants.VerrazzanoSystemNamespace, true),
			pods:          []*v1.Pod{initFakePodWithIstioInject("pod1", []string{someImage}, "")},
			expectRestart: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Setup fake client to provide workloads for restart platform testing
			clientSet := fake.NewSimpleClientset(test.namespace, initFakeDeployment(), initFakeStatefulSet(), initFakeDaemonSet(), test.pods[0])
			k8sutil.SetFakeClient(clientSet)

			namespaces := []string{constants.VerrazzanoSystemNamespace}
			err := RestartComponents(vzlog.DefaultLogger(), namespaces, 1, &OutdatedSidecarPodMatcher{
				istioProxyImage: proxyImage,
			})

			var expectedVal string
			if test.expectRestart {
				expectedVal = "1"
			}
			// Validate the results
			asserts.NoError(err)
			dep, err := clientSet.AppsV1().Deployments("verrazzano-system").Get(context.TODO(), "test", metav1.GetOptions{})
			asserts.NoError(err)
			val := dep.Spec.Template.Annotations[vzconst.VerrazzanoRestartAnnotation]
			asserts.Equal(expectedVal, val, "Incorrect Deployment restart annotation")

			sts, err := clientSet.AppsV1().StatefulSets("verrazzano-system").Get(context.TODO(), "test", metav1.GetOptions{})
			asserts.NoError(err)
			val = sts.Spec.Template.Annotations[vzconst.VerrazzanoRestartAnnotation]
			asserts.Equal(expectedVal, val, "Incorrect StatefulSet restart annotation")

			daemonSet, err := clientSet.AppsV1().DaemonSets("verrazzano-system").Get(context.TODO(), "test", metav1.GetOptions{})
			asserts.NoError(err)
			val = daemonSet.Spec.Template.Annotations[vzconst.VerrazzanoRestartAnnotation]
			asserts.Equal(expectedVal, val, "Incorrect DaemonSet restart annotation")
		})
	}

}

// TestNoRestartAllWorkloadTypesNoOldProxy tests the RestartComponents method for the following use case
// GIVEN a request to RestartComponents a component passing DoesPodHaveOutdatedImages
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
	err := RestartComponents(vzlog.DefaultLogger(), namespaces, 1, &OutdatedSidecarPodMatcher{})

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
	err := RestartComponents(vzlog.DefaultLogger(), namespaces, 1, &OutdatedSidecarPodMatcher{})

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
	err := RestartComponents(vzlog.DefaultLogger(), namespaces, 1, &OutdatedSidecarPodMatcher{})

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
			Namespace: constants.VerrazzanoSystemNamespace,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "foo"},
			},
		},
	}
}

// initFakePodWithIstioInject inits a fake Pod with optional Istio injection label
func initFakePodWithIstioInject(podName string, imageNames []string, istioInject string) *v1.Pod {
	labels := make(map[string]string)
	labels["app"] = "foo"
	if len(istioInject) > 0 {
		labels[podIstioInjectLabel] = istioInject
	}
	return initFakePodWithLabels(podName, imageNames, labels)
}

// initFakePod inits a fake Pod with specified image
func initFakePod(image string) *v1.Pod {
	return initFakePodWithLabels("testPod", []string{image}, map[string]string{"app": "foo"})
}

// initFakePodWithLabels inits a fake Pod with specified image and labels
func initFakePodWithLabels(podname string, imageNames []string, labels map[string]string) *v1.Pod {
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podname,
			Namespace: constants.VerrazzanoSystemNamespace,
			Labels:    labels,
		},
	}
	for i := range imageNames {
		pod.Spec.Containers = append(pod.Spec.Containers,
			v1.Container{
				Name:  fmt.Sprintf("c%v", i),
				Image: imageNames[i],
			})
	}
	return pod
}

// initNamespace inits a namespace with optional istio inject label
func initNamespace(name string, istioInject bool) *v1.Namespace {
	var labels = make(map[string]string)
	if istioInject {
		labels[namespaceIstioInjectLabel] = "enabled"
	}
	return &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
	}
}
