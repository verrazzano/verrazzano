// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certmanager

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"testing"
	"time"

	certv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	profileDir      = "../../../../manifests/profiles"
	testNamespace   = "testNamespace"
	testBomFile     = "../../testdata/test_bom.json"
	fooDomainSuffix = "foo.com"
)

const (
	// Make the code smells go away
	myvz             = "my-verrazzano"
	myvzns           = "default"
	zoneName         = "zone.name.io"
	ociDNSSecretName = "oci"
	zoneID           = "zoneID"
	compartmentID    = "compartmentID"
)

// Default Verrazzano object
var vz = &vzapi.Verrazzano{
	ObjectMeta: metav1.ObjectMeta{Name: myvz, Namespace: myvzns, CreationTimestamp: metav1.Now()},
	Spec: vzapi.VerrazzanoSpec{
		EnvironmentName: "myenv",
		Components: vzapi.ComponentSpec{
			DNS: &vzapi.DNSComponent{},
		},
	},
}

// Default Verrazzano v1beta1 object
var vzv1beta1 = &v1beta1.Verrazzano{
	ObjectMeta: metav1.ObjectMeta{Name: myvz, Namespace: myvzns, CreationTimestamp: metav1.Now()},
	Spec: v1beta1.VerrazzanoSpec{
		EnvironmentName: "myenv",
		Components: v1beta1.ComponentSpec{
			DNS: &v1beta1.DNSComponent{},
		},
	},
}

var oci = &vzapi.OCI{
	OCIConfigSecret:        ociDNSSecretName,
	DNSZoneCompartmentOCID: compartmentID,
	DNSZoneOCID:            zoneID,
	DNSZoneName:            zoneName,
}

var ociV1Beta1 = &v1beta1.OCI{
	OCIConfigSecret:        ociDNSSecretName,
	DNSZoneCompartmentOCID: compartmentID,
	DNSZoneOCID:            zoneID,
	DNSZoneName:            zoneName,
}

var ociLongDNSZoneName = &vzapi.OCI{
	OCIConfigSecret:        ociDNSSecretName,
	DNSZoneCompartmentOCID: compartmentID,
	DNSZoneOCID:            zoneID,
	DNSZoneName:            "veryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryverylong.name.io",
	DNSScope:               "#jhwuyusj!!!",
}

var ociLongDNSZoneNameV1Beta1 = &v1beta1.OCI{
	OCIConfigSecret:        ociDNSSecretName,
	DNSZoneCompartmentOCID: compartmentID,
	DNSZoneOCID:            zoneID,
	DNSZoneName:            "veryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryverylong.name.io",
	DNSScope:               "#jhwuyusj!!!",
}

// default CA object
var ca = vzapi.CA{
	SecretName:               "testSecret",
	ClusterResourceNamespace: testNamespace,
}

// Default Acme object
var acme = vzapi.Acme{
	Provider:     vzapi.LetsEncrypt,
	EmailAddress: "testEmail@foo.com",
	Environment:  letsEncryptStaging,
}

// Default Verrazzano object
var defaultVZConfig = &vzapi.Verrazzano{
	Spec: vzapi.VerrazzanoSpec{
		EnvironmentName: "myenv",
		Components: vzapi.ComponentSpec{
			CertManager: &vzapi.CertManagerComponent{
				Certificate: vzapi.Certificate{},
			},
		},
	},
}

// Fake certManagerComponent resource for function calls
var fakeComponent = certManagerComponent{}

var testScheme = runtime.NewScheme()

func init() {
	_ = k8scheme.AddToScheme(testScheme)
	_ = certv1.AddToScheme(testScheme)
	_ = vzapi.AddToScheme(testScheme)
}

// TestIsCertManagerEnabled tests the IsCertManagerEnabled fn
// GIVEN a call to IsCertManagerEnabled
// WHEN cert-manager is enabled
// THEN the function returns true
func TestIsCertManagerEnabled(t *testing.T) {
	localvz := defaultVZConfig.DeepCopy()
	localvz.Spec.Components.CertManager.Enabled = getBoolPtr(true)
	assert.True(t, fakeComponent.IsEnabled(spi.NewFakeContext(nil, localvz, nil, false).EffectiveCR()))
}

