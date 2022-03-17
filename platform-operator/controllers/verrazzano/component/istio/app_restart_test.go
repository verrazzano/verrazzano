// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	"context"
	"testing"

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

func TestRestartWebLogic(t *testing.T) {
	asserts := assert.New(t)
	config.SetDefaultBomFilePath(unitTestBomFile)

	defer config.Set(config.Get())
	config.Set(config.OperatorConfig{VersionCheckEnabled: false})

	// Setup fake client to provide workloads for restart platform testing
	clientSet := fake.NewSimpleClientset(initFakePod(oldIstioImage), initFakeDeployment(), initFakeStatefulSet(), initFakeDaemonSet())
	k8sutil.SetFakeClient(clientSet)

	StopDomainsUsingOldEnvoy(vzlog.DefaultLogger(), getWebLogicAppMock(t, "test", constants.VerrazzanoSystemNamespace))

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

func getWebLogicAppMock(t *testing.T, name string, namespace string) *mocks.MockClient {
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *oam.ApplicationConfigurationList, opts ...client.ListOption) error {
			appconfig := oam.ApplicationConfiguration{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
				Spec: oam.ApplicationConfigurationSpec{},
				Status: oam.ApplicationConfigurationStatus{
					Workloads: []oam.WorkloadStatus{{
						Reference: oamrt.TypedReference{
							Kind: vzconst.VerrazzanoWebLogicWorkloadKind,
						},
					}},
				},
			}

			list.Items = append(list.Items, appconfig)
			return nil
		})

	return mock
}
