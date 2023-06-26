// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"context"
	"encoding/base64"
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
	cmconstants "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/time"
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

	missingIssuerMessage = "Failed to find clusterIssuer component in effective cr"
)

var getKubernetesTestVersion = func() (string, error) { return "v1.23.5", nil }

func init() {

	getKubernetesClusterVersion = getKubernetesTestVersion
}

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
	// Create a fake ComponentContext with the profiles dir to create an EffectiveCR; this is required to
	// convert the CertManager config to the ClusterIssuer config
	ctx := spi.NewFakeContext(fake.NewClientBuilder().WithScheme(getScheme()).Build(), &vzAcmeDev, nil,
		false, profilesRelativePath)
	config.SetDefaultBomFilePath(testBomFilePath)
	registry := "foobar"
	imageRepo := "barfoo"
	kvs, _ := AppendOverrides(ctx, "", "", "", []bom.KeyValue{})
	assert.Equal(t, 28, len(kvs)) // should only have LetsEncrypt + useBundledSystemChart + RancherImage Overrides
	_ = os.Setenv(constants.RegistryOverrideEnvVar, registry)
	kvs, _ = AppendOverrides(ctx, "", "", "", []bom.KeyValue{})
	assert.Equal(t, 29, len(kvs)) // one extra for the systemDefaultRegistry override
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

// TestApplendLetsEncryptDefaultEnvOverrides verifies that Helm overrides are added as appropriate for LE Prod
// GIVEN a Verrazzano CR
//
//	WHEN AppendOverrides is called with an LE prod configuration where the env is not specified
//	THEN AppendOverrides should add the appropriate LE prod overrides
func TestApplendLetsEncryptDefaultEnvOverrides(t *testing.T) {
	// Create a fake ComponentContext with the profiles dir to create an EffectiveCR; this is required to
	// convert the CertManager config to the ClusterIssuer config
	vzACMEProd := vzAcmeDev.DeepCopy()
	vzACMEProd.Spec.Components.CertManager.Certificate.Acme.Environment = ""
	ctx := spi.NewFakeContext(fake.NewClientBuilder().WithScheme(getScheme()).Build(), vzACMEProd, nil,
		false, profilesRelativePath)
	config.SetDefaultBomFilePath(testBomFilePath)

	kvs, _ := AppendOverrides(ctx, "", "", "", []bom.KeyValue{})
	assert.Contains(t, kvs, bom.KeyValue{Key: letsEncryptIngressClassKey, Value: common.RancherName})
	assert.Contains(t, kvs, bom.KeyValue{Key: letsEncryptEmailKey, Value: vzACMEProd.Spec.Components.CertManager.Certificate.Acme.EmailAddress})
	assert.Contains(t, kvs, bom.KeyValue{Key: letsEncryptEnvironmentKey, Value: cmconstants.LetsEncryptProduction})
	assert.Contains(t, kvs, bom.KeyValue{Key: ingressTLSSourceKey, Value: letsEncryptTLSSource})
	assert.Contains(t, kvs, bom.KeyValue{Key: additionalTrustedCAsKey, Value: "false"})
	assert.NotContains(t, kvs, bom.KeyValue{Key: ingressTLSSourceKey, Value: caTLSSource})
	assert.NotContains(t, kvs, bom.KeyValue{Key: privateCAKey, Value: privateCAValue})
}

