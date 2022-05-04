// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package status

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestDaemonSetsReady(t *testing.T) {

	selector := &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"app": "foo",
		},
	}
	enoughReplicas := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "bar",
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: selector,
		},
		Status: appsv1.DaemonSetStatus{
			NumberAvailable:        1,
			UpdatedNumberScheduled: 1,
		},
	}
	enoughReplicasMultiple := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "bar",
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: selector,
		},
		Status: appsv1.DaemonSetStatus{
			NumberAvailable:        2,
			UpdatedNumberScheduled: 2,
		},
	}
	notEnoughReadyReplicas := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "bar",
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: selector,
		},
		Status: appsv1.DaemonSetStatus{
			NumberAvailable:        0,
			UpdatedNumberScheduled: 1,
		},
	}
	notEnoughUpdatedReplicas := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "bar",
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: selector,
		},
		Status: appsv1.DaemonSetStatus{
			NumberAvailable:        1,
			UpdatedNumberScheduled: 0,
		},
	}

	readyPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo-95d8c5d96-m6mbr",
			Namespace: "bar",
			Labels: map[string]string{
				controllerRevisionHashLabel: "95d8c5d96",
				"app":                       "foo",
			},
		},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Ready: true,
				},
			},
		},
	}
	notReadyContainerPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo-95d8c5d96-m6y76",
			Namespace: "bar",
			Labels: map[string]string{
				controllerRevisionHashLabel: "95d8c5d96",
				"app":                       "foo",
			},
		},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Ready: false,
				},
			},
		},
	}
	notReadyInitContainerPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo-95d8c5d96-m6mbr",
			Namespace: "bar",
			Labels: map[string]string{
				controllerRevisionHashLabel: "95d8c5d96",
				"app":                       "foo",
			},
		},
		Status: corev1.PodStatus{
			InitContainerStatuses: []corev1.ContainerStatus{
				{
					Ready: false,
				},
			},
		},
	}
	noPodHashLabel := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo-95d8c5d96-m6mbr",
			Namespace: "bar",
			Labels: map[string]string{
				"app": "foo",
			},
		},
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Ready: true,
				},
			},
		},
	}

	controllerRevision := &appsv1.ControllerRevision{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo-95d8c5d96",
			Namespace: "bar",
		},
		Revision: 1,
	}

	namespacedName := []types.NamespacedName{
		{
			Name:      "foo",
			Namespace: "bar",
		},
	}

	var tests = []struct {
		name     string
		c        client.Client
		n        []types.NamespacedName
		ready    bool
		expected int32
	}{
		{
			"should be ready when daemonset has enough replicas and pod is ready",
			fake.NewFakeClientWithScheme(k8scheme.Scheme, enoughReplicas, readyPod, controllerRevision),
			namespacedName,
			true,
			1,
		},
		{
			"should be ready when daemonset has enough replicas and one pod of two pods is ready",
			fake.NewFakeClientWithScheme(k8scheme.Scheme, enoughReplicasMultiple, notReadyContainerPod, readyPod, controllerRevision),
			namespacedName,
			true,
			1,
		},
		{
			"should be not ready when expected pods ready not reached",
			fake.NewFakeClientWithScheme(k8scheme.Scheme, enoughReplicasMultiple, readyPod, controllerRevision),
			namespacedName,
			false,
			2,
		},
		{
			"should be not ready when daemonset has enough replicas but pod init container pods is not ready",
			fake.NewFakeClientWithScheme(k8scheme.Scheme, enoughReplicas, notReadyInitContainerPod, controllerRevision),
			namespacedName,
			false,
			1,
		},
		{
			"should be not ready when daemonset has enough replicas but pod container pods is not ready",
			fake.NewFakeClientWithScheme(k8scheme.Scheme, enoughReplicas, notReadyContainerPod, controllerRevision),
			namespacedName,
			false,
			1,
		},
		{
			"should be not ready when daemonset not found",
			fake.NewFakeClientWithScheme(k8scheme.Scheme),
			namespacedName,
			false,
			1,
		},
		{
			"should be not ready when controller-revision-hash label not found",
			fake.NewFakeClientWithScheme(k8scheme.Scheme, enoughReplicas, noPodHashLabel),
			namespacedName,
			false,
			1,
		},
		{
			"should be not ready when controllerrevision resource not found",
			fake.NewFakeClientWithScheme(k8scheme.Scheme, enoughReplicas, readyPod),
			namespacedName,
			false,
			1,
		},
		{
			"should be not ready when daemonset doesn't have enough ready replicas",
			fake.NewFakeClientWithScheme(k8scheme.Scheme, notEnoughReadyReplicas),
			namespacedName,
			false,
			1,
		},
		{
			"should be not ready when daemonset doesn't have enough updated replicas",
			fake.NewFakeClientWithScheme(k8scheme.Scheme, notEnoughUpdatedReplicas),
			namespacedName,
			false,
			1,
		},
		{
			"should be ready when no PodReadyCheck provided",
			fake.NewFakeClientWithScheme(k8scheme.Scheme),
			[]types.NamespacedName{},
			true,
			1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.ready, DaemonSetsAreReady(vzlog.DefaultLogger(), tt.c, tt.n, tt.expected, ""))
		})
	}
}
