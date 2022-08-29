// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/istio"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"io/ioutil"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"os/exec"
	"strings"
	"testing"

	"github.com/verrazzano/verrazzano/pkg/k8sutil"

	"github.com/verrazzano/verrazzano/pkg/log/vzlog"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/helm"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/mocks"

	"github.com/verrazzano/verrazzano/pkg/test/ip"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gofake "k8s.io/client-go/kubernetes/fake"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// fakeRunner is used to test istio without actually running an OS exec command
type fakeRunner struct {
}

var (
	falseValue = false
	trueValue  = true
)

const profilesRelativePath = "../../../../manifests/profiles"

var testExternalIP = ip.RandomIP()

var crEnabled = vzapi.Verrazzano{
	Spec: vzapi.VerrazzanoSpec{
		Components: vzapi.ComponentSpec{
			Istio: &vzapi.IstioComponent{
				Enabled: getBoolPtr(true),
				Ingress: &vzapi.IstioIngressSection{
					Kubernetes: &vzapi.IstioKubernetesSection{
						CommonKubernetesSpec: vzapi.CommonKubernetesSpec{
							Replicas: 2,
							Affinity: &v1.Affinity{
								PodAntiAffinity: &v1.PodAntiAffinity{
									RequiredDuringSchedulingIgnoredDuringExecution: nil,
									PreferredDuringSchedulingIgnoredDuringExecution: []v1.WeightedPodAffinityTerm{
										{
											Weight: 100,
											PodAffinityTerm: v1.PodAffinityTerm{
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
				Egress: &vzapi.IstioEgressSection{
					Kubernetes: &vzapi.IstioKubernetesSection{
						CommonKubernetesSpec: vzapi.CommonKubernetesSpec{
							Replicas: 2,
							Affinity: &v1.Affinity{
								PodAntiAffinity: &v1.PodAntiAffinity{
									RequiredDuringSchedulingIgnoredDuringExecution: nil,
									PreferredDuringSchedulingIgnoredDuringExecution: []v1.WeightedPodAffinityTerm{
										{
											Weight: 100,
											PodAffinityTerm: v1.PodAffinityTerm{
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

var crInstall = &vzapi.Verrazzano{
	Spec: vzapi.VerrazzanoSpec{
		Version: "1.0",
		Components: vzapi.ComponentSpec{
			Istio: &vzapi.IstioComponent{
				IstioInstallArgs: []vzapi.InstallArgs{{
					Name:  "arg1",
					Value: "val1",
				}},
				Ingress: &vzapi.IstioIngressSection{
					Kubernetes: &vzapi.IstioKubernetesSection{
						CommonKubernetesSpec: vzapi.CommonKubernetesSpec{
							Replicas: 2,
							Affinity: &v1.Affinity{
								PodAntiAffinity: &v1.PodAntiAffinity{
									RequiredDuringSchedulingIgnoredDuringExecution: nil,
									PreferredDuringSchedulingIgnoredDuringExecution: []v1.WeightedPodAffinityTerm{
										{
											Weight: 100,
											PodAffinityTerm: v1.PodAffinityTerm{
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
				Egress: &vzapi.IstioEgressSection{
					Kubernetes: &vzapi.IstioKubernetesSection{
						CommonKubernetesSpec: vzapi.CommonKubernetesSpec{
							Replicas: 2,
							Affinity: &v1.Affinity{
								PodAntiAffinity: &v1.PodAntiAffinity{
									RequiredDuringSchedulingIgnoredDuringExecution: nil,
									PreferredDuringSchedulingIgnoredDuringExecution: []v1.WeightedPodAffinityTerm{
										{
											Weight: 100,
											PodAffinityTerm: v1.PodAffinityTerm{
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

var comp = istioComponent{}

const testBomFilePath = "../../testdata/test_bom.json"

// TestGetName tests the component name
// GIVEN a Verrazzano component
//  WHEN I call Name
//  THEN the correct Verrazzano name is returned
func TestGetName(t *testing.T) {
	a := assert.New(t)
	a.Equal("istio", comp.Name(), "Wrong component name")
}

// TestUpgrade tests the component upgrade
// GIVEN a component
//  WHEN I call Upgrade
//  THEN the upgrade returns success and passes the correct values to the upgrade function
func TestUpgrade(t *testing.T) {
	a := assert.New(t)

	comp := istioComponent{
		ValuesFile:               "test-values-file.yaml",
		Revision:                 "1-1-1",
		InjectedSystemNamespaces: config.GetInjectedSystemNamespaces(),
	}

	config.SetDefaultBomFilePath(testBomFilePath)
	SetIstioUpgradeFunction(fakeUpgrade)
	defer SetDefaultIstioUpgradeFunction()

	err := comp.Upgrade(spi.NewFakeContext(getMock(t), crInstall, nil, false))
	a.NoError(err, "Upgrade returned an error")
}

// fakeUpgrade verifies that the correct parameter values are passed to upgrade
func fakeUpgrade(log vzlog.VerrazzanoLogger, imageOverridesString string, overridesFiles ...string) (stdout []byte, stderr []byte, err error) {
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

func TestPostUpgrade(t *testing.T) {
	a := assert.New(t)

	comp := istioComponent{}

	// Setup fake client to provide workloads for restart platform testing
	clientSet := gofake.NewSimpleClientset()
	k8sutil.SetFakeClient(clientSet)

	config.SetDefaultBomFilePath(testBomFilePath)
	helm.SetCmdRunner(fakeRunner{})
	defer helm.SetDefaultRunner()
	SetHelmUninstallFunction(fakeHelmUninstall)
	SetDefaultHelmUninstallFunction()
	err := comp.PostUpgrade(spi.NewFakeContext(getMock(t), crInstall, nil, false))
	a.NoError(err, "PostUpgrade returned an error")
}

func fakeHelmUninstall(_ vzlog.VerrazzanoLogger, releaseName string, namespace string, dryRun bool) (stdout []byte, stderr []byte, err error) {
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
		List(gomock.Any(), &v1.SecretList{}, gomock.Any()).
		DoAndReturn(func(ctx context.Context, secretList *v1.SecretList, options ...client.ListOption) error {
			secretList.Items = []v1.Secret{{Type: HelmScrtType}, {Type: "generic"}, {Type: HelmScrtType}}
			return nil
		}).AnyTimes()

	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Name: constants.GlobalImagePullSecName, Namespace: "default"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, _ client.ObjectKey, _ *v1.Secret) error {
			return nil
		}).AnyTimes()

	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Name: constants.GlobalImagePullSecName, Namespace: IstioNamespace}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, _ client.ObjectKey, _ *v1.Secret) error {
			return nil
		}).AnyTimes()

	mock.EXPECT().
		Delete(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, secret *v1.Secret, opts ...client.DeleteOption) error {
			return nil
		}).AnyTimes()

	mock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Name: IstioIngressgatewayDeployment, Namespace: IstioNamespace}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, _ client.ObjectKey, svc *v1.Service) error {
			svc.Status.LoadBalancer.Ingress = []v1.LoadBalancerIngress{
				{
					IP: "0.0.0.0",
				},
			}
			return nil
		}).AnyTimes()

	return mock
}

// fakeRunner overrides the istio run command
func (r fakeRunner) Run(cmd *exec.Cmd) (stdout []byte, stderr []byte, err error) {
	return []byte("success"), []byte(""), nil
}

// TestIsReady tests the IsReady function
// GIVEN a call to IsReady
//  WHEN the deployment object has enough replicas available and istioctl ran successfully
//  THEN true is returned
func TestIsReady(t *testing.T) {

	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: IstioNamespace,
				Name:      IstiodDeployment,
				Labels:    map[string]string{"app": IstiodDeployment},
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": IstiodDeployment},
				},
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 1,
				Replicas:          1,
				UpdatedReplicas:   1,
			},
		},
		&v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: IstioNamespace,
				Name:      IstiodDeployment + "-95d8c5d96-m6mbr",
				Labels: map[string]string{
					"pod-template-hash": "95d8c5d96",
					"app":               IstiodDeployment,
				},
			},
		},
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   IstioNamespace,
				Name:        IstiodDeployment + "-95d8c5d96",
				Annotations: map[string]string{"deployment.kubernetes.io/revision": "1"},
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: IstioNamespace,
				Name:      IstioIngressgatewayDeployment,
				Labels:    map[string]string{"app": IstioIngressgatewayDeployment},
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": IstioIngressgatewayDeployment},
				},
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 1,
				Replicas:          1,
				UpdatedReplicas:   1,
			},
		},
		&v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: IstioNamespace,
				Name:      IstioIngressgatewayDeployment + "-95d8c5d96-m6mbr",
				Labels: map[string]string{
					"pod-template-hash": "95d8c5d96",
					"app":               IstioIngressgatewayDeployment,
				},
			},
		},
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   IstioNamespace,
				Name:        IstioIngressgatewayDeployment + "-95d8c5d96",
				Annotations: map[string]string{"deployment.kubernetes.io/revision": "1"},
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: IstioNamespace,
				Name:      IstioEgressgatewayDeployment,
				Labels:    map[string]string{"app": IstioEgressgatewayDeployment},
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": IstioEgressgatewayDeployment},
				},
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 1,
				Replicas:          1,
				UpdatedReplicas:   1,
			},
		},
		&v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: IstioNamespace,
				Name:      IstioEgressgatewayDeployment + "-95d8c5d96-m6mbr",
				Labels: map[string]string{
					"pod-template-hash": "95d8c5d96",
					"app":               IstioEgressgatewayDeployment,
				},
			},
		},
		&appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   IstioNamespace,
				Name:        IstioEgressgatewayDeployment + "-95d8c5d96",
				Annotations: map[string]string{"deployment.kubernetes.io/revision": "1"},
			},
		},
		&v1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: IstioNamespace,
				Name:      IstioIngressgatewayDeployment,
			},
			Status: v1.ServiceStatus{
				LoadBalancer: v1.LoadBalancerStatus{
					Ingress: []v1.LoadBalancerIngress{
						{
							IP: "0.0.0.0",
						},
					},
				},
			},
		},
	).Build()

	isInstalledFunc = func(log vzlog.VerrazzanoLogger) (bool, error) {
		return true, nil
	}
	defer func() { isInstalledFunc = istio.IsInstalled }()

	var iComp istioComponent
	compContext := spi.NewFakeContext(fakeClient, &vzapi.Verrazzano{}, nil, false)
	assert.True(t, iComp.IsReady(compContext))
}

// TestIsEnabledNilIstio tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Istio component is nil
//  THEN true is returned
func TestIsEnabledNilIstio(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Istio = nil
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, nil, false, profilesRelativePath).EffectiveCR()))
}

// TestIsEnabledNilComponent tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Istio component is nil
//  THEN false is returned
func TestIsEnabledNilComponent(t *testing.T) {
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &vzapi.Verrazzano{}, nil, false, profilesRelativePath).EffectiveCR()))
}

// TestIsEnabledNilEnabled tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Istio component enabled is nil
//  THEN true is returned
func TestIsEnabledNilEnabled(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Istio.Enabled = nil
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, nil, false, profilesRelativePath).EffectiveCR()))
}

// TestIsEnabledExplicit tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Istio component is explicitly enabled
//  THEN true is returned
func TestIsEnabledExplicit(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Istio.Enabled = getBoolPtr(true)
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, nil, false, profilesRelativePath).EffectiveCR()))
}