// TestApplendLetsEncryptProdEnvOverrides verifies that Helm overrides are added as appropriate for LE Prod
// GIVEN a Verrazzano CR
//
//	WHEN AppendOverrides is called with an LE prod configuration where the env is explicitly specified
//	THEN AppendOverrides should add the appropriate LE prod overrides
func TestApplendLetsEncryptProdEnvOverrides(t *testing.T) {
	// Create a fake ComponentContext with the profiles dir to create an EffectiveCR; this is required to
	// convert the CertManager config to the ClusterIssuer config
	vzACMEProd := vzAcmeDev.DeepCopy()
	vzACMEProd.Spec.Components.CertManager.Certificate.Acme.Environment = cmconstants.LetsEncryptProduction
	ctx := spi.NewFakeContext(fake.NewClientBuilder().WithScheme(getScheme()).Build(), vzACMEProd, nil,
		false, profilesRelativePath)
	config.SetDefaultBomFilePath(testBomFilePath)

	kvs, _ := AppendOverrides(ctx, "", "", "", []bom.KeyValue{})
	assert.Contains(t, kvs, bom.KeyValue{Key: letsEncryptIngressClassKey, Value: common.RancherName})
	assert.Contains(t, kvs, bom.KeyValue{Key: letsEncryptEmailKey, Value: vzACMEProd.Spec.Components.CertManager.Certificate.Acme.EmailAddress})
	assert.Contains(t, kvs, bom.KeyValue{Key: letsEncryptEnvironmentKey, Value: cmconstants.LetsEncryptProduction})
	assert.Contains(t, kvs, bom.KeyValue{Key: ingressTLSSourceKey, Value: letsEncryptTLSSource})
	assert.Contains(t, kvs, bom.KeyValue{Key: additionalTrustedCAsKey, Value: "false"})
	assert.NotContains(t, kvs, bom.KeyValue{Key: ingressTLSSourceKey, Value: caTLSSource})
	assert.NotContains(t, kvs, bom.KeyValue{Key: privateCAKey, Value: privateCAValue})
}

// TestApplendLetsEncryptStagingEnvOverrides verifies that Helm overrides are added as appropriate for LE Staging env
// GIVEN a Verrazzano CR
//
//	WHEN AppendOverrides is called with an LE staging configuration
//	THEN AppendOverrides should add the appropriate LE prod overrides
func TestApplendLetsEncryptStagingEnvOverrides(t *testing.T) {
	// Create a fake ComponentContext with the profiles dir to create an EffectiveCR; this is required to
	// convert the CertManager config to the ClusterIssuer config
	vzACMEProd := vzAcmeDev.DeepCopy()
	vzACMEProd.Spec.Components.CertManager.Certificate.Acme.Environment = cmconstants.LetsEncryptStaging
	ctx := spi.NewFakeContext(fake.NewClientBuilder().WithScheme(getScheme()).Build(), vzACMEProd, nil,
		false, profilesRelativePath)
	config.SetDefaultBomFilePath(testBomFilePath)

	kvs, _ := AppendOverrides(ctx, "", "", "", []bom.KeyValue{})
	assert.Contains(t, kvs, bom.KeyValue{Key: letsEncryptIngressClassKey, Value: common.RancherName})
	assert.Contains(t, kvs, bom.KeyValue{Key: letsEncryptEmailKey, Value: vzACMEProd.Spec.Components.CertManager.Certificate.Acme.EmailAddress})
	assert.Contains(t, kvs, bom.KeyValue{Key: letsEncryptEnvironmentKey, Value: cmconstants.LetsEncryptStaging})
	assert.Contains(t, kvs, bom.KeyValue{Key: ingressTLSSourceKey, Value: letsEncryptTLSSource})
	assert.Contains(t, kvs, bom.KeyValue{Key: additionalTrustedCAsKey, Value: "true"})
	assert.NotContains(t, kvs, bom.KeyValue{Key: ingressTLSSourceKey, Value: caTLSSource})
	assert.NotContains(t, kvs, bom.KeyValue{Key: privateCAKey, Value: privateCAValue})
}

