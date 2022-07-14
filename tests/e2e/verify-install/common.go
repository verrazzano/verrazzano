// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"time"

	appsv1 "k8s.io/api/apps/v1"

	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	waitTimeout     = 3 * time.Minute
	pollingInterval = 5 * time.Second
)

// ValidateCorrectNumberOfPodsRunning - validate the expected number of pods is running for a deployment
func ValidateCorrectNumberOfPodsRunning(deployName string, nameSpace string) error {
	// Get the deployment
	var deployment *appsv1.Deployment
	Eventually(func() (*appsv1.Deployment, error) {
		var err error
		deployment, err = pkg.GetDeployment(nameSpace, deployName)
		return deployment, err
	}, waitTimeout, pollingInterval).ShouldNot(BeNil())

	var expectedPods = deployment.Spec.Replicas
	var pods []corev1.Pod
	Eventually(func() bool {
		var err error
		pods, err = pkg.GetPodsFromSelector(&metav1.LabelSelector{MatchLabels: map[string]string{"app": deployName}}, nameSpace)
		if err != nil {
			return false
		}
		// Compare the number of running pods to the expected number
		var runningPods int32 = 0
		for _, pod := range pods {
			if pod.Status.Phase == corev1.PodRunning {
				runningPods++
			}
		}
		return runningPods == *expectedPods
	}, waitTimeout, pollingInterval).Should(BeTrue())
	return nil
}

// ValidateCorrectNumberOfPodsRunningSts - validate the expected number of pods is running for a statefulset
func ValidateCorrectNumberOfPodsRunningSts(stsName string, nameSpace string, label string) error {
	// Get the deployment
	var statefulset *appsv1.StatefulSet
	Eventually(func() (*appsv1.StatefulSet, error) {
		var err error
		statefulset, err = pkg.GetStatefulSet(nameSpace, stsName)
		return statefulset, err
	}, waitTimeout, pollingInterval).ShouldNot(BeNil())

	var expectedPods = statefulset.Spec.Replicas
	var pods []corev1.Pod
	Eventually(func() bool {
		var err error
		pods, err = pkg.GetPodsFromSelector(&metav1.LabelSelector{MatchLabels: map[string]string{label: stsName}}, nameSpace)
		if err != nil {
			return false
		}
		// Compare the number of running pods to the expected number
		var runningPods int32 = 0
		for _, pod := range pods {
			if pod.Status.Phase == corev1.PodRunning {
				runningPods++
			}
		}
		return runningPods == *expectedPods
	}, waitTimeout, pollingInterval).Should(BeTrue())
	return nil
}

// AssertPodAntiAffinity - assert expected pod affinity definition exists
func AssertPodAntiAffinity(matchLabels map[string]string, namespace string) {
	var pods []corev1.Pod
	Eventually(func() error {
		var err error
		pods, err = pkg.GetPodsFromSelector(&metav1.LabelSelector{MatchLabels: matchLabels}, namespace)
		return err
	}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())

	// Check the affinity configuration. Verify only a pod anti-affinity definition exists.
	for _, pod := range pods {
		affinity := pod.Spec.Affinity
		Expect(affinity).ToNot(BeNil())
		Expect(affinity.PodAffinity).To(BeNil())
		Expect(affinity.NodeAffinity).To(BeNil())
		Expect(affinity.PodAntiAffinity).ToNot(BeNil())
		Expect(len(affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution)).To(Equal(1))
	}
}
