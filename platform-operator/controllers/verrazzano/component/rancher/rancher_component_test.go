// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"

	certapiv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/bom"
	ctrlerrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
	helmcli "github.com/verrazzano/verrazzano/pkg/helm"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	k8sutilfake "github.com/verrazzano/verrazzano/pkg/k8sutil/fake"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	dynfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testBomFilePath      = "../../testdata/test_bom.json"
	profilesRelativePath = "../../../../manifests/profiles"
)

func getValue(kvs []bom.KeyValue, key string) (string, bool) {
	for _, kv := range kvs {
		if strings.EqualFold(key, kv.Key) {
			return kv.Value, true
		}
	}
	return "", false
}

// TestAppendRegistryOverrides verifies that registry overrides are added as appropriate
// GIVEN a Verrazzano CR
//
//	WHEN AppendOverrides is called
//	THEN AppendOverrides should add registry overrides
func TestAppendRegistryOverrides(t *testing.T) {
	ctx := spi.NewFakeContext(fake.NewClientBuilder().WithScheme(getScheme()).Build(), &vzAcmeDev, nil, false)
	config.SetDefaultBomFilePath(testBomFilePath)
	registry := "foobar"
	imageRepo := "barfoo"
	kvs, _ := AppendOverrides(ctx, "", "", "", []bom.KeyValue{})
	assert.Equal(t, 29, len(kvs)) // should only have LetsEncrypt + useBundledSystemChart + RancherImage Overrides
	_ = os.Setenv(constants.RegistryOverrideEnvVar, registry)
	kvs, _ = AppendOverrides(ctx, "", "", "", []bom.KeyValue{})
	assert.Equal(t, 29, len(kvs))
	v, ok := getValue(kvs, systemDefaultRegistryKey)
	assert.True(t, ok)
	assert.Equal(t, registry, v)

	_ = os.Setenv(constants.ImageRepoOverrideEnvVar, imageRepo)
	kvs, _ = AppendOverrides(ctx, "", "", "", []bom.KeyValue{})
	assert.Equal(t, 29, len(kvs))
	v, ok = getValue(kvs, systemDefaultRegistryKey)
	assert.True(t, ok)
	assert.Equal(t, fmt.Sprintf("%s/%s", registry, imageRepo), v)
}

// TestAppendImageOverrides verifies that Rancher image overrides are added
// GIVEN a Verrazzano CR
// WHEN appendImageOverrides is called
// THEN appendImageOverrides should add the image overrides
func TestAppendImageOverrides(t *testing.T) {
	a := assert.New(t)
	ctx := spi.NewFakeContext(fake.NewClientBuilder().WithScheme(getScheme()).Build(), &vzapi.Verrazzano{}, nil, false)
	config.SetDefaultBomFilePath(testBomFilePath)
	_ = os.Unsetenv(constants.RegistryOverrideEnvVar)

	// construct an expected image list
	expectedImages := map[string]bool{}
	for key := range imageEnvVars {
		expectedImages[key] = false
	}

	kvs, err := appendImageOverrides(ctx, []bom.KeyValue{})
	a.Nil(err)
	a.Equal(21, len(kvs))
	for _, kv := range kvs {
		// special exception for the extra arguments
		if kv.Value == "true" || kv.Value == "ghcr.io" {
			continue
		}
		if regexp.MustCompile(`extraEnv\[\d+]\.name`).Match([]byte(kv.Key)) {
			a.NotEmpty(kv.Value)
			continue
		}
		// catch image tags and ignore them
		if regexp.MustCompile(`^v\d+.\d+.\d+-\d+-\w+`).Match([]byte(kv.Value)) {
			continue
		}
		if strings.Contains(kv.Value, cattleShellImageName) {
			expectedImages[cattleShellImageName] = true
			continue
		}
		splitImage := strings.Split(kv.Value, "/")
		expectedImages[splitImage[len(splitImage)-1]] = true
	}

	for key, val := range expectedImages {
		a.True(val, fmt.Sprintf("Image %s was not found in the key value arguments:\n%v", key, expectedImages))
	}
}