// TestAppendImageOverrides verifies that Rancher image overrides are added
// GIVEN a Verrazzano CR
// AND  there is no registry override
// WHEN appendImageOverrides is called
// THEN appendImageOverrides should add the image overrides with the registry prepended
func TestAppendImageOverrides(t *testing.T) {
	a := assert.New(t)

	// Create a fake ComponentContext with the profiles dir to create an EffectiveCR; this is required to
	// convert the CertManager config to the ClusterIssuer config
	ctx := spi.NewFakeContext(fake.NewClientBuilder().WithScheme(getScheme()).Build(), &vzapi.Verrazzano{}, nil, false, profilesRelativePath)

	config.SetDefaultBomFilePath(testBomFilePath)
	_ = os.Unsetenv(constants.RegistryOverrideEnvVar)

	// construct an expected image list
	expectedImages := map[string]bool{}
	for key := range imageEnvVars {
		expectedImages[key] = false
	}

	kvs, err := appendImageOverrides(ctx, []bom.KeyValue{})
	a.Nil(err)
	a.Equal(20, len(kvs))
	for _, kv := range kvs {
		// special exception for the extra arguments
		if kv.Value == "true" {
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
		a.Equal(splitImage[0], "ghcr.io", "Expected image to have the ghcr.io prefix")
	}

	for key, val := range expectedImages {
		a.True(val, fmt.Sprintf("Image %s was not found in the key value arguments:\n%v", key, expectedImages))
	}
}

// TestPSPEnabledOverrides verifies that pspEnabled override is added if K8s version is 1.25 or above
func TestPSPEnabledOverrides(t *testing.T) {
	tests := []struct {
		name                     string
		getKubernetesVersionFunc func() (string, error)
		isError                  bool
		overrideAdded            bool
	}{
		{
			name:                     "testPSPEnabledOverrideNotAdded",
			getKubernetesVersionFunc: func() (string, error) { return "v1.23.5", nil },
			isError:                  false,
			overrideAdded:            false,
		},
		{
			name:                     "testPSPEnabledOverrideAddedFor_1_25",
			getKubernetesVersionFunc: func() (string, error) { return "v1.25.3", nil },
			isError:                  false,
			overrideAdded:            true,
		},
		{
			name:                     "testPSPEnabledOverrideAddedFor_1_25_Above",
			getKubernetesVersionFunc: func() (string, error) { return "1.26.5", nil },
			isError:                  false,
			overrideAdded:            true,
		},
		{
			name:                     "testPSPEnabledOverrideError",
			getKubernetesVersionFunc: func() (string, error) { return "xx1.26.5", fmt.Errorf("errored out") },
			isError:                  true,
			overrideAdded:            true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getKubernetesClusterVersion = tt.getKubernetesVersionFunc
			defer func() {
				getKubernetesClusterVersion = getKubernetesTestVersion

			}()
			ctx := spi.NewFakeContext(fake.NewClientBuilder().WithScheme(getScheme()).Build(), &vzapi.Verrazzano{}, nil, false)
			kvs, err := appendPSPEnabledOverrides(ctx, []bom.KeyValue{})
			if !tt.isError {
				assert.Nil(t, err)
				if tt.overrideAdded {
					assert.Equal(t, 1, len(kvs))
					v, ok := getValue(kvs, pspEnabledKey)
					assert.True(t, ok)
					assert.Equal(t, "false", v)
				} else {
					assert.Equal(t, 0, len(kvs))
				}
			} else {
				assert.Error(t, err)
			}
		})
	}
}

// TestAppendImageOverrides verifies that Rancher image overrides are added
// GIVEN a Verrazzano CR
// AND  there a registry override
// WHEN appendImageOverrides is called
// THEN appendImageOverrides should add the image overrides without the registry prepended
func TestAppendImageOverridesWithRegistryOverride(t *testing.T) {
	a := assert.New(t)
	ctx := spi.NewFakeContext(fake.NewClientBuilder().WithScheme(getScheme()).Build(), &vzapi.Verrazzano{}, nil, false)
	config.SetDefaultBomFilePath(testBomFilePath)
	err := os.Setenv(constants.RegistryOverrideEnvVar, "my-private-reg")
	a.NoError(err)

	// construct an expected image list
	expectedImages := map[string]bool{}
	for key := range imageEnvVars {
		expectedImages[key] = false
	}

	kvs, err := appendImageOverrides(ctx, []bom.KeyValue{})
	a.Nil(err)
	a.Equal(20, len(kvs))
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
		a.NotEqual(splitImage[0], "ghcr.io", "Did not expect image to have the ghcr.io prefix")
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
	// Create a fake ComponentContext with the profiles dir to create an EffectiveCR; this is required to
	// convert the CertManager config to the ClusterIssuer config
	ctx := spi.NewFakeContext(fake.NewClientBuilder().WithScheme(getScheme()).Build(), &vzDefaultCA, nil,
		false, profilesRelativePath)

	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() { config.SetDefaultBomFilePath("") }()

	kvs, err := AppendOverrides(ctx, "", "", "", []bom.KeyValue{})
	assert.Nil(t, err)
	v, ok := getValue(kvs, ingressTLSSourceKey)
	assert.True(t, ok)
	assert.Equal(t, caTLSSource, v)
	v, ok = getValue(kvs, privateCAKey)
	assert.True(t, ok)
	assert.Equal(t, privateCAValue, v)
}

