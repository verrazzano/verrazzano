// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	"context"
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	spi2 "github.com/verrazzano/verrazzano/pkg/controller/errors"
	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/istio"
	"github.com/verrazzano/verrazzano/platform-operator/mocks"
	"go.uber.org/zap"
	"io/ioutil"
	istiosec "istio.io/api/security/v1beta1"
	istioclisec "istio.io/client-go/pkg/apis/security/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"os/exec"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"testing"
	"time"
)

// fakeIstioInstalledRunner is used to test if Istio is installed
type fakeIstioInstalledRunner struct {
}

var installCR = &installv1alpha1.Verrazzano{
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

type fakeMonitor struct {
	result  bool
	err     error
	running bool
}

func (f *fakeMonitor) run(args installRoutineParams) {
}

func (f *fakeMonitor) checkResult() (bool, error) { return f.result, f.err }

func (f *fakeMonitor) reset() {}

func (f *fakeMonitor) init() {}

func (f *fakeMonitor) sendResult(r bool) {}

func (f *fakeMonitor) isRunning() bool { return f.running }

var _ installMonitor = &fakeMonitor{}

// TestIsOperatorInstallSupported tests if the install is supported
// GIVEN a component
//  WHEN I call IsOperatorInstallSupported
//  THEN true is returned
func TestIsOperatorInstallSupported(t *testing.T) {
	assert := assert.New(t)

	b := comp.IsOperatorInstallSupported()
	assert.True(b, "IsOperatorInstallSupported returned the wrong value")
}

// TestIsInstalled tests if the component is installed
// GIVEN a component
//  WHEN I call IsInstalled
//  THEN true is returned
func TestIsInstalled(t *testing.T) {
	assert := assert.New(t)

	istio.SetCmdRunner(fakeIstioInstalledRunner{})
	b, err := comp.IsInstalled(spi.NewFakeContext(getIsInstalledMock(t), installCR, false))
	assert.NoError(err, "IsInstalled returned an error")
	assert.True(b, "IsInstalled returned false")
}

func getIsInstalledMock(t *testing.T) *mocks.MockClient {
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	// Expect a call to create the PeerAuthentication
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: IstioNamespace, Name: IstiodDeployment}, gomock.Not(gomock.Nil())).
		Return(nil)
	return mock
}

// TestIsNotInstalled tests if the component is not installed
// GIVEN a component
//  WHEN I call IsInstalled
//  THEN false is returned
func TestIsNotInstalled(t *testing.T) {
	assert := assert.New(t)

	istio.SetCmdRunner(fakeIstioInstalledRunner{})
	b, err := comp.IsInstalled(spi.NewFakeContext(getIsNotInstalledMock(t), installCR, false))
	assert.NoError(err, "IsInstalled returned an error")
	assert.False(b, "IsInstalled returned true")
}

func getIsNotInstalledMock(t *testing.T) *mocks.MockClient {
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	// Expect a call to create the PeerAuthentication
	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: IstioNamespace, Name: IstiodDeployment}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: IstioNamespace, Resource: "Deployment"}, IstiodDeployment))
	return mock
}

// TestInstall tests the component install
// GIVEN a component
//  WHEN I call Install
//  THEN the install starts a new attempt and returns a RetryableError to requeue
func TestInstall(t *testing.T) {
	assert := assert.New(t)

	comp := istioComponent{
		ValuesFile: "test-values-file.yaml",
		monitor:    &fakeMonitor{result: true, running: false},
	}

	expectedErr := spi2.RetryableError{Source: ComponentName}
	forkInstallFunc = func(_ spi.ComponentContext, _ installMonitor, _ string, _ []string) error {
		return expectedErr
	}
	defer func() { forkInstallFunc = forkInstall }()

	config.SetDefaultBomFilePath(testBomFilePath)
	istio.SetCmdRunner(fakeRunner{})
	setInstallFunc(fakeInstall)
	setBashFunc(fakeBash)

	err := comp.Install(spi.NewFakeContext(getIstioInstallMock(t), installCR, false))
	assert.Equal(expectedErr, err, "Upgrade returned an unexpected error")
}

