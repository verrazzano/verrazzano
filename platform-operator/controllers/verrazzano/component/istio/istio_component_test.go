// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	"context"
	"fmt"
	"github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/golang/mock/gomock"
	"io/ioutil"
	v1 "k8s.io/api/core/v1"
	"os"
	"os/exec"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"testing"

	"github.com/verrazzano/verrazzano/pkg/log/vzlog"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/helm"
	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/mocks"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// fakeRunner is used to test istio without actually running an OS exec command
type fakeRunner struct {
}

const profilesRelativePath = "../../../../manifests/profiles"

var crEnabled = installv1alpha1.Verrazzano{
	Spec: installv1alpha1.VerrazzanoSpec{
		Components: installv1alpha1.ComponentSpec{
			Istio: &installv1alpha1.IstioComponent{
				Enabled: getBoolPtr(true),
				Ingress: &installv1alpha1.IstioIngressSection{
					Kubernetes: &installv1alpha1.IstioKubernetesSection{
						CommonKubernetesSpec: installv1alpha1.CommonKubernetesSpec{
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
				Egress: &installv1alpha1.IstioEgressSection{
					Kubernetes: &installv1alpha1.IstioKubernetesSection{
						CommonKubernetesSpec: installv1alpha1.CommonKubernetesSpec{
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

var crInstall = &installv1alpha1.Verrazzano{
	Spec: installv1alpha1.VerrazzanoSpec{
		Version: "1.0",
		Components: installv1alpha1.ComponentSpec{
			Istio: &installv1alpha1.IstioComponent{
				IstioInstallArgs: []installv1alpha1.InstallArgs{{
					Name:  "arg1",
					Value: "val1",
				}},
				Ingress: &installv1alpha1.IstioIngressSection{
					Kubernetes: &installv1alpha1.IstioKubernetesSection{
						CommonKubernetesSpec: installv1alpha1.CommonKubernetesSpec{
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
				Egress: &installv1alpha1.IstioEgressSection{
					Kubernetes: &installv1alpha1.IstioKubernetesSection{
						CommonKubernetesSpec: installv1alpha1.CommonKubernetesSpec{
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
	assert := assert.New(t)
	assert.Equal("istio", comp.Name(), "Wrong component name")
}

// TestUpgrade tests the component upgrade
// GIVEN a component
//  WHEN I call Upgrade
//  THEN the upgrade returns success and passes the correct values to the upgrade function
func TestUpgrade(t *testing.T) {
	assert := assert.New(t)

	comp := istioComponent{
		ValuesFile:               "test-values-file.yaml",
		Revision:                 "1-1-1",
		InjectedSystemNamespaces: config.GetInjectedSystemNamespaces(),
	}

	config.SetDefaultBomFilePath(testBomFilePath)
	SetIstioUpgradeFunction(fakeUpgrade)
	defer SetDefaultIstioUpgradeFunction()

	err := comp.Upgrade(spi.NewFakeContext(upgradeMocks(t), crInstall, false))
	assert.NoError(err, "Upgrade returned an error")
}

func TestPostUpgrade(t *testing.T) {
	assert := assert.New(t)

	comp := istioComponent{}

	config.SetDefaultBomFilePath(testBomFilePath)
	helm.SetCmdRunner(fakeRunner{})
	defer helm.SetDefaultRunner()
	SetHelmUninstallFunction(fakeHelmUninstall)
	SetDefaultHelmUninstallFunction()
	err := comp.PostUpgrade(spi.NewFakeContext(upgradeMocks(t), crInstall, false))
	assert.NoError(err, "PostUpgrade returned an error")
}

// TestAppendIstioOverrides tests the Istio override for the global hub
// GIVEN the registry ovverride env var is set
//  WHEN I call AppendIstioOverrides
//  THEN the Istio global.hub helm override is added to the provided array/slice.
func TestAppendIstioOverrides(t *testing.T) {
	assert := assert.New(t)

	config.SetDefaultBomFilePath(testBomFilePath)

	os.Setenv(constants.RegistryOverrideEnvVar, "myreg.io")
	defer os.Unsetenv(constants.RegistryOverrideEnvVar)

	kvs, err := AppendIstioOverrides(nil, "istiod", "", "", nil)
	assert.NoError(err, "AppendIstioOverrides returned an error ")
	assert.Len(kvs, 1, "AppendIstioOverrides returned wrong number of Key:Value pairs")
	assert.Equal(istioGlobalHubKey, kvs[0].Key)
	assert.Equal("myreg.io/verrazzano", kvs[0].Value)

	os.Setenv(constants.ImageRepoOverrideEnvVar, "myrepo")
	defer os.Unsetenv(constants.ImageRepoOverrideEnvVar)
	kvs, err = AppendIstioOverrides(nil, "istiod", "", "", nil)
	assert.NoError(err, "AppendIstioOverrides returned an error ")
	assert.Len(kvs, 1, "AppendIstioOverrides returned wrong number of Key:Value pairs")
	assert.Equal(istioGlobalHubKey, kvs[0].Key)
	assert.Equal("myreg.io/myrepo/verrazzano", kvs[0].Value)
}

// TestAppendIstioOverridesNoRegistryOverride tests the Istio override for the global hub when no registry override is specified
// GIVEN the registry ovverride env var is NOT set
//  WHEN I call AppendIstioOverrides
//  THEN no overrides are added to the provided array/slice
func TestAppendIstioOverridesNoRegistryOverride(t *testing.T) {
	assert := assert.New(t)

	config.SetDefaultBomFilePath(testBomFilePath)

	kvs, err := AppendIstioOverrides(nil, "istiod", "", "", nil)
	assert.NoError(err, "AppendIstioOverrides returned an error ")
	assert.Len(kvs, 0, "AppendIstioOverrides returned wrong number of Key:Value pairs")
}

// TestIsReady tests the IsReady function
// GIVEN a call to IsReady
//  WHEN the deployment object has enough replicas available
//  THEN true is returned
func TestIsReady(t *testing.T) {

	fakeClient := fake.NewFakeClientWithScheme(k8scheme.Scheme, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: constants.IstioSystemNamespace,
			Name:      IstiodDeployment,
		},
		Status: appsv1.DeploymentStatus{
			Replicas:            1,
			ReadyReplicas:       1,
			AvailableReplicas:   1,
			UnavailableReplicas: 0,
		},
	},
	)
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
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, false, profilesRelativePath)))
}

// TestIsEnabledNilComponent tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Istio component is nil
//  THEN false is returned
func TestIsEnabledNilComponent(t *testing.T) {
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &installv1alpha1.Verrazzano{}, false, profilesRelativePath)))
}

// TestIsEnabledNilEnabled tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Istio component enabled is nil
//  THEN true is returned
func TestIsEnabledNilEnabled(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Istio.Enabled = nil
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, false, profilesRelativePath)))
}

// TestIsEnabledExplicit tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Istio component is explicitly enabled
//  THEN true is returned
func TestIsEnabledExplicit(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Istio.Enabled = getBoolPtr(true)
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, false, profilesRelativePath)))
}

// TestIsDisableExplicit tests the IsEnabled function
// GIVEN a call to IsEnabled
//  WHEN The Istio component is explicitly disabled
//  THEN false is returned
func TestIsDisableExplicit(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Istio.Enabled = getBoolPtr(false)
	assert.False(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, false, profilesRelativePath)))
}

func getBoolPtr(b bool) *bool {
	return &b
}

func TestGetIstioVersion(t *testing.T) {
	config.SetDefaultBomFilePath(testBomFilePath)
	defer config.SetDefaultBomFilePath("")
	istioVersion, err := getIstioVersion()
	assert.Nil(t, err, "getIstioVersion should not return an error")
	assert.Equal(t, "1.10.4", istioVersion, "the istio proxyv2 image tag should match the one in test_bom.json")
}

func upgradeMocks(t *testing.T) *mocks.MockClient {
	mocker := gomock.NewController(t)
	mock := mocks.NewMockClient(mocker)

	mocks.RestartMocks(mock)

	mock.EXPECT().
		List(gomock.Any(), &v1.SecretList{}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, secretList *v1.SecretList, options *client.ListOptions) error {
			secretList.Items = []v1.Secret{{Type: HelmScrtType}, {Type: "generic"}, {Type: HelmScrtType}}
			return nil
		})

	mock.EXPECT().
		Delete(gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, secret *v1.Secret) error {
			return nil
		}).Times(2)

	mock.EXPECT().
		List(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *v1alpha2.ApplicationConfigurationList, opts ...client.ListOption) error {
			return nil
		}).Times(2)

	return mock
}

// fakeRunner overrides the istio run command
func (r fakeRunner) Run(cmd *exec.Cmd) (stdout []byte, stderr []byte, err error) {
	return []byte("success"), []byte(""), nil
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

func fakeHelmUninstall(_ vzlog.VerrazzanoLogger, releaseName string, namespace string, dryRun bool) (stdout []byte, stderr []byte, err error) {
	if releaseName != "istiocoredns" {
		return []byte("error"), []byte(""), fmt.Errorf("expected release name istiocoredns does not match provided release name of %v", releaseName)
	}
	if releaseName != "istio-system" {
		return []byte("error"), []byte(""), fmt.Errorf("expected namespace istio-system does not match provided namespace of %v", namespace)
	}
	return []byte("success"), []byte(""), nil
}