// TestIsOCIDNS tests whether the Effective CR is using OCI DNS
// GIVEN a call to isOCIDNS
// WHEN OCI DNS is specified in the Verrazzano Spec
// THEN isOCIDNS should return true
func TestIsOCIDNS(t *testing.T) {
	var tests = []struct {
		name   string
		vz     *vzapi.Verrazzano
		ocidns bool
	}{
		{
			"shouldn't be enabled when nil",
			&vzapi.Verrazzano{},
			false,
		},
		{
			"should be enabled when OCI DNS present",
			&vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						DNS: &vzapi.DNSComponent{
							OCI: &vzapi.OCI{},
						},
					},
				},
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.ocidns, isOCIDNS(tt.vz))
		})
	}
}

// TestWriteCRD tests writing out the OCI DNS metadata to CertManager CRDs
// GIVEN a call to writeCRD
// WHEN the input file exists
// THEN the output should have ocidns added.
func TestWriteCRD(t *testing.T) {
	inputFile := "../../../../thirdparty/manifests/cert-manager/cert-manager.crds.yaml"
	outputFile := "../../../../thirdparty/manifests/cert-manager/output.crd.yaml"
	err := writeCRD(inputFile, outputFile, true)
	assert.NoError(t, err)
}

// TestCleanTempFiles tests cleaning up temp files
// GIVEN a call to cleanTempFiles
// WHEN a file is not found
// THEN cleanTempFiles should return an error
func TestCleanTempFiles(t *testing.T) {
	assert.Error(t, cleanTempFiles("blahblah"))
}

// TestIsCertManagerDisabled tests the IsCertManagerEnabled fn
// GIVEN a call to IsCertManagerEnabled
// WHEN cert-manager is disabled
// THEN the function returns false
func TestIsCertManagerDisabled(t *testing.T) {
	localvz := defaultVZConfig.DeepCopy()
	localvz.Spec.Components.CertManager.Enabled = getBoolPtr(false)
	assert.False(t, fakeComponent.IsEnabled(spi.NewFakeContext(nil, localvz, nil, false).EffectiveCR()))
}

// TestAppendCertManagerOverrides tests the AppendOverrides fn
// GIVEN a call to AppendOverrides
// WHEN a VZ spec is passed with defaults
// THEN the values created properly
func TestAppendCertManagerOverrides(t *testing.T) {
	config.SetDefaultBomFilePath(testBomFile)
	kvs, err := AppendOverrides(spi.NewFakeContext(nil, &vzapi.Verrazzano{}, nil, false, profileDir), ComponentName, ComponentNamespace, "", []bom.KeyValue{})
	assert.NoError(t, err)
	assert.Len(t, kvs, 1)
}

// TestAppendCertManagerOverridesWithInstallArgs tests the AppendOverrides fn
// GIVEN a call to AppendOverrides
// WHEN a VZ spec is passed with install args
// THEN the values created properly
func TestAppendCertManagerOverridesWithInstallArgs(t *testing.T) {
	localvz := defaultVZConfig.DeepCopy()
	localvz.Spec.Components.CertManager.Certificate.CA = ca
	defer func() { getClientFunc = k8sutil.GetCoreV1Client }()
	getClientFunc = createClientFunc(localvz.Spec.Components.CertManager.Certificate.CA, "defaultVZConfig-cn")
	kvs, err := AppendOverrides(spi.NewFakeContext(nil, localvz, nil, false), ComponentName, ComponentNamespace, "", []bom.KeyValue{})
	assert.NoError(t, err)
	assert.Len(t, kvs, 1)
	assert.Contains(t, kvs, bom.KeyValue{Key: clusterResourceNamespaceKey, Value: testNamespace})
}

// TestCertManagerPreInstall tests the PreInstall fn
// GIVEN a call to this fn
// WHEN I call PreInstall with dry-run = true
// THEN no errors are returned
func TestCertManagerPreInstallDryRun(t *testing.T) {
	client := fake.NewClientBuilder().WithScheme(testScheme).Build()
	err := fakeComponent.PreInstall(spi.NewFakeContext(client, &vzapi.Verrazzano{}, nil, true))
	assert.NoError(t, err)
}

