package mocks

import (
	"context"
	"github.com/golang/mock/gomock"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"k8s.io/api/apps/v1"
	v12 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func RestartMocks(mock *MockClient) {
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, deployList *v1.DeploymentList) error {
			deployList.Items = []v1.Deployment{{}}
			return nil
		})

	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, ssList *v1.StatefulSetList) error {
			ssList.Items = []v1.StatefulSet{{}}
			return nil
		})

	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, dsList *v1.DaemonSetList) error {
			dsList.Items = []v1.DaemonSet{{}}
			return nil
		})

	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, podList *v12.PodList, option client.ListOption) error {
			return nil
		}).AnyTimes()

	mock.EXPECT().
		Update(gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, deploy *v1.Deployment) error {
			deploy.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
			deploy.Spec.Template.ObjectMeta.Annotations[constants.VerrazzanoRestartAnnotation] = "some time"
			return nil
		}).AnyTimes()

	mock.EXPECT().
		Update(gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, ss *v1.StatefulSet) error {
			ss.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
			ss.Spec.Template.ObjectMeta.Annotations[constants.VerrazzanoRestartAnnotation] = "some time"
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
