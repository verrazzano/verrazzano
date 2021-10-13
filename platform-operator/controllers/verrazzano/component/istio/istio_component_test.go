// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	"context"
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/istio"
	"github.com/verrazzano/verrazzano/platform-operator/mocks"
	"go.uber.org/zap"
	"io/ioutil"
	appsv1 "k8s.io/api/apps/v1"
	"os/exec"
	"strings"
	"testing"
)

// fakeRunner is used to test istio without actually running an OS exec command
type fakeRunner struct {
}

var vz = &installv1alpha1.Verrazzano{
	Spec: installv1alpha1.VerrazzanoSpec{
		Components: installv1alpha1.ComponentSpec{
			Istio: &installv1alpha1.IstioComponent{
				IstioInstallArgs: []installv1alpha1.InstallArgs{{
					Name:  "arg1",
					Value: "val1",
				}},
			},
		},
	},
}

var comp = IstioComponent{}

const testBomFilePath = "../../testdata/test_bom_istio_1.10.2.json"

// TestGetName tests the component name
// GIVEN a Verrazzano component
//  WHEN I call Name
//  THEN the correct verrazzano name is returned
func TestGetName(t *testing.T) {
	assert := assert.New(t)
	assert.Equal("istio", comp.Name(), "Wrong component name")
}

// TestUpgrade tests the component upgrade
// GIVEN a component
//  WHEN I call Upgrade
//  THEN the upgrade returns success and passes the correct values to the upgrade function
func TestUpgrade(t *testing.T) {
	assert := assert.New(t)

	comp := IstioComponent{
		ValuesFile:               "test-values-file.yaml",
		Revision:                 "1-1-1",
		InjectedSystemNamespaces: config.GetInjectedSystemNamespaces(),
	}

	config.SetDefaultBomFilePath(testBomFilePath)
	SetIstioUpgradeFunction(fakeUpgrade)
	defer ResetIstioUpgradeFunction()
	err := comp.Upgrade(spi.NewContext(zap.S(), getMock(t), vz, false))
	assert.NoError(err, "Upgrade returned an error")
}

// fakeUpgrade verifies that the correct parameter values are passed to upgrade
func fakeUpgrade(log *zap.SugaredLogger, imageOverridesString string, overridesFiles ...string) (stdout []byte, stderr []byte, err error) {
	if len(overridesFiles) != 2 {
		return []byte("error"), []byte(""), fmt.Errorf("incorrect number of override files: expected 2, received %v", len(overridesFiles))
	}
	if overridesFiles[0] != "test-values-file.yaml" {
		return []byte("error"), []byte(""), fmt.Errorf("invalid values file")
	}
	if !strings.Contains(overridesFiles[1], "values-") || !strings.Contains(overridesFiles[1], ".yaml") {
		return []byte("error"), []byte(""), fmt.Errorf("incorrect install args overrides file")
	}
	installArgsFromFile, err := ioutil.ReadFile(overridesFiles[1])
	if err != nil {
		return []byte("error"), []byte(""), fmt.Errorf("unable to read install args overrides file")
	}
	if !strings.Contains(string(installArgsFromFile), "val1") {
		return []byte("error"), []byte(""), fmt.Errorf("install args overrides file does not contain install args")
	}
	return []byte("success"), []byte(""), nil
}

func TestPostUpgrade(t *testing.T) {
	assert := assert.New(t)

	comp := IstioComponent{}

	config.SetDefaultBomFilePath(testBomFilePath)
	istio.SetCmdRunner(fakeRunner{})
	defer istio.SetDefaultRunner()
	SetIstioUpgradeFunction(fakeUpgrade)
	defer ResetIstioUpgradeFunction()
	err := comp.PostUpgrade(spi.NewContext(zap.S(), nil, vz, false))
	assert.NoError(err, "PostUpgrade returned an error")
}

func fakePostUpgrade(log *zap.SugaredLogger, releaseName string, namespace string, dryRun bool) (stdout []byte, stderr []byte, err error) {
	if releaseName != "istiocoredns" {
		return []byte("error"), []byte(""), fmt.Errorf("expected release name istiocoredns does not match provided release name of %v", releaseName)
	}
	if releaseName != "istio-system" {
		return []byte("error"), []byte(""), fmt.Errorf("expected namespace istio-system does not match provided namespace of %v", namespace)
	}
	return []byte("success"), []byte(""), nil
}

func getMock(t *testing.T) *mocks.MockClient {
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, deployList *appsv1.DeploymentList) error {
			deployList.Items = []appsv1.Deployment{{}}
			return nil
		})

	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, ssList *appsv1.StatefulSetList) error {
			ssList.Items = []appsv1.StatefulSet{{}}
			return nil
		})

	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, dsList *appsv1.DaemonSetList) error {
			dsList.Items = []appsv1.DaemonSet{{}}
			return nil
		})

	mock.EXPECT().
		Update(gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, deploy *appsv1.Deployment) error {
			deploy.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
			deploy.Spec.Template.ObjectMeta.Annotations["verrazzano.io/restartedAt"] = "some time"
			return nil
		}).AnyTimes()

	mock.EXPECT().
		Update(gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, ss *appsv1.StatefulSet) error {
			ss.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
			ss.Spec.Template.ObjectMeta.Annotations["verrazzano.io/restartedAt"] = "some time"
			return nil
		}).AnyTimes()

	mock.EXPECT().
		Update(gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, ds *appsv1.DaemonSet) error {
			ds.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
			ds.Spec.Template.ObjectMeta.Annotations["verrazzano.io/restartedAt"] = "some time"
			return nil
		}).AnyTimes()
	return mock
}

// fakeRunner overrides the istio run command
func (r fakeRunner) Run(cmd *exec.Cmd) (stdout []byte, stderr []byte, err error) {
	return []byte("success"), []byte(""), nil
}
