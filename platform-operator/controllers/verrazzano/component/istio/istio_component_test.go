// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	"context"
	"fmt"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/types"
	"os"
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

const profilesRelativePath = "../../../../manifests/profiles"

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

	err := comp.Upgrade(spi.NewFakeContext(getMock(t), crInstall, false))
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
	err := comp.PostUpgrade(spi.NewFakeContext(getMock(t), crInstall, false))
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

	return mock
}

// fakeRunner overrides the istio run command
func (r fakeRunner) Run(cmd *exec.Cmd) (stdout []byte, stderr []byte, err error) {
	return []byte("success"), []byte(""), nil
}

// TestAppendIstioOverrides tests the Istio override for the global hub
// GIVEN the registry ovverride env var is set
//  WHEN I call AppendIstioOverrides
//  THEN the Istio global.hub helm override is added to the provided array/slice.
func TestAppendIstioOverrides(t *testing.T) {
	a := assert.New(t)

	config.SetDefaultBomFilePath(testBomFilePath)

	_ = os.Setenv(constants.RegistryOverrideEnvVar, "myreg.io")
	defer func() { _ = os.Unsetenv(constants.RegistryOverrideEnvVar) }()

	kvs, err := AppendIstioOverrides(nil, "istiod", "", "", nil)
	a.NoError(err, "AppendIstioOverrides returned an error ")
	a.Len(kvs, 1, "AppendIstioOverrides returned wrong number of Key:Value pairs")
	a.Equal(istioGlobalHubKey, kvs[0].Key)
	a.Equal("myreg.io/verrazzano", kvs[0].Value)

	_ = os.Setenv(constants.ImageRepoOverrideEnvVar, "myrepo")
	defer func() { _ = os.Unsetenv(constants.ImageRepoOverrideEnvVar) }()
	kvs, err = AppendIstioOverrides(nil, "istiod", "", "", nil)
	a.NoError(err, "AppendIstioOverrides returned an error ")
	a.Len(kvs, 1, "AppendIstioOverrides returned wrong number of Key:Value pairs")
	a.Equal(istioGlobalHubKey, kvs[0].Key)
	a.Equal("myreg.io/myrepo/verrazzano", kvs[0].Value)
}

// TestAppendIstioOverridesNoRegistryOverride tests the Istio override for the global hub when no registry override is specified
// GIVEN the registry ovverride env var is NOT set
//  WHEN I call AppendIstioOverrides
//  THEN no overrides are added to the provided array/slice
func TestAppendIstioOverridesNoRegistryOverride(t *testing.T) {
	a := assert.New(t)

	config.SetDefaultBomFilePath(testBomFilePath)

	kvs, err := AppendIstioOverrides(nil, "istiod", "", "", nil)
	a.NoError(err, "AppendIstioOverrides returned an error ")
	a.Len(kvs, 0, "AppendIstioOverrides returned wrong number of Key:Value pairs")
}

// TestIsReady tests the IsReady function
// GIVEN a call to IsReady
//  WHEN the deployment object has enough replicas available
//  THEN true is returned
func TestIsReady(t *testing.T) {

	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: IstioNamespace,
				Name:      IstiodDeployment,
				Labels:    map[string]string{"app": IstiodDeployment},
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 1,
				Replicas:          1,
				UpdatedReplicas:   1,
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: IstioNamespace,
				Name:      IstioIngressgatewayDeployment,
				Labels:    map[string]string{"app": IstioIngressgatewayDeployment},
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 1,
				Replicas:          1,
				UpdatedReplicas:   1,
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: IstioNamespace,
				Name:      IstioEgressgatewayDeployment,
				Labels:    map[string]string{"app": IstioEgressgatewayDeployment},
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 1,
				Replicas:          1,
				UpdatedReplicas:   1,
			},
		},
	).Build()
	var iComp istioComponent
	compContext := spi.NewFakeContext(fakeClient, nil, false)
	assert.True(t, iComp.IsReady(compContext))
}

// TestIsEnabledNilIstio tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Istio component is nil
//  THEN true is returned
func TestIsEnabledNilIstio(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Istio = nil
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, false, profilesRelativePath).EffectiveCR()))
}

// TestIsEnabledNilComponent tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Istio component is nil
//  THEN false is returned
func TestIsEnabledNilComponent(t *testing.T) {
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &vzapi.Verrazzano{}, false, profilesRelativePath).EffectiveCR()))
}

// TestIsEnabledNilEnabled tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Istio component enabled is nil
//  THEN true is returned
func TestIsEnabledNilEnabled(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Istio.Enabled = nil
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, false, profilesRelativePath).EffectiveCR()))
}

// TestIsEnabledExplicit tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Istio component is explicitly enabled
//  THEN true is returned
func TestIsEnabledExplicit(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Istio.Enabled = getBoolPtr(true)
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, false, profilesRelativePath).EffectiveCR()))
}

// TestIsDisableExplicit tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Istio component is explicitly disabled
//  THEN false is returned
func TestIsDisableExplicit(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Istio.Enabled = getBoolPtr(false)
	assert.False(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, false, profilesRelativePath).EffectiveCR()))
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
									ValueList: []string{"1.2.3.4"},
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
									ValueList: []string{"1.1.1.1"},
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
									Value: "1.1.1.1",
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
									ValueList: []string{"1.1.1.1.1"},
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
									ValueList: []string{"1.2.3.4"},
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