// TestIsDisableExplicit tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Istio component is explicitly disabled
//  THEN false is returned
func TestIsDisableExplicit(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Istio.Enabled = getBoolPtr(false)
	assert.False(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, nil, false, profilesRelativePath).EffectiveCR()))
}

func getBoolPtr(b bool) *bool {
	return &b
}

func Test_istioComponent_ValidateUpdate(t *testing.T) {
	disabled := false
	affinityChange := &v1.Affinity{
		NodeAffinity: &v1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &v1.NodeSelector{
				NodeSelectorTerms: []v1.NodeSelectorTerm{
					{MatchExpressions: []v1.NodeSelectorRequirement{{Key: "foo"}}},
				},
			},
		},
	}
	tests := []struct {
		name    string
		old     *vzapi.Verrazzano
		new     *vzapi.Verrazzano
		wantErr bool
	}{
		{
			name: "enable",
			old: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Istio: &vzapi.IstioComponent{
							Enabled: &disabled,
						},
					},
				},
			},
			new:     &vzapi.Verrazzano{},
			wantErr: false,
		},
		{
			name: "disable",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Istio: &vzapi.IstioComponent{
							Enabled: &disabled,
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "change-install-args",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Istio: &vzapi.IstioComponent{
							IstioInstallArgs: []vzapi.InstallArgs{{Name: "foo", Value: "bar"}},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "change-ingress-replicas",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Istio: &vzapi.IstioComponent{
							Ingress: &vzapi.IstioIngressSection{
								Kubernetes: &vzapi.IstioKubernetesSection{
									CommonKubernetesSpec: vzapi.CommonKubernetesSpec{
										Replicas: 5,
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "change-ingress-affinity",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Istio: &vzapi.IstioComponent{
							Ingress: &vzapi.IstioIngressSection{
								Kubernetes: &vzapi.IstioKubernetesSection{
									CommonKubernetesSpec: vzapi.CommonKubernetesSpec{
										Affinity: affinityChange,
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "change-egress-replicas",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Istio: &vzapi.IstioComponent{
							Egress: &vzapi.IstioEgressSection{
								Kubernetes: &vzapi.IstioKubernetesSection{
									CommonKubernetesSpec: vzapi.CommonKubernetesSpec{
										Replicas: 5,
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "change-eggress-affinity",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Istio: &vzapi.IstioComponent{
							Egress: &vzapi.IstioEgressSection{
								Kubernetes: &vzapi.IstioKubernetesSection{
									CommonKubernetesSpec: vzapi.CommonKubernetesSpec{
										Affinity: affinityChange,
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "change-type-to-nodeport-without-externalIPs",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Istio: &vzapi.IstioComponent{
							Ingress: &vzapi.IstioIngressSection{
								Type: vzapi.NodePort,
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "change-type-to-nodeport-with-externalIPs",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Istio: &vzapi.IstioComponent{
							Ingress: &vzapi.IstioIngressSection{
								Type: vzapi.NodePort,
							},
							IstioInstallArgs: []vzapi.InstallArgs{
								{
									Name:      ExternalIPArg,
									ValueList: []string{testExternalIP},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "change-type-from-nodeport",
			old: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Istio: &vzapi.IstioComponent{
							Ingress: &vzapi.IstioIngressSection{
								Type: vzapi.NodePort,
							},
						},
					},
				},
			},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Istio: &vzapi.IstioComponent{
							Ingress: &vzapi.IstioIngressSection{
								Type: vzapi.LoadBalancer,
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "change-ports",
			old:  &vzapi.Verrazzano{},
			new: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Istio: &vzapi.IstioComponent{
							Ingress: &vzapi.IstioIngressSection{
								Ports: []v1.ServicePort{{Name: "https2", NodePort: 30057}},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "no change",
			old:     &vzapi.Verrazzano{},
			new:     &vzapi.Verrazzano{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewComponent()
			if err := c.ValidateUpdate(tt.old, tt.new); (err != nil) != tt.wantErr {
				t.Errorf("ValidateUpdate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_istioComponent_ValidateInstall(t *testing.T) {
	tests := []struct {
		name    string
		vz      *vzapi.Verrazzano
		wantErr bool
	}{
		{
			name: "IstioComponentEmpty",
			vz: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Istio: &vzapi.IstioComponent{},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "IstioInstallArgsEmpty",
			vz: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Istio: &vzapi.IstioComponent{
							Ingress: &vzapi.IstioIngressSection{
								Type: vzapi.NodePort,
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "IstioInstallMissingKey",
			vz: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Istio: &vzapi.IstioComponent{
							Ingress: &vzapi.IstioIngressSection{
								Type: vzapi.NodePort,
							},
							IstioInstallArgs: []vzapi.InstallArgs{
								{
									Name:      "foo",
									ValueList: []string{testExternalIP},
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "IstioInstallMissingIP",
			vz: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Istio: &vzapi.IstioComponent{
							Ingress: &vzapi.IstioIngressSection{
								Type: vzapi.NodePort,
							},
							IstioInstallArgs: []vzapi.InstallArgs{
								{
									Name:  ExternalIPArg,
									Value: testExternalIP,
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "IstioInstallInvalidIP",
			vz: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Istio: &vzapi.IstioComponent{
							Ingress: &vzapi.IstioIngressSection{
								Type: vzapi.NodePort,
							},
							IstioInstallArgs: []vzapi.InstallArgs{
								{
									Name:      ExternalIPArg,
									ValueList: []string{testExternalIP + ".1"},
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "IstioInstallValidConfig",
			vz: &vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Istio: &vzapi.IstioComponent{
							Ingress: &vzapi.IstioIngressSection{
								Type: vzapi.NodePort,
							},
							IstioInstallArgs: []vzapi.InstallArgs{
								{
									Name:      ExternalIPArg,
									ValueList: []string{testExternalIP},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewComponent()
			if err := c.ValidateInstall(tt.vz); (err != nil) != tt.wantErr {
				t.Errorf("ValidateInstall() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestValidateUpdate tests the istio ValidateUpdate function
func TestValidateUpdate(t *testing.T) {
	oldVZ := vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Istio: &vzapi.IstioComponent{
					Enabled: &trueValue,
				},
			},
		},
	}
	newVZ := vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Istio: &vzapi.IstioComponent{
					Enabled: &falseValue,
				},
			},
		},
	}
	assert.Error(t, NewComponent().ValidateUpdate(&oldVZ, &newVZ))
}

// TestPostUninstall tests the PostUninstall function
// GIVEN a call to PostUninstall
//  WHEN the istio-system namespace exists with a finalizer
//  THEN true is returned and istio-system namespace is deleted
func TestPostUninstall(t *testing.T) {

	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:       IstioNamespace,
				Finalizers: []string{"fake-finalizer"},
			},
		},
		&rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name: istioReaderIstioSystem,
			},
		},
		&rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name: istiodIstioSystem,
			},
		},
		&rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: istioReaderIstioSystem,
			},
		},
		&rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: istiodIstioSystem,
			},
		},
	).Build()

	var iComp istioComponent
	compContext := spi.NewFakeContext(fakeClient, &vzapi.Verrazzano{}, nil, false)
	assert.NoError(t, iComp.PostUninstall(compContext))

	// Validate that the namespace does not exist
	ns := v1.Namespace{}
	err := compContext.Client().Get(context.TODO(), types.NamespacedName{Name: IstioNamespace}, &ns)
	assert.True(t, errors.IsNotFound(err))

	//Validate that clusterroles and clusterroles do not exist
	cr := rbacv1.ClusterRole{}
	err = compContext.Client().Get(context.TODO(), types.NamespacedName{Name: istioReaderIstioSystem}, &cr)
	assert.True(t, errors.IsNotFound(err))

	cr = rbacv1.ClusterRole{}
	err = compContext.Client().Get(context.TODO(), types.NamespacedName{Name: istiodIstioSystem}, &cr)
	assert.True(t, errors.IsNotFound(err))

	crb := rbacv1.ClusterRoleBinding{}
	err = compContext.Client().Get(context.TODO(), types.NamespacedName{Name: istioReaderIstioSystem}, &crb)
	assert.True(t, errors.IsNotFound(err))

	crb = rbacv1.ClusterRoleBinding{}
	err = compContext.Client().Get(context.TODO(), types.NamespacedName{Name: istiodIstioSystem}, &crb)
	assert.True(t, errors.IsNotFound(err))

}

// TestUninstall tests the Uninstall function is working
// GIVEN a call to Uninstall
//  WHEN the uninstall function is called
//  THEN success is returned
func TestUninstall(t *testing.T) {

	fakeUnInstallFunc := func(log vzlog.VerrazzanoLogger) (stdout []byte, stderr []byte, err error) {
		return []byte(""), []byte(""), nil
	}
	SetIstioUninstallFunction(fakeUnInstallFunc)
	defer SetDefaultIstioUninstallFunction()

	var iComp istioComponent
	compContext := spi.NewFakeContext(nil, nil, nil, false)
	assert.NoError(t, iComp.Uninstall(compContext))
}

func TestGetOverrides(t *testing.T) {
	ref := &corev1.ConfigMapKeySelector{
		Key: "foo",
	}
	o := v1beta1.InstallOverrides{
		ValueOverrides: []v1beta1.Overrides{
			{
				ConfigMapRef: ref,
			},
		},
	}
	oV1Alpha1 := vzapi.InstallOverrides{
		ValueOverrides: []vzapi.Overrides{
			{
				ConfigMapRef: ref,
			},
		},
	}
	var tests = []struct {
		name string
		cr   runtime.Object
		res  interface{}
	}{
		{
			"overrides when component not nil, v1alpha1",
			&vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Istio: &vzapi.IstioComponent{
							InstallOverrides: oV1Alpha1,
						},
					},
				},
			},
			oV1Alpha1.ValueOverrides,
		},
		{
			"Empty overrides when component nil",
			&v1beta1.Verrazzano{},
			[]v1beta1.Overrides{},
		},
		{
			"overrides when component not nil",
			&v1beta1.Verrazzano{
				Spec: v1beta1.VerrazzanoSpec{
					Components: v1beta1.ComponentSpec{
						Istio: &v1beta1.IstioComponent{
							InstallOverrides: o,
						},
					},
				},
			},
			o.ValueOverrides,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			override := GetOverrides(tt.cr)
			assert.EqualValues(t, tt.res, override)
		})
	}
}
