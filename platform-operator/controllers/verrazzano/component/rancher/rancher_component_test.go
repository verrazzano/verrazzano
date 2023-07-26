// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/test/keycloakutil"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	certapiv1 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/bom"
	constants2 "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	k8sutilfake "github.com/verrazzano/verrazzano/pkg/k8sutil/fake"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/mocks"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	corev1Cli "k8s.io/client-go/kubernetes/typed/core/v1"
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
	registry := "foobar"
	imageRepo := "barfoo"
	kvs, _ := AppendOverrides(ctx, "", "", "", []bom.KeyValue{})
	assert.Equal(t, 8, len(kvs)) // should only have LetsEncrypt + useBundledSystemChart Overrides
	_ = os.Setenv(constants.RegistryOverrideEnvVar, registry)
	kvs, _ = AppendOverrides(ctx, "", "", "", []bom.KeyValue{})
	assert.Equal(t, 9, len(kvs))
	v, ok := getValue(kvs, systemDefaultRegistryKey)
	assert.True(t, ok)
	assert.Equal(t, registry, v)

	_ = os.Setenv(constants.ImageRepoOverrideEnvVar, imageRepo)
	kvs, _ = AppendOverrides(ctx, "", "", "", []bom.KeyValue{})
	assert.Equal(t, 9, len(kvs))
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
	assert.Contains(t, kvs, bom.KeyValue{Key: letsEncryptEnvironmentKey, Value: letsencryptProduction})
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
	vzACMEProd.Spec.Components.CertManager.Certificate.Acme.Environment = letsencryptProduction
	ctx := spi.NewFakeContext(fake.NewClientBuilder().WithScheme(getScheme()).Build(), vzACMEProd, nil,
		false, profilesRelativePath)
	config.SetDefaultBomFilePath(testBomFilePath)

	kvs, _ := AppendOverrides(ctx, "", "", "", []bom.KeyValue{})
	assert.Contains(t, kvs, bom.KeyValue{Key: letsEncryptIngressClassKey, Value: common.RancherName})
	assert.Contains(t, kvs, bom.KeyValue{Key: letsEncryptEmailKey, Value: vzACMEProd.Spec.Components.CertManager.Certificate.Acme.EmailAddress})
	assert.Contains(t, kvs, bom.KeyValue{Key: letsEncryptEnvironmentKey, Value: letsencryptProduction})
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
	vzACMEProd.Spec.Components.CertManager.Certificate.Acme.Environment = letsEncryptStaging
	ctx := spi.NewFakeContext(fake.NewClientBuilder().WithScheme(getScheme()).Build(), vzACMEProd, nil,
		false, profilesRelativePath)
	config.SetDefaultBomFilePath(testBomFilePath)

	kvs, _ := AppendOverrides(ctx, "", "", "", []bom.KeyValue{})
	assert.Contains(t, kvs, bom.KeyValue{Key: letsEncryptIngressClassKey, Value: common.RancherName})
	assert.Contains(t, kvs, bom.KeyValue{Key: letsEncryptEmailKey, Value: vzACMEProd.Spec.Components.CertManager.Certificate.Acme.EmailAddress})
	assert.Contains(t, kvs, bom.KeyValue{Key: letsEncryptEnvironmentKey, Value: letsEncryptStaging})
	assert.Contains(t, kvs, bom.KeyValue{Key: ingressTLSSourceKey, Value: letsEncryptTLSSource})
	assert.Contains(t, kvs, bom.KeyValue{Key: additionalTrustedCAsKey, Value: "true"})
	assert.NotContains(t, kvs, bom.KeyValue{Key: ingressTLSSourceKey, Value: caTLSSource})
	assert.NotContains(t, kvs, bom.KeyValue{Key: privateCAKey, Value: privateCAValue})
}

// TestAppendCAOverrides verifies that CA overrides are added as appropriate for private CAs
// GIVEN a Verrzzano CR
//
//	WHEN AppendOverrides is called
//	THEN AppendOverrides should add private CA overrides
func TestAppendCAOverrides(t *testing.T) {
	ctx := spi.NewFakeContext(fake.NewClientBuilder().WithScheme(getScheme()).Build(), &vzDefaultCA, nil, false)
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
// GIVEN a Verrzzano CR
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
	ctx := spi.NewFakeContext(fake.NewClientBuilder().WithScheme(getScheme()).Build(), vzCustomCA, nil, false)
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
	_, ctxWithIngress := prepareContexts()

	err := component.PostInstall(ctxWithIngress)
	assert.NoError(t, err)
}

