// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	"context"
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
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
	b, err := comp.IsInstalled(spi.NewContext(zap.S(), nil, installCR, false))
	assert.NoError(err, "IsInstalled returned an error")
	assert.True(b, "IsInstalled returned false")
}

// TestInstall tests the component install
// GIVEN a component
//  WHEN I call Install
//  THEN the install returns success and passes the correct values to the install function
func TestInstall(t *testing.T) {
	assert := assert.New(t)

	comp := IstioComponent{
		ValuesFile: "test-values-file.yaml",
	}

	config.SetDefaultBomFilePath(testBomFilePath)
	istio.SetCmdRunner(fakeRunner{})
	setInstallFunc(fakeInstall)
	setBashFunc(fakeBash)
	err := comp.Install(spi.NewContext(zap.S(), getIstioInstallMock(t), installCR, false))
	assert.NoError(err, "Upgrade returned an error")
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
	err := createCertSecret(spi.NewContext(zap.S(), createCertSecretMock(t), installCR, false))
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
	err := createPeerAuthentication(spi.NewContext(zap.S(), createPeerAuthenticationMock(t), installCR, false))
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
	err := labelNamespace(spi.NewContext(zap.S(), labelNamespaceMock(t), installCR, false))
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

// TestCreateEnvoyFilter tests creating the Envoy filter
// GIVEN a component
//  WHEN I call createEnvoyFilter
//  THEN the bash function is called to create the filter
func TestCreateEnvoyFilter(t *testing.T) {
	assert := assert.New(t)

	setBashFunc(fakeBash)
	err := createEnvoyFilter(spi.NewContext(zap.S(), createEnvoyFilterMock(t), installCR, false))
	assert.NoError(err, "createEnvoyFilter returned an error")
}

func createEnvoyFilterMock(t *testing.T) *mocks.MockClient {
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: IstioNamespace, Name: IstioEnvoyFilter}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: IstioNamespace, Resource: "EnvoyFilter"}, IstioEnvoyFilter))

	return mock
}

// fakeUpgrade verifies that the correct parameter values are passed to upgrade
func fakeInstall(log *zap.SugaredLogger, imageOverridesString string, overridesFiles ...string) (stdout []byte, stderr []byte, err error) {
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
