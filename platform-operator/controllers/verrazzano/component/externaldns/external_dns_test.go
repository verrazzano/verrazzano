// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package externaldns

import (
	"github.com/verrazzano/verrazzano/pkg/helm"
	"k8s.io/apimachinery/pkg/runtime"
	"os/exec"
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
	k8scheme.AddToScheme(testScheme)
	vzapi.AddToScheme(testScheme)
}

// genericTestRunner is used to run generic OS commands with expected results
type genericTestRunner struct {
	stdOut []byte
	stdErr []byte
	err    error
}

// Run genericTestRunner executor
func (r genericTestRunner) Run(_ *exec.Cmd) (stdout []byte, stderr []byte, err error) {
	return r.stdOut, r.stdErr, r.err
}

// TestIsExternalDNSEnabled tests the IsEnabled fn
// GIVEN a call to IsEnabled
// WHEN OCI DNS is enabled
// THEN the function returns true
func TestIsExternalDNSEnabled(t *testing.T) {
	localvz := vz.DeepCopy()
	localvz.Spec.Components.DNS.OCI = &vzapi.OCI{}
	assert.True(t, fakeComponent.IsEnabled(spi.NewFakeContext(nil, localvz, false).EffectiveCR()))
}

// TestIsExternalDNSDisabled tests the IsEnabled fn
// GIVEN a call to IsEnabled
// WHEN OCI DNS is disabled
// THEN the function returns true
func TestIsExternalDNSDisabled(t *testing.T) {
	assert.False(t, fakeComponent.IsEnabled(spi.NewFakeContext(nil, vz, false).EffectiveCR()))
}

// TestIsExternalDNSReady tests the isExternalDNSReady fn
// GIVEN a call to isExternalDNSReady
// WHEN the external dns deployment is ready
// THEN the function returns true
func TestIsExternalDNSReady(t *testing.T) {
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme,
		newDeployment(ComponentName, true),
	)
	assert.True(t, isExternalDNSReady(spi.NewFakeContext(client, nil, false)))
}

// TestIsExternalDNSNotReady tests the isExternalDNSReady fn
// GIVEN a call to isExternalDNSReady
// WHEN the external dns deployment is not ready
// THEN the function returns false
func TestIsExternalDNSNotReady(t *testing.T) {
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme,
		newDeployment(ComponentName, false),
	)
	assert.False(t, isExternalDNSReady(spi.NewFakeContext(client, nil, false)))
}

// TestAppendExternalDNSOverrides tests the AppendOverrides fn
// GIVEN a call to AppendOverrides
// WHEN a VZ spec is passed with defaults
// THEN the values created properly
func TestAppendExternalDNSOverrides(t *testing.T) {
	localvz := vz.DeepCopy()
	localvz.Spec.Components.DNS.OCI = oci

	helm.SetCmdRunner(genericTestRunner{
		stdOut: []byte(""),
		stdErr: []byte{},
		err:    nil,
	})
	defer helm.SetDefaultRunner()

	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartNotFound, nil
	})
	defer helm.SetDefaultChartStatusFunction()

	kvs, err := AppendOverrides(spi.NewFakeContext(nil, localvz, false, profileDir), ComponentName, ComponentNamespace, "", []bom.KeyValue{})
	assert.NoError(t, err)
	assert.Len(t, kvs, 9)
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

// TestOwnerIDTextPrefix_HelmValueExists tests the getOrBuildIDs and getOrBuildTXTRecordPrefix functions
// GIVEN calls to getOrBuildIDs and getOrBuildTXTRecordPrefix
//  WHEN a valid helm release and namespace are deployed and the txtOwnerId and txtPrefix values exist in the release values
//  THEN the function returns the stored helm values and no error
func TestOwnerIDTextPrefix_HelmValueExists(t *testing.T) {
	jsonOut := []byte(`
{
  "domainFilters": [
    "my.domain.io"
  ],
  "triggerLoopOnEvent": true,
  "txtOwnerId": "storedOwnerId",
  "txtPrefix": "storedPrefix",
  "zoneIDFilters": [
    "ocid1.dns-zone.oc1..blahblahblah"
  ]
}
`)

	helm.SetCmdRunner(genericTestRunner{
		stdOut: jsonOut,
		stdErr: []byte{},
		err:    nil,
	})
	defer helm.SetDefaultRunner()

	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusDeployed, nil
	})
	defer helm.SetDefaultChartStatusFunction()

	localvz := vz.DeepCopy()
	localvz.UID = "uid"
	localvz.Spec.Components.DNS.OCI = oci

	client := fake.NewFakeClientWithScheme(testScheme, localvz)
	compContext := spi.NewFakeContext(client, vz, false)

	ids, err := getOrBuildIDs(compContext, ComponentName, ComponentNamespace)
	assert.NoError(t, err)
	assert.Len(t, ids, 2)

	assert.NoError(t, err)
	assert.Equal(t, "storedOwnerId", ids[0])
	assert.Equal(t, "storedPrefix", ids[1])
}

// TestOwnerIDTextPrefix_NoHelmValueExists tests the getOrBuildIDs and getOrBuildTXTRecordPrefix functions
// GIVEN calls to getOrBuildIDs and getOrBuildTXTRecordPrefix
//  WHEN no stored helm values exist
//  THEN the function returns the generated values and no error
func Test_getOrBuildOwnerID_NoHelmValueExists(t *testing.T) {
	helm.SetCmdRunner(genericTestRunner{
		stdOut: []byte(""),
		stdErr: []byte{},
		err:    nil,
	})
	defer helm.SetDefaultRunner()

	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartNotFound, nil
	})
	defer helm.SetDefaultChartStatusFunction()

	localvz := vz.DeepCopy()
	localvz.UID = "uid"
	localvz.Spec.Components.DNS.OCI = oci

	client := fake.NewFakeClientWithScheme(testScheme, localvz)
	compContext := spi.NewFakeContext(client, vz, false)

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
