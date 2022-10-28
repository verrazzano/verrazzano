// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package fluentd

import (
	"context"
	"github.com/stretchr/testify/assert"
	globalconst "github.com/verrazzano/verrazzano/pkg/constants"
	ctrlerrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vpoconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

var keycloakEnabledCR = &vzapi.Verrazzano{
	Spec: vzapi.VerrazzanoSpec{
		Components: vzapi.ComponentSpec{
			Keycloak: &vzapi.KeycloakComponent{
				Enabled: &enabled,
			},
		},
	},
}

var keycloakDisabledCR = &vzapi.Verrazzano{
	Spec: vzapi.VerrazzanoSpec{
		Components: vzapi.ComponentSpec{
			Keycloak: &vzapi.KeycloakComponent{
				Enabled: &notEnabled,
			},
		},
	},
}

func TestIsFluentdReady(t *testing.T) {
	boolTrue, boolFalse := true, false
	var tests = []struct {
		testName string
		spec     vzapi.Verrazzano
		client   clipkg.Client
		expected bool
	}{
		{"1", vzapi.Verrazzano{
			Spec: vzapi.VerrazzanoSpec{
				Components: vzapi.ComponentSpec{Fluentd: &vzapi.FluentdComponent{
					Enabled: &boolTrue,
				}},
			},
		}, getFakeClient(1), true},
		{"2", vzapi.Verrazzano{
			Spec: vzapi.VerrazzanoSpec{
				Components: vzapi.ComponentSpec{Fluentd: &vzapi.FluentdComponent{
					Enabled: &boolFalse,
				}},
			},
		}, getFakeClient(1), false},
		{"3", vzapi.Verrazzano{
			Spec: vzapi.VerrazzanoSpec{
				Components: vzapi.ComponentSpec{Fluentd: &vzapi.FluentdComponent{
					Enabled: &boolTrue,
				}}, Profile: vzapi.Prod,
			},
		}, getFakeClient(1), true},
		{"4", vzapi.Verrazzano{
			Spec: vzapi.VerrazzanoSpec{
				Components: vzapi.ComponentSpec{Fluentd: &vzapi.FluentdComponent{
					Enabled: &boolTrue,
				}}, Profile: vzapi.Prod,
			},
		}, getFakeClient(0), false},
		{"5", vzapi.Verrazzano{
			Spec: vzapi.VerrazzanoSpec{
				Components: vzapi.ComponentSpec{Fluentd: &vzapi.FluentdComponent{
					Enabled: &boolTrue,
				}}, Profile: vzapi.ManagedCluster,
			},
		}, getFakeClient(0), false},
	}
	for _, test := range tests {
		ctx := spi.NewFakeContext(test.client, &test.spec, nil, false)
		if actual := isFluentdReady(ctx); actual != test.expected {
			t.Errorf("test name %s: got fluent ready = %v, want %v", test.testName, actual, test.expected)
		}
	}
}

// TestFixupFluentdDaemonset tests calls to fixupFluentdDaemonset
func TestFixupFluentdDaemonset(t *testing.T) {
	const defNs = vpoconst.VerrazzanoSystemNamespace
	a := assert.New(t)
	scheme := runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	c := fake.NewClientBuilder().WithScheme(scheme).Build()
	log := vzlog.DefaultLogger()

	ns := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: defNs,
		},
	}
	err := c.Create(context.TODO(), &ns)
	a.NoError(err)

	// Should return with no error since the fluentd daemonset does not exist.
	// This is valid case when fluentd is not installed.
	err = fixupFluentdDaemonset(log, c, defNs)
	a.NoError(err)

	// Create a fluentd daemonset for test purposes
	const someURL = "some-url"
	daemonSet := appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: defNs,
			Name:      globalconst.FluentdDaemonSetName,
		},
		Spec: appsv1.DaemonSetSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "wrong-name",
							Env: []corev1.EnvVar{
								{
									Name:  vpoconst.ClusterNameEnvVar,
									Value: "managed1",
								},
								{
									Name:  vpoconst.OpensearchURLEnvVar,
									Value: someURL,
								},
							},
						},
					},
				},
			},
		},
	}
	err = c.Create(context.TODO(), &daemonSet)
	a.NoError(err)

	// should return error that fluentd container is missing
	err = fixupFluentdDaemonset(log, c, defNs)
	a.Contains(err.Error(), "fluentd container not found in fluentd daemonset: fluentd")

	daemonSet.Spec.Template.Spec.Containers[0].Name = "fluentd"
	err = c.Update(context.TODO(), &daemonSet)
	a.NoError(err)

	// should return no error since the env variables don't need fixing up
	err = fixupFluentdDaemonset(log, c, defNs)
	a.NoError(err)

	// create a secret with needed keys
	data := make(map[string][]byte)
	data[vpoconst.ClusterNameData] = []byte("managed1")
	data[vpoconst.OpensearchURLData] = []byte(someURL)
	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: defNs,
			Name:      vpoconst.MCRegistrationSecret,
		},
		Data: data,
	}
	err = c.Create(context.TODO(), &secret)
	a.NoError(err)

	// Update env variables to use ValueFrom instead of Value
	clusterNameRef := corev1.EnvVarSource{
		SecretKeyRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: vpoconst.MCRegistrationSecret,
			},
			Key: vpoconst.ClusterNameData,
		},
	}
	esURLRef := corev1.EnvVarSource{
		SecretKeyRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: vpoconst.MCRegistrationSecret,
			},
			Key: vpoconst.OpensearchURLData,
		},
	}
	daemonSet.Spec.Template.Spec.Containers[0].Env[0].Value = ""
	daemonSet.Spec.Template.Spec.Containers[0].Env[0].ValueFrom = &clusterNameRef
	daemonSet.Spec.Template.Spec.Containers[0].Env[1].Value = ""
	daemonSet.Spec.Template.Spec.Containers[0].Env[1].ValueFrom = &esURLRef
	err = c.Update(context.TODO(), &daemonSet)
	a.NoError(err)

	// should return no error
	err = fixupFluentdDaemonset(log, c, defNs)
	a.NoError(err)

	// env variables should be fixed up to use Value instead of ValueFrom
	fluentdNamespacedName := types.NamespacedName{Name: globalconst.FluentdDaemonSetName, Namespace: defNs}
	updatedDaemonSet := appsv1.DaemonSet{}
	err = c.Get(context.TODO(), fluentdNamespacedName, &updatedDaemonSet)
	a.NoError(err)
	a.Equal("managed1", updatedDaemonSet.Spec.Template.Spec.Containers[0].Env[0].Value)
	a.Nil(updatedDaemonSet.Spec.Template.Spec.Containers[0].Env[0].ValueFrom)
	a.Equal(someURL, updatedDaemonSet.Spec.Template.Spec.Containers[0].Env[1].Value)
	a.Nil(updatedDaemonSet.Spec.Template.Spec.Containers[0].Env[1].ValueFrom)
}