// TestPostInstallNoIngress tests PostInstall()
// GIVEN a call to PostInstall
//
//	WHEN the ingress is not present
//	THEN PostInstall should return an error
func TestPostInstallNoIngress(t *testing.T) {
	component := NewComponent()
	ctxWithoutIngress, _ := prepareContexts()
	err := component.PostInstall(ctxWithoutIngress)
	assert.Error(t, err)
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
	verrazzanoAdminClusterRole := createClusterRoles("verrazzano-admin")
	verrazzanoMonitorClusterRole := createClusterRoles("verrazzano-monitor")

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
	authConfig := createKeycloakAuthConfig()
	localAuthConfig := createLocalAuthConfig()
	kcSecret := keycloakutil.CreateTestKeycloakLoginSecret()
	firstLoginSetting := createFirstLoginSetting()
	rancherPod := newPod("cattle-system", "rancher")
	rancherPod.Status = corev1.PodStatus{
		Phase: corev1.PodRunning,
	}
	keycloakPod := keycloakutil.CreateTestKeycloakPod()

	clientWithoutIngress := fake.NewClientBuilder().WithScheme(getScheme()).WithObjects(&caSecret, &rootCASecret, &adminSecret, &rancherPodList.Items[0], &serverURLSetting, &ociDriver, &okeDriver, &authConfig, &kcIngress, kcSecret, &localAuthConfig, &firstLoginSetting, &verrazzanoAdminClusterRole, &verrazzanoMonitorClusterRole, rancherPod, keycloakPod).Build()
	ctxWithoutIngress := spi.NewFakeContext(clientWithoutIngress, &vzDefaultCA, nil, false)

	clientWithIngress := fake.NewClientBuilder().WithScheme(getScheme()).WithObjects(&caSecret, &rootCASecret, &adminSecret, &rancherPodList.Items[0], &ingress, &cert, &serverURLSetting, &ociDriver, &okeDriver, &authConfig, &kcIngress, kcSecret, &localAuthConfig, &firstLoginSetting, &verrazzanoAdminClusterRole, &verrazzanoMonitorClusterRole, rancherPod, keycloakPod).Build()
	ctxWithIngress := spi.NewFakeContext(clientWithIngress, &vzDefaultCA, nil, false)
	// mock the pod executor when resetting the Rancher admin password
	scheme.Scheme.AddKnownTypes(schema.GroupVersion{Group: "", Version: "v1"}, &corev1.PodExecOptions{})
	k8sutil.NewPodExecutor = k8sutilfake.NewPodExecutor
	k8sutilfake.PodExecResult = func(url *url.URL) (string, string, error) {
		var commands []string
		if commands = url.Query()["command"]; len(commands) == 3 {
			if strings.Contains(commands[2], "id,clientId") {
				return "[{\"id\":\"something\", \"clientId\":\"" + AuthConfigKeycloakClientIDRancher + "\"}]", "", nil
			}

			if strings.Contains(commands[2], "client-secret") {
				return "{\"type\":\"secret\",\"value\":\"abcdef\"}", "", nil
			}

			if strings.Contains(commands[2], "get users") {
				return "[{\"id\":\"something\", \"username\":\"verrazzano\"}]", "", nil
			}

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

// TestGetSecret tests the getSecret func
// GIVEN a all to getSecret
//
//	THEN the secret is returned, or an error is returned if the secret does not exist
func TestGetSecret(t *testing.T) {
	type args struct {
		namespace string
		name      string
	}
	tests := []struct {
		name    string
		args    args
		want    *corev1.Secret
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "GetSecretFound",
			args: args{name: "mysecret", namespace: ComponentNamespace},
			want: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: "mysecret", Namespace: ComponentNamespace},
			},
			wantErr: assert.NoError,
		},
		{
			name:    "GetSecretNotFound",
			args:    args{name: "mysecret", namespace: ComponentNamespace},
			want:    nil,
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.want == nil {
				k8sutil.GetCoreV1Func = common.MockGetCoreV1()
			} else {
				k8sutil.GetCoreV1Func = common.MockGetCoreV1(tt.want)
			}
			defer k8sutil.ResetCoreV1Client()

			got, err := getSecret(tt.args.namespace, tt.args.name)
			if !tt.wantErr(t, err, fmt.Sprintf("getSecret(%v, %v)", tt.args.namespace, tt.args.name)) {
				return
			}
			if tt.want == nil {
				assert.Nil(t, got)
			} else {
				assert.Equalf(t, tt.want, got, "getSecret(%v, %v)", tt.args.namespace, tt.args.name)
			}
		})
	}
}