// TestBackgroundInstallCompletedSuccessfully tests the component install
// GIVEN a call to istioComponent.Install()
//  WHEN when the monitor goroutine failed to successfully complete
//  THEN the Install() method returns nil without calling the forkInstall function
func TestBackgroundInstallCompletedSuccessfully(t *testing.T) {
	assert := assert.New(t)

	comp := istioComponent{
		ValuesFile: "test-values-file.yaml",
	}

	forkInstallFunc = func(_ spi.ComponentContext, _ installMonitor, _ string, _ []string) error {
		assert.Fail("Unexpected call to forkInstall() function")
		return nil
	}
	defer func() { forkInstallFunc = forkInstall }()

	config.SetDefaultBomFilePath(testBomFilePath)
	istio.SetCmdRunner(fakeRunner{})
	setInstallFunc(fakeInstall)
	setBashFunc(fakeBash)

	comp.monitor = &fakeMonitor{result: true, running: true}
	err := comp.Install(spi.NewFakeContext(getIstioInstallMock(t), installCR, false))
	assert.NoError(err)
}

// TestBackgroundInstallRetryOnFailure tests the component install
// GIVEN a call to istioComponent.Install()
//  WHEN when the monitor goroutine failed to successfully complete
//  THEN the Install() method calls the forkInstall function and returns a retry error
func TestBackgroundInstallRetryOnFailure(t *testing.T) {
	assert := assert.New(t)

	comp := istioComponent{
		ValuesFile: "test-values-file.yaml",
	}

	forkFuncCalled := false
	expectedErr := spi2.RetryableError{Source: ComponentName}
	forkInstallFunc = func(_ spi.ComponentContext, _ installMonitor, _ string, _ []string) error {
		forkFuncCalled = true
		return expectedErr
	}
	defer func() { forkInstallFunc = forkInstall }()

	config.SetDefaultBomFilePath(testBomFilePath)
	istio.SetCmdRunner(fakeRunner{})
	setInstallFunc(fakeInstall)
	setBashFunc(fakeBash)

	comp.monitor = &fakeMonitor{result: false, running: true}

	err := comp.Install(spi.NewFakeContext(getIstioInstallMock(t), installCR, false))
	assert.True(forkFuncCalled)
	assert.Equal(expectedErr, err)
}

// Test_forkInstallSuccess tests the forkInstall function
// GIVEN a call to istioComponent.forkInstall()
//  WHEN when the monitor install successfully runs istioctl install
//  THEN retryerrors are returned until the goroutine completes, and sends a success message
func Test_forkInstallSuccess(t *testing.T) {
	assert := assert.New(t)

	comp := istioComponent{
		ValuesFile: "test-values-file.yaml",
	}

	config.SetDefaultBomFilePath(testBomFilePath)
	istio.SetCmdRunner(fakeRunner{})

	expectedOverridesFiles := []string{comp.ValuesFile, "istio-overrides.yaml"}
	expectedOverridesString := "myoverride=true"

	setInstallFunc(func(log *zap.SugaredLogger, overridesString string, overridesFiles ...string) (stdout []byte, stderr []byte, err error) {
		assert.Equal(expectedOverridesFiles, overridesFiles, "Did not get expected override files")
		assert.Equal(expectedOverridesString, overridesString)
		return []byte(""), []byte(""), nil
	})
	defer func() { installFunc = istio.Install }()

	setBashFunc(fakeBash)

	var monitor installMonitor = &installMonitorType{}
	err := forkInstall(spi.NewFakeContext(getIstioInstallMock(t), installCR, false), monitor, expectedOverridesString, expectedOverridesFiles)
	assert.Equal(spi2.RetryableError{Source: ComponentName}, err)
	for i := 0; i < 100; i++ {
		result, retryError := monitor.checkResult()
		if retryError != nil {
			t.Log("Waiting for result...")
			time.Sleep(100 * time.Millisecond)
			continue
		}
		assert.True(result)
		assert.Nil(retryError)
		return
	}
	assert.Fail("Did not detect completion in time")
}

