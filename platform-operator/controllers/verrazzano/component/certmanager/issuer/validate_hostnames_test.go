// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package issuer

import (
	"fmt"

	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	cmcommonfake "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager/common/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

const (
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
	cr3.Spec.Components.CertManager = &vzapi.CertManagerComponent{Enabled: cmcommonfake.GetBoolPtr(false)}
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
	cr1.Spec.Components.DNS = &v1beta1.DNSComponent{OCI: ociLongDNSZoneNameV1Beta1}
	cr2.Spec.Components.DNS = &v1beta1.DNSComponent{OCI: ociV1Beta1}
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
		err := validateLongestHostName(&test.cr)
		if test.wantError {
			asserts.EqualError(err, test.want)
		} else {
			asserts.NoError(err)
		}
	}
}
