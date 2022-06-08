// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package fluentd

import (
	"context"
	"github.com/stretchr/testify/assert"
	globalconst "github.com/verrazzano/verrazzano/pkg/constants"
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

func TestIsFluentdReady(t *testing.T) {
	boolTrue, boolFalse := true, false
	var tests = []struct {
		spec     vzapi.Verrazzano
		client   clipkg.Client
		expected bool
	}{
		{vzapi.Verrazzano{
			Spec: vzapi.VerrazzanoSpec{
				Components: vzapi.ComponentSpec{Fluentd: &vzapi.FluentdComponent{
					Enabled: &boolTrue,
				}},
			},
		}, getFakeClient(1), true},
		{vzapi.Verrazzano{
			Spec: vzapi.VerrazzanoSpec{
				Components: vzapi.ComponentSpec{Fluentd: &vzapi.FluentdComponent{
					Enabled: &boolFalse,
				}},
			},
		}, getFakeClient(1), false},
		{vzapi.Verrazzano{
			Spec: vzapi.VerrazzanoSpec{
				Components: vzapi.ComponentSpec{Fluentd: &vzapi.FluentdComponent{
					Enabled: &boolTrue,
				}}, Profile: vzapi.Prod,
			},
		}, getFakeClient(1), true},
		{vzapi.Verrazzano{
			Spec: vzapi.VerrazzanoSpec{
				Components: vzapi.ComponentSpec{Fluentd: &vzapi.FluentdComponent{
					Enabled: &boolTrue,
				}}, Profile: vzapi.Prod,
			},
		}, getFakeClient(0), false},
		{vzapi.Verrazzano{
			Spec: vzapi.VerrazzanoSpec{
				Components: vzapi.ComponentSpec{Fluentd: &vzapi.FluentdComponent{
					Enabled: &boolTrue,
				}}, Profile: vzapi.ManagedCluster,
			},
		}, getFakeClient(0), false},
	}
	for _, test := range tests {
		//client := createPreInstallTestClient()
		ctx := spi.NewFakeContext(test.client, &test.spec, false)
		if actual := isFluentdReady(ctx); actual != test.expected {
			t.Errorf("got fluent ready = %v, want %v", actual, test.expected)
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
									Name:  vpoconst.ElasticsearchURLEnvVar,
									Value: "some-url",
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
	data[vpoconst.ElasticsearchURLData] = []byte("some-url")
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
			Key: vpoconst.ElasticsearchURLData,
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
	a.Equal("some-url", updatedDaemonSet.Spec.Template.Spec.Containers[0].Env[1].Value)
	a.Nil(updatedDaemonSet.Spec.Template.Spec.Containers[0].Env[1].ValueFrom)
}

func getFakeClient(scheduled int32) clipkg.Client {
	return fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: globalconst.VerrazzanoSystemNamespace,
				Name:      fluentDaemonset,
			},
			Status: appsv1.DaemonSetStatus{
				UpdatedNumberScheduled: scheduled,
				NumberAvailable:        1,
			},
		},
	).Build()
}