// TestAppendCAOverrides verifies that CA overrides are added as appropriate for private CAs
// GIVEN a Verrzzano CR
//
//	WHEN AppendOverrides is called
//	THEN AppendOverrides should add private CA overrides
func TestAppendCAOverrides(t *testing.T) {
	ctx := spi.NewFakeContext(fake.NewClientBuilder().WithScheme(getScheme()).Build(), &vzDefaultCA, nil, false)
	config.SetDefaultBomFilePath(testBomFilePath)
	kvs, err := AppendOverrides(ctx, "", "", "", []bom.KeyValue{})
	assert.Nil(t, err)
	v, ok := getValue(kvs, ingressTLSSourceKey)
	assert.True(t, ok)
	assert.Equal(t, caTLSSource, v)
	v, ok = getValue(kvs, privateCAKey)
	assert.True(t, ok)
	assert.Equal(t, privateCAValue, v)
}

// TestIsReady verifies Rancher is enabled or disabled as expected
// GIVEN a Verrzzano CR
//
//	WHEN IsEnabled is called
//	THEN IsEnabled should return true/false depending on the enabled state of the CR
func TestIsEnabled(t *testing.T) {
	enabled := true
	disabled := false
	c := fake.NewClientBuilder().WithScheme(getScheme()).Build()
	vzWithRancher := vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Rancher: &vzapi.RancherComponent{
					Enabled: &enabled,
				},
			},
		},
	}
	vzNoRancher := vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Rancher: &vzapi.RancherComponent{
					Enabled: &disabled,
				},
			},
		},
	}
	var tests = []struct {
		testName string
		ctx      spi.ComponentContext
		enabled  bool
	}{
		{
			"should be enabled",
			spi.NewFakeContext(c, &vzWithRancher, nil, false),
			true,
		},
		{
			"should not be enabled",
			spi.NewFakeContext(c, &vzNoRancher, nil, false),
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			r := NewComponent()
			assert.Equal(t, tt.enabled, r.IsEnabled(tt.ctx.EffectiveCR()))
		})
	}
}

func TestPreInstall(t *testing.T) {
	caSecret := createCASecret()
	c := fake.NewClientBuilder().WithScheme(getScheme()).WithObjects(&caSecret).Build()
	ctx := spi.NewFakeContext(c, &vzDefaultCA, nil, false)
	assert.Nil(t, NewComponent().PreInstall(ctx))
}

// TestPreUpgrade tests the PreUpgrade func call
func TestPreUpgrade(t *testing.T) {
	asserts := assert.New(t)
	three := int32(3)
	// create a fake dynamic client to serve the Setting and ClusterRepo resources
	fakeDynamicClient := dynfake.NewSimpleDynamicClient(getScheme(), newClusterRepoResources()...)

	// override the getDynamicClientFunc for unit testing and reset it when done
	prevGetDynamicClientFunc := getDynamicClientFunc
	getDynamicClientFunc = func() (dynamic.Interface, error) { return fakeDynamicClient, nil }
	defer func() {
		getDynamicClientFunc = prevGetDynamicClientFunc
	}()

	tests := []struct {
		name              string
		rancherDeployment appsv1.Deployment
		wantErr           bool
	}{
		// GIVEN rancher deployment with 0 available replicas
		// WHEN PreUpgrade is called
		// THEN no error is returned and ClusterRepos resources are deleted
		{
			name: "PreUpgrade should not return an error when rancher deployment status has 0 available replicas",
			rancherDeployment: appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: ComponentName, Namespace: ComponentNamespace},
				Spec:       appsv1.DeploymentSpec{Replicas: &three},
				Status: appsv1.DeploymentStatus{
					AvailableReplicas: 0,
				},
			},
			wantErr: false,
		},

		// GIVEN a rancher deployment with some available and non zero replicas
		// WHEN PreUpgrade func is called
		// THEN the replicas are set to 0 in the deployment spec and a RetryableError is thrown
		{
			name: "PreUpgrade should return a retryable error when rancher deployment has available replicas",
			rancherDeployment: appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: ComponentName, Namespace: ComponentNamespace},
				Spec:       appsv1.DeploymentSpec{Replicas: &three},
				Status: appsv1.DeploymentStatus{
					AvailableReplicas: three,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := fake.NewClientBuilder().WithScheme(getScheme()).WithObjects(&tt.rancherDeployment).Build()
			fakeCtx := spi.NewFakeContext(cli, &vzapi.Verrazzano{}, nil, false)
			err := NewComponent().PreUpgrade(fakeCtx)
			if tt.wantErr {
				asserts.Equal(err, ctrlerrors.RetryableError{Source: ComponentName})
				obj := appsv1.Deployment{}
				asserts.NoError(cli.Get(context.Background(), types.NamespacedName{Namespace: ComponentNamespace, Name: ComponentName}, &obj))
				asserts.Equal(*obj.Spec.Replicas, int32(0))
			} else {
				asserts.Nil(err)
				// validate that the Setting and ClusterRepo resources have been deleted
				_, err = fakeDynamicClient.Resource(cattleSettingsGVR).Get(context.TODO(), chartDefaultBranchName, metav1.GetOptions{})
				asserts.True(errors.IsNotFound(err))

				_, err = fakeDynamicClient.Resource(cattleClusterReposGVR).Get(context.TODO(), rancherChartsClusterRepoName, metav1.GetOptions{})
				asserts.True(errors.IsNotFound(err))
				_, err = fakeDynamicClient.Resource(cattleClusterReposGVR).Get(context.TODO(), rancherPartnerChartsClusterRepoName, metav1.GetOptions{})
				asserts.True(errors.IsNotFound(err))
				_, err = fakeDynamicClient.Resource(cattleClusterReposGVR).Get(context.TODO(), rancherRke2ChartsClusterRepoName, metav1.GetOptions{})
				asserts.True(errors.IsNotFound(err))

				// this ClusterRepo should not have been deleted
				_, err = fakeDynamicClient.Resource(cattleClusterReposGVR).Get(context.TODO(), "app-charts", metav1.GetOptions{})
				assert.NoError(t, err)
			}
		})
	}
}

