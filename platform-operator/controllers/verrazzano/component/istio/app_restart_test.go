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

type testAppConfigInfo struct {
	namespace     string
	appConfigName string
	workloadKind  string
	workloadName  string
}

// TestDoNotStopWebLogic tests the StopDomainsUsingOldEnvoy method for the following use case
// GIVEN a request to StopDomainsUsingOldEnvoy
// WHEN where there are no WebLogic domains with an old Istio sidecar
// THEN the WebLogic workloads should not be updated with a stop annotation
func TestDoNotStopWebLogic(t *testing.T) {
	asserts := assert.New(t)
	config.SetDefaultBomFilePath(unitTestBomFile)

	defer config.Set(config.Get())
	config.Set(config.OperatorConfig{VersionCheckEnabled: false})

	// Setup fake client to provide workloads for restart platform testing
	clientSet := fake.NewSimpleClientset(initFakePod(oldIstioImage), initFakeDeployment(), initFakeStatefulSet(), initFakeDaemonSet())
	k8sutil.SetFakeClient(clientSet)

	err := StopDomainsUsingOldEnvoy(vzlog.DefaultLogger(), getNoAppRestartMock(t))

	// Validate the results
	asserts.NoError(err)
}

// TestStopWebLogic tests the StopDomainsUsingOldEnvoy method for the following use case
// GIVEN a request to StopDomainsUsingOldEnvoy
// WHEN where there are WebLogic domains with an old Istio sidecar
// THEN the WebLogic workloads should be updated with a stop annotation
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

	config := testAppConfigInfo{
		namespace:     constants.VerrazzanoSystemNamespace,
		appConfigName: appConfigName,
		workloadKind:  vzconst.VerrazzanoWebLogicWorkloadKind,
		workloadName:  wlName,
	}
	expectListAppConfigs(t, mock, config)
	expectGetWebLogicWorkload(t, mock, wlName, "")
	expectUpdateWebLogicWorkload(t, mock, wlName, vzconst.LifecycleActionStop)

	err := StopDomainsUsingOldEnvoy(vzlog.DefaultLogger(), mock)

	// Validate the results
	asserts.NoError(err)
}

// TestStartWebLogic tests the StartDomainsStoppedByUpgrade method for the following use case
// GIVEN a request to StartDomainsStoppedByUpgrade
// WHEN where there are WebLogic domains were stopped by upgrade
// THEN the WebLogic workloads should be updated with a start annotation
func TestStartWebLogic(t *testing.T) {
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

	config := testAppConfigInfo{
		namespace:     constants.VerrazzanoSystemNamespace,
		appConfigName: appConfigName,
		workloadKind:  vzconst.VerrazzanoWebLogicWorkloadKind,
		workloadName:  wlName,
	}
	expectListAppConfigs(t, mock, config)
	expectGetWebLogicWorkload(t, mock, wlName, vzconst.LifecycleActionStop)
	expectUpdateWebLogicWorkload(t, mock, wlName, vzconst.LifecycleActionStart)

	version := "1"
	err := StartDomainsStoppedByUpgrade(vzlog.DefaultLogger(), mock, version)

	// Validate the results
	asserts.NoError(err)
}

// TestStartWebLogic tests the StartDomainsStoppedByUpgrade method for the following use case
// GIVEN a request to StartDomainsStoppedByUpgrade
// WHEN where there are not WebLogic domains were stopped by upgrade
// THEN the WebLogic workloads should not be updated with a start annotation
func TestDoNotStartWebLogic(t *testing.T) {
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

	config := testAppConfigInfo{
		namespace:     constants.VerrazzanoSystemNamespace,
		appConfigName: appConfigName,
		workloadKind:  vzconst.VerrazzanoWebLogicWorkloadKind,
		workloadName:  wlName,
	}
	expectListAppConfigs(t, mock, config)
	expectGetWebLogicWorkload(t, mock, wlName, "")

	version := "1"
	err := StartDomainsStoppedByUpgrade(vzlog.DefaultLogger(), mock, version)

	// Validate the results
	asserts.NoError(err)
}

// TestRestartHelidonApps tests the RestartAllApps method for the following use case
// GIVEN a request to RestartAllApps
// WHEN where there are Helidon applications
// THEN the AppConfig should be annotated with a restart version
func TestRestartHelidonApps(t *testing.T) {
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

	config := testAppConfigInfo{
		namespace:     constants.VerrazzanoSystemNamespace,
		appConfigName: appConfigName,
		workloadKind:  vzconst.VerrazzanoHelidonWorkloadKind,
		workloadName:  wlName,
	}
	version := "1"
	expectListAppConfigs(t, mock, config)
	expectGetAppConfig(t, mock, appConfigName, vzconst.RestartVersionAnnotation, version)
	expectUpdateAppConfig(t, mock, vzconst.RestartVersionAnnotation, version)

	err := RestartAllApps(vzlog.DefaultLogger(), mock, version)

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

func expectListAppConfigs(_ *testing.T, mock *mocks.MockClient, config testAppConfigInfo) {
	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *oam.ApplicationConfigurationList, opts ...client.ListOption) error {
			appconfig := oam.ApplicationConfiguration{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      config.appConfigName,
					Namespace: config.namespace,
				},
				Status: oam.ApplicationConfigurationStatus{
					Workloads: []oam.WorkloadStatus{{
						Reference: oamrt.TypedReference{
							Kind: config.workloadKind,
							Name: config.workloadName,
						},
					}},
				},
			}

			list.Items = append(list.Items, appconfig)
			return nil
		})
}

func expectGetAppConfig(_ *testing.T, mock *mocks.MockClient, appConfigName string, annotationKey string, annotationVal string) {
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: appConfigName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, nsName types.NamespacedName, appConfig *oam.ApplicationConfiguration) error {
			if len(annotationVal) > 0 {
				appConfig.Annotations = map[string]string{}
				appConfig.Annotations[annotationKey] = annotationVal
			}
			return nil
		})
}

func expectUpdateAppConfig(t *testing.T, mock *mocks.MockClient, annotationKey string, annotationVal string) {
	mock.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, appConfig *oam.ApplicationConfiguration, opts ...client.UpdateOption) error {
			if len(annotationVal) > 0 {
				assert.Equal(t, annotationVal, appConfig.Annotations[annotationKey], "Incorrect Appconfig lifecycle annotation")
			}
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
