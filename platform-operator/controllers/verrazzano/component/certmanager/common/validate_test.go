// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"fmt"
	certv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

const (
	profileDir      = "../../../../../manifests/profiles"
	testNamespace   = "testNamespace"
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

var testScheme = runtime.NewScheme()

func init() {
	_ = k8scheme.AddToScheme(testScheme)
	_ = certv1.AddToScheme(testScheme)
	_ = vzapi.AddToScheme(testScheme)
}

// TestIsCANilTrue tests the isCA function
// GIVEN a call to isCA
// WHEN the Certificate CA is populated
// THEN true is returned
func TestIsCATrue(t *testing.T) {
	localvz := defaultVZConfig.DeepCopy()
	localvz.Spec.Components.CertManager.Certificate.CA = ca

	defer func() { GetClientFunc = k8sutil.GetCoreV1Client }()
	GetClientFunc = createClientFunc(localvz.Spec.Components.CertManager.Certificate.CA, "defaultVZConfig-cn")

	client := fake.NewClientBuilder().WithScheme(testScheme).Build()

	isCAValue, err := IsCA(spi.NewFakeContext(client, localvz, nil, false, profileDir))
	assert.Nil(t, err)
	assert.True(t, isCAValue)
}

func createClientFunc(caConfig vzapi.CA, cn string, otherObjs ...runtime.Object) GetCoreV1ClientFuncType {
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

func createCertSecretNoParent(name string, namespace string, cn string) (*v1.Secret, error) {
	fakeCertBytes, err := CreateFakeCertBytes(cn, nil)
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

// TestIsOCIDNS tests whether the Effective CR is using OCI DNS
// GIVEN a call to IsOCIDNS
// WHEN OCI DNS is specified in the Verrazzano Spec
// THEN IsOCIDNS should return true
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
			assert.Equal(t, tt.ocidns, IsOCIDNS(tt.vz))
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
	// Verify that we check the hostname length even if CM is disabled
	cr3.Spec.EnvironmentName = "veryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryverylong"
	cr3.Spec.Components.CertManager = &vzapi.CertManagerComponent{Enabled: GetBoolPtr(false)}
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
			want:      fmt.Sprintf("Failed: spec.environmentName %s and DNS suffix %s are too long. For the given configuration they must have at most %v characters in combination", cr1.Spec.EnvironmentName, cr1.Spec.Components.DNS.OCI.DNSZoneName, 64-preOccupiedspace),
		},
		{
			cr:        cr2,
			wantError: false,
		},
		{
			cr:        cr3,
			wantError: true,
			want:      fmt.Sprintf("Failed: spec.environmentName %s is too long. For the given configuration it must have at most %v characters", cr3.Spec.EnvironmentName, 64-(14+preOccupiedspace)),
		},
		{
			cr:        cr4,
			wantError: false,
		},
		{
			cr:        cr5,
			wantError: true,
			want:      fmt.Sprintf("Failed: spec.environmentName %s and DNS suffix %s are too long. For the given configuration they must have at most %v characters in combination", cr5.Spec.EnvironmentName, cr5.Spec.Components.DNS.External.Suffix, 64-preOccupiedspace),
		},
		{
			cr:        cr6,
			wantError: false,
		},
	}
	for _, test := range tests {
		err := ValidateLongestHostName(&test.cr)
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
			want:      fmt.Sprintf("Failed: spec.environmentName %s and DNS suffix %s are too long. For the given configuration they must have at most %v characters in combination", cr1.Spec.EnvironmentName, cr1.Spec.Components.DNS.OCI.DNSZoneName, 64-preOccupiedspace),
		},
		{
			cr:        cr2,
			wantError: false,
		},
		{
			cr:        cr3,
			wantError: true,
			want:      fmt.Sprintf("Failed: spec.environmentName %s is too long. For the given configuration it must have at most %v characters", cr3.Spec.EnvironmentName, 64-(14+preOccupiedspace)),
		},
		{
			cr:        cr4,
			wantError: false,
		},
		{
			cr:        cr5,
			wantError: true,
			want:      fmt.Sprintf("Failed: spec.environmentName %s and DNS suffix %s are too long. For the given configuration they must have at most %v characters in combination", cr5.Spec.EnvironmentName, cr5.Spec.Components.DNS.External.Suffix, 64-preOccupiedspace),
		},
		{
			cr:        cr6,
			wantError: false,
		},
	}
	for _, test := range tests {
		err := ValidateLongestHostName(&test.cr)
		if test.wantError {
			asserts.EqualError(err, test.want)
		} else {
			asserts.NoError(err)
		}
	}
}
