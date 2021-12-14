// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package externaldns

import (
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/bom"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

const (
	profileDir = "../../../../manifests/profiles"
)

// Default Verrazzano object
var vz = &vzapi.Verrazzano{
	Spec: vzapi.VerrazzanoSpec{
		EnvironmentName: "myenv",
		Components: vzapi.ComponentSpec{
			DNS: &vzapi.DNSComponent{},
		},
	},
}

var oci = &vzapi.OCI{
	OCIConfigSecret:        "oci",
	DNSZoneCompartmentOCID: "compartmentID",
	DNSZoneOCID:            "zoneID",
	DNSZoneName:            "zone.name.io",
}

var ociGlobalScope = &vzapi.OCI{
	OCIConfigSecret:        "oci",
	DNSZoneCompartmentOCID: "compartmentID",
	DNSZoneOCID:            "zoneID",
	DNSZoneName:            "zone.name.io",
	DNSScope:               "Global",
}
var ociPrivateScope = &vzapi.OCI{
	OCIConfigSecret:        "oci",
	DNSZoneCompartmentOCID: "compartmentID",
	DNSZoneOCID:            "zoneID",
	DNSZoneName:            "zone.name.io",
	DNSScope:               "Private",
}

var ociInvalidScope = &vzapi.OCI{
	OCIConfigSecret:        "oci",
	DNSZoneCompartmentOCID: "compartmentID",
	DNSZoneOCID:            "zoneID",
	DNSZoneName:            "zone.name.io",
	DNSScope:               "#jhwuyusj!!!",
}

var fakeComponent = externalDNSComponent{}

// TestIsExternalDNSEnabled tests the IsEnabled fn
// GIVEN a call to IsEnabled
// WHEN OCI DNS is enabled
// THEN the function returns true
func TestIsExternalDNSEnabled(t *testing.T) {
	localvz := vz.DeepCopy()
	localvz.Spec.Components.DNS.OCI = &vzapi.OCI{}
	assert.True(t, fakeComponent.IsEnabled(spi.NewFakeContext(nil, localvz, false)))
}

// TestIsExternalDNSDisabled tests the IsEnabled fn
// GIVEN a call to IsEnabled
// WHEN OCI DNS is disabled
// THEN the function returns true
func TestIsExternalDNSDisabled(t *testing.T) {
	assert.False(t, fakeComponent.IsEnabled(spi.NewFakeContext(nil, vz, false)))
}

// TestIsExternalDNSReady tests the IsReady fn
// GIVEN a call to IsReady
// WHEN the external dns deployment is ready
// THEN the function returns true
func TestIsExternalDNSReady(t *testing.T) {
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme,
		newDeployment(externalDNSDeploymentName, true),
	)
	assert.True(t, fakeComponent.IsReady(spi.NewFakeContext(client, nil, false)))
}

// TestIsExternalDNSNotReady tests the IsReady fn
// GIVEN a call to IsReady
// WHEN the external dns deployment is not ready
// THEN the function returns false
func TestIsExternalDNSNotReady(t *testing.T) {
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme,
		newDeployment(externalDNSDeploymentName, false),
	)
	assert.False(t, fakeComponent.IsReady(spi.NewFakeContext(client, nil, false)))
}

// TestAppendExternalDNSOverrides tests the AppendOverrides fn
// GIVEN a call to AppendOverrides
// WHEN a VZ spec is passed with defaults
// THEN the values created properly
func TestAppendExternalDNSOverrides(t *testing.T) {
	localvz := vz.DeepCopy()
	localvz.Spec.Components.DNS.OCI = oci
	kvs, err := AppendOverrides(spi.NewFakeContext(nil, localvz, false, profileDir), ComponentName, externalDNSNamespace, "", []bom.KeyValue{})
	assert.NoError(t, err)
	assert.Len(t, kvs, 10)
}

// TestExternalDNSPreInstallDryRun tests the PreInstall fn
// GIVEN a call to this fn
// WHEN I call PreInstall with dry-run = true
// THEN no errors are returned
func TestExternalDNSPreInstallDryRun(t *testing.T) {
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme)
	err := fakeComponent.PreInstall(spi.NewFakeContext(client, &vzapi.Verrazzano{}, true))
	assert.NoError(t, err)
}

// TestExternalDNSPreInstall tests the PreInstall fn
// GIVEN a call to this fn
// WHEN I call PreInstall
// THEN no errors are returned
func TestExternalDNSPreInstall(t *testing.T) {
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme,
		&v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "oci",
				Namespace: constants.VerrazzanoInstallNamespace,
			},
			Data: map[string][]byte{"oci.yaml": []byte("fake data")},
		})
	localvz := vz.DeepCopy()
	localvz.Spec.Components.DNS.OCI = oci
	err := fakeComponent.PreInstall(spi.NewFakeContext(client, localvz, false))
	assert.NoError(t, err)
}

func TestExternalDNSPreInstallGlobalScope(t *testing.T) {
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme,
		&v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "oci",
				Namespace: constants.VerrazzanoInstallNamespace,
			},
			Data: map[string][]byte{"oci.yaml": []byte("fake data")},
		})
	localvz := vz.DeepCopy()
	localvz.Spec.Components.DNS.OCI = ociGlobalScope
	err := fakeComponent.PreInstall(spi.NewFakeContext(client, localvz, false))
	assert.NoError(t, err)
}

func TestExternalDNSPreInstallPrivateScope(t *testing.T) {
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme,
		&v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "oci",
				Namespace: constants.VerrazzanoInstallNamespace,
			},
			Data: map[string][]byte{"oci.yaml": []byte("fake data")},
		})
	localvz := vz.DeepCopy()
	localvz.Spec.Components.DNS.OCI = ociPrivateScope
	err := fakeComponent.PreInstall(spi.NewFakeContext(client, localvz, false))
	assert.NoError(t, err)
}

func TestExternalDNSPreInstall3InvalidScope(t *testing.T) {
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme,
		&v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "oci",
				Namespace: constants.VerrazzanoInstallNamespace,
			},
			Data: map[string][]byte{"oci.yaml": []byte("fake data")},
		})
	localvz := vz.DeepCopy()
	localvz.Spec.Components.DNS.OCI = ociInvalidScope
	err := fakeComponent.PreInstall(spi.NewFakeContext(client, localvz, false))
	assert.Error(t, err)
}

// Create a new deployment object for testing
func newDeployment(name string, ready bool) *appsv1.Deployment {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: externalDNSNamespace,
			Name:      name,
		},
		Status: appsv1.DeploymentStatus{
			Replicas:            1,
			ReadyReplicas:       1,
			AvailableReplicas:   1,
			UnavailableReplicas: 0,
		},
	}

	if !ready {
		deployment.Status = appsv1.DeploymentStatus{
			Replicas:            1,
			ReadyReplicas:       0,
			AvailableReplicas:   0,
			UnavailableReplicas: 1,
		}
	}
	return deployment
}