// TestInstall tests the Install func call
func TestInstall(t *testing.T) {
	config.SetDefaultBomFilePath(testBomFilePath)
	helm.SetUpgradeFunc(fakeUpgrade)
	defer helm.SetDefaultUpgradeFunc()
	helmcli.SetChartStateFunction(func(releaseName string, namespace string) (string, error) {
		return helmcli.ChartStatusDeployed, nil
	})
	defer helmcli.SetDefaultChartStateFunction()

	cli := createFakeTestClient(&v1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ComponentName,
			Namespace: ComponentNamespace,
		},
	}, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: ComponentName, Namespace: ComponentNamespace},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: ComponentName},
					},
				}},
		},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: 3,
		},
	})
	cliMissingIngress := createFakeTestClient(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: ComponentName, Namespace: ComponentNamespace},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: ComponentName},
					},
				}},
		},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: 3,
		},
	})

	cliMissingDeployment := createFakeTestClient(&v1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ComponentName,
			Namespace: ComponentNamespace,
		},
	})

	cliMissingContainerSpec := createFakeTestClient(&v1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ComponentName,
			Namespace: ComponentNamespace,
		},
	}, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: ComponentName, Namespace: ComponentNamespace},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: 3,
		},
	})
	tests := []struct {
		name        string
		c           client.Client
		vz          vzapi.Verrazzano
		wantErr     bool
		errContains string
	}{
		// GIVEN an environment with correct rancher deployment and ingress along with the Verrazzano resource
		// WHEN a call to Rancher Install is made
		// THEN no error is returned and rancher resources are patched as expected
		{
			name: "Install should not return an error for default values",
			c:    cli,
			vz: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Rancher: &vzapi.RancherComponent{
							Enabled: getBoolPtr(true),
						},
						DNS: &vzapi.DNSComponent{
							External: &vzapi.External{Suffix: "blah"},
						},
						CertManager: &vzapi.CertManagerComponent{
							Enabled: getBoolPtr(true),
						},
					},
				},
			},
			wantErr:     false,
			errContains: "",
		},
		// GIVEN an env with correct rancher deployment and Verrazzano resource but missing rancher ingress
		// WHEN a call to rancher Install is made
		// THEN an error is returned complaining about missing rancher ingress
		{
			name: "Install should return an error in case of missing rancher ingress",
			c:    cliMissingIngress,
			vz: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Rancher: &vzapi.RancherComponent{
							Enabled: getBoolPtr(true),
						},
						DNS: &vzapi.DNSComponent{
							External: &vzapi.External{Suffix: "blah"},
						},
						CertManager: &vzapi.CertManagerComponent{
							Enabled: getBoolPtr(true),
						},
					},
				},
			},
			wantErr:     true,
			errContains: "ingresses.networking.k8s.io \"rancher\" not found",
		},
		// GIVEN an env with correct rancher ingress and Verrazzano resource but missing rancher deployment
		// WHEN a call to rancher Install is made
		// THEN an error is returned complaining about missing rancher deployment
		{
			name: "Install should return an error in case of missing rancher deployment",
			c:    cliMissingDeployment,
			vz: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Rancher: &vzapi.RancherComponent{
							Enabled: getBoolPtr(true),
						},
						DNS: &vzapi.DNSComponent{
							External: &vzapi.External{Suffix: "blah"},
						},
						CertManager: &vzapi.CertManagerComponent{
							Enabled: getBoolPtr(true),
						},
					},
				},
			},
			wantErr:     true,
			errContains: "deployments.apps \"rancher\" not found",
		},
		// GIVEN an env with correct rancher ingress and Verrazzano resource but the deployment is missing rancher container
		// WHEN a call to rancher Install is made
		// THEN an error is returned complaining about the missing rancher container
		{
			name: "Install should return an error in case of deployment missing the rancher container",
			c:    cliMissingContainerSpec,
			vz: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Rancher: &vzapi.RancherComponent{
							Enabled: getBoolPtr(true),
						},
						DNS: &vzapi.DNSComponent{
							External: &vzapi.External{Suffix: "blah"},
						},
						CertManager: &vzapi.CertManagerComponent{
							Enabled: getBoolPtr(true),
						},
					},
				},
			},
			wantErr:     true,
			errContains: "container 'rancher' was not found",
		},
		// GIVEN an env with correct rancher deployment and ingress but the Verrazzano resource is missing cm component
		// WHEN a call to rancher install is made
		// THEN an error is returned complaining about missing cm component from the CR
		{
			name: "Install should return an error if cm component is missing from the VZ CR",
			c:    cli,
			vz: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Rancher: &vzapi.RancherComponent{
							Enabled: getBoolPtr(true),
						},
						DNS: &vzapi.DNSComponent{
							External: &vzapi.External{Suffix: "blah"},
						},
					},
				},
			},
			wantErr:     true,
			errContains: "Failed to find certManager component in effective cr",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := spi.NewFakeContext(tt.c, &tt.vz, nil, false)
			err := NewComponent().Install(ctx)
			if !tt.wantErr {
				assert.NoError(t, err)
				ingress := v1.Ingress{}
				deployment := appsv1.Deployment{}
				assert.NoError(t, tt.c.Get(context.Background(), types.NamespacedName{Name: ComponentName, Namespace: ComponentNamespace}, &ingress))
				assert.NoError(t, tt.c.Get(context.Background(), types.NamespacedName{Name: ComponentName, Namespace: ComponentNamespace}, &deployment))

				assert.Equal(t, "true", ingress.Annotations["kubernetes.io/tls-acme"])
				assert.Equal(t, "HTTPS", ingress.Annotations["nginx.ingress.kubernetes.io/backend-protocol"])
				assert.Equal(t, "true", ingress.Annotations["nginx.ingress.kubernetes.io/force-ssl-redirect"])

				assert.Equal(t, deployment.Spec.Template.Spec.Containers[0].SecurityContext.Capabilities.Add[0], corev1.Capability("MKNOD"))
			} else {
				assert.ErrorContains(t, err, tt.errContains)
			}
		})
	}
}