// TestRestartRancherDeployment tests the getSecret func
// GIVEN a call to restartRancherDeployment
//
//	THEN the Rancher deployment is annotated for a rolling restart if present, or an error is returned for unexpected errors
func TestRestartRancherDeployment(t *testing.T) {
	log := vzlog.DefaultLogger()
	deploymentName := types.NamespacedName{Namespace: constants2.RancherSystemNamespace, Name: ComponentName}

	tests := []struct {
		name             string
		deploymentExists bool
		createClientFunc func() client.Client
		wantErr          assert.ErrorAssertionFunc
	}{
		{
			name: "RestartSuccessful",
			createClientFunc: func() client.Client {
				mocker := gomock.NewController(t)
				mockClient := mocks.NewMockClient(mocker)
				mockClient.EXPECT().Get(context.TODO(),
					deploymentName,
					gomock.AssignableToTypeOf(&appsv1.Deployment{})).
					DoAndReturn(func(ctx context.Context, key types.NamespacedName, deployment *appsv1.Deployment) error {
						deployment.Name = deploymentName.Name
						deployment.Namespace = deploymentName.Namespace
						return nil
					}).Times(1)
				mockClient.EXPECT().Update(context.TODO(), gomock.AssignableToTypeOf(&appsv1.Deployment{})).
					DoAndReturn(func(ctx context.Context, deployment *appsv1.Deployment, opts ...client.UpdateOption) error {
						assert.Equal(t, deploymentName, client.ObjectKeyFromObject(deployment))
						_, restartAnnotationFound := deployment.Spec.Template.ObjectMeta.Annotations[constants2.VerrazzanoRestartAnnotation]
						assert.Truef(t, restartAnnotationFound, "Restart annotation %s not found", constants2.RestartVersionAnnotation)
						return nil
					}).Times(1)
				return mockClient
			},
			wantErr: assert.NoError,
		},
		{
			name: "DeploymentNotFound",
			createClientFunc: func() client.Client {
				mocker := gomock.NewController(t)
				mockClient := mocks.NewMockClient(mocker)
				mockClient.EXPECT().Get(context.TODO(),
					deploymentName,
					gomock.AssignableToTypeOf(&appsv1.Deployment{})).
					Return(errors.NewNotFound(schema.GroupResource{Group: "appsv1", Resource: "Deployment"},
						deploymentName.Name))
				mockClient.EXPECT().Update(context.TODO(), gomock.AssignableToTypeOf(&appsv1.Deployment{})).Times(0)
				return mockClient
			},
			wantErr: assert.NoError,
		},
		{
			name: "GetUnexpectedError",
			createClientFunc: func() client.Client {
				mocker := gomock.NewController(t)
				mockClient := mocks.NewMockClient(mocker)
				mockClient.EXPECT().Get(context.TODO(),
					deploymentName,
					gomock.AssignableToTypeOf(&appsv1.Deployment{})).
					Return(fmt.Errorf("unexpected error"))
				mockClient.EXPECT().Update(context.TODO(), gomock.AssignableToTypeOf(&appsv1.Deployment{})).Times(0)
				return mockClient
			},
			wantErr: assert.Error,
		},
		{
			name: "UpdateFailed",
			createClientFunc: func() client.Client {
				mocker := gomock.NewController(t)
				mockClient := mocks.NewMockClient(mocker)
				mockClient.EXPECT().Get(context.TODO(),
					deploymentName,
					gomock.AssignableToTypeOf(&appsv1.Deployment{})).
					DoAndReturn(func(ctx context.Context, key types.NamespacedName, deployment *appsv1.Deployment) error {
						deployment.Name = deploymentName.Name
						deployment.Namespace = deploymentName.Namespace
						return nil
					}).Times(1)
				mockClient.EXPECT().Update(context.TODO(), gomock.AssignableToTypeOf(&appsv1.Deployment{})).
					Return(fmt.Errorf("update failed")).
					Times(1)
				return mockClient
			},
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.wantErr(t, restartRancherDeployment(log, tt.createClientFunc()))
		})
	}
}

