// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package externaldns

import (
	"github.com/verrazzano/verrazzano/pkg/helm"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/time"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"strings"
	"testing"

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
)

const (
	profileDir = "../../../../manifests/profiles"
)

// Default Verrazzano object
var vz = &vzapi.Verrazzano{
	ObjectMeta: metav1.ObjectMeta{Name: "my-verrazzano", Namespace: "default", CreationTimestamp: metav1.Now()},
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
	DNSScope:               "GLOBAL",
}
var ociPrivateScope = &vzapi.OCI{
	OCIConfigSecret:        "oci",
	DNSZoneCompartmentOCID: "compartmentID",
	DNSZoneOCID:            "zoneID",
	DNSZoneName:            "zone.name.io",
	DNSScope:               "PRIVATE",
}

var ociInvalidScope = &vzapi.OCI{
	OCIConfigSecret:        "oci",
	DNSZoneCompartmentOCID: "compartmentID",
	DNSZoneOCID:            "zoneID",
	DNSZoneName:            "zone.name.io",
	DNSScope:               "#jhwuyusj!!!",
}

var fakeComponent = externalDNSComponent{}

var testScheme = runtime.NewScheme()

func init() {
	_ = k8scheme.AddToScheme(testScheme)
	_ = vzapi.AddToScheme(testScheme)
}

func getChart() *chart.Chart {
	return &chart.Chart{
		Metadata: &chart.Metadata{
			APIVersion: "v1",
			Name:       "hello",
			Version:    "0.1.0",
			AppVersion: "1.0",
		},
		Templates: []*chart.File{
			{Name: "templates/hello", Data: []byte("hello: world")},
		},
	}
}

func createReleaseWithValues(name string, status release.Status) *release.Release {
	now := time.Now()
	return &release.Release{
		Name:      name,
		Namespace: ComponentNamespace,
		Info: &release.Info{
			FirstDeployed: now,
			LastDeployed:  now,
			Status:        status,
			Description:   "Named Release Stub",
		},
		Chart: getChart(),
		Config: map[string]interface{}{
			"domainFilters": []string{
				"my.domain.io",
			},
			"triggerLoopOnEvent": true,
			"txtOwnerId":         "storedOwnerId",
			"txtPrefix":          "storedPrefix",
			"zoneIDFilters": []string{
				"ocid1.dns-zone.oc1..blahblahblah",
			},
		},
		Version: 1,
	}
}

func createReleaseWithoutValues(name string, status release.Status) *release.Release {
	now := time.Now()
	return &release.Release{
		Name:      name,
		Namespace: ComponentNamespace,
		Info: &release.Info{
			FirstDeployed: now,
			LastDeployed:  now,
			Status:        status,
			Description:   "Named Release Stub",
		},
		Chart:   getChart(),
		Version: 1,
	}
}

func testActionConfigWithInstallationAndValues(log vzlog.VerrazzanoLogger, settings *cli.EnvSettings, namespace string) (*action.Configuration, error) {
	return helm.CreateActionConfig(true, "external-dns", release.StatusDeployed, createReleaseWithValues, vzlog.DefaultLogger())
}

func testActionConfigWithInstallationNoValues(log vzlog.VerrazzanoLogger, settings *cli.EnvSettings, namespace string) (*action.Configuration, error) {
	return helm.CreateActionConfig(true, "external-dns", release.StatusDeployed, createReleaseWithoutValues, vzlog.DefaultLogger())
}

// TestIsExternalDNSEnabled tests the IsEnabled fn
// GIVEN a call to IsEnabled
// WHEN OCI DNS is enabled
// THEN the function returns true
func TestIsExternalDNSEnabled(t *testing.T) {
	localvz := vz.DeepCopy()
	localvz.Spec.Components.DNS.OCI = &vzapi.OCI{}
	assert.True(t, fakeComponent.IsEnabled(spi.NewFakeContext(nil, localvz, nil, false).EffectiveCR()))
}

// TestIsExternalDNSDisabled tests the IsEnabled fn
// GIVEN a call to IsEnabled
// WHEN OCI DNS is disabled
// THEN the function returns true
func TestIsExternalDNSDisabled(t *testing.T) {
	assert.False(t, fakeComponent.IsEnabled(spi.NewFakeContext(nil, vz, nil, false).EffectiveCR()))
}