// Test_forkInstallFailure tests the forkInstall function
// GIVEN a call to istioComponent.forkInstall()
//  WHEN when the monitor install unsuccessfully runs istioctl install
//  THEN retryerrors are returned until the goroutine completes, and sends a failure message when istioctl fails
func Test_forkInstallFailure(t *testing.T) {
	assert := assert.New(t)

	comp := istioComponent{
		ValuesFile: "test-values-file.yaml",
	}

	config.SetDefaultBomFilePath(testBomFilePath)
	istio.SetCmdRunner(fakeRunner{})

	cause := fmt.Errorf("Unexpected error on install")
	setInstallFunc(func(log *zap.SugaredLogger, imageOverridesString string, overridesFiles ...string) (stdout []byte, stderr []byte, err error) {
		return []byte(""), []byte(""), cause
	})
	defer func() { installFunc = istio.Install }()

	setBashFunc(fakeBash)

	var monitor installMonitor = &installMonitorType{}
	err := forkInstall(spi.NewFakeContext(getIstioInstallMock(t), installCR, false), monitor, "myoverride=true", []string{comp.ValuesFile, "istio-overrides.yaml"})
	assert.Equal(spi2.RetryableError{Source: ComponentName}, err)
	for i := 0; i < 100; i++ {
		result, retryError := monitor.checkResult()
		if retryError != nil {
			t.Log("Waiting for result...")
			time.Sleep(100 * time.Millisecond)
			continue
		}
		assert.False(result)
		assert.Nil(retryError)
		return
	}
	assert.Fail("Did not detect completion in time")
}

func getIstioInstallMock(t *testing.T) *mocks.MockClient {
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
			deploy.Spec.Template.ObjectMeta.Annotations[constants.VerrazzanoRestartAnnotation] = "some time"
			return nil
		}).AnyTimes()

	mock.EXPECT().
		Update(gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, ss *appsv1.StatefulSet) error {
			ss.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
			ss.Spec.Template.ObjectMeta.Annotations[constants.VerrazzanoRestartAnnotation] = "some time"
			return nil
		}).AnyTimes()

	mock.EXPECT().
		Update(gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, ds *appsv1.DaemonSet) error {
			ds.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
			ds.Spec.Template.ObjectMeta.Annotations[constants.VerrazzanoRestartAnnotation] = "some time"
			return nil
		}).AnyTimes()

	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.DefaultNamespace, Name: constants.GlobalImagePullSecName}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: constants.DefaultNamespace, Resource: "Secret"}, constants.GlobalImagePullSecName))

	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: IstioNamespace, Name: constants.GlobalImagePullSecName}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: IstioNamespace, Resource: "Secret"}, constants.GlobalImagePullSecName))

	return mock
}

// TestCreateCertSecret tests the cert secret
// GIVEN a component
//  WHEN I call createCertSecret
//  THEN the bash function is called to create the secret
func TestCreateCertSecret(t *testing.T) {
	assert := assert.New(t)

	setBashFunc(fakeBash)
	err := createCertSecret(spi.NewFakeContext(createCertSecretMock(t), installCR, false))
	assert.NoError(err, "createCertSecret returned an error")
}

