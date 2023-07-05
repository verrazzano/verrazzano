// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certmanager

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	cmcommon "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

var (
	mockNamespaceCoreV1Client = common.MockGetCoreV1(&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
		Name:   ComponentName,
		Labels: map[string]string{constants.VerrazzanoManagedKey: ComponentNamespace},
	}})
	mockNamespaceWithoutLabelClient = common.MockGetCoreV1(&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
		Name: ComponentName,
	}})
)

// TestValidateUpdate tests the ValidateUpdate function
// GIVEN a call to ValidateUpdate
//
//	WHEN for various CM configurations
//	THEN an error is returned if anything is misconfigured
func TestValidateUpdate(t *testing.T) {
	validationTests(t, true)
}

// TestValidateInstall tests the ValidateInstall function
// GIVEN a call to ValidateInstall
//
//	WHEN for various CM configurations
//	THEN an error is returned if anything is misconfigured
func TestValidateInstall(t *testing.T) {
	validationTests(t, false)
}

func createFakeClient(objs ...runtime.Object) *k8sfake.Clientset {
	return k8sfake.NewSimpleClientset(objs...)
}

// All of this below is to make Sonar happy
type validationTestStruct struct {
	name      string
	old       *vzapi.Verrazzano
	new       *vzapi.Verrazzano
	coreV1Cli func(_ ...vzlog.VerrazzanoLogger) (v1.CoreV1Interface, error)
	caSecret  *corev1.Secret
	wantErr   bool
}

var disabled = false

const secretName = "newsecret"
const secretNamespace = "ns"

var tests = []validationTestStruct{
	{
		name: "enable",
		old: &vzapi.Verrazzano{
			Spec: vzapi.VerrazzanoSpec{
				Components: vzapi.ComponentSpec{
					CertManager: &vzapi.CertManagerComponent{
						Enabled: &disabled,
					},
				},
			},
		},
		new:       &vzapi.Verrazzano{},
		coreV1Cli: mockNamespaceCoreV1Client,
		wantErr:   false,
	},
	{
		name: "Cert Manager Namespace already exists",
		old: &vzapi.Verrazzano{
			Spec: vzapi.VerrazzanoSpec{
				Components: vzapi.ComponentSpec{
					CertManager: &vzapi.CertManagerComponent{
						Enabled: &disabled,
					},
				},
			},
		},
		new:       &vzapi.Verrazzano{},
		coreV1Cli: mockNamespaceWithoutLabelClient,
		wantErr:   true,
	},
	{
		name: "disable",
		old:  &vzapi.Verrazzano{},
		new: &vzapi.Verrazzano{
			Spec: vzapi.VerrazzanoSpec{
				Components: vzapi.ComponentSpec{
					CertManager: &vzapi.CertManagerComponent{
						Enabled: &disabled,
					},
				},
			},
		},
		coreV1Cli: mockNamespaceCoreV1Client,
		wantErr:   true,
	},
	{
		name:    "no change",
		old:     &vzapi.Verrazzano{},
		new:     &vzapi.Verrazzano{},
		wantErr: false,
	},
	{
		name: "singleOverride",
		new:  getSingleOverrideCR(),
		caSecret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: secretNamespace},
		},
		wantErr: false,
	},
	{
		name: "multipleOverridesInOneListValue",
		new:  getMultipleOverrideCR(),
		caSecret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: secretNamespace},
		},
		wantErr: true,
	},
}

func validationTests(t *testing.T, isUpdate bool) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "Cert Manager Namespace already exists" && isUpdate { // will throw error only during installation
				tt.wantErr = false
			}
			c := NewComponent()
			cmcommon.GetClientFunc = getTestClient(tt)
			runValidationTest(t, tt, isUpdate, c)
		})
	}
}

func runValidationTest(t *testing.T, tt validationTestStruct, isUpdate bool, c spi.Component) {
	defer func() { cmcommon.ResetCoreV1ClientFunc() }()
	if isUpdate {
		if err := c.ValidateUpdate(tt.old, tt.new); (err != nil) != tt.wantErr {
			t.Errorf("ValidateUpdate() error = %v, wantErr %v", err, tt.wantErr)
		}
		v1beta1New := &v1beta1.Verrazzano{}
		v1beta1Old := &v1beta1.Verrazzano{}
		err := tt.new.ConvertTo(v1beta1New)
		assert.NoError(t, err)
		err = tt.old.ConvertTo(v1beta1Old)
		assert.NoError(t, err)
		if err := c.ValidateUpdateV1Beta1(v1beta1Old, v1beta1New); (err != nil) != tt.wantErr {
			t.Errorf("ValidateUpdateV1Beta1() error = %v, wantErr %v", err, tt.wantErr)
		}

	} else {
		wantErr := tt.name != "disable" && tt.wantErr // hack for disable validation, allowed on initial install but not on update
		if tt.coreV1Cli != nil {
			k8sutil.GetCoreV1Func = tt.coreV1Cli
		} else {
			k8sutil.GetCoreV1Func = common.MockGetCoreV1()
		}
		if err := c.ValidateInstall(tt.new); (err != nil) != wantErr {
			t.Errorf("ValidateInstall() error = %v, wantErr %v", err, tt.wantErr)
		}
		v1beta1Vz := &v1beta1.Verrazzano{}
		err := tt.new.ConvertTo(v1beta1Vz)
		assert.NoError(t, err)
		if err := c.ValidateInstallV1Beta1(v1beta1Vz); (err != nil) != wantErr {
			t.Errorf("ValidateInstallV1Beta1() error = %v, wantErr %v", err, tt.wantErr)
		}
	}
}

func getTestClient(tt validationTestStruct) func(log ...vzlog.VerrazzanoLogger) (v1.CoreV1Interface, error) {
	return func(log ...vzlog.VerrazzanoLogger) (v1.CoreV1Interface, error) {
		if tt.caSecret != nil {
			return createFakeClient(tt.caSecret).CoreV1(), nil
		}
		return createFakeClient().CoreV1(), nil
	}
}

func getMultipleOverrideCR() *vzapi.Verrazzano {
	return &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				CertManager: &vzapi.CertManagerComponent{
					Certificate: vzapi.Certificate{
						CA: vzapi.CA{
							SecretName:               secretName,
							ClusterResourceNamespace: secretNamespace,
						},
					},
					InstallOverrides: vzapi.InstallOverrides{
						ValueOverrides: []vzapi.Overrides{
							{
								Values: &apiextensionsv1.JSON{
									Raw: []byte("certManagerCROverride"),
								},
								ConfigMapRef: &corev1.ConfigMapKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "overrideConfigMapSecretName",
									},
									Key: "Key",
								},
							},
						},
					},
				},
			},
		},
	}
}

func getSingleOverrideCR() *vzapi.Verrazzano {
	return &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				CertManager: &vzapi.CertManagerComponent{
					Certificate: vzapi.Certificate{
						CA: vzapi.CA{
							SecretName:               secretName,
							ClusterResourceNamespace: secretNamespace,
						},
					},
					InstallOverrides: vzapi.InstallOverrides{
						ValueOverrides: []vzapi.Overrides{
							{
								Values: &apiextensionsv1.JSON{
									Raw: []byte("certManagerCROverride"),
								},
							},
						},
					},
				},
			},
		},
	}
}