// TestMonitorOverrides tests the monitor overrides function
func TestMonitorOverrides(t *testing.T) {
	tests := []struct {
		name       string
		actualCR   *vzapi.Verrazzano
		expectTrue bool
	}{
		// GIVEN a default Verrazzano custom resource,
		// WHEN we call MonitorOverrides on the Rancher component,
		// THEN it returns false
		{
			"Monitor changes should be true by default when actual VZ spec does not have a Rancher Component section",
			&vzapi.Verrazzano{},
			// True because Rancher is enabled be default in the effective CR
			true,
		},
		// GIVEN a Verrazzano custom resource with a Rancher Component in the spec section,
		// WHEN we call MonitorOverrides on the Rancher component,
		// THEN it returns true
		{
			"Monitor changes should be true by default when the actual VZ spec has a Rancher Component section",
			&vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Rancher: &vzapi.RancherComponent{},
					},
				},
			},
			true,
		},
		// GIVEN a Verrazzano custom resource with a Rancher Component in the spec section
		//       with monitor changes flag explicitly set to true,
		// WHEN we call MonitorOverrides on the Rancher component,
		// THEN it returns true
		{
			"Monitor changes should be true when set explicitly in the VZ CR",
			&vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Rancher: &vzapi.RancherComponent{
							InstallOverrides: vzapi.InstallOverrides{
								MonitorChanges: getBoolPtr(true),
							},
						},
					},
				},
			},
			true,
		},
		// GIVEN a Verrazzano custom resource with a Rancher Component in the spec section
		//       with monitor changes flag explicitly set to false,
		// WHEN we call MonitorOverrides on the Rancher component,
		// THEN it returns false
		{
			"Monitor changes should be false when set explicitly in the actual VZ CR",
			&vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Rancher: &vzapi.RancherComponent{
							InstallOverrides: vzapi.InstallOverrides{
								MonitorChanges: getBoolPtr(false),
							},
						},
					},
				},
			},
			false,
		},
	}
	client := createFakeTestClient()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := spi.NewFakeContext(client, tt.actualCR, nil, false, profilesRelativePath)
			if tt.expectTrue {
				assert.True(t, NewComponent().MonitorOverrides(ctx), tt.name)
			} else {
				assert.False(t, NewComponent().MonitorOverrides(ctx), tt.name)
			}
		})
	}
}