func createCertSecretMock(t *testing.T) *mocks.MockClient {
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
			deploy.Spec.Template.ObjectMeta.Annotations[constants.VerrazzanoRestartAnnotation] = "some time"
			return nil
		}).AnyTimes()

	mock.EXPECT().
		Update(gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, ss *appsv1.StatefulSet) error {
			ss.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
			ss.Spec.Template.ObjectMeta.Annotations[constants.VerrazzanoRestartAnnotation] = "some time"
			return nil
		}).AnyTimes()

	mock.EXPECT().
		Update(gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, ds *appsv1.DaemonSet) error {
			ds.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
			ds.Spec.Template.ObjectMeta.Annotations[constants.VerrazzanoRestartAnnotation] = "some time"
			return nil
		}).AnyTimes()

	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.DefaultNamespace, Name: constants.GlobalImagePullSecName}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: constants.DefaultNamespace, Resource: "Secret"}, constants.GlobalImagePullSecName))

	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: IstioNamespace, Name: IstioCertSecret}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: IstioNamespace, Resource: "Secret"}, IstioCertSecret))

	return mock
}

// TestCreatePeerAuthentication tests creating the PeerAuthentication resource
// GIVEN a component
//  WHEN I call createPeerAuthentication
//  THEN the PeerAuthentication resource is created with STRICT MTLS
func TestCreatePeerAuthentication(t *testing.T) {
	assert := assert.New(t)

	setBashFunc(fakeBash)
	err := createPeerAuthentication(spi.NewFakeContext(createPeerAuthenticationMock(t), installCR, false))
	assert.NoError(err, "createPeerAuthentication returned an error")
}

func createPeerAuthenticationMock(t *testing.T) *mocks.MockClient {
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: IstioNamespace, Name: "default"}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: IstioNamespace, Resource: "PeerAuthentication"}, "default"))

	// Expect a call to create the PeerAuthentication
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, peer *istioclisec.PeerAuthentication, opts ...client.CreateOption) error {
			if peer.Spec.Mtls.Mode != istiosec.PeerAuthentication_MutualTLS_STRICT {
				return errors.NewBadRequest("MTLS should be STRICT")
			}
			return nil
		})

	return mock
}

// TestLabelNamespace tests creating the labelNamespace resource
// GIVEN a component
//  WHEN I call labelNamespace
//  THEN the namespace is labelled
func TestLabelNamespace(t *testing.T) {
	assert := assert.New(t)

	setBashFunc(fakeBash)
	err := labelNamespace(spi.NewFakeContext(labelNamespaceMock(t), installCR, false))
	assert.NoError(err, "labelNamespace returned an error")
}

func labelNamespaceMock(t *testing.T) *mocks.MockClient {
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "", Name: IstioNamespace}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: IstioNamespace, Resource: "Namespace"}, IstioNamespace))

	// Expect a call to create the NameSpace
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, ns *corev1.Namespace, opts ...client.CreateOption) error {
			if ns.ObjectMeta.Labels == nil {
				ns.ObjectMeta.Labels["verrazzano.io/namespace"] = IstioNamespace
			}
			return nil
		})

	return mock
}

// fakeUpgrade verifies that the correct parameter values are passed to upgrade
func fakeInstall(log *zap.SugaredLogger, _ string, overridesFiles ...string) (stdout []byte, stderr []byte, err error) {
	if len(overridesFiles) != 2 {
		return []byte("error"), []byte(""), fmt.Errorf("incorrect number of override files: expected 2, received %v", len(overridesFiles))
	}
	if overridesFiles[0] != "test-values-file.yaml" {
		return []byte("error"), []byte(""), fmt.Errorf("invalid values file")
	}
	if !strings.Contains(overridesFiles[1], "istio-") || !strings.Contains(overridesFiles[1], ".yaml") {
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

// fakeBash verifies that the correct parameter values are passed to upgrade
func fakeBash(_ ...string) (string, string, error) {
	return "succes", "", nil
}

// fakeIsInstalledRunner overrides the istio run command
func (r fakeIstioInstalledRunner) Run(cmd *exec.Cmd) (stdout []byte, stderr []byte, err error) {
	return []byte("Istio is installed and verified successfully"), []byte(""), nil
}