// TestLoggingPreInstall tests the Fluentd loggingPreInstall call
func TestLoggingPreInstall(t *testing.T) {
	// GIVEN a Fluentd component
	//  WHEN I call loggingPreInstall with fluentd overrides for ES and a custom ES secret
	//  THEN no error is returned and the secret has been copied
	trueValue := true
	secretName := "my-es-secret" //nolint:gosec //#gosec G101
	c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Namespace: vpoconst.VerrazzanoInstallNamespace, Name: secretName},
	}).Build()

	ctx := spi.NewFakeContext(c, &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Fluentd: &vzapi.FluentdComponent{
					Enabled:             &trueValue,
					ElasticsearchURL:    "https://myes.mydomain.com:9200",
					ElasticsearchSecret: secretName,
				},
			},
		},
	}, nil, false)
	err := loggingPreInstall(ctx)
	assert.NoError(t, err)

	secret := &corev1.Secret{}
	err = c.Get(context.TODO(), types.NamespacedName{Name: secretName, Namespace: ComponentNamespace}, secret)
	assert.NoError(t, err)

	// GIVEN a Verrazzano component
	//  WHEN I call loggingPreInstall with fluentd overrides for OCI logging, including an OCI API secret name
	//  THEN no error is returned and the secret has been copied
	secretName = "my-oci-api-secret" //nolint:gosec //#gosec G101
	cs := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Namespace: vpoconst.VerrazzanoInstallNamespace, Name: secretName},
		},
	).Build()
	ctx = spi.NewFakeContext(cs, &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Fluentd: &vzapi.FluentdComponent{
					Enabled: &trueValue,
					OCI: &vzapi.OciLoggingConfiguration{
						APISecret: secretName,
					},
				},
			},
		},
	}, nil, false)
	err = loggingPreInstall(ctx)
	assert.NoError(t, err)

	err = cs.Get(context.TODO(), types.NamespacedName{Name: secretName, Namespace: ComponentNamespace}, secret)
	assert.NoError(t, err)
}

