// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package restart

import (
	"context"
	v1 "k8s.io/api/core/v1"
	"testing"

	"k8s.io/apimachinery/pkg/types"

	oamrt "github.com/crossplane/crossplane-runtime/apis/core/v1alpha1"
	oam "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	vzapp "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/mocks"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type testAppConfigInfo struct {
	namespace     string
	appConfigName string
	workloadKind  string
	workloadName  string
}

// TestWebLogicStopStart tests the starting and stopping of WebLogic
// GIVEN a AppConfig that contains WebLogic workloads
// WHEN the WebLogic pods have old Istio envoy sidecar with skew more than 2 minor versions.
// THEN the domain should be stopped
// WHEN the WebLogic pods do NOT have an old Istio envoy sidecar with skew more than 2 minor versions.
// THEN the domain should NOT be stopped
// WHEN the WebLogic pods do not have an old Istio envoy sidecar with skew max 2 minor versions.
// THEN the domain should be restarted
// IF the domain was stopped
// THEN it should be started after upgrade
func TestWebLogicStopStart(t *testing.T) {
	asserts := assert.New(t)
	config.SetDefaultBomFilePath(appRestartUnitTestBomFile)

	tests := []struct {
		name                   string
		expectGetAndUpdate     bool
		image                  string
		initialLifeCycleAction string
		updatedLifeCycleAction string
		f                      func(mock *mocks.MockClient) error
	}{
		// Test stopping WebLogic by setting annotation on WebLogic workload because it has an old Istio image with skew more than 2 minor versions.
		{
			name:                   "StopWebLogic",
			expectGetAndUpdate:     true,
			image:                  "proxyv2:1.4.5",
			initialLifeCycleAction: "",
			updatedLifeCycleAction: vzconst.LifecycleActionStop,
			f: func(mock *mocks.MockClient) error {
				return StopDomainsUsingOldEnvoy(vzlog.DefaultLogger(), mock)
			},
		},
		// Test NOT stopping WebLogic by setting annotation on WebLogic workload because it has an old Istio image with skew max 2 minor versions.
		{
			name:                   "DoNotStopWebLogic",
			expectGetAndUpdate:     false,
			image:                  "proxyv2:1.5.5",
			initialLifeCycleAction: "",
			f: func(mock *mocks.MockClient) error {
				return StopDomainsUsingOldEnvoy(vzlog.DefaultLogger(), mock)
			},
		},
		// Test starting WebLogic by setting annotation on WebLogic workload because it has an old Istio image with skew max 2 minor versions.
		{
			name:                   "StartWebLogic",
			expectGetAndUpdate:     true,
			image:                  oldIstioImage,
			initialLifeCycleAction: vzconst.LifecycleActionStop,
			updatedLifeCycleAction: vzconst.LifecycleActionStart,
			f: func(mock *mocks.MockClient) error {
				return startDomainsStoppedByUpgrade(vzlog.DefaultLogger(), mock, "1")
			},
		},
		// Test NOT starting WebLogic because workload is missing stop annotation
		{
			name:                   "DoNotStopWebLogic",
			image:                  oldIstioImage,
			expectGetAndUpdate:     true,
			initialLifeCycleAction: "",
			f: func(mock *mocks.MockClient) error {
				return startDomainsStoppedByUpgrade(vzlog.DefaultLogger(), mock, "1")
			},
		},

		// Test rolling restart of WebLogic because it has an old Istio image with skew max 2 minor versions.
		{
			name:                   "RollingRestartWebLogic",
			image:                  "proxyv2:1.5.5",
			expectGetAndUpdate:     true,
			initialLifeCycleAction: "",
			f: func(mock *mocks.MockClient) error {
				return RestartDomainsUsingOldEnvoyMaxSkewTwoMinorVersions(vzlog.DefaultLogger(), mock, "1")
			},
		},

		// Test rolling restart of WebLogic because it has an old Istio image with skew max 2 minor versions.
		{
			name:                   "DoNotRollingRestartWebLogic",
			image:                  "proxyv2:1.4.5",
			expectGetAndUpdate:     false,
			initialLifeCycleAction: "",
			f: func(mock *mocks.MockClient) error {
				return RestartDomainsUsingOldEnvoyMaxSkewTwoMinorVersions(vzlog.DefaultLogger(), mock, "1")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
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

			clientSet := fake.NewSimpleClientset(initFakePodWithLabels(test.image, podLabels), initFakeDeployment(), initFakeStatefulSet(), initFakeDaemonSet())
			k8sutil.SetFakeClient(clientSet)

			conf := testAppConfigInfo{
				namespace:     constants.VerrazzanoSystemNamespace,
				appConfigName: appConfigName,
				workloadKind:  vzconst.VerrazzanoWebLogicWorkloadKind,
				workloadName:  wlName,
			}
			expectListAppConfigs(t, mock, conf)
			if test.expectGetAndUpdate {
				expectGetWebLogicWorkload(t, mock, wlName, test.initialLifeCycleAction)
				expectUpdateWebLogicWorkload(t, mock, wlName, test.updatedLifeCycleAction)
			}

			err := test.f(mock)

			// Validate the results
			asserts.NoError(err)
		})
	}
}

