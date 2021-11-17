// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/bom"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"strings"
	"testing"
)

func getValue(kvs []bom.KeyValue, key string) (string, bool) {
	for _, kv := range kvs {
		if strings.EqualFold(key, kv.Key) {
			return kv.Value, true
		}
	}
	return "", false
}

func TestAppendRegistryOverrides(t *testing.T) {
	ctx := spi.NewFakeContext(fake.NewFakeClientWithScheme(getScheme()), &vzAcmeDev, false)
	registry := "foobar"
	imageRepo := "barfoo"
	kvs, _ := AppendOverrides(ctx, "", "", "", []bom.KeyValue{})
	assert.Equal(t, 6, len(kvs)) // should only have LetsEncrypt Overrides
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
	setBashFunc(func(inArgs ...string) (string, string, error) {
		return "", "", nil
	})
	caSecret := createCASecret()
	c := fake.NewFakeClientWithScheme(getScheme(), &caSecret)
	ctx := spi.NewFakeContext(c, &vzDefaultCA, false)
	assert.Nil(t, NewComponent().PreInstall(ctx))
}

func TestIsReady(t *testing.T) {
	deploy := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ComponentNamespace,
			Name:      ComponentName,
		},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: 1,
		},
	}
	unreadyDeploy := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ComponentNamespace,
			Name:      ComponentName,
		},
	}
	rancherIngress := createRancherIngress()
	readyClient := fake.NewFakeClientWithScheme(getScheme(), &deploy, &rancherIngress)
	unreadyDeployClient := fake.NewFakeClientWithScheme(getScheme(), &unreadyDeploy, &rancherIngress)
	unreadyIngressClient := fake.NewFakeClientWithScheme(getScheme(), &deploy)

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
		{
			"should not be ready due to ingress",
			spi.NewFakeContext(unreadyIngressClient, &vzDefaultCA, true),
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			assert.Equal(t, tt.isReady, NewComponent().IsReady(tt.ctx))
		})
	}
}

func TestPostInstall(t *testing.T) {
	setBashFunc(func(inArgs ...string) (string, string, error) {
		return "", "", nil
	})
	caSecret := createCASecret()
	adminSecret := createAdminSecret()
	rancherPodList := createRancherPodList()
	rancherIngress := createRancherIngress()
	c := fake.NewFakeClientWithScheme(getScheme(), &caSecret, &adminSecret, &rancherPodList, &rancherIngress)
	ctx := spi.NewFakeContext(c, &vzDefaultCA, false)
	assert.Nil(t, NewComponent().PostInstall(ctx))
}