// TestCertManagerPreInstall tests the PreInstall fn
// GIVEN a call to this fn
// WHEN I call PreInstall
// THEN no errors are returned
func TestCertManagerPreInstall(t *testing.T) {
	config.Set(config.OperatorConfig{
		VerrazzanoRootDir: "../../../../..", //since we are running inside the cert manager package, root is up 5 directories
	})
	client := fake.NewClientBuilder().WithScheme(testScheme).Build()
	err := fakeComponent.PreInstall(spi.NewFakeContext(client, &vzapi.Verrazzano{}, nil, false))
	assert.NoError(t, err)

	secret := &v1.Secret{}
	err = client.Get(context.TODO(), types.NamespacedName{Name: ociDNSSecretName, Namespace: ComponentNamespace}, secret)
	assert.Error(t, err)
	assert.True(t, errors.IsNotFound(err))
}

// TestCertManagerPreInstallOCIDNS tests the PreInstall fn
// GIVEN a call to this fn
// WHEN I call PreInstall and OCI DNS is enabled
// THEN no errors are returned and the DNS secret is set up
func TestCertManagerPreInstallOCIDNS(t *testing.T) {
	config.Set(config.OperatorConfig{
		VerrazzanoRootDir: "../../../../..", //since we are running inside the cert manager package, root is up 5 directories
	})
	client := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
		&v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ociDNSSecretName,
				Namespace: constants.VerrazzanoInstallNamespace,
			},
			Data: map[string][]byte{"oci.yaml": []byte("fake data")},
		}).Build()
	vz := &vzapi.Verrazzano{
		ObjectMeta: metav1.ObjectMeta{Name: myvz, Namespace: myvzns, CreationTimestamp: metav1.Now()},
		Spec: vzapi.VerrazzanoSpec{
			EnvironmentName: "myenv",
			Components: vzapi.ComponentSpec{
				DNS: &vzapi.DNSComponent{
					OCI: &vzapi.OCI{
						OCIConfigSecret:        ociDNSSecretName,
						DNSZoneCompartmentOCID: compartmentID,
						DNSZoneOCID:            zoneID,
						DNSZoneName:            zoneName,
					},
				},
			},
		},
	}
	err := fakeComponent.PreInstall(spi.NewFakeContext(client, vz, nil, false))
	assert.NoError(t, err)
}

// TestIsCertManagerReady tests the isCertManagerReady function
// GIVEN a call to isCertManagerReady
// WHEN the deployment object has enough replicas available
// THEN true is returned
func TestIsCertManagerReady(t *testing.T) {
	client := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
		newDeployment(certManagerDeploymentName, map[string]string{"app": certManagerDeploymentName}, true),
		newPod(certManagerDeploymentName, map[string]string{"app": certManagerDeploymentName}),
		newReplicaSet(certManagerDeploymentName),
		newDeployment(cainjectorDeploymentName, map[string]string{"app": "cainjector"}, true),
		newPod(cainjectorDeploymentName, map[string]string{"app": "cainjector"}),
		newReplicaSet(cainjectorDeploymentName),
		newDeployment(webhookDeploymentName, map[string]string{"app": "webhook"}, true),
		newPod(webhookDeploymentName, map[string]string{"app": "webhook"}),
		newReplicaSet(webhookDeploymentName),
	).Build()
	certManager := NewComponent().(certManagerComponent)
	assert.True(t, certManager.isCertManagerReady(spi.NewFakeContext(client, nil, nil, false)))
}

// TestIsCertManagerNotReady tests the isCertManagerReady function
// GIVEN a call to isCertManagerReady
// WHEN the deployment object does not have enough replicas available
// THEN false is returned
func TestIsCertManagerNotReady(t *testing.T) {
	client := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
		newDeployment(certManagerDeploymentName, map[string]string{"app": certManagerDeploymentName}, false),
		newDeployment(cainjectorDeploymentName, map[string]string{"app": "cainjector"}, false),
		newDeployment(webhookDeploymentName, map[string]string{"app": "webhook"}, false),
	).Build()
	certManager := NewComponent().(certManagerComponent)
	assert.False(t, certManager.isCertManagerReady(spi.NewFakeContext(client, nil, nil, false)))
}

// TestIsCANilWithProfile tests the isCA function
// GIVEN a call to isCA
// WHEN the CertManager component is populated by the profile
// THEN true is returned
func TestIsCANilWithProfile(t *testing.T) {
	client := fake.NewClientBuilder().WithScheme(testScheme).Build()
	isCAValue, err := IsCA(spi.NewFakeContext(client, &vzapi.Verrazzano{}, nil, false, profileDir))
	assert.Nil(t, err)
	assert.True(t, isCAValue)
}