// TestGetCurrentCABundleSecretsValue tests the getCurrentCABundleSecretsValue  func of the rancherComonent
// GIVEN a call to rancherComponent.getCurrentCABundleSecretsValue
//
//	THEN the bundle data is returned if the secret exists and the bundle is present, or an error is returned and the found bool is false otherwise
func TestGetCurrentCABundleSecretsValue(t *testing.T) {
	bundleData1 := "cabundledata"
	emptyBundle := ""
	bundleDataWithWhitespace := "  \t " + bundleData1 + "\n\t"
	tests := []struct {
		name                string
		cli                 client.Client
		corev1ClientFunc    func(log ...vzlog.VerrazzanoLogger) (corev1Cli.CoreV1Interface, error)
		bundleDataExpected  string
		bundleFoundExpected bool
		wantErr             assert.ErrorAssertionFunc
	}{
		{
			name: "SecretAndBundleKeyExist",
			corev1ClientFunc: common.MockGetCoreV1(&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: rancherTLSSecretName, Namespace: ComponentNamespace},
				Data: map[string][]byte{
					caCertsPem: []byte(bundleData1),
				},
			}),
			bundleDataExpected:  bundleData1,
			bundleFoundExpected: true,
			wantErr:             assert.NoError,
		},
		{
			name:                "SecretDoesNotExist",
			corev1ClientFunc:    common.MockGetCoreV1(),
			bundleDataExpected:  emptyBundle,
			bundleFoundExpected: false,
			wantErr:             assert.NoError,
		},
		{
			name: "BundleKeyDoesNotExist",
			corev1ClientFunc: common.MockGetCoreV1(&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: rancherTLSSecretName, Namespace: ComponentNamespace},
			}),
			bundleDataExpected:  emptyBundle,
			bundleFoundExpected: false,
			wantErr:             assert.Error,
		},
		{
			name: "BundleWithLeadingAndTrailingWhitespace",
			corev1ClientFunc: common.MockGetCoreV1(&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: rancherTLSSecretName, Namespace: ComponentNamespace},
				Data: map[string][]byte{
					caCertsPem: []byte(bundleDataWithWhitespace),
				},
			}),
			bundleDataExpected:  bundleData1,
			bundleFoundExpected: true,
			wantErr:             assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewComponent().(rancherComponent)
			ctx := spi.NewFakeContext(fake.NewClientBuilder().Build(), &vzapi.Verrazzano{}, nil, false)
			k8sutil.GetCoreV1Func = tt.corev1ClientFunc
			defer k8sutil.ResetCoreV1Client()

			bundleData, bundleFound, err := r.getCurrentCABundleSecretsValue(ctx, rancherTLSSecretName, caCertsPem)
			if !tt.wantErr(t, err) {
				return
			}
			assert.Equal(t, tt.bundleFoundExpected, bundleFound)
			assert.Equal(t, tt.bundleDataExpected, bundleData)
		})
	}
}