// TestIsKeycloakAuthEnabled tests the isKeycloakAuthEnabled func call
func TestIsKeycloakAuthEnabled(t *testing.T) {
	tests := []struct {
		name      string
		vz        vzapi.Verrazzano
		isEnabled bool
	}{
		// GIVEN a VZ CR with empty component spec
		// WHEN a call to isKeycloakAuthEnabled func is made
		// THEN the func returns a true boolean value
		{
			name:      "Return true for empty CR i.e. default values",
			vz:        vzapi.Verrazzano{},
			isEnabled: true,
		},
		// GIVEN a VZ CR with keycloak explicitly disabled
		// WHEN a call to isKeycloakAuthEnabled func is made
		// THEN the func returns a false boolean value
		{
			name: "Return false if keycloak component is explicitly disabled in the CR",
			vz: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Rancher: &vzapi.RancherComponent{
							KeycloakAuthEnabled: getBoolPtr(true),
						},
						Keycloak: &vzapi.KeycloakComponent{
							Enabled: getBoolPtr(false),
						},
					},
				},
			},
			isEnabled: false,
		},
		// GIVEN a VZ CR with keycloak auth explicitly disabled in the rancher component spec
		// WHEN a call to isKeycloakAuthEnabled func is made
		// THEN the func returns a false boolean value
		{
			name: "Return false if keycloak auth is explicitly disabled in the rancher component spec",
			vz: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Rancher: &vzapi.RancherComponent{
							KeycloakAuthEnabled: getBoolPtr(false),
						},
						Keycloak: &vzapi.KeycloakComponent{
							Enabled: getBoolPtr(true),
						},
					},
				},
			},
			isEnabled: false,
		},
		// GIVEN a VZ CR with keycloak and rancher keycloak auth explicitly enabled
		// WHEN a call to isKeycloakAuthEnabled func is made
		// THEN the func returns a true boolean value
		{
			name: "Return true if the required values are explicitly set to true in the CR",
			vz: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Rancher: &vzapi.RancherComponent{
							KeycloakAuthEnabled: getBoolPtr(true),
						},
						Keycloak: &vzapi.KeycloakComponent{
							Enabled: getBoolPtr(true),
						},
					},
				},
			},
			isEnabled: true,
		},
		// GIVEN a VZ CR with nil rancher component value
		// WHEN a call to isKeycloakAuthEnabled func is made
		// THEN the func returns a true boolean value
		{
			name: "Return true if rancher component is nil in the CR and keycloak is explicitly enabled",
			vz: vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Keycloak: &vzapi.KeycloakComponent{
							Enabled: getBoolPtr(true),
						},
					},
				},
			},
			isEnabled: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val := isKeycloakAuthEnabled(&tt.vz)
			assert.Equal(t, val, tt.isEnabled)
		})
	}
}