// TestIsCANilTrue tests the isCA function
// GIVEN a call to isCA
// WHEN the Certificate CA is populated
// THEN true is returned
func TestIsCATrue(t *testing.T) {
	localvz := defaultVZConfig.DeepCopy()
	localvz.Spec.Components.CertManager.Certificate.CA = ca

	defer func() { getClientFunc = k8sutil.GetCoreV1Client }()
	getClientFunc = createClientFunc(localvz.Spec.Components.CertManager.Certificate.CA, "defaultVZConfig-cn")

	client := fake.NewClientBuilder().WithScheme(testScheme).Build()

	isCAValue, err := IsCA(spi.NewFakeContext(client, localvz, nil, false, profileDir))
	assert.Nil(t, err)
	assert.True(t, isCAValue)
}

func createClientFunc(caConfig vzapi.CA, cn string, otherObjs ...runtime.Object) getCoreV1ClientFuncType {
	return func(...vzlog.VerrazzanoLogger) (corev1.CoreV1Interface, error) {
		secret, err := createCertSecretNoParent(caConfig.SecretName, caConfig.ClusterResourceNamespace, cn)
		if err != nil {
			return nil, err
		}
		objs := []runtime.Object{secret}
		objs = append(objs, otherObjs...)
		return k8sfake.NewSimpleClientset(objs...).CoreV1(), nil
	}
}

// TestIsCANilFalse tests the isCA function
// GIVEN a call to isCA
// WHEN the Certificate Acme is populated
// THEN false is returned
func TestIsCAFalse(t *testing.T) {
	localvz := defaultVZConfig.DeepCopy()
	localvz.Spec.Components.CertManager.Certificate.Acme = acme
	client := fake.NewClientBuilder().WithScheme(testScheme).Build()
	isCAValue, err := IsCA(spi.NewFakeContext(client, localvz, nil, false, profileDir))
	assert.Nil(t, err)
	assert.False(t, isCAValue)
}

// TestIsCANilFalse tests the isCA function
// GIVEN a call to isCA
// WHEN the Certificate Acme is populated
// THEN false is returned
func TestIsCABothPopulated(t *testing.T) {
	localvz := defaultVZConfig.DeepCopy()
	localvz.Spec.Components.CertManager.Certificate.CA = ca
	localvz.Spec.Components.CertManager.Certificate.Acme = acme
	client := fake.NewClientBuilder().WithScheme(testScheme).Build()
	_, err := IsCA(spi.NewFakeContext(client, localvz, nil, false, profileDir))
	assert.Error(t, err)
}

// TestUninstallCertManager tests the cert-manager uninstall process
// GIVEN a call to uninstallCertManager
// WHEN the objects exist in the cluster
// THEN no error is returned and all objects are deleted
func TestUninstallCertManager(t *testing.T) {
	vz := defaultVZConfig.DeepCopy()

	controllerCM := v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: constants.KubeSystem,
			Name:      controllerConfigMap,
		},
	}
	caCM := v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: constants.KubeSystem,
			Name:      caInjectorConfigMap,
		},
	}
	certNS := v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: constants.CertManagerNamespace,
		},
	}

	tests := []struct {
		name    string
		objects []clipkg.Object
	}{
		{
			name: "test empty cluster",
		},
		{
			name: "test controller configmap",
			objects: []clipkg.Object{
				&controllerCM,
			},
		},
		{
			name: "test ca configmap",
			objects: []clipkg.Object{
				&controllerCM,
				&caCM,
			},
		},
		{
			name: "test namespace",
			objects: []clipkg.Object{
				&controllerCM,
				&caCM,
				&certNS,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(tt.objects...).Build()
			fakeContext := spi.NewFakeContext(c, vz, nil, false, profileDir)
			err := uninstallCertManager(fakeContext)
			assert.NoError(t, err)
			// expect the controller ConfigMap to get deleted
			err = c.Get(context.TODO(), types.NamespacedName{Name: controllerConfigMap, Namespace: constants.KubeSystem}, &v1.ConfigMap{})
			assert.Error(t, err, "Expected the ConfigMap %s to be deleted", controllerConfigMap)
			// expect the CA injector ConfigMap to get deleted
			err = c.Get(context.TODO(), types.NamespacedName{Name: caInjectorConfigMap, Namespace: constants.KubeSystem}, &v1.ConfigMap{})
			assert.Error(t, err, "Expected the ConfigMap %s to be deleted", caInjectorConfigMap)
		})
	}
}