// TestIsPrivateCABundleInSync tests the isPrivateCABundleInSync  func of the rancherComonent
// GIVEN a call to rancherComponent.isPrivateCABundleInSync
//
//	THEN true is returned if the bundle data in tls-ca is out of sync with the cacerts settings value, or an error
func TestIsPrivateCABundleInSync(t *testing.T) {
	bundleData1 := "cabundledata"
	bundleDataWithWhitespace := "  \t " + bundleData1 + "\n\t"
	tests := []struct {
		name             string
		corev1ClientFunc func(log ...vzlog.VerrazzanoLogger) (corev1Cli.CoreV1Interface, error)
		crtClientFunc    func() client.Client
		exepectedResult  bool
		wantErr          assert.ErrorAssertionFunc
	}{
		{
			name: "SecretAndSettingsInSync",
			corev1ClientFunc: common.MockGetCoreV1(&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: rancherTLSSecretName, Namespace: ComponentNamespace},
				Data: map[string][]byte{
					caCertsPem: []byte(bundleData1),
				},
			}),
			crtClientFunc: func() client.Client {
				return fake.NewClientBuilder().WithScheme(getScheme()).
					WithRuntimeObjects(newCASetting(bundleData1)).Build()
			},
			exepectedResult: true,
			wantErr:         assert.NoError,
		},
		{
			name: "SecretAndSettingsInSyncWithWhiteSpace",
			corev1ClientFunc: common.MockGetCoreV1(&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: rancherTLSSecretName, Namespace: ComponentNamespace},
				Data: map[string][]byte{
					caCertsPem: []byte(bundleDataWithWhitespace),
				},
			}),
			crtClientFunc: func() client.Client {
				return fake.NewClientBuilder().WithScheme(getScheme()).
					WithRuntimeObjects(newCASetting(bundleData1)).Build()
			},
			exepectedResult: true,
			wantErr:         assert.NoError,
		},
		{
			name: "SecretAndSettingsNotInSync",
			corev1ClientFunc: common.MockGetCoreV1(&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: rancherTLSSecretName, Namespace: ComponentNamespace},
				Data: map[string][]byte{
					caCertsPem: []byte(bundleDataWithWhitespace),
				},
			}),
			crtClientFunc: func() client.Client {
				return fake.NewClientBuilder().WithScheme(getScheme()).
					WithRuntimeObjects(newCASetting("old bundle data")).Build()
			},
			exepectedResult: false,
			wantErr:         assert.NoError,
		},
		{
			name:             "SecretDoesNotExist",
			corev1ClientFunc: common.MockGetCoreV1(),
			crtClientFunc: func() client.Client {
				return fake.NewClientBuilder().WithScheme(getScheme()).
					WithRuntimeObjects(newCASetting(bundleData1)).Build()
			},
			exepectedResult: true,
			wantErr:         assert.NoError,
		},
		{
			name: "SettingDoesNotExist",
			corev1ClientFunc: common.MockGetCoreV1(&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: rancherTLSSecretName, Namespace: ComponentNamespace},
				Data: map[string][]byte{
					caCertsPem: []byte(bundleData1),
				},
			}),
			crtClientFunc: func() client.Client {
				return fake.NewClientBuilder().WithScheme(getScheme()).WithRuntimeObjects().Build()
			},
			exepectedResult: false,
			wantErr:         assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewComponent().(rancherComponent)
			ctx := spi.NewFakeContext(tt.crtClientFunc(), &vzapi.Verrazzano{}, nil, false)
			k8sutil.GetCoreV1Func = tt.corev1ClientFunc
			defer k8sutil.ResetCoreV1Client()

			inSync, err := r.isPrivateCABundleInSync(ctx)
			if !tt.wantErr(t, err) {
				return
			}
			assert.Equal(t, tt.exepectedResult, inSync)
		})
	}
}