// TestAppendCustomCAOverrides verifies that CA overrides are added as appropriate for custom CAs
// GIVEN a Verrzzano CR with a Custom CA configured in the Certificates field
//
//	WHEN AppendOverrides is called
//	THEN AppendOverrides should add private CA overrides
func TestAppendCustomCAOverrides(t *testing.T) {
	vzCustomCA := vzDefaultCA.DeepCopy()
	namespace := "customnamespace"
	secretName := "customSecret"
	vzCustomCA.Spec.Components.CertManager.Certificate.CA = vzapi.CA{
		ClusterResourceNamespace: namespace,
		SecretName:               secretName,
	}

	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() { config.SetDefaultBomFilePath("") }()
	ctx := spi.NewFakeContext(fake.NewClientBuilder().WithScheme(getScheme()).Build(), vzCustomCA, nil, false,
		profilesRelativePath)

	kvs, err := AppendOverrides(ctx, "", "", "", []bom.KeyValue{})
	assert.Nil(t, err)
	v, ok := getValue(kvs, ingressTLSSourceKey)
	assert.True(t, ok)
	assert.Equal(t, caTLSSource, v)
	v, ok = getValue(kvs, privateCAKey)
	assert.True(t, ok)
	assert.Equal(t, privateCAValue, v)
}

// TestAppendIssuerCustomCAOverrides verifies that CA overrides are added as appropriate for custom CAs using the ClusterIssuer component
// GIVEN a Verrzzano CR with a Custom CA configured in the ClusterIssuerComponent
//
//	WHEN AppendOverrides is called
//	THEN AppendOverrides should add private CA overrides
func TestAppendIssuerCustomCAOverrides(t *testing.T) {
	namespace := "customnamespace"
	secretName := "customSecret"
	vzCustomCA := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				DNS: &vzapi.DNSComponent{
					External: &vzapi.External{Suffix: common.RancherName},
				},
				ClusterIssuer: &vzapi.ClusterIssuerComponent{
					ClusterResourceNamespace: namespace,
					IssuerConfig: vzapi.IssuerConfig{
						CA: &vzapi.CAIssuer{
							SecretName: secretName,
						},
					},
				},
			},
		},
	}

	config.SetDefaultBomFilePath(testBomFilePath)
	defer func() { config.SetDefaultBomFilePath("") }()

	ctx := spi.NewFakeContext(fake.NewClientBuilder().WithScheme(getScheme()).Build(), vzCustomCA, nil, false)

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
	defer helmcli.SetDefaultActionConfigFunction()
	helmcli.SetActionConfigFunction(func(log vzlog.VerrazzanoLogger, settings *cli.EnvSettings, namespace string) (*action.Configuration, error) {
		return helmcli.CreateActionConfig(true, ComponentName, release.StatusDeployed, vzlog.DefaultLogger(), func(name string, releaseStatus release.Status) *release.Release {
			now := time.Now()
			return &release.Release{
				Name:      ComponentName,
				Namespace: ComponentNamespace,
				Info: &release.Info{
					FirstDeployed: now,
					LastDeployed:  now,
					Status:        releaseStatus,
					Description:   "Named Release Stub",
				},
				Version: 1,
			}
		})
	})

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

	defer config.Set(config.Get())
	config.Set(config.OperatorConfig{VerrazzanoRootDir: "../../../../../"})

	defer helmcli.SetDefaultActionConfigFunction()
	helmcli.SetActionConfigFunction(func(log vzlog.VerrazzanoLogger, settings *cli.EnvSettings, namespace string) (*action.Configuration, error) {
		return helmcli.CreateActionConfig(true, ComponentName, release.StatusDeployed, vzlog.DefaultLogger(), func(name string, releaseStatus release.Status) *release.Release {
			now := time.Now()
			return &release.Release{
				Name:      ComponentName,
				Namespace: ComponentNamespace,
				Info: &release.Info{
					FirstDeployed: now,
					LastDeployed:  now,
					Status:        releaseStatus,
					Description:   "Named Release Stub",
				},
				Version: 1,
			}
		})
	})

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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fake ComponentContext with the profiles dir to create an EffectiveCR; this is required to
			// convert the legacy CertManager config to the ClusterIssuer config
			ctx := spi.NewFakeContext(tt.c, &tt.vz, nil, false, profilesRelativePath)

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

			} else {
				assert.ErrorContains(t, err, tt.errContains)
			}
		})
	}
}

