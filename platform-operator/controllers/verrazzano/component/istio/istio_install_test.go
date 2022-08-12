// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"sigs.k8s.io/yaml"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	globalconst "github.com/verrazzano/verrazzano/pkg/constants"
	spi2 "github.com/verrazzano/verrazzano/pkg/controller/errors"
	"github.com/verrazzano/verrazzano/pkg/istio"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/mocks"
	istiosec "istio.io/api/security/v1beta1"
	istioclisec "istio.io/client-go/pkg/apis/security/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
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
				Ingress: &installv1alpha1.IstioIngressSection{
					Kubernetes: &installv1alpha1.IstioKubernetesSection{
						CommonKubernetesSpec: installv1alpha1.CommonKubernetesSpec{
							Replicas: 1,
							Affinity: &corev1.Affinity{
								PodAntiAffinity: &corev1.PodAntiAffinity{
									RequiredDuringSchedulingIgnoredDuringExecution: nil,
									PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
										{
											Weight: 100,
											PodAffinityTerm: corev1.PodAffinityTerm{
												LabelSelector: &metav1.LabelSelector{
													MatchLabels: nil,
													MatchExpressions: []metav1.LabelSelectorRequirement{
														{
															Key:      "app",
															Operator: "In",
															Values: []string{
																"istio-ingressgateway",
															},
														},
													},
												},
												Namespaces:  nil,
												TopologyKey: "kubernetes.io/hostname",
											},
										},
									},
								},
							},
						},
					},
				},
				Egress: &installv1alpha1.IstioEgressSection{
					Kubernetes: &installv1alpha1.IstioKubernetesSection{
						CommonKubernetesSpec: installv1alpha1.CommonKubernetesSpec{
							Replicas: 1,
							Affinity: &corev1.Affinity{
								PodAntiAffinity: &corev1.PodAntiAffinity{
									RequiredDuringSchedulingIgnoredDuringExecution: nil,
									PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
										{
											Weight: 100,
											PodAffinityTerm: corev1.PodAffinityTerm{
												LabelSelector: &metav1.LabelSelector{
													MatchLabels: nil,
													MatchExpressions: []metav1.LabelSelectorRequirement{
														{
															Key:      "app",
															Operator: "In",
															Values: []string{
																"istio-egressgateway",
															},
														},
													},
												},
												Namespaces:  nil,
												TopologyKey: "kubernetes.io/hostname",
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	},
}

func testCR() *installv1alpha1.Verrazzano {
	return installCR.DeepCopy()
}

type fakeMonitor struct {
	result          bool
	istioctlSuccess bool
	err             error
	running         bool
}

func (f *fakeMonitor) run(args installRoutineParams) {
}

func (f *fakeMonitor) checkResult() (bool, error) { return f.result, f.err }

func (f *fakeMonitor) reset() {}

func (f *fakeMonitor) init() {}

func (f *fakeMonitor) sendResult(r bool) {}

func (f *fakeMonitor) isRunning() bool { return f.running }

func (f *fakeMonitor) isIstioctlSuccess() bool { return f.istioctlSuccess }

var _ installMonitor = &fakeMonitor{}

// TestAppendOverrideFilesInOrder tests if the override files are appended in reverse order
// GIVEN a component
//  WHEN I call appendOverrideFilesInOrder
//  THEN the overrides are appended in reverse order
func TestAppendOverrideFilesInOrder(t *testing.T) {
	type Foo struct {
		Foo string `json:"foo"`
	}
	cr := testCR()
	dat1 := []byte(`{"foo": "a"}`)
	dat2 := []byte(`{"foo": "b"}`)
	cr.Spec.Components.Istio.ValueOverrides = []installv1alpha1.Overrides{
		{
			Values: &apiextensionsv1.JSON{
				Raw: dat1,
			},
		},
		{
			Values: &apiextensionsv1.JSON{
				Raw: dat2,
			},
		},
	}
	c := fake.NewClientBuilder().WithScheme(testScheme).Build()
	ctx := spi.NewFakeContext(c, cr, false)
	files, err := appendOverrideFilesInOrder(ctx, []string{})

	equalFoos := func(jsonFoo, yamlFoo []byte) {
		jsonFooObj := &Foo{}
		yamlFooObj := &Foo{}
		err := yaml.Unmarshal(yamlFoo, yamlFooObj)
		assert.NoError(t, err)
		err = json.Unmarshal(jsonFoo, jsonFooObj)
		assert.NoError(t, err)
		assert.Equal(t, jsonFooObj, yamlFooObj)
	}
	assert.NoError(t, err)
	assert.Len(t, files, 2)
	fileDat1, err := os.ReadFile(files[0])
	assert.NoError(t, err)
	equalFoos(dat2, fileDat1)
	fileDat2, err := os.ReadFile(files[1])
	assert.NoError(t, err)
	equalFoos(dat1, fileDat2)
}

// TestIsOperatorInstallSupported tests if the install is supported
// GIVEN a component
//  WHEN I call IsOperatorInstallSupported
//  THEN true is returned
func TestIsOperatorInstallSupported(t *testing.T) {
	a := assert.New(t)

	b := comp.IsOperatorInstallSupported()
	a.True(b, "IsOperatorInstallSupported returned the wrong value")
}

// TestIsInstalled tests if the component is installed
// GIVEN a component
//  WHEN I call IsInstalled
//  THEN true is returned
func TestIsInstalled(t *testing.T) {
	a := assert.New(t)

	istio.SetCmdRunner(fakeIstioInstalledRunner{})
	b, err := comp.IsInstalled(spi.NewFakeContext(getIsInstalledMock(t), installCR, false))
	a.NoError(err, "IsInstalled returned an error")
	a.True(b, "IsInstalled returned false")
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
	a := assert.New(t)

	istio.SetCmdRunner(fakeIstioInstalledRunner{})
	b, err := comp.IsInstalled(spi.NewFakeContext(getIsNotInstalledMock(t), installCR, false))
	a.NoError(err, "IsInstalled returned an error")
	a.False(b, "IsInstalled returned true")
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
	a := assert.New(t)

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
	a.Equal(expectedErr, err, "Upgrade returned an unexpected error")
}

// TestBackgroundInstallCompletedSuccessfully tests the component install
// GIVEN a call to istioComponent.Install()
//  WHEN when the monitor goroutine failed to successfully complete
//  THEN the Install() method returns nil without calling the forkInstall function
func TestBackgroundInstallCompletedSuccessfully(t *testing.T) {
	a := assert.New(t)

	comp := istioComponent{
		ValuesFile: "test-values-file.yaml",
	}

	forkInstallFunc = func(_ spi.ComponentContext, _ installMonitor, _ string, _ []string) error {
		a.Fail("Unexpected call to forkInstall() function")
		return nil
	}
	defer func() { forkInstallFunc = forkInstall }()

	config.SetDefaultBomFilePath(testBomFilePath)
	istio.SetCmdRunner(fakeRunner{})
	setInstallFunc(fakeInstall)
	setBashFunc(fakeBash)

	comp.monitor = &fakeMonitor{result: true, running: true}
	err := comp.Install(spi.NewFakeContext(getIstioInstallMock(t), installCR, false))
	a.NoError(err)
}

// TestBackgroundInstallRetryOnFailure tests the component install
// GIVEN a call to istioComponent.Install()
//  WHEN when the monitor goroutine failed to successfully complete
//  THEN the Install() method calls the forkInstall function and returns a retry error
func TestBackgroundInstallRetryOnFailure(t *testing.T) {
	a := assert.New(t)

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
	a.True(forkFuncCalled)
	a.Equal(expectedErr, err)
}

// Test_forkInstallSuccess tests the forkInstall function
// GIVEN a call to istioComponent.forkInstall()
//  WHEN when the monitor install successfully runs istioctl install
//  THEN retryerrors are returned until the goroutine completes, and sends a success message
func Test_forkInstallSuccess(t *testing.T) {
	a := assert.New(t)

	comp := istioComponent{
		ValuesFile: "test-values-file.yaml",
	}

	config.SetDefaultBomFilePath(testBomFilePath)
	istio.SetCmdRunner(fakeRunner{})

	expectedOverridesFiles := []string{comp.ValuesFile, "istio-overrides.yaml"}
	expectedOverridesString := "myoverride=true"

	setInstallFunc(func(log vzlog.VerrazzanoLogger, overridesString string, overridesFiles ...string) (stdout []byte, stderr []byte, err error) {
		a.Equal(expectedOverridesFiles, overridesFiles, "Did not get expected override files")
		a.Equal(expectedOverridesString, overridesString)
		return []byte(""), []byte(""), nil
	})
	defer func() { installFunc = istio.Install }()

	setBashFunc(fakeBash)

	var monitor installMonitor = &installMonitorType{}
	err := forkInstall(spi.NewFakeContext(getIstioInstallMock(t), installCR, false), monitor, expectedOverridesString, expectedOverridesFiles)
	a.Equal(spi2.RetryableError{Source: ComponentName}, err)
	for i := 0; i < 100; i++ {
		result, retryError := monitor.checkResult()
		if retryError != nil {
			t.Log("Waiting for result...")
			time.Sleep(100 * time.Millisecond)
			continue
		}
		a.True(result)
		a.Nil(retryError)
		return
	}
	a.Fail("Did not detect completion in time")
}

// Test_forkInstallFailure tests the forkInstall function
// GIVEN a call to istioComponent.forkInstall()
//  WHEN when the monitor install unsuccessfully runs istioctl install
//  THEN retryerrors are returned until the goroutine completes, and sends a failure message when istioctl fails
func Test_forkInstallFailure(t *testing.T) {
	a := assert.New(t)

	comp := istioComponent{
		ValuesFile: "test-values-file.yaml",
	}

	config.SetDefaultBomFilePath(testBomFilePath)
	istio.SetCmdRunner(fakeRunner{})

	cause := fmt.Errorf("Unexpected error on install")
	setInstallFunc(func(log vzlog.VerrazzanoLogger, imageOverridesString string, overridesFiles ...string) (stdout []byte, stderr []byte, err error) {
		return []byte(""), []byte(""), cause
	})
	defer func() { installFunc = istio.Install }()

	setBashFunc(fakeBash)

	var monitor installMonitor = &installMonitorType{}
	err := forkInstall(spi.NewFakeContext(getIstioInstallMock(t), installCR, false), monitor, "myoverride=true", []string{comp.ValuesFile, "istio-overrides.yaml"})
	a.Equal(spi2.RetryableError{Source: ComponentName}, err)
	for i := 0; i < 100; i++ {
		result, retryError := monitor.checkResult()
		if retryError != nil {
			t.Log("Waiting for result...")
			time.Sleep(100 * time.Millisecond)
			continue
		}
		a.False(result)
		a.Nil(retryError)
		return
	}
	a.Fail("Did not detect completion in time")
}

func getIstioInstallMock(t *testing.T) *mocks.MockClient {
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.DefaultNamespace, Name: constants.GlobalImagePullSecName}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: constants.DefaultNamespace, Resource: "Secret"}, constants.GlobalImagePullSecName)).AnyTimes()

	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: IstioNamespace, Name: constants.GlobalImagePullSecName}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: IstioNamespace, Resource: "Secret"}, constants.GlobalImagePullSecName)).AnyTimes()
	return mock
}

