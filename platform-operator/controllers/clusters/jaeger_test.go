// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusters

import (
	"github.com/stretchr/testify/assert"
	globalconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/mcconstants"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	corev1 "k8s.io/api/core/v1"
	k8net "k8s.io/api/networking/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

var (
	falseValue = false
	trueValue  = true
)

var keycloakDisabledCR = &vzapi.Verrazzano{
	Spec: vzapi.VerrazzanoSpec{
		Components: vzapi.ComponentSpec{
			Keycloak: &vzapi.KeycloakComponent{
				Enabled: &falseValue,
			},
			JaegerOperator: &vzapi.JaegerOperatorComponent{
				Enabled: &trueValue,
			},
		},
	},
}

var openSearchDisabledCR = &vzapi.Verrazzano{
	Spec: vzapi.VerrazzanoSpec{
		Components: vzapi.ComponentSpec{
			Elasticsearch: &vzapi.ElasticsearchComponent{
				Enabled: &falseValue,
			},
			JaegerOperator: &vzapi.JaegerOperatorComponent{
				Enabled: &trueValue,
			},
		},
	},
}

var jaegerEnabledCR = &vzapi.Verrazzano{
	Spec: vzapi.VerrazzanoSpec{
		Components: vzapi.ComponentSpec{
			JaegerOperator: &vzapi.JaegerOperatorComponent{
				Enabled: &trueValue,
			},
		},
	},
}

var defaultJaegerSecret = &corev1.Secret{
	ObjectMeta: v12.ObjectMeta{Name: globalconst.DefaultJaegerSecretName,
		Namespace: constants.VerrazzanoMonitoringNamespace},
	Data: map[string][]byte{
		mcconstants.JaegerOSUsernameKey: []byte("username"),
		mcconstants.JaegerOSPasswordKey: []byte("password"),
	},
}

var jaegerExternalOSSecret = &corev1.Secret{
	ObjectMeta: v12.ObjectMeta{Name: customJaegerSecName,
		Namespace: constants.VerrazzanoMonitoringNamespace},
	Data: map[string][]byte{
		mcconstants.JaegerOSUsernameKey: []byte("externalusername"),
		mcconstants.JaegerOSPasswordKey: []byte("externalpassword"),
		"ca-bundle":                     []byte("jaegeropensearchtlscakey"),
		"os.tls.key":                    []byte("jaegeropensearchtlskey"),
		"os.tls.cert":                   []byte("jaegeropensearchtlscert"),
	},
}

var vzTLSSecret = &corev1.Secret{
	ObjectMeta: v12.ObjectMeta{Name: "verrazzano-tls",
		Namespace: constants.VerrazzanoSystemNamespace},
	Data: map[string][]byte{
		mcconstants.CaCrtKey: []byte("adminCAbundle"),
	},
}

const (
	externalOSURL        = "https://external-os"
	customJaegerSecName  = "custom-jaeger-secret"
	jaegerDisabledJSON   = "{\"jaeger\": {\"create\": false}}"
	jaegerExternalOSJSON = `
{
	"jaeger": {
		"create": true,
		"spec": {
			"strategy": "production",
			"storage": {
				"type": "elasticsearch",
				"options": {
					"es": {
						"server-urls": "` + externalOSURL + `",
						"index-prefix": "jaeger",
						"tls": {
							"ca": "/verrazzano/certificates/ca-bundle"
						}
					}
				},
				"secretName": "` + customJaegerSecName + `"
			}
		}
	}
}`
	jaegerExternalOSMutualTLSJSON = `
{
	"jaeger": {
		"create": true,
		"spec": {
			"strategy": "production",
			"storage": {
				"type": "elasticsearch",
				"options": {
					"es": {
						"server-urls": "` + externalOSURL + `",
						"index-prefix": "jaeger",
						"tls": {
							"ca": "/verrazzano/certificates/ca-bundle",
							"key": "/verrazzano/certificates/os.tls.key",
							"cert": "/verrazzano/certificates/os.tls.cert"
						}
					}
				},
				"secretName": "` + customJaegerSecName + `"
			}
		}
	}
}`
	jaegerStorageTypeOverrideJSON = `
{
	"jaeger": {
		"create": true,
		"spec": {
			"strategy": "production",
			"storage": {
				"type": "inmemory",
				"options": {
					"es": {
						"server-urls": "` + externalOSURL + `",
						"index-prefix": "jaeger",
						"tls": {
							"ca": "/verrazzano/certificates/ca-bundle"
						}
					}
				},
				"secretName": "` + customJaegerSecName + `"
			}
		}
	}
}`
)

func TestGetJaegerOpenSearchConfig(t *testing.T) {
	type fields struct {
		vz            *vzapi.Verrazzano
		jaegerCreate  bool
		externalOS    bool
		mutualTLS     bool
		OSStorageType bool
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{"Default Jaeger instance", fields{vz: jaegerEnabledCR, jaegerCreate: true, OSStorageType: true}},
		{"Jaeger instance with external OpenSearch", fields{vz: createVZJaegerOverride(jaegerExternalOSJSON),
			jaegerCreate: true, externalOS: true, OSStorageType: true}},
		{"Jaeger instance with external OpenSearch mutual TLS", fields{vz: createVZJaegerOverride(jaegerExternalOSMutualTLSJSON),
			jaegerCreate: true, externalOS: true, mutualTLS: true, OSStorageType: true}},
		{"Jaeger instance creation disabled", fields{vz: createVZJaegerOverride(jaegerDisabledJSON), jaegerCreate: false,
			OSStorageType: true}},
		{"Jaeger instance with different storage type", fields{vz: createVZJaegerOverride(jaegerStorageTypeOverrideJSON),
			jaegerCreate: false, OSStorageType: false}},
		{"Keycloak disabled", fields{vz: keycloakDisabledCR, jaegerCreate: false}},
		{"OpenSearch disabled", fields{vz: openSearchDisabledCR, jaegerCreate: false}},
		{"Jaeger instance with default OpenSearch disabled and external OpenSearch configured",
			fields{vz: createVZWithOSDisabledAndJaegerOverride(jaegerExternalOSJSON),
				jaegerCreate: true, externalOS: true, OSStorageType: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			_ = corev1.AddToScheme(scheme)
			_ = k8net.AddToScheme(scheme)
			c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(defaultJaegerSecret, jaegerExternalOSSecret,
				vzTLSSecret, newIngress()).Build()
			rc := &VerrazzanoManagedClusterReconciler{
				Client: c,
				log:    vzlog.DefaultLogger(),
			}
			vzList := &vzapi.VerrazzanoList{}
			vzList.Items = append(vzList.Items, *tt.fields.vz)
			jc, err := rc.getJaegerOpenSearchConfig(vzList)
			assert.NoError(t, err)
			if tt.fields.jaegerCreate {
				if tt.fields.externalOS {
					assert.Equal(t, externalOSURL, jc.URL)
					assert.Equal(t, "jaegeropensearchtlscakey", string(jc.CA))
					if tt.fields.mutualTLS {
						assert.Equal(t, "jaegeropensearchtlskey", string(jc.TLSKey))
						assert.Equal(t, "jaegeropensearchtlscert", string(jc.TLSCert))
					}
					assert.Equal(t, "externalusername", string(jc.username))
					assert.Equal(t, "externalpassword", string(jc.password))
				} else {
					assert.Equal(t, "https://jaeger.unit-test.com:443", jc.URL)
					assert.Equal(t, "username", string(jc.username))
					assert.Equal(t, "password", string(jc.password))
					assert.Equal(t, "adminCAbundle", string(jc.CA))
					assert.Equal(t, "", string(jc.TLSKey))
					assert.Equal(t, "", string(jc.TLSCert))
				}
			} else {
				assert.Equal(t, "", jc.URL)
			}
		})
	}
}

func createVZJaegerOverride(json string) *vzapi.Verrazzano {
	return &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				JaegerOperator: &vzapi.JaegerOperatorComponent{
					Enabled: &trueValue,
					InstallOverrides: vzapi.InstallOverrides{
						MonitorChanges: &trueValue,
						ValueOverrides: []vzapi.Overrides{
							{
								Values: &apiextensionsv1.JSON{
									Raw: []byte(json),
								},
							},
						},
					},
				},
			},
		},
	}
}

func createVZWithOSDisabledAndJaegerOverride(json string) *vzapi.Verrazzano {
	return &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Elasticsearch: &vzapi.ElasticsearchComponent{
					Enabled: &falseValue,
				},
				JaegerOperator: &vzapi.JaegerOperatorComponent{
					Enabled: &trueValue,
					InstallOverrides: vzapi.InstallOverrides{
						MonitorChanges: &trueValue,
						ValueOverrides: []vzapi.Overrides{
							{
								Values: &apiextensionsv1.JSON{
									Raw: []byte(json),
								},
							},
						},
					},
				},
			},
		},
	}
}

func newIngress() *k8net.Ingress {
	ingress := &k8net.Ingress{}
	ingress.Namespace = constants.VerrazzanoSystemNamespace
	ingress.Name = vmiIngest
	rule := k8net.IngressRule{Host: "jaeger.unit-test.com"}
	ingress.Spec.Rules = append(ingress.Spec.Rules, rule)
	return ingress
}