// TestIsExternalDNSReady tests the isExternalDNSReady fn
// GIVEN a call to isExternalDNSReady
// WHEN the external dns deployment is ready
// THEN the function returns true
func TestIsExternalDNSReady(t *testing.T) {
	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		newDeployment(ComponentName, true), newPod(ComponentName), newReplicaSet(ComponentName),
	).Build()
	externalDNS := NewComponent().(externalDNSComponent)
	assert.True(t, externalDNS.isExternalDNSReady(spi.NewFakeContext(client, nil, nil, false)))
}

// TestIsExternalDNSNotReady tests the isExternalDNSReady fn
// GIVEN a call to isExternalDNSReady
// WHEN the external dns deployment is not ready
// THEN the function returns false
func TestIsExternalDNSNotReady(t *testing.T) {
	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		newDeployment(ComponentName, false),
	).Build()
	externalDNS := NewComponent().(externalDNSComponent)
	assert.False(t, externalDNS.isExternalDNSReady(spi.NewFakeContext(client, nil, nil, false)))
}

// TestAppendExternalDNSOverrides tests the AppendOverrides fn
// GIVEN a call to AppendOverrides
// WHEN a VZ spec is passed with defaults
// THEN the values created properly
func TestAppendExternalDNSOverrides(t *testing.T) {
	localvz := vz.DeepCopy()
	localvz.Spec.Components.DNS.OCI = oci

	defer helm.SetDefaultActionConfigFunction()
	helm.SetActionConfigFunction(testActionConfigWithInstallationNoValues)

	kvs, err := AppendOverrides(spi.NewFakeContext(nil, localvz, nil, false, profileDir), ComponentName, ComponentNamespace, "", []bom.KeyValue{})
	assert.NoError(t, err)
	assert.Len(t, kvs, 9)
}

// TestExternalDNSPreInstallDryRun tests the PreInstall fn
// GIVEN a call to this fn
// WHEN I call PreInstall with dry-run = true
// THEN no errors are returned
func TestExternalDNSPreInstallDryRun(t *testing.T) {
	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
	err := fakeComponent.PreInstall(spi.NewFakeContext(client, &vzapi.Verrazzano{}, nil, true))
	assert.NoError(t, err)
}

// TestExternalDNSPreInstall tests the PreInstall fn
// GIVEN a call to this fn
// WHEN I call PreInstall
// THEN no errors are returned
func TestExternalDNSPreInstall(t *testing.T) {
	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "oci",
				Namespace: constants.VerrazzanoInstallNamespace,
			},
			Data: map[string][]byte{"oci.yaml": []byte("fake data")},
		}).Build()
	localvz := vz.DeepCopy()
	localvz.Spec.Components.DNS.OCI = oci
	err := fakeComponent.PreInstall(spi.NewFakeContext(client, localvz, nil, false))
	assert.NoError(t, err)
}

func TestExternalDNSPreInstallGlobalScope(t *testing.T) {
	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "oci",
				Namespace: constants.VerrazzanoInstallNamespace,
			},
			Data: map[string][]byte{"oci.yaml": []byte("fake data")},
		}).Build()
	localvz := vz.DeepCopy()
	localvz.Spec.Components.DNS.OCI = ociGlobalScope
	err := fakeComponent.PreInstall(spi.NewFakeContext(client, localvz, nil, false))
	assert.NoError(t, err)
}

func TestExternalDNSPreInstallPrivateScope(t *testing.T) {
	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "oci",
				Namespace: constants.VerrazzanoInstallNamespace,
			},
			Data: map[string][]byte{"oci.yaml": []byte("fake data")},
		}).Build()
	localvz := vz.DeepCopy()
	localvz.Spec.Components.DNS.OCI = ociPrivateScope
	err := fakeComponent.PreInstall(spi.NewFakeContext(client, localvz, nil, false))
	assert.NoError(t, err)
}

