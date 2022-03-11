// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/bom"
	spi2 "github.com/verrazzano/verrazzano/pkg/controller/errors"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	k8sutilfake "github.com/verrazzano/verrazzano/pkg/k8sutil/fake"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
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
//  WHEN AppendOverrides is called
//  THEN AppendOverrides should add registry overrides
func TestAppendRegistryOverrides(t *testing.T) {
	ctx := spi.NewFakeContext(fake.NewFakeClientWithScheme(getScheme()), &vzAcmeDev, false)
	registry := "foobar"
	imageRepo := "barfoo"
	kvs, _ := AppendOverrides(ctx, "", "", "", []bom.KeyValue{})
	assert.Equal(t, 7, len(kvs)) // should only have LetsEncrypt + useBundledSystemChart Overrides
	_ = os.Setenv(constants.RegistryOverrideEnvVar, registry)
	kvs, _ = AppendOverrides(ctx, "", "", "", []bom.KeyValue{})
	assert.Equal(t, 8, len(kvs))
	v, ok := getValue(kvs, systemDefaultRegistryKey)
	assert.True(t, ok)
	assert.Equal(t, registry, v)

	_ = os.Setenv(constants.ImageRepoOverrideEnvVar, imageRepo)
	kvs, _ = AppendOverrides(ctx, "", "", "", []bom.KeyValue{})
	assert.Equal(t, 8, len(kvs))
	v, ok = getValue(kvs, systemDefaultRegistryKey)
	assert.True(t, ok)
	assert.Equal(t, fmt.Sprintf("%s/%s", registry, imageRepo), v)
}

// TestAppendCAOverrides verifies that CA overrides are added as appropriate for private CAs
// GIVEN a Verrzzano CR
//  WHEN AppendOverrides is called
//  THEN AppendOverrides should add private CA overrides
func TestAppendCAOverrides(t *testing.T) {
	ctx := spi.NewFakeContext(fake.NewFakeClientWithScheme(getScheme()), &vzDefaultCA, false)
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
//  WHEN IsEnabled is called
//  THEN IsEnabled should return true/false depending on the enabled state of the CR
func TestIsEnabled(t *testing.T) {
	enabled := true
	disabled := false
	c := fake.NewFakeClientWithScheme(getScheme())
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
			spi.NewFakeContext(c, &vzWithRancher, false),
			true,
		},
		{
			"should not be enabled",
			spi.NewFakeContext(c, &vzNoRancher, false),
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			r := NewComponent()
			assert.Equal(t, tt.enabled, r.IsEnabled(tt.ctx))
		})
	}
}

func TestPreInstall(t *testing.T) {
	caSecret := createCASecret()
	c := fake.NewFakeClientWithScheme(getScheme(), &caSecret)
	ctx := spi.NewFakeContext(c, &vzDefaultCA, false)
	assert.Nil(t, NewComponent().PreInstall(ctx))
}

// TestIsReady verifies that a ready-state Rancher shows as ready
// GIVEN a ready Rancher install
//  WHEN IsReady is called
//  THEN IsReady should return true
func TestIsReady(t *testing.T) {
	readyClient := fake.NewFakeClientWithScheme(getScheme(),
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      ComponentName,
				Labels:    map[string]string{"app": ComponentName},
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 1,
				Replicas:          1,
				UpdatedReplicas:   1,
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      rancherWebhookDeployment,
				Labels:    map[string]string{"app": rancherWebhookDeployment},
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 1,
				Replicas:          1,
				UpdatedReplicas:   1,
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: OperatorNamespace,
				Name:      rancherOperatorDeployment,
				Labels:    map[string]string{"app": rancherOperatorDeployment},
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 1,
				Replicas:          1,
				UpdatedReplicas:   1,
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: fleetSystemNamespace,
				Name:      fleetAgentDeployment,
				Labels:    map[string]string{"app": fleetAgentDeployment},
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 1,
				Replicas:          1,
				UpdatedReplicas:   1,
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: fleetSystemNamespace,
				Name:      fleetControllerDeployment,
				Labels:    map[string]string{"app": fleetControllerDeployment},
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 1,
				Replicas:          1,
				UpdatedReplicas:   1,
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: fleetSystemNamespace,
				Name:      gitjobDeployment,
				Labels:    map[string]string{"app": gitjobDeployment},
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 1,
				Replicas:          1,
				UpdatedReplicas:   1,
			},
		},
	)
	unreadyDeployClient := fake.NewFakeClientWithScheme(getScheme(),
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
				Namespace: OperatorNamespace,
				Name:      rancherOperatorDeployment,
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 0,
				Replicas:          1,
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: fleetSystemNamespace,
				Name:      fleetAgentDeployment,
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 0,
				Replicas:          1,
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: fleetSystemNamespace,
				Name:      fleetControllerDeployment,
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 0,
				Replicas:          1,
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: fleetSystemNamespace,
				Name:      gitjobDeployment,
			},
			Status: appsv1.DeploymentStatus{
				AvailableReplicas: 0,
				Replicas:          1,
			},
		},
	)

	var tests = []struct {
		testName string
		ctx      spi.ComponentContext
		isReady  bool
	}{
		{
			"should be ready",
			spi.NewFakeContext(readyClient, &vzDefaultCA, true),
			true,
		},
		{
			"should not be ready due to deployment",
			spi.NewFakeContext(unreadyDeployClient, &vzDefaultCA, true),
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
//  WHEN PostInstall is called
//  THEN PostInstall should return nil
func TestPostInstall(t *testing.T) {
	// mock the k8s resources used in post install
	caSecret := createCASecret()
	rootCASecret := createRootCASecret()
	adminSecret := createAdminSecret()
	rancherPodList := createRancherPodListWithAllRunning()
	ingress := v1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: common.CattleSystem,
			Name:      constants.RancherIngress,
		},
	}
	clientWithoutIngress := fake.NewFakeClientWithScheme(getScheme(), &caSecret, &rootCASecret, &adminSecret, &rancherPodList)
	ctxWithoutIngress := spi.NewFakeContext(clientWithoutIngress, &vzDefaultCA, false)

	clientWithIngress := fake.NewFakeClientWithScheme(getScheme(), &caSecret, &rootCASecret, &adminSecret, &rancherPodList, &ingress)
	ctxWithIngress := spi.NewFakeContext(clientWithIngress, &vzDefaultCA, false)

	// mock the pod executor when resetting the Rancher admin password
	k8sutil.NewPodExecutor = k8sutilfake.NewPodExecutor
	k8sutilfake.PodSTDOUT = "password"
	k8sutil.ClientConfig = func() (*rest.Config, kubernetes.Interface, error) {
		config, k := k8sutilfake.NewClientsetConfig()
		return config, k, nil
	}

	// mock the HTTP responses for the Rancher API
	common.HTTPDo = func(hc *http.Client, req *http.Request) (*http.Response, error) {
		url := req.URL.String()
		if strings.Contains(url, common.RancherServerURLPath) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("blahblah")),
			}, nil
		}
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`{"token":"token"}`)),
		}, nil
	}

	component := NewComponent()
	assert.IsType(t, spi2.RetryableError{}, component.PostInstall(ctxWithoutIngress))
	assert.Nil(t, component.PostInstall(ctxWithIngress))
}