// TestMissingCertificateIssuerConfiguration tests the Install() method such that
// GIVEN a call to Install()
// WHEN there is an env with correct rancher deployment and ingress but the Verrazzano resource is missing a cluster issuer configuration
// THEN an error is returned complaining about missing the issuer configuration in the CR
func TestMissingCertificateIssuerConfiguration(t *testing.T) {
	c := createInstallTestClient()
	vz :=
		vzapi.Verrazzano{
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
		}
	// In this case we expressly do NOT create an effective CR to ensure we create the error condition; otherwise the
	// Effective CR will always have a minimal/default issuer configuration
	ctx := spi.NewFakeContext(c, &vz, nil, false)
	err := NewComponent().Install(ctx)
	assert.Error(t, err)
	assert.ErrorContains(t, err, missingIssuerMessage)
}

func createInstallTestClient() client.Client {
	return createFakeTestClient(&v1.Ingress{
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
	ctxWithoutIngress, _ := prepareContexts()
	assert.IsType(t, fmt.Errorf(""), component.PostInstall(ctxWithoutIngress))
	//	assert.Nil(t, component.PostInstall(ctxWithIngress))
}

// TestPostUpgrade tests a happy path post upgrade run
// GIVEN a Rancher install state where all components are ready
//
//	WHEN PostUpgrade is called
//	THEN PostUpgrade should return nil
func TestPostUpgrade(t *testing.T) {
	s := getScheme()
	s.AddKnownTypeWithName(GVKNodeDriverList, &unstructured.UnstructuredList{})
	fakeDynamicClient := dynfake.NewSimpleDynamicClient(s)
	prevGetDynamicClientFunc := getDynamicClientFunc
	getDynamicClientFunc = func() (dynamic.Interface, error) { return fakeDynamicClient, nil }
	defer func() {
		getDynamicClientFunc = prevGetDynamicClientFunc
	}()
	component := NewComponent()
	ctxWithoutIngress, _ := prepareContexts()
	assert.Error(t, component.PostUpgrade(ctxWithoutIngress))
	//	assert.Nil(t, component.PostUpgrade(ctxWithIngress))
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
	okeDriver := createOkeDriver()
	rancherPod := newPod("cattle-system", "rancher")
	rancherPod.Status = corev1.PodStatus{
		Phase: corev1.PodRunning,
	}

	// Create both fake ComponentContexts with the profiles dir to create an EffectiveCR; this is required to
	// convert the legacy CertManager config to the ClusterIssuer config
	clientWithoutIngress := fake.NewClientBuilder().WithScheme(getScheme()).WithObjects(&caSecret, &rootCASecret,
		&adminSecret, &rancherPodList.Items[0], &serverURLSetting, &okeDriver, &kcIngress, rancherPod).
		Build()
	ctxWithoutIngress := spi.NewFakeContext(clientWithoutIngress, &vzDefaultCA, nil, false, profilesRelativePath)

	clientWithIngress := fake.NewClientBuilder().WithScheme(getScheme()).WithObjects(&caSecret, &rootCASecret,
		&adminSecret, &rancherPodList.Items[0], &ingress, &cert, &serverURLSetting, &okeDriver, &kcIngress, rancherPod).
		Build()
	ctxWithIngress := spi.NewFakeContext(clientWithIngress, &vzDefaultCA, nil, false, profilesRelativePath)

	// mock the pod executor when resetting the Rancher admin password
	scheme.Scheme.AddKnownTypes(schema.GroupVersion{Group: "", Version: "v1"}, &corev1.PodExecOptions{})
	k8sutil.NewPodExecutor = k8sutilfake.NewPodExecutor
	k8sutilfake.PodExecResult = func(url *url.URL) (string, string, error) {
		var commands []string
		if commands = url.Query()["command"]; len(commands) == 3 {
			if strings.Contains(commands[2], fmt.Sprintf("base64 %s", SettingUILogoDarkLogoFilePath)) {
				return base64.StdEncoding.EncodeToString([]byte("<svg>dark</svg>")), "", nil
			}

			if strings.Contains(commands[2], fmt.Sprintf("base64 %s", SettingUILogoLightLogoFilePath)) {
				return base64.StdEncoding.EncodeToString([]byte("<svg>light</svg>")), "", nil
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
	namespaceWithoutLabels := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:      FleetSystemNamespace,
			Namespace: FleetSystemNamespace,
		},
	}
	labelledNamespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:      FleetSystemNamespace,
			Namespace: FleetSystemNamespace,
			Labels: map[string]string{
				constants.VerrazzanoManagedKey: FleetSystemNamespace,
			},
		},
	}
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Rancher: &vzapi.RancherComponent{},
			},
		},
	}
	common.RunValidateInstallTest(t, NewComponent,
		common.ValidateInstallTest{
			Name:       "ValidRancherNamespace",
			WantErr:    "",
			Appsv1Cli:  common.MockGetAppsV1(),
			Corev1Cli:  common.MockGetCoreV1(labelledNamespace),
			DynamicCli: common.MockDynamicClient(),
			Vz:         vz,
		},
		common.ValidateInstallTest{
			Name:       "InvalidRancherNamespace",
			WantErr:    FleetSystemNamespace,
			Appsv1Cli:  common.MockGetAppsV1(),
			Corev1Cli:  common.MockGetCoreV1(namespaceWithoutLabels),
			DynamicCli: common.MockDynamicClient(),
			Vz:         vz,
		},
		common.ValidateInstallTest{
			Name:       "ClusterNotProvisionedByRancher",
			WantErr:    "",
			Appsv1Cli:  common.MockGetAppsV1(),
			Corev1Cli:  common.MockGetCoreV1(getLocalNamespaceNotProvisioned()),
			DynamicCli: common.MockDynamicClient(),
			Vz:         vz,
		},
		common.ValidateInstallTest{
			Name:       "ClusterProvisionedByRancher",
			WantErr:    "",
			Appsv1Cli:  common.MockGetAppsV1(),
			Corev1Cli:  common.MockGetCoreV1(getLocalNamespaceProvisioned()),
			DynamicCli: common.MockDynamicClient(getLocalClusterManagementCattleIo()),
			Vz:         vz,
		})

}