func TestExternalDNSPreInstall3InvalidScope(t *testing.T) {
	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "oci",
				Namespace: constants.VerrazzanoInstallNamespace,
			},
			Data: map[string][]byte{"oci.yaml": []byte("fake data")},
		}).Build()
	localvz := vz.DeepCopy()
	localvz.Spec.Components.DNS.OCI = ociInvalidScope
	err := fakeComponent.PreInstall(spi.NewFakeContext(client, localvz, nil, false))
	assert.Error(t, err)
}

// TestOwnerIDTextPrefix_HelmValueExists tests the getOrBuildIDs and getOrBuildTXTRecordPrefix functions
// GIVEN calls to getOrBuildIDs and getOrBuildTXTRecordPrefix
//
//	WHEN a valid helm release and namespace are deployed and the txtOwnerId and txtPrefix values exist in the release values
//	THEN the function returns the stored helm values and no error
func TestOwnerIDTextPrefix_HelmValueExists(t *testing.T) {
	defer helm.SetDefaultActionConfigFunction()
	helm.SetActionConfigFunction(testActionConfigWithInstallationAndValues)

	localvz := vz.DeepCopy()
	localvz.UID = "uid"
	localvz.Spec.Components.DNS.OCI = oci

	client := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(localvz).Build()
	compContext := spi.NewFakeContext(client, vz, nil, false)

	ids, err := getOrBuildIDs(compContext, ComponentName, ComponentNamespace)
	assert.NoError(t, err)
	assert.Len(t, ids, 2)

	assert.NoError(t, err)
	assert.Equal(t, "storedOwnerId", ids[0])
	assert.Equal(t, "storedPrefix", ids[1])
}

// TestOwnerIDTextPrefix_NoHelmValueExists tests the getOrBuildIDs and getOrBuildTXTRecordPrefix functions
// GIVEN calls to getOrBuildIDs and getOrBuildTXTRecordPrefix
//
//	WHEN no stored helm values exist
//	THEN the function returns the generated values and no error
func Test_getOrBuildOwnerID_NoHelmValueExists(t *testing.T) {
	defer helm.SetDefaultActionConfigFunction()
	helm.SetActionConfigFunction(testActionConfigWithInstallationNoValues)

	localvz := vz.DeepCopy()
	localvz.UID = "uid"
	localvz.Spec.Components.DNS.OCI = oci
	schemeGroupVersion := schema.GroupVersion{Group: "install.verrazzano.io", Version: "v1alpha1"}
	testScheme.AddKnownTypes(schemeGroupVersion, &vzapi.Verrazzano{})
	client := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(localvz).Build()
	compContext := spi.NewFakeContext(client, vz, nil, false)

	ids, err := getOrBuildIDs(compContext, ComponentName, ComponentNamespace)
	assert.NoError(t, err)
	assert.Len(t, ids, 2)

	assert.True(t, strings.HasPrefix(ids[0], "v8o-"))
	assert.NotContains(t, ids[0], vz.Spec.EnvironmentName)

	assert.NoError(t, err)
	assert.True(t, strings.HasPrefix(ids[1], "_"+ids[0]))
}

// Create a new deployment object for testing
func newDeployment(name string, updated bool) *appsv1.Deployment {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ComponentNamespace,
			Name:      name,
			Labels:    map[string]string{"app.kubernetes.io/instance": "external-dns"},
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app.kubernetes.io/instance": "external-dns"},
			},
		},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: 1,
			Replicas:          1,
			UpdatedReplicas:   1,
		},
	}

	if !updated {
		deployment.Status = appsv1.DeploymentStatus{
			AvailableReplicas: 1,
			Replicas:          1,
			UpdatedReplicas:   0,
		}
	}
	return deployment
}

func newPod(name string) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ComponentNamespace,
			Name:      name + "-95d8c5d96-m6mbr",
			Labels: map[string]string{
				"app.kubernetes.io/instance": "external-dns",
				"pod-template-hash":          "95d8c5d96",
			},
		},
	}
}

func newReplicaSet(name string) *appsv1.ReplicaSet {
	return &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   ComponentNamespace,
			Name:        name + "-95d8c5d96",
			Annotations: map[string]string{"deployment.kubernetes.io/revision": "1"},
		},
	}
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
						DNS: &vzapi.DNSComponent{
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
						DNS: &v1beta1.DNSComponent{
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