// TestLoggingPreInstallSecretNotFound tests the Verrazzano loggingPreInstall call
// GIVEN a Verrazzano component
//
//	WHEN I call loggingPreInstall with fluentd overrides for ES and a custom ES secret and the secret does not exist
//	THEN an error is returned
func TestLoggingPreInstallSecretNotFound(t *testing.T) {
	trueValue := true
	c := fake.NewClientBuilder().WithScheme(testScheme).Build()
	ctx := spi.NewFakeContext(c, &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Fluentd: &vzapi.FluentdComponent{
					Enabled:             &trueValue,
					ElasticsearchURL:    "https://myes.mydomain.com:9200",
					ElasticsearchSecret: "my-es-secret",
				},
			},
		},
	}, nil, false)
	err := loggingPreInstall(ctx)
	assert.Error(t, err)
}

// TestLoggingPreInstallFluentdNotEnabled tests the Verrazzano loggingPreInstall call
// GIVEN a Verrazzano component
//
//	WHEN I call loggingPreInstall and fluentd is disabled
//	THEN no error is returned
func TestLoggingPreInstallFluentdNotEnabled(t *testing.T) {
	falseValue := false
	c := fake.NewClientBuilder().WithScheme(testScheme).Build()
	ctx := spi.NewFakeContext(c, &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Fluentd: &vzapi.FluentdComponent{
					Enabled: &falseValue,
				},
			},
		},
	}, nil, false)
	err := loggingPreInstall(ctx)
	assert.NoError(t, err)
}

// TestCheckSecretExists tests the verrazzano-es-internal secret exists.
func TestCheckSecretExists(t *testing.T) {
	var tests = []struct {
		name   string
		spec   *vzapi.Verrazzano
		client clipkg.Client
		err    error
	}{
		{
			"should fail when verrazzano-es-internal secret does not exist and keycloak is enabled",
			keycloakEnabledCR,
			createFakeClient(),
			ctrlerrors.RetryableError{Source: ComponentName},
		},
		{
			"should pass when verrazzano-es-internal secret does exist and keycloak is enabled",
			keycloakEnabledCR,
			createFakeClient(vzEsInternalSecret),
			nil,
		},
		{
			"always nil error when keycloak is disabled",
			keycloakDisabledCR,
			createFakeClient(),
			nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := spi.NewFakeContext(tt.client, tt.spec, nil, false)
			err := checkSecretExists(ctx)
			if tt.err != nil {
				assert.Error(t, err)
				assert.IsTypef(t, tt.err, err, "")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestExternalSecretExists tests the Fluentd checkSecretExists call
func TestExternalSecretExists(t *testing.T) {
	// GIVEN a Fluentd component
	//  WHEN I call checkSecretExists with fluentd overrides for ES and a custom ES secret
	//  THEN no error is returned and external secret is checked and not the internal.
	trueValue := true
	secretName := "my-es-secret" //nolint:gosec //#gosec G101
	c := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Namespace: vpoconst.VerrazzanoSystemNamespace, Name: secretName},
	}).Build()

	ctx := spi.NewFakeContext(c, &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Fluentd: &vzapi.FluentdComponent{
					Enabled:             &trueValue,
					ElasticsearchURL:    "https://myes.mydomain.com:9200",
					ElasticsearchSecret: secretName,
				},
			},
		},
	}, nil, false)

	err := checkSecretExists(ctx)
	assert.NoError(t, err)
}

func getFakeClient(scheduled int32) clipkg.Client {
	return fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: globalconst.VerrazzanoSystemNamespace,
				Name:      ComponentName,
				Labels:    map[string]string{"app": "test"},
			},
			Spec: appsv1.DaemonSetSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": "test"},
				},
			},
			Status: appsv1.DaemonSetStatus{
				UpdatedNumberScheduled: scheduled,
				NumberAvailable:        1,
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      ComponentName,
				Labels: map[string]string{
					"app":                      "test",
					"controller-revision-hash": "test-95d8c5d96",
				},
			},
		},
		&appsv1.ControllerRevision{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ComponentName + "-test-95d8c5d96",
				Namespace: ComponentNamespace,
			},
			Revision: 1,
		},
	).Build()
}