func getLocalNamespaceNotProvisioned() *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: ClusterLocal,
			Labels: map[string]string{
				constants.VerrazzanoManagedKey: ClusterLocal,
			},
		},
	}
}

func getLocalNamespaceProvisioned() *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: ClusterLocal,
			Labels: map[string]string{
				ProviderCattleIoLabel: "rke2",
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: common.APIGroupRancherManagement + "/" + "v3",
					Kind:       ClusterKind,
					Name:       ClusterLocal,
				},
			},
		},
	}
}

func getLocalClusterManagementCattleIo() *unstructured.Unstructured {
	localClusterManagementCattleIo := &unstructured.Unstructured{
		Object: map[string]interface{}{},
	}
	localClusterManagementCattleIo.SetName(ClusterLocal)
	localClusterManagementCattleIo.SetKind(ClusterKind)
	localClusterManagementCattleIo.SetAPIVersion(common.APIGroupRancherManagement + "/" + "v3")
	localClusterManagementCattleIo.SetLabels(map[string]string{ProviderCattleIoLabel: "rke2"})
	return localClusterManagementCattleIo
}

func createFakeTestClient(extraObjs ...client.Object) client.Client {
	objs := []client.Object{}
	objs = append(objs, extraObjs...)
	c := fake.NewClientBuilder().WithScheme(getScheme()).WithObjects(objs...).Build()
	return c
}

func getBoolPtr(b bool) *bool {
	return &b
}