// TestCreateOrUpdateClusterRoleTemplateBindings tests the following scenario
// GIVEN a slice of group role pairs
// WHEN createOrUpdateClusterRoleTemplateBindings is called
// THEN the cluster role template bindings are created or updated for the given cluster role and group
func TestCreateOrUpdateClusterRoleTemplateBindings(t *testing.T) {
	asserts := assert.New(t)
	cli := fake.NewClientBuilder().WithScheme(getScheme()).WithObjects(
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: ClusterLocal,
			},
		},
	).Build()
	fakeCtx := spi.NewFakeContext(cli, &vzapi.Verrazzano{}, nil, false)

	asserts.NoError(createOrUpdateClusterRoleTemplateBindings(fakeCtx))
	for _, grp := range GroupRolePairs {
		obj := unstructured.Unstructured{}
		obj.SetGroupVersionKind(GVKClusterRoleTemplateBinding)
		nsn := types.NamespacedName{Name: fmt.Sprintf("crtb-%s-%s", grp[ClusterRoleKey], grp[GroupKey]), Namespace: ClusterLocal}
		asserts.NoError(cli.Get(context.Background(), nsn, &obj))

		data := obj.UnstructuredContent()

		asserts.Equal(ClusterLocal, data[ClusterRoleTemplateBindingAttributeClusterName])
		asserts.Equal(GroupPrincipalKeycloakPrefix+grp[GroupKey], data[ClusterRoleTemplateBindingAttributeGroupPrincipalName])
		asserts.Equal(grp[ClusterRoleKey], data[ClusterRoleTemplateBindingAttributeRoleTemplateName])
	}
}

// TestIsReady verifies that a ready-state Rancher shows as ready
// GIVEN a ready Rancher install
//
//	WHEN IsReady is called
//	THEN IsReady should return true
func TestIsReady(t *testing.T) {
	readyClient := fake.NewClientBuilder().WithScheme(getScheme()).WithObjects(
		newReadyDeployment(ComponentNamespace, ComponentName),
		newPod(ComponentNamespace, ComponentName),
		newReplicaSet(ComponentNamespace, ComponentName),
		newReadyDeployment(ComponentNamespace, rancherWebhookDeployment),
		newPod(ComponentNamespace, rancherWebhookDeployment),
		newReplicaSet(ComponentNamespace, rancherWebhookDeployment),
		newReadyDeployment(FleetLocalSystemNamespace, fleetAgentDeployment),
		newPod(FleetLocalSystemNamespace, fleetAgentDeployment),
		newReplicaSet(FleetLocalSystemNamespace, fleetAgentDeployment),
		newReadyDeployment(FleetSystemNamespace, fleetControllerDeployment),
		newPod(FleetSystemNamespace, fleetControllerDeployment),
		newReplicaSet(FleetSystemNamespace, fleetControllerDeployment),
		newReadyDeployment(FleetSystemNamespace, gitjobDeployment),
		newPod(FleetSystemNamespace, gitjobDeployment),
		newReplicaSet(FleetSystemNamespace, gitjobDeployment),
	).Build()
	unreadyDeployClient := fake.NewClientBuilder().WithScheme(getScheme()).WithObjects(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      ComponentName,
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 0,
				Replicas:          1,
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      rancherWebhookDeployment,
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 0,
				Replicas:          1,
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: FleetLocalSystemNamespace,
				Name:      fleetAgentDeployment,
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 0,
				Replicas:          1,
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: FleetSystemNamespace,
				Name:      fleetControllerDeployment,
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 0,
				Replicas:          1,
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: FleetSystemNamespace,
				Name:      gitjobDeployment,
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 0,
				Replicas:          1,
			},
		},
	).Build()

	var tests = []struct {
		testName string
		ctx      spi.ComponentContext
		isReady  bool
	}{
		{
			"should be ready",
			spi.NewFakeContext(readyClient, &vzDefaultCA, nil, true),
			true,
		},
		{
			"should not be ready due to deployment",
			spi.NewFakeContext(unreadyDeployClient, &vzDefaultCA, nil, true),
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			assert.Equal(t, tt.isReady, NewComponent().IsReady(tt.ctx))
		})
	}
}