// TestGetOverrides tests the GetOverrides function in various permutations
// GIVEN a call to GetOverrides
//
//	THEN the overrides are merged and returned correctly
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
						CertManager: &vzapi.CertManagerComponent{
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
						CertManager: &v1beta1.CertManagerComponent{
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

// TestValidateLongestHostName tests the following scenarios
// GIVEN a call to validateLongestHostName func
// WHEN the CR passed is v1alpha1
// THEN it is inspected to validate the host name length of endpoints
func TestValidateLongestHostName(t *testing.T) {
	asserts := assert.New(t)
	cr1, cr2, cr3, cr4, cr5, cr6 := *vz.DeepCopy(), *vz.DeepCopy(), *vz.DeepCopy(), *vz.DeepCopy(), *vz.DeepCopy(), *vz.DeepCopy()
	cr1.Spec.Components.DNS.OCI = ociLongDNSZoneName
	cr2.Spec.Components.DNS.OCI = oci
	cr3.Spec.EnvironmentName = "veryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryverylong"
	cr3.Spec.Components.DNS = nil
	cr4.Spec.Components.DNS = nil
	cr5.Spec.Components.DNS = &vzapi.DNSComponent{External: &vzapi.External{Suffix: ociLongDNSZoneNameV1Beta1.DNSZoneName}}
	cr6.Spec.Components.DNS = &vzapi.DNSComponent{External: &vzapi.External{Suffix: fooDomainSuffix}}
	tests := []struct {
		cr        vzapi.Verrazzano
		wantError bool
		want      string
	}{
		{
			cr:        cr1,
			wantError: true,
			want:      fmt.Sprintf("spec.environmentName %s and DNS suffix %s are too long. For the given configuration they must have at most %v characters in combination", cr1.Spec.EnvironmentName, cr1.Spec.Components.DNS.OCI.DNSZoneName, 64-preOccupiedspace),
		},
		{
			cr:        cr2,
			wantError: false,
		},
		{
			cr:        cr3,
			wantError: true,
			want:      fmt.Sprintf("spec.environmentName %s is too long. For the given configuration it must have at most %v characters", cr3.Spec.EnvironmentName, 64-(14+preOccupiedspace)),
		},
		{
			cr:        cr4,
			wantError: false,
		},
		{
			cr:        cr5,
			wantError: true,
			want:      fmt.Sprintf("spec.environmentName %s and DNS suffix %s are too long. For the given configuration they must have at most %v characters in combination", cr5.Spec.EnvironmentName, cr5.Spec.Components.DNS.External.Suffix, 64-preOccupiedspace),
		},
		{
			cr:        cr6,
			wantError: false,
		},
	}
	for _, test := range tests {
		err := validateLongestHostName(&test.cr)
		if test.wantError {
			asserts.EqualError(err, test.want)
		} else {
			asserts.NoError(err)
		}
	}
}

// TestValidateLongestHostNameV1Beta1 tests the following scenarios
// GIVEN a call to validateLongestHostName func
// WHEN the CR passed is v1beta1
// THEN it is inspected to validate the host name length of endpoints
func TestValidateLongestHostNameV1Beta1(t *testing.T) {
	asserts := assert.New(t)
	cr1, cr2, cr3, cr4, cr5, cr6 := *vzv1beta1.DeepCopy(), *vzv1beta1.DeepCopy(), *vzv1beta1.DeepCopy(), *vzv1beta1.DeepCopy(), *vzv1beta1.DeepCopy(), *vzv1beta1.DeepCopy()
	cr1.Spec.Components.DNS.OCI = ociLongDNSZoneNameV1Beta1
	cr2.Spec.Components.DNS.OCI = ociV1Beta1
	cr3.Spec.EnvironmentName = "veryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryverylong"
	cr3.Spec.Components.DNS = nil
	cr4.Spec.Components.DNS = nil
	cr5.Spec.Components.DNS = &v1beta1.DNSComponent{External: &v1beta1.External{Suffix: ociLongDNSZoneNameV1Beta1.DNSZoneName}}
	cr6.Spec.Components.DNS = &v1beta1.DNSComponent{External: &v1beta1.External{Suffix: fooDomainSuffix}}
	tests := []struct {
		cr        v1beta1.Verrazzano
		wantError bool
		want      string
	}{
		{
			cr:        cr1,
			wantError: true,
			want:      fmt.Sprintf("spec.environmentName %s and DNS suffix %s are too long. For the given configuration they must have at most %v characters in combination", cr1.Spec.EnvironmentName, cr1.Spec.Components.DNS.OCI.DNSZoneName, 64-preOccupiedspace),
		},
		{
			cr:        cr2,
			wantError: false,
		},
		{
			cr:        cr3,
			wantError: true,
			want:      fmt.Sprintf("spec.environmentName %s is too long. For the given configuration it must have at most %v characters", cr3.Spec.EnvironmentName, 64-(14+preOccupiedspace)),
		},
		{
			cr:        cr4,
			wantError: false,
		},
		{
			cr:        cr5,
			wantError: true,
			want:      fmt.Sprintf("spec.environmentName %s and DNS suffix %s are too long. For the given configuration they must have at most %v characters in combination", cr5.Spec.EnvironmentName, cr5.Spec.Components.DNS.External.Suffix, 64-preOccupiedspace),
		},
		{
			cr:        cr6,
			wantError: false,
		},
	}
	for _, test := range tests {
		err := validateLongestHostName(&test.cr)
		if test.wantError {
			asserts.EqualError(err, test.want)
		} else {
			asserts.NoError(err)
		}
	}
}

// Create a new deployment object for testing
func newDeployment(name string, labels map[string]string, updated bool) *appsv1.Deployment {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ComponentNamespace,
			Name:      name,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
		},
		Status: appsv1.DeploymentStatus{
			Replicas:          1,
			AvailableReplicas: 1,
			UpdatedReplicas:   1,
		},
	}

	if !updated {
		deployment.Status = appsv1.DeploymentStatus{
			Replicas:          1,
			AvailableReplicas: 1,
			UpdatedReplicas:   0,
		}
	}
	return deployment
}