// TestHelidonStopStart tests the starting and stopping of Helidon
// GIVEN a AppConfig that contains Helidon workloads
// WHEN the Helidon pods have old Istio envoy sidecar
// THEN the pods should be restarted
// WHEN the Helidon pods do NOT have old Istio envoy sidecar
// THEN the pods should NOT be restarted
// WHEN the Helidon pods do NOT have an old istio sidecar but with istio injected namespace
// THEN the pods should be restarted
// WHEN the Helidon pods do NOT have an old istio sidecar and without istio injected namespace
// THEN the pods should not be restarted
func TestHelidonStopStart(t *testing.T) {
	asserts := assert.New(t)
	config.SetDefaultBomFilePath(unitTestBomFile)

	tests := []struct {
		name                string
		expectGetAndUpdate  bool
		image               string
		isNSIstioEnabled    bool
		isIstioLabelPresent bool
		f                   func(mock *mocks.MockClient) error
	}{
		// Test restarting Helidon workload because it has an old Istio image
		{
			name:               "RestartHelidon",
			expectGetAndUpdate: true,
			image:              oldIstioImage,
			f: func(mock *mocks.MockClient) error {
				return restartAllApps(vzlog.DefaultLogger(), mock, "1")
			},
		},
		// Test restarting Helidon workload because it doesn't have an old Istio image
		{
			name:               "DoNotRestartHelidon",
			expectGetAndUpdate: false,
			image:              "randomImage",
			f: func(mock *mocks.MockClient) error {
				return restartAllApps(vzlog.DefaultLogger(), mock, "1")
			},
		},
		// Test restarting Helidon workload without old istio sidecar but with istio injected namespace
		{
			name:                "RestartHelidonWithIsioInjection",
			expectGetAndUpdate:  true,
			image:               "randomImage",
			isIstioLabelPresent: true,
			isNSIstioEnabled:    true,
			f: func(mock *mocks.MockClient) error {
				return restartAllApps(vzlog.DefaultLogger(), mock, "1")
			},
		},
		// Test restarting Helidon workload without old istio sidecar and without istio injected namespace
		{
			name:                "DoNotRestartHelidonWithoutIstioInjection",
			expectGetAndUpdate:  false,
			image:               "randomImage",
			isIstioLabelPresent: true,
			isNSIstioEnabled:    false,
			f: func(mock *mocks.MockClient) error {
				return restartAllApps(vzlog.DefaultLogger(), mock, "1")
			},
		},
		// Test restarting Helidon workload when namespace doesn't have an istio injection label
		{
			name:                "DoNotRestartHelidonWithoutIstioNSLabel",
			expectGetAndUpdate:  false,
			image:               "randomImage",
			isIstioLabelPresent: false,
			f: func(mock *mocks.MockClient) error {
				return restartAllApps(vzlog.DefaultLogger(), mock, "1")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mocker := gomock.NewController(t)
			mock := mocks.NewMockClient(mocker)

			defer config.Set(config.Get())
			config.Set(config.OperatorConfig{VersionCheckEnabled: false})

			// Setup fake client to provide workloads for restart platform testing
			wlName := "test"
			appConfigName := "myApp"
			podLabels := map[string]string{"app.oam.dev/name": appConfigName}
			podNamespace := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{
				Name:   "verrazzano-system",
				Labels: map[string]string{"istio-injection": "enabled"},
			}}

			if test.isIstioLabelPresent == false {
				delete(podNamespace.Labels, "istio-injection")
			} else if test.isIstioLabelPresent == true && test.isNSIstioEnabled == false {
				podNamespace.Labels["istio-injection"] = "disabled"
			}

			clientSet := fake.NewSimpleClientset(initFakePodWithLabels(test.image, podLabels), initFakeDeployment(), initFakeStatefulSet(), initFakeDaemonSet(), podNamespace)
			k8sutil.SetFakeClient(clientSet)

			conf := testAppConfigInfo{
				namespace:     constants.VerrazzanoSystemNamespace,
				appConfigName: appConfigName,
				workloadKind:  vzconst.VerrazzanoHelidonWorkloadKind,
				workloadName:  wlName,
			}
			expectListAppConfigs(t, mock, conf)
			version := "1"
			if test.expectGetAndUpdate {
				expectGetAppConfig(t, mock, appConfigName, vzconst.RestartVersionAnnotation, version)
				expectUpdateAppConfig(t, mock, vzconst.RestartVersionAnnotation, version)
			}

			err := test.f(mock)

			// Validate the results
			asserts.NoError(err)
		})
	}
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
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, appConfig *oam.ApplicationConfiguration, opts ...client.UpdateOption) error {
			if len(annotationVal) > 0 {
				assert.Equal(t, annotationVal, appConfig.Annotations[annotationKey], "Incorrect Appconfig lifecycle annotation")
			}
			return nil
		}).AnyTimes()
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
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, wl *vzapp.VerrazzanoWebLogicWorkload, opts ...client.UpdateOption) error {
			if len(lifecycleAction) > 0 {
				assert.Equal(t, lifecycleAction, wl.Annotations[vzconst.LifecycleActionAnnotation], "Incorrect WebLogic lifecycle action")
			}
			return nil
		}).AnyTimes()
}
