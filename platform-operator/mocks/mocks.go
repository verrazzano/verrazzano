// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mocks

import (
	"context"

	"github.com/verrazzano/verrazzano/pkg/constants"

	"github.com/golang/mock/gomock"
	"k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// RestartMocks takes a MockClient and gives it the EXPECTs necessary to pass the unit tests
func RestartMocks(mock *MockClient) {
	mock.EXPECT().
		List(gomock.Any(), &v1.DeploymentList{}).
		DoAndReturn(func(ctx context.Context, deployList *v1.DeploymentList) error {
			deployList.Items = []v1.Deployment{{}}
			return nil
		}).AnyTimes()

	mock.EXPECT().
		List(gomock.Any(), &v1.StatefulSetList{}).
		DoAndReturn(func(ctx context.Context, ssList *v1.StatefulSetList) error {
			ssList.Items = []v1.StatefulSet{{}}
			return nil
		}).AnyTimes()

	s := metav1.LabelSelector{
		MatchLabels: map[string]string{"app": "fluentdd"},
	}
	mock.EXPECT().
		List(gomock.Any(), &v1.DaemonSetList{}).
		DoAndReturn(func(ctx context.Context, dsList *v1.DaemonSetList) error {
			dsList.Items = []v1.DaemonSet{{
				ObjectMeta: metav1.ObjectMeta{Namespace: constants.VerrazzanoSystemNamespace},
				Spec: v1.DaemonSetSpec{
					Selector: &s,
				},
			}}
			return nil
		}).AnyTimes()

	mock.EXPECT().
		List(gomock.Any(), &corev1.PodList{}, gomock.Any()).
		DoAndReturn(func(ctx context.Context, podList *corev1.PodList, labels client.MatchingLabels) error {
			podList.Items = []corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{"app": "fluentd"},
					},
					Spec: corev1.PodSpec{

						Containers: []corev1.Container{
							{
								Image: "ghcr.io/verrazzano/fluentd-kubernetes-daemonset:v1.12.3-20211206061401-0302423",
							},
							{
								Image: "ghcr.io/verrazzano/proxyv2:1.10.4",
							},
						},
					},
					Status: corev1.PodStatus{},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{"app": "fluentd"},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Image: "ghcr.io/verrazzano/fluentd-kubernetes-daemonset:v1.12.3-20211206061401-0302423",
							},
							{
								Image: "ghcr.io/verrazzano/proxyv2:1.7.3",
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{"app": "fluentd"},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Image: "ghcr.io/verrazzano/fluentd-kubernetes-daemonset:v1.12.3-20211206061401-0302423",
							},
							{
								Image: "ghcr.io/verrazzano/proxyv2:1.10.4",
							},
						},
					},
				},
			}
			return nil
		}).AnyTimes()

	mock.EXPECT().
		Update(gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, ds *v1.DaemonSet) error {
			ds.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
			ds.Spec.Template.ObjectMeta.Annotations[constants.VerrazzanoRestartAnnotation] = "some time"
			return nil
		}).AnyTimes()
}