// TestCheckRestartRequired tests the checkRestartRequired  func of the rancherComonent
// GIVEN a call to rancherComponent.checkRestartRequired
//
//	THEN the Rancher deployment is restarted if the CA bundle is out of sync with the secret AND a Rancher deployment
//	  	rolling update is NOT already in progress
func TestCheckRestartRequired(t *testing.T) {
	deploymentName := types.NamespacedName{Namespace: constants2.RancherSystemNamespace, Name: ComponentName}
	bundleData1 := "cabundledata"
	bundleDataWithWhitespace := "  \t " + bundleData1 + "\n\t"
	staleBundleData := "otherData"

	tests := []struct {
		name             string
		description      string
		corev1ClientFunc func(log ...vzlog.VerrazzanoLogger) (corev1Cli.CoreV1Interface, error)
		crtClientFunc    func() client.Client
		restartExpected  bool
		wantErr          assert.ErrorAssertionFunc
	}{
		{
			name: "SecretAndSettingsInSyncRancherReady",
			description: `Tests that the cattle-system/rancher deployment is NOT restarted when the
				tls-ca bundle is in sync with the cacerts settings, and the deployment is in steady state.  This
				means that there is no need to restart the Rancher pods`,
			corev1ClientFunc: common.MockGetCoreV1(&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: rancherTLSSecretName, Namespace: ComponentNamespace},
				Data: map[string][]byte{
					caCertsPem: []byte(bundleData1),
				},
			}),
			crtClientFunc: func() client.Client {
				return fake.NewClientBuilder().WithScheme(getScheme()).
					WithRuntimeObjects(
						newReadyDeployment(ComponentNamespace, ComponentName),
						newPod(ComponentNamespace, ComponentName),
						newReplicaSet(ComponentNamespace, ComponentName),
						newCASetting(bundleData1)).
					Build()
			},
			restartExpected: false,
			wantErr:         assert.NoError,
		},
		{
			name: "RestartRequiredNotInSync",
			description: `Tests that the cattle-system/rancher deployment is restarted when the
				tls-ca bundle is out of sync with the cacerts settings, and the deployment is in steady state.  This
				means that the we need to restart the Rancher pods in order to pick up the new private CA bundle`,
			corev1ClientFunc: common.MockGetCoreV1(&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: rancherTLSSecretName, Namespace: ComponentNamespace},
				Data: map[string][]byte{
					caCertsPem: []byte(bundleDataWithWhitespace),
				},
			}),
			crtClientFunc: func() client.Client {
				return fake.NewClientBuilder().WithScheme(getScheme()).
					WithRuntimeObjects(
						newReadyDeployment(ComponentNamespace, ComponentName),
						newPod(ComponentNamespace, ComponentName),
						newReplicaSet(ComponentNamespace, ComponentName),
						newCASetting(staleBundleData)).
					Build()
			},
			restartExpected: true,
			wantErr:         assert.NoError,
		},
		{
			name: "UpdateInProgressNoRestartRequired",
			description: `Tests that the cattle-system/rancher deployment is NOT restarted when the
				tls-ca bundle is out of sync with the cacerts settings, and the deployment is already in the middle of
				a rolling restart.  The restart check is done immediately after applying the Rancher Helm chart, so
				other updates to the Rancher configuration have already triggered the deployment to update`,
			corev1ClientFunc: common.MockGetCoreV1(&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: rancherTLSSecretName, Namespace: ComponentNamespace},
				Data: map[string][]byte{
					caCertsPem: []byte(bundleDataWithWhitespace),
				},
			}),
			crtClientFunc: func() client.Client {
				return fake.NewClientBuilder().WithScheme(getScheme()).
					WithRuntimeObjects(
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
						newCASetting(staleBundleData)).
					Build()
			},
			restartExpected: false,
			wantErr:         assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewComponent().(rancherComponent)
			crtClient := tt.crtClientFunc()
			ctx := spi.NewFakeContext(crtClient, &vzapi.Verrazzano{}, nil, false)
			k8sutil.GetCoreV1Func = tt.corev1ClientFunc
			defer k8sutil.ResetCoreV1Client()
			tt.wantErr(t, r.checkRestartRequired(ctx))

			depObject := &appsv1.Deployment{}
			if !assert.NoError(t, crtClient.Get(context.TODO(), deploymentName, depObject)) {
				return
			}
			_, restarted := depObject.Spec.Template.ObjectMeta.Annotations[constants2.VerrazzanoRestartAnnotation]
			assert.Equalf(t, tt.restartExpected, restarted, "Did not get expected restart value")
		})
	}
}

func newCASetting(bundleData1 string) *unstructured.Unstructured {
	expectedSetting := &unstructured.Unstructured{}

	expectedSetting.SetGroupVersionKind(common.GVKSetting)
	expectedSetting.SetName(SettingCACerts)
	unstructuredContent := expectedSetting.UnstructuredContent()

	unstructuredContent["value"] = bundleData1
	return expectedSetting
}