// TestCreateCertSecret tests the cert secret
// GIVEN a component
//  WHEN I call createCertSecret
//  THEN the bash function is called to create the secret
func TestCreateCertSecret(t *testing.T) {
	a := assert.New(t)

	setBashFunc(fakeBash)
	err := createCertSecret(spi.NewFakeContext(createCertSecretMock(t), installCR, false))
	a.NoError(err, "createCertSecret returned an error")
}

func createCertSecretMock(t *testing.T) *mocks.MockClient {
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.DefaultNamespace, Name: constants.GlobalImagePullSecName}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: constants.DefaultNamespace, Resource: "Secret"}, constants.GlobalImagePullSecName)).AnyTimes()

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
	a := assert.New(t)

	setBashFunc(fakeBash)
	err := createPeerAuthentication(spi.NewFakeContext(createPeerAuthenticationMock(t), installCR, false))
	a.NoError(err, "createPeerAuthentication returned an error")
}

func createPeerAuthenticationMock(t *testing.T) *mocks.MockClient {
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: IstioNamespace, Name: "default"}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: IstioNamespace, Resource: "PeerAuthentication"}, "default"))

	// Expect a call to create the PeerAuthentication
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
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
	a := assert.New(t)

	setBashFunc(fakeBash)
	err := labelNamespace(spi.NewFakeContext(labelNamespaceMock(t), installCR, false))
	a.NoError(err, "labelNamespace returned an error")
}