func newPod(name string, labelsIn map[string]string) *v1.Pod {
	labels := make(map[string]string)
	labels["pod-template-hash"] = "95d8c5d96"
	for key, element := range labelsIn {
		labels[key] = element
	}
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ComponentNamespace,
			Name:      name + "-95d8c5d96-m6mbr",
			Labels:    labels,
		},
	}
	return pod
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

func createCertSecretNoParent(name string, namespace string, cn string) (*v1.Secret, error) {
	fakeCertBytes, err := createFakeCertBytes(cn, nil)
	if err != nil {
		return nil, err
	}
	secret := &v1.Secret{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			v1.TLSCertKey: fakeCertBytes,
		},
		Type: v1.SecretTypeTLS,
	}
	return secret, nil
}

var testRSAKey *rsa.PrivateKey

func getRSAKey() (*rsa.PrivateKey, error) {
	if testRSAKey == nil {
		var err error
		if testRSAKey, err = rsa.GenerateKey(rand.Reader, 2048); err != nil {
			return nil, err
		}
	}
	return testRSAKey, nil
}

func createFakeCertBytes(cn string, parent *x509.Certificate) ([]byte, error) {
	rsaKey, err := getRSAKey()
	if err != nil {
		return []byte{}, err
	}

	cert := createFakeCertificate(cn)
	if parent == nil {
		parent = cert
	}
	pubKey := &rsaKey.PublicKey
	certBytes, err := x509.CreateCertificate(rand.Reader, cert, parent, pubKey, rsaKey)
	if err != nil {
		return []byte{}, err
	}
	certPem := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})
	return certPem, nil
}

func createFakeCertificate(cn string) *x509.Certificate {
	duration30, _ := time.ParseDuration("-30h")
	notBefore := time.Now().Add(duration30) // valid 30 hours ago
	duration1Year, _ := time.ParseDuration("90h")
	notAfter := notBefore.Add(duration1Year) // for 90 hours
	serialNo := big.NewInt(int64(123123413123))
	cert := &x509.Certificate{
		Subject: pkix.Name{
			Country:      []string{"US"},
			Organization: []string{"BarOrg"},
			SerialNumber: "2234",
			CommonName:   cn,
		},
		SerialNumber:          serialNo,
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  false,
		SignatureAlgorithm:    x509.SHA256WithRSA,
	}

	cert.IPAddresses = append(cert.IPAddresses, net.ParseIP("127.0.0.1"))
	cert.IPAddresses = append(cert.IPAddresses, net.ParseIP("::"))
	cert.DNSNames = append(cert.DNSNames, "localhost")
	return cert
}

// Create a bool pointer
func getBoolPtr(b bool) *bool {
	return &b
}