// TestPostInstall tests a happy path post install run
// GIVEN a Rancher install state where all components are ready
//
//	WHEN PostInstall is called
//	THEN PostInstall should return nil
func TestPostInstall(t *testing.T) {
	component := NewComponent()
	ctxWithoutIngress, ctxWithIngress := prepareContexts()
	assert.IsType(t, fmt.Errorf(""), component.PostInstall(ctxWithoutIngress))
	assert.Nil(t, component.PostInstall(ctxWithIngress))
}

// TestPostUpgrade tests a happy path post upgrade run
// GIVEN a Rancher install state where all components are ready
//
//	WHEN PostUpgrade is called
//	THEN PostUpgrade should return nil
func TestPostUpgrade(t *testing.T) {
	component := NewComponent()
	ctxWithoutIngress, ctxWithIngress := prepareContexts()
	assert.Error(t, component.PostUpgrade(ctxWithoutIngress))
	assert.Nil(t, component.PostUpgrade(ctxWithIngress))
}

func TestValidateUpdate(t *testing.T) {
	disabled := false
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
						Rancher: &vzapi.RancherComponent{
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
						Rancher: &vzapi.RancherComponent{
							Enabled: &disabled,
						},
					},
				},
			},
			wantErr: true,
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

func TestValidateUpdateV1beta1(t *testing.T) {
	disabled := false
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
						Rancher: &v1beta1.RancherComponent{
							Enabled: &disabled,
						},
					},
				},
			},
			new:     &v1beta1.Verrazzano{},
			wantErr: false,
		},
		{
			name: "disable",
			old:  &v1beta1.Verrazzano{},
			new: &v1beta1.Verrazzano{
				Spec: v1beta1.VerrazzanoSpec{
					Components: v1beta1.ComponentSpec{
						Rancher: &v1beta1.RancherComponent{
							Enabled: &disabled,
						},
					},
				},
			},
			wantErr: true,
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

func prepareContexts() (spi.ComponentContext, spi.ComponentContext) {
	// mock the k8s resources used in post install
	caSecret := createCASecret()
	rootCASecret := createRootCASecret()
	adminSecret := createAdminSecret()
	rancherPodList := createRancherPodListWithAllRunning()

	ingress := v1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   common.CattleSystem,
			Name:        constants.RancherIngress,
			Annotations: map[string]string{},
		},
		Spec: v1.IngressSpec{
			Rules: []v1.IngressRule{
				{
					Host: "rancher",
				},
			},
		},
	}
	kcIngress := v1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "keycloak",
			Name:      "keycloak",
		},
		Spec: v1.IngressSpec{
			Rules: []v1.IngressRule{
				{
					Host: "keycloak",
				},
			},
		},
	}
	time := metav1.Now()
	cert := certapiv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{Name: certificates[0].Name, Namespace: certificates[0].Namespace},
		Status: certapiv1.CertificateStatus{
			Conditions: []certapiv1.CertificateCondition{
				{Type: certapiv1.CertificateConditionReady, Status: cmmeta.ConditionTrue, LastTransitionTime: &time},
			},
		},
	}
	serverURLSetting := createServerURLSetting()
	ociDriver := createOciDriver()
	okeDriver := createOkeDriver()
	rancherPod := newPod("cattle-system", "rancher")
	rancherPod.Status = corev1.PodStatus{
		Phase: corev1.PodRunning,
	}

	clientWithoutIngress := fake.NewClientBuilder().WithScheme(getScheme()).WithObjects(&caSecret, &rootCASecret, &adminSecret, &rancherPodList.Items[0], &serverURLSetting, &ociDriver, &okeDriver, &kcIngress, rancherPod).Build()
	ctxWithoutIngress := spi.NewFakeContext(clientWithoutIngress, &vzDefaultCA, nil, false)

	clientWithIngress := fake.NewClientBuilder().WithScheme(getScheme()).WithObjects(&caSecret, &rootCASecret, &adminSecret, &rancherPodList.Items[0], &ingress, &cert, &serverURLSetting, &ociDriver, &okeDriver, &kcIngress, rancherPod).Build()
	ctxWithIngress := spi.NewFakeContext(clientWithIngress, &vzDefaultCA, nil, false)
	// mock the pod executor when resetting the Rancher admin password
	scheme.Scheme.AddKnownTypes(schema.GroupVersion{Group: "", Version: "v1"}, &corev1.PodExecOptions{})
	k8sutil.NewPodExecutor = k8sutilfake.NewPodExecutor
	k8sutilfake.PodExecResult = func(url *url.URL) (string, string, error) {
		var commands []string
		if commands = url.Query()["command"]; len(commands) == 3 {
			if strings.Contains(commands[2], fmt.Sprintf("cat %s", SettingUILogoDarkLogoFilePath)) {
				return "dark", "", nil
			}

			if strings.Contains(commands[2], fmt.Sprintf("cat %s", SettingUILogoLightLogoFilePath)) {
				return "light", "", nil
			}

		}
		return "", "", nil
	}
	k8sutil.ClientConfig = func() (*rest.Config, kubernetes.Interface, error) {
		config, k := k8sutilfake.NewClientsetConfig()
		return config, k, nil
	}
	return ctxWithoutIngress, ctxWithIngress
}