// TestVerifyIstioIngressGatewayIP tests verifying the external IP is created for the Istio service
// GIVEN a call to verifyIstioIngressGatewayIP
// WHEN the service has an external IP
// THEN no error is returned
func TestVerifyIstioIngressGatewayIP(t *testing.T) {
	a := assert.New(t)

	ipaddr := "0.0.0.0"
	svcName := IstioIngressgatewayDeployment
	svcNamespace := globalconst.IstioSystemNamespace
	svcNoIP := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      svcName,
			Namespace: svcNamespace,
		},
	}

	svcIP := svcNoIP.DeepCopy()
	svcIP.Status.LoadBalancer.Ingress = []corev1.LoadBalancerIngress{
		{
			IP: ipaddr,
		},
	}

	lbVZ := &installv1alpha1.Verrazzano{}
	lbVZ.Spec.Components.Istio = &installv1alpha1.IstioComponent{
		Ingress: &installv1alpha1.IstioIngressSection{
			Type: installv1alpha1.LoadBalancer,
		},
	}

	npVZ := &installv1alpha1.Verrazzano{}
	npVZ.Spec.Components.Istio = &installv1alpha1.IstioComponent{
		Ingress: &installv1alpha1.IstioIngressSection{
			Type: installv1alpha1.NodePort,
		},
	}

	tests := []struct {
		name        string
		service     *corev1.Service
		vz          *installv1alpha1.Verrazzano
		expectError bool
	}{
		{
			name:        "test no service",
			service:     &corev1.Service{},
			vz:          lbVZ,
			expectError: true,
		},
		{
			name:        "test no IP",
			service:     svcNoIP,
			vz:          lbVZ,
			expectError: true,
		},
		{
			name:        "test external IP",
			service:     svcIP,
			vz:          lbVZ,
			expectError: false,
		},
		{
			name:        "test NodePort",
			service:     svcIP,
			vz:          npVZ,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			cli := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(tt.service).Build()
			ip, err := verifyIstioIngressGatewayIP(cli, tt.vz)
			if tt.expectError {
				a.Error(err)
				return
			}
			a.NoError(err)
			a.Equal(ip, ipaddr)
		})
	}
}

func labelNamespaceMock(t *testing.T) *mocks.MockClient {
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "", Name: IstioNamespace}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: IstioNamespace, Resource: "Namespace"}, IstioNamespace))

	// Expect a call to create the NameSpace
	mock.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, ns *corev1.Namespace, opts ...client.CreateOption) error {
			if ns.ObjectMeta.Labels == nil {
				ns.ObjectMeta.Labels["verrazzano.io/namespace"] = IstioNamespace
			}
			return nil
		})

	return mock
}

// fakeUpgrade verifies that the correct parameter values are passed to upgrade
func fakeInstall(log vzlog.VerrazzanoLogger, _ string, overridesFiles ...string) (stdout []byte, stderr []byte, err error) {
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
