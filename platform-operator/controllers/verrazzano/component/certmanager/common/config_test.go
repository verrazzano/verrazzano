// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	cmconstants "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager/constants"
	"testing"

	acmev1 "github.com/cert-manager/cert-manager/pkg/apis/acme/v1"
	certv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	cmcommonfake "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager/common/fake"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	v1 "k8s.io/api/core/v1"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	profileDir    = "../../../../../manifests/profiles"
	testNamespace = "testNamespace"
)

// default CA object
var defaultCA = vzapi.CA{
	SecretName:               "testSecret",
	ClusterResourceNamespace: testNamespace,
}

// Default Acme object
var acmeTestConfig = vzapi.Acme{
	Provider:     vzapi.LetsEncrypt,
	EmailAddress: "testEmail@foo.com",
	Environment:  cmconstants.LetsEncryptStaging,
}

// Default Verrazzano object
var defaultValidationConfig = &vzapi.Verrazzano{
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
	_ = acmev1.AddToScheme(testScheme)
	_ = vzapi.AddToScheme(testScheme)
	_ = apiextv1.AddToScheme(testScheme)
}

// TestIsCANilTrue tests the isCA function
// GIVEN a call to isCA
// WHEN the Certificate CA is populated
// THEN true is returned
func TestIsCATrue(t *testing.T) {
	localvz := defaultValidationConfig.DeepCopy()
	localvz.Spec.Components.CertManager.Certificate.CA = defaultCA

	defer func() { GetClientFunc = k8sutil.GetCoreV1Client }()
	GetClientFunc = createClientFunc(localvz.Spec.Components.CertManager.Certificate.CA, "defaultValidationConfig-cn")

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
	fakeCertBytes, err := cmcommonfake.CreateFakeCertBytes(cn, nil)
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
	localvz := defaultValidationConfig.DeepCopy()
	localvz.Spec.Components.CertManager.Certificate.Acme = acmeTestConfig
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