func newReadyDeployment(namespace string, name string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			Labels:    map[string]string{"app": name},
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": name},
			},
		},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: 1,
			Replicas:          1,
			UpdatedReplicas:   1,
		},
	}
}

func newPod(namespace string, name string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name + "-95d8c5d96-m6mbr",
			Labels: map[string]string{
				"pod-template-hash": "95d8c5d96",
				"app":               name,
			},
		},
	}
}

func newReplicaSet(namespace string, name string) *appsv1.ReplicaSet {
	return &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   namespace,
			Name:        name + "-95d8c5d96",
			Annotations: map[string]string{"deployment.kubernetes.io/revision": "1"},
		},
	}
}

// TestValidateInstall verifies the ValidateInstall function of Rancher Component
// When there is namespace without the required label,
// Then ValidateInstall should throw error
func TestValidateInstall(t *testing.T) {
	namespaceWithoutLabels := &corev1.Namespace{}
	namespaceWithoutLabels.Name = FleetSystemNamespace
	namespaceWithoutLabels.Namespace = FleetSystemNamespace
	labelledNamespace := &corev1.Namespace{}
	labelledNamespace.Name = FleetSystemNamespace
	labelledNamespace.Namespace = FleetSystemNamespace
	labelledNamespace.Labels = map[string]string{constants.VerrazzanoManagedKey: FleetSystemNamespace}
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Rancher: &vzapi.RancherComponent{},
			},
		},
	}
	common.RunValidateInstallTest(t, NewComponent,
		common.ValidateInstallTest{
			Name:      "ValidRancherNamespace",
			WantErr:   "",
			Appsv1Cli: common.MockGetAppsV1(),
			Corev1Cli: common.MockGetCoreV1(labelledNamespace),
			Vz:        vz,
		},
		common.ValidateInstallTest{
			Name:      "InvalidRancherNamespace",
			WantErr:   FleetSystemNamespace,
			Appsv1Cli: common.MockGetAppsV1(),
			Corev1Cli: common.MockGetCoreV1(namespaceWithoutLabels),
			Vz:        vz,
		})
}

func createFakeTestClient(extraObjs ...client.Object) client.Client {
	objs := []client.Object{}
	objs = append(objs, extraObjs...)
	c := fake.NewClientBuilder().WithScheme(getScheme()).WithObjects(objs...).Build()
	return c
}

// fakeUpgrade override the upgrade function during unit tests
func fakeUpgrade(_ vzlog.VerrazzanoLogger, releaseName string, namespace string, chartDir string, wait bool, dryRun bool, overrides []helmcli.HelmOverrides) (stdout []byte, stderr []byte, err error) {
	return []byte("success"), []byte(""), nil
}

func getBoolPtr(b bool) *bool {
	return &b
}
