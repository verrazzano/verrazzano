// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"text/template"

	"github.com/verrazzano/verrazzano/pkg/istio"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	"github.com/verrazzano/verrazzano/pkg/k8sutil"

	"github.com/verrazzano/verrazzano/pkg/log/vzlog"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/helm"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
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

const (
	profilesRelativePath = "../../../../manifests/profiles"

	testBomFilePath = "../../testdata/test_bom.json"

	externalIPOverrideJSONTemplate = `
{
  "apiVersion": "install.istio.io/v1alpha1",
  "kind": "IstioOperator",
  "spec": {
    "components": {
      "ingressGateways": [
        {
          "enabled": true,
          "k8s": {
            "service": {
              "type": "{{.ServiceType}}",
              "ports": [
                {
                  "name": "port1",
                  "protocol": "TCP",
                  "port": 8000,
                  "nodePort": 32443,
                  "targetPort": 2000
                }
              ],
              "externalIPs": [
                "{{.IPAddress}}"
              ]
            }
          },
          "name": "istio-ingressgateway"
        }
      ]
    }
  }
}
`
)

type testData struct {
	ServiceType string
	IPAddress   string
}

var testExternalIP = ip.RandomIP()

var crEnabled = v1alpha1.Verrazzano{
	Spec: v1alpha1.VerrazzanoSpec{
		Components: v1alpha1.ComponentSpec{
			Istio: &v1alpha1.IstioComponent{
				Enabled: getBoolPtr(true),
				Ingress: &v1alpha1.IstioIngressSection{
					Kubernetes: &v1alpha1.IstioKubernetesSection{
						CommonKubernetesSpec: v1alpha1.CommonKubernetesSpec{
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
				Egress: &v1alpha1.IstioEgressSection{
					Kubernetes: &v1alpha1.IstioKubernetesSection{
						CommonKubernetesSpec: v1alpha1.CommonKubernetesSpec{
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

var crInstall = &v1alpha1.Verrazzano{
	Spec: v1alpha1.VerrazzanoSpec{
		Version: "1.0",
		Components: v1alpha1.ComponentSpec{
			Istio: &v1alpha1.IstioComponent{
				IstioInstallArgs: []v1alpha1.InstallArgs{{
					Name:  "arg1",
					Value: "val1",
				}},
				Ingress: &v1alpha1.IstioIngressSection{
					Kubernetes: &v1alpha1.IstioKubernetesSection{
						CommonKubernetesSpec: v1alpha1.CommonKubernetesSpec{
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
				Egress: &v1alpha1.IstioEgressSection{
					Kubernetes: &v1alpha1.IstioKubernetesSection{
						CommonKubernetesSpec: v1alpha1.CommonKubernetesSpec{
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

// TestGetName tests the component name
// GIVEN a Verrazzano component
//
//	WHEN I call Name
//	THEN the correct Verrazzano name is returned
func TestGetName(t *testing.T) {
	a := assert.New(t)
	a.Equal("istio", comp.Name(), "Wrong component name")
}

// TestUpgrade tests the component upgrade
// GIVEN a component
//
//	WHEN I call Upgrade
//	THEN the upgrade returns success and passes the correct values to the upgrade function
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
	//if len(overridesFiles) != 2 {
	//	return []byte("error"), []byte(""), fmt.Errorf("incorrect number of override files: expected 2, received %v", len(overridesFiles))
	//}
	//if overridesFiles[0] != "test-values-file.yaml" {
	//	return []byte("error"), []byte(""), fmt.Errorf("invalid values file")
	//}
	//if !strings.Contains(overridesFiles[1], "istio-") || !strings.Contains(overridesFiles[1], ".yaml") {
	//	return []byte("error"), []byte(""), fmt.Errorf("incorrect install args overrides file")
	//}
	//installArgsFromFile, err := ioutil.ReadFile(overridesFiles[1])
	//if err != nil {
	//	return []byte("error"), []byte(""), fmt.Errorf("unable to read install args overrides file")
	//}
	//if !strings.Contains(string(installArgsFromFile), "val1") {
	//	return []byte("error"), []byte(""), fmt.Errorf("install args overrides file does not contain install args")
	//}
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
//
//	WHEN the deployment object has enough replicas available and istioctl ran successfully
//	THEN true is returned
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
	compContext := spi.NewFakeContext(fakeClient, &v1alpha1.Verrazzano{}, nil, false)
	assert.True(t, iComp.IsReady(compContext))
}

// TestIsEnabledNilIstio tests the IsEnabled function
// GIVEN a call to IsEnabled
//
//	WHEN The Istio component is nil
//	THEN true is returned
func TestIsEnabledNilIstio(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Istio = nil
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, nil, false, profilesRelativePath).EffectiveCR()))
}

// TestIsEnabledNilComponent tests the IsEnabled function
// GIVEN a call to IsEnabled
//
//	WHEN The Istio component is nil
//	THEN false is returned
func TestIsEnabledNilComponent(t *testing.T) {
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &v1alpha1.Verrazzano{}, nil, false, profilesRelativePath).EffectiveCR()))
}

// TestIsEnabledNilEnabled tests the IsEnabled function
// GIVEN a call to IsEnabled
//
//	WHEN The Istio component enabled is nil
//	THEN true is returned
func TestIsEnabledNilEnabled(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Istio.Enabled = nil
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, nil, false, profilesRelativePath).EffectiveCR()))
}

// TestIsEnabledExplicit tests the IsEnabled function
// GIVEN a call to IsEnabled
//
//	WHEN The Istio component is explicitly enabled
//	THEN true is returned
func TestIsEnabledExplicit(t *testing.T) {
	cr := crEnabled
	cr.Spec.Components.Istio.Enabled = getBoolPtr(true)
	assert.True(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, &cr, nil, false, profilesRelativePath).EffectiveCR()))
}

// TestIsDisableExplicit tests the IsEnabled function
// GIVEN a call to IsEnabled
//
//	WHEN The Istio component is explicitly disabled
//	THEN false is returned
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
		old     *v1alpha1.Verrazzano
		new     *v1alpha1.Verrazzano
		wantErr bool
	}{
		{
			name: "enable",
			old: &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						Istio: &v1alpha1.IstioComponent{
							Enabled: &disabled,
						},
					},
				},
			},
			new:     &v1alpha1.Verrazzano{},
			wantErr: false,
		},
		{
			name: "disable",
			old:  &v1alpha1.Verrazzano{},
			new: &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						Istio: &v1alpha1.IstioComponent{
							Enabled: &disabled,
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "change-install-args",
			old:  &v1alpha1.Verrazzano{},
			new: &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						Istio: &v1alpha1.IstioComponent{
							IstioInstallArgs: []v1alpha1.InstallArgs{{Name: "foo", Value: "bar"}},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "change-ingress-replicas",
			old:  &v1alpha1.Verrazzano{},
			new: &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						Istio: &v1alpha1.IstioComponent{
							Ingress: &v1alpha1.IstioIngressSection{
								Kubernetes: &v1alpha1.IstioKubernetesSection{
									CommonKubernetesSpec: v1alpha1.CommonKubernetesSpec{
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
			old:  &v1alpha1.Verrazzano{},
			new: &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						Istio: &v1alpha1.IstioComponent{
							Ingress: &v1alpha1.IstioIngressSection{
								Kubernetes: &v1alpha1.IstioKubernetesSection{
									CommonKubernetesSpec: v1alpha1.CommonKubernetesSpec{
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
			old:  &v1alpha1.Verrazzano{},
			new: &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						Istio: &v1alpha1.IstioComponent{
							Egress: &v1alpha1.IstioEgressSection{
								Kubernetes: &v1alpha1.IstioKubernetesSection{
									CommonKubernetesSpec: v1alpha1.CommonKubernetesSpec{
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
			old:  &v1alpha1.Verrazzano{},
			new: &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						Istio: &v1alpha1.IstioComponent{
							Egress: &v1alpha1.IstioEgressSection{
								Kubernetes: &v1alpha1.IstioKubernetesSection{
									CommonKubernetesSpec: v1alpha1.CommonKubernetesSpec{
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
			old:  &v1alpha1.Verrazzano{},
			new: &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						Istio: &v1alpha1.IstioComponent{
							Ingress: &v1alpha1.IstioIngressSection{
								Type: v1alpha1.NodePort,
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "change-type-to-nodeport-with-externalIPs",
			old:  &v1alpha1.Verrazzano{},
			new: &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						Istio: &v1alpha1.IstioComponent{
							Ingress: &v1alpha1.IstioIngressSection{
								Type: v1alpha1.NodePort,
							},
							IstioInstallArgs: []v1alpha1.InstallArgs{
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
			old: &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						Istio: &v1alpha1.IstioComponent{
							Ingress: &v1alpha1.IstioIngressSection{
								Type: v1alpha1.NodePort,
							},
						},
					},
				},
			},
			new: &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						Istio: &v1alpha1.IstioComponent{
							Ingress: &v1alpha1.IstioIngressSection{
								Type: v1alpha1.LoadBalancer,
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "change-ports",
			old:  &v1alpha1.Verrazzano{},
			new: &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						Istio: &v1alpha1.IstioComponent{
							Ingress: &v1alpha1.IstioIngressSection{
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
			old:     &v1alpha1.Verrazzano{},
			new:     &v1alpha1.Verrazzano{},
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

func Test_istioComponent_ValidateUpdateV1Beta1(t *testing.T) {
	disabled := false
	nodePortWithoutIPAddressJSON := createIPOverrideJSON(t, externalIPOverrideJSONTemplate, v1beta1.NodePort, "")
	loadBalancerWithoutIPAddressJSON := createIPOverrideJSON(t, externalIPOverrideJSONTemplate, v1beta1.LoadBalancer, "")
	nodePortIPAddressJSON := createIPOverrideJSON(t, externalIPOverrideJSONTemplate, v1beta1.NodePort, testExternalIP)
	tests := []struct {
		name    string
		old     *v1beta1.Verrazzano
		new     *v1beta1.Verrazzano
		wantErr bool
	}{
		{
			name: "enable",
			old: &v1beta1.Verrazzano{
				Spec: v1beta1.VerrazzanoSpec{
					Components: v1beta1.ComponentSpec{
						Istio: &v1beta1.IstioComponent{
							Enabled: &disabled,
						},
					},
				},
			},
			new:     &v1beta1.Verrazzano{},
			wantErr: false,
		},
		{
			name:    "change-type-to-nodeport-without-externalIPs",
			old:     &v1beta1.Verrazzano{},
			new:     createVerrazzanoData(nodePortWithoutIPAddressJSON),
			wantErr: true,
		},
		{
			name:    "change-type-to-nodeport-with-externalIPs",
			old:     &v1beta1.Verrazzano{},
			new:     createVerrazzanoData(nodePortIPAddressJSON),
			wantErr: false,
		},
		{
			name:    "change-type-from-nodeport",
			old:     createVerrazzanoData(nodePortWithoutIPAddressJSON),
			new:     createVerrazzanoData(loadBalancerWithoutIPAddressJSON),
			wantErr: false,
		},
		{
			name:    "no change",
			old:     &v1beta1.Verrazzano{},
			new:     &v1beta1.Verrazzano{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewComponent()
			if err := c.ValidateUpdateV1Beta1(tt.old, tt.new); (err != nil) != tt.wantErr {
				t.Errorf("ValidateUpdateV1Beta1() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_istioComponent_ValidateInstall(t *testing.T) {
	tests := []struct {
		name        string
		vz          *v1alpha1.Verrazzano
		wantErr     bool
		v1beta1Test bool
	}{
		{
			name: "IstioComponentEmpty",
			vz: &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						Istio: &v1alpha1.IstioComponent{},
					},
				},
			},
			wantErr:     false,
			v1beta1Test: true,
		},
		{
			name: "IstioInstallArgsEmpty",
			vz: &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						Istio: &v1alpha1.IstioComponent{
							Ingress: &v1alpha1.IstioIngressSection{
								Type: v1alpha1.NodePort,
							},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "IstioInstallMissingKey",
			vz: &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						Istio: &v1alpha1.IstioComponent{
							Ingress: &v1alpha1.IstioIngressSection{
								Type: v1alpha1.NodePort,
							},
							IstioInstallArgs: []v1alpha1.InstallArgs{
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
			vz: &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						Istio: &v1alpha1.IstioComponent{
							Ingress: &v1alpha1.IstioIngressSection{
								Type: v1alpha1.NodePort,
							},
							IstioInstallArgs: []v1alpha1.InstallArgs{
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
			vz: &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						Istio: &v1alpha1.IstioComponent{
							Ingress: &v1alpha1.IstioIngressSection{
								Type: v1alpha1.NodePort,
							},
							IstioInstallArgs: []v1alpha1.InstallArgs{
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
			vz: &v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						Istio: &v1alpha1.IstioComponent{
							Ingress: &v1alpha1.IstioIngressSection{
								Type: v1alpha1.NodePort,
							},
							IstioInstallArgs: []v1alpha1.InstallArgs{
								{
									Name:      ExternalIPArg,
									ValueList: []string{testExternalIP},
								},
							},
						},
					},
				},
			},
			wantErr:     false,
			v1beta1Test: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewComponent()
			if err := c.ValidateInstall(tt.vz); (err != nil) != tt.wantErr {
				t.Errorf("ValidateInstall() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.v1beta1Test {
				vzV1Beta1 := &v1beta1.Verrazzano{}
				err := tt.vz.ConvertTo(vzV1Beta1)
				assert.NoError(t, err)
				if err := c.ValidateInstallV1Beta1(vzV1Beta1); (err != nil) != tt.wantErr {
					t.Errorf("ValidateInstallV1Beta1() error = %v, wantErr %v", err, tt.wantErr)
				}
			}

		})
	}
}

func Test_istioComponent_ValidateInstallV1Beta1(t *testing.T) {
	ipAddressInvalid := "1.2.3.we"
	nodePortIPAddressValidJSON := createIPOverrideJSON(t, externalIPOverrideJSONTemplate, v1beta1.NodePort, testExternalIP)
	nodePortIPAddressInvalidJSON := createIPOverrideJSON(t, externalIPOverrideJSONTemplate, v1beta1.NodePort, ipAddressInvalid)
	loadBalancerIPAddressValidJSON := createIPOverrideJSON(t, externalIPOverrideJSONTemplate, v1beta1.LoadBalancer, testExternalIP)
	loadBalancerIPAddressInvalidJSON := createIPOverrideJSON(t, externalIPOverrideJSONTemplate, v1beta1.LoadBalancer, ipAddressInvalid)

	tests := []struct {
		name    string
		vz      *v1beta1.Verrazzano
		wantErr bool
	}{
		{
			name:    "IstioInstallOverrides_InvalidNodePortIPAddressJson",
			vz:      createVerrazzanoData(nodePortIPAddressInvalidJSON),
			wantErr: true,
		},
		{
			name:    "IstioInstallOverrides_ValidNodePortIPAddressJson",
			vz:      createVerrazzanoData(nodePortIPAddressValidJSON),
			wantErr: false,
		},
		{
			// No external IP Address validation when service type is not Node Port and valid IP address
			name:    "IstioInstallOverrides_ValidLoadBalancerIPAddressJson",
			vz:      createVerrazzanoData(loadBalancerIPAddressValidJSON),
			wantErr: false,
		},
		{
			// No external IP Address validation when service type is not Node Port and invalid IP address
			name:    "IstioInstallOverrides_InvalidLoadBalancerIPAddressJson",
			vz:      createVerrazzanoData(loadBalancerIPAddressInvalidJSON),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewComponent()
			if err := c.ValidateInstallV1Beta1(tt.vz); (err != nil) != tt.wantErr {
				t.Errorf("ValidateInstallV1Beta1() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func createIPOverrideJSON(t *testing.T, jsonTemplate string, ingressType v1beta1.IngressType, ipAddress string) string {
	dataSample := testData{}
	if len(ingressType) > 0 {
		dataSample.ServiceType = string(ingressType)
	}
	if strings.TrimSpace(ipAddress) != "" {
		dataSample.IPAddress = ipAddress
	}
	tmpl, err := template.New("dataSample").Parse(jsonTemplate)
	if err != nil {
		t.Errorf("Error while parsing template: %v", err)
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, dataSample)
	if err != nil {
		t.Errorf("Error while executing template: %v", err)
	}
	nodePortIPAddressValidJSON := buf.String()
	return nodePortIPAddressValidJSON
}

func createVerrazzanoData(externalIPOverrideJSON string) *v1beta1.Verrazzano {
	return &v1beta1.Verrazzano{
		Spec: v1beta1.VerrazzanoSpec{
			Components: v1beta1.ComponentSpec{
				Istio: &v1beta1.IstioComponent{
					InstallOverrides: v1beta1.InstallOverrides{
						ValueOverrides: []v1beta1.Overrides{
							{
								Values: &apiextensionsv1.JSON{
									Raw: []byte(externalIPOverrideJSON),
								},
							},
						},
					},
				},
			},
		},
	}
}

// TestValidateUpdate tests the istio ValidateUpdate function
func TestValidateUpdate(t *testing.T) {
	oldVZ := &v1alpha1.Verrazzano{
		Spec: v1alpha1.VerrazzanoSpec{
			Components: v1alpha1.ComponentSpec{
				Istio: &v1alpha1.IstioComponent{
					Enabled: &trueValue,
				},
			},
		},
	}
	newVZ := &v1alpha1.Verrazzano{
		Spec: v1alpha1.VerrazzanoSpec{
			Components: v1alpha1.ComponentSpec{
				Istio: &v1alpha1.IstioComponent{
					Enabled: &falseValue,
				},
			},
		},
	}
	assert.Error(t, NewComponent().ValidateUpdate(oldVZ, newVZ))

	oldV1Beta1 := &v1beta1.Verrazzano{}
	err := oldVZ.ConvertTo(oldV1Beta1)
	assert.NoError(t, err)
	newV1Beta1 := &v1beta1.Verrazzano{}
	err = newVZ.ConvertTo(newV1Beta1)
	assert.NoError(t, err)
	assert.Error(t, NewComponent().ValidateUpdateV1Beta1(oldV1Beta1, newV1Beta1))

}

// TestPostUninstall tests the PostUninstall function
// GIVEN a call to PostUninstall
//
//	WHEN the istio-system namespace exists with a finalizer
//	THEN true is returned and istio-system namespace is deleted
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
	compContext := spi.NewFakeContext(fakeClient, &v1alpha1.Verrazzano{}, nil, false)
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
//
//	WHEN the uninstall function is called
//	THEN success is returned
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
	ref := &v1.ConfigMapKeySelector{
		Key: "foo",
	}
	o := v1beta1.InstallOverrides{
		ValueOverrides: []v1beta1.Overrides{
			{
				ConfigMapRef: ref,
			},
		},
	}
	oV1Alpha1 := v1alpha1.InstallOverrides{
		ValueOverrides: []v1alpha1.Overrides{
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
			&v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						Istio: &v1alpha1.IstioComponent{
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
			override := NewComponent().GetOverrides(tt.cr)
			assert.EqualValues(t, tt.res, override)
		})
	}
}
