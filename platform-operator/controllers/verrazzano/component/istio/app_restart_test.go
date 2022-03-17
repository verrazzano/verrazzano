// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	"context"
	"testing"

	"k8s.io/apimachinery/pkg/types"

	vzapp "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"

	oamrt "github.com/crossplane/crossplane-runtime/apis/core/v1alpha1"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"

	oam "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/golang/mock/gomock"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/verrazzano/verrazzano/platform-operator/mocks"

	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/constants"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestNoRestart(t *testing.T) {
	asserts := assert.New(t)
	config.SetDefaultBomFilePath(unitTestBomFile)

	defer config.Set(config.Get())
	config.Set(config.OperatorConfig{VersionCheckEnabled: false})

	// Setup fake client to provide workloads for restart platform testing
	clientSet := fake.NewSimpleClientset(initFakePod(oldIstioImage), initFakeDeployment(), initFakeStatefulSet(), initFakeDaemonSet())
	k8sutil.SetFakeClient(clientSet)

	StopDomainsUsingOldEnvoy(vzlog.DefaultLogger(), getNoAppRestartMock(t))

	namespaces := []string{constants.VerrazzanoSystemNamespace}
	err := RestartComponents(vzlog.DefaultLogger(), namespaces, 1)

	// Validate the results
	asserts.NoError(err)
}

func TestStopWebLogic(t *testing.T) {
	asserts := assert.New(t)
	config.SetDefaultBomFilePath(unitTestBomFile)
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	defer config.Set(config.Get())
	config.Set(config.OperatorConfig{VersionCheckEnabled: false})

	// Setup fake client to provide workloads for restart platform testing
	wlName := "test"
	appConfigName := "myApp"
	podLabels := map[string]string{"verrazzano.io/workload-type": "weblogic",
		"app.oam.dev/component": wlName,
		"app.oam.dev/name":      appConfigName}

	clientSet := fake.NewSimpleClientset(initFakePodWithLabels(oldIstioImage, podLabels), initFakeDeployment(), initFakeStatefulSet(), initFakeDaemonSet())
	k8sutil.SetFakeClient(clientSet)

	expectListWebLogicAppConfigs(t, mock, appConfigName, wlName, constants.VerrazzanoSystemNamespace)
	expectGetWebLogicWorkload(t, mock, wlName, "")
	expectUpdateWebLogicWorkload(t, mock, wlName, vzconst.LifecycleActionStop)

	StopDomainsUsingOldEnvoy(vzlog.DefaultLogger(), mock)

	namespaces := []string{constants.VerrazzanoSystemNamespace}
	err := RestartComponents(vzlog.DefaultLogger(), namespaces, 1)

	// Validate the results
	asserts.NoError(err)
}

func getNoAppRestartMock(t *testing.T) *mocks.MockClient {
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *oam.ApplicationConfigurationList, opts ...client.ListOption) error {
			return nil
		})

	return mock
}

func expectListWebLogicAppConfigs(_ *testing.T, mock *mocks.MockClient, appConfigName string, wlName string, namespace string) {
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *oam.ApplicationConfigurationList, opts ...client.ListOption) error {
			appconfig := oam.ApplicationConfiguration{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      appConfigName,
					Namespace: namespace,
				},
				Status: oam.ApplicationConfigurationStatus{
					Workloads: []oam.WorkloadStatus{{
						Reference: oamrt.TypedReference{
							Kind: vzconst.VerrazzanoWebLogicWorkloadKind,
							Name: wlName,
						},
					}},
				},
			}

			list.Items = append(list.Items, appconfig)
			return nil
		})
}

func expectGetWebLogicWorkload(_ *testing.T, mock *mocks.MockClient, wlName string, lifecycleAction string) {
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: wlName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, nsName types.NamespacedName, wl *vzapp.VerrazzanoWebLogicWorkload) error {
			if len(lifecycleAction) > 0 {
				wl.Annotations = map[string]string{}
				wl.Annotations[vzconst.LifecycleActionAnnotation] = lifecycleAction
			}
			return nil
		})
}

func expectUpdateWebLogicWorkload(t *testing.T, mock *mocks.MockClient, wlName string, lifecycleAction string) {
	mock.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, wl *vzapp.VerrazzanoWebLogicWorkload, opts ...client.UpdateOption) error {
			if len(lifecycleAction) > 0 {
				assert.Equal(t, lifecycleAction, wl.Annotations[vzconst.LifecycleActionAnnotation], "Incorrect WebLogic lifecycle action")
			}
			return nil
		})
}
