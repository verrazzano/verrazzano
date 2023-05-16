// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package thanos

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/authproxy"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	cliruntime "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const profilesRelativePath = "../../../../manifests/profiles"

// TestThanosEnabled tests if Thanos is enabled
// GIVEN a call to IsEnabled
// WHEN the VZ CR is populated
// THEN a boolean is returned
func TestThanosEnabled(t *testing.T) {
	trueVal := true
	falseVal := false
	crA1 := &v1alpha1.Verrazzano{}
	crB1 := &v1beta1.Verrazzano{}

	crA1NilComp := crA1.DeepCopy()
	crA1NilComp.Spec.Components.Thanos = nil
	crA1NilEnabled := crA1.DeepCopy()
	crA1NilEnabled.Spec.Components.Thanos = &v1alpha1.ThanosComponent{Enabled: nil}
	crA1Enabled := crA1.DeepCopy()
	crA1Enabled.Spec.Components.Thanos = &v1alpha1.ThanosComponent{Enabled: &trueVal}
	crA1Disabled := crA1.DeepCopy()
	crA1Disabled.Spec.Components.Thanos = &v1alpha1.ThanosComponent{Enabled: &falseVal}

	crB1NilComp := crB1.DeepCopy()
	crB1NilComp.Spec.Components.Thanos = nil
	crB1NilEnabled := crB1.DeepCopy()
	crB1NilEnabled.Spec.Components.Thanos = &v1beta1.ThanosComponent{Enabled: nil}
	crB1Enabled := crB1.DeepCopy()
	crB1Enabled.Spec.Components.Thanos = &v1beta1.ThanosComponent{Enabled: &trueVal}
	crB1Disabled := crB1.DeepCopy()
	crB1Disabled.Spec.Components.Thanos = &v1beta1.ThanosComponent{Enabled: &falseVal}

	tests := []struct {
		name         string
		verrazzanoA1 *v1alpha1.Verrazzano
		verrazzanoB1 *v1beta1.Verrazzano
		assertion    func(t assert.TestingT, value bool, msgAndArgs ...interface{}) bool
	}{
		{
			name:         "test v1alpha1 component nil",
			verrazzanoA1: crA1NilComp,
			assertion:    assert.False,
		},
		{
			name:         "test v1alpha1 enabled nil",
			verrazzanoA1: crA1NilEnabled,
			assertion:    assert.False,
		},
		{
			name:         "test v1alpha1 enabled",
			verrazzanoA1: crA1Enabled,
			assertion:    assert.True,
		},
		{
			name:         "test v1alpha1 disabled",
			verrazzanoA1: crA1Disabled,
			assertion:    assert.False,
		},
		{
			name:         "test v1beta1 component nil",
			verrazzanoB1: crB1NilComp,
			assertion:    assert.False,
		},
		{
			name:         "test v1beta1 enabled nil",
			verrazzanoB1: crB1NilEnabled,
			assertion:    assert.False,
		},
		{
			name:         "test v1beta1 enabled",
			verrazzanoB1: crB1Enabled,
			assertion:    assert.True,
		},
		{
			name:         "test v1beta1 disabled",
			verrazzanoB1: crB1Disabled,
			assertion:    assert.False,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.verrazzanoA1 != nil {
				tt.assertion(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, tt.verrazzanoA1, tt.verrazzanoB1, false, profilesRelativePath).EffectiveCR()))
			}
			if tt.verrazzanoB1 != nil {
				tt.assertion(t, NewComponent().IsEnabled(spi.NewFakeContext(nil, tt.verrazzanoA1, tt.verrazzanoB1, false, profilesRelativePath).EffectiveCRV1Beta1()))
			}
		})
	}
}

// TestGetIngressNames tests the GetIngressNames for the Thanos component
func TestGetIngressNames(t *testing.T) {
	enabled := true
	disabled := false

	scheme := k8scheme.Scheme

	var tests = []struct {
		name     string
		vz       v1alpha1.Verrazzano
		ingNames []types.NamespacedName
	}{
		// GIVEN a call to GetIngressNames
		// WHEN all components are disabled
		// THEN no ingresses are returned
		{
			name: "TestGetIngress when all disabled",
			vz: v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						AuthProxy: &v1alpha1.AuthProxyComponent{Enabled: &disabled},
						Ingress:   &v1alpha1.IngressNginxComponent{Enabled: &disabled},
						Thanos:    &v1alpha1.ThanosComponent{Enabled: &disabled},
						DNS: &v1alpha1.DNSComponent{
							OCI: &v1alpha1.OCI{
								DNSZoneName: "mydomain.com",
							},
						},
					},
				},
			},
		},
		// GIVEN a call to GetIngressNames
		// WHEN all Thanos is disabled
		// THEN no ingresses are returned
		{
			name: "TestGetIngress when Thanos disabled",
			vz: v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						AuthProxy: &v1alpha1.AuthProxyComponent{Enabled: &enabled},
						Ingress:   &v1alpha1.IngressNginxComponent{Enabled: &enabled},
						Thanos:    &v1alpha1.ThanosComponent{Enabled: &disabled},
						DNS: &v1alpha1.DNSComponent{
							OCI: &v1alpha1.OCI{
								DNSZoneName: "mydomain.com",
							},
						},
					},
				},
			},
		},
		// GIVEN a call to GetIngressNames
		// WHEN all NGINX is disabled
		// THEN no ingresses are returned
		{
			name: "TestGetIngress when NGINX disabled",
			vz: v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						AuthProxy: &v1alpha1.AuthProxyComponent{Enabled: &enabled},
						Ingress:   &v1alpha1.IngressNginxComponent{Enabled: &disabled},
						Thanos:    &v1alpha1.ThanosComponent{Enabled: &enabled},
						DNS: &v1alpha1.DNSComponent{
							OCI: &v1alpha1.OCI{
								DNSZoneName: "mydomain.com",
							},
						},
					},
				},
			},
		},
		// GIVEN a call to GetIngressNames
		// WHEN the authproxy is disabled
		// THEN and ingress in the verrazzano-system namespace is returned
		{
			name: "TestGetIngress when Authproxy disabled",
			vz: v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						AuthProxy: &v1alpha1.AuthProxyComponent{Enabled: &disabled},
						Ingress:   &v1alpha1.IngressNginxComponent{Enabled: &enabled},
						Thanos:    &v1alpha1.ThanosComponent{Enabled: &enabled},
						DNS: &v1alpha1.DNSComponent{
							OCI: &v1alpha1.OCI{
								DNSZoneName: "mydomain.com",
							},
						},
					},
				},
			},
			ingNames: []types.NamespacedName{
				{Namespace: constants.VerrazzanoSystemNamespace, Name: vzconst.ThanosQueryIngress},
				{Namespace: constants.VerrazzanoSystemNamespace, Name: vzconst.ThanosQueryStoreIngress},
			},
		},
		// GIVEN a call to GetIngressNames
		// WHEN all components are enabled
		// THEN an ingress in the authproxy namespace is returned
		{
			name: "TestGetIngress when all enabled",
			vz: v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						AuthProxy: &v1alpha1.AuthProxyComponent{Enabled: &enabled},
						Ingress:   &v1alpha1.IngressNginxComponent{Enabled: &enabled},
						Thanos:    &v1alpha1.ThanosComponent{Enabled: &enabled},
						DNS: &v1alpha1.DNSComponent{
							OCI: &v1alpha1.OCI{
								DNSZoneName: "mydomain.com",
							},
						},
					},
				},
			},
			ingNames: []types.NamespacedName{
				{Namespace: authproxy.ComponentNamespace, Name: vzconst.ThanosQueryIngress},
				{Namespace: constants.VerrazzanoSystemNamespace, Name: vzconst.ThanosQueryStoreIngress},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := fake.NewClientBuilder().WithScheme(scheme).Build()
			ctx := spi.NewFakeContext(client, &test.vz, nil, false)
			nsn := NewComponent().GetIngressNames(ctx)
			assert.Equal(t, nsn, test.ingNames)
		})
	}
}

// TestGetCertificateNames tests the GetCertificateNames for the Thanos component
func TestGetCertificateNames(t *testing.T) {
	enabled := true
	disabled := false

	scheme := k8scheme.Scheme

	var tests = []struct {
		name     string
		vz       v1alpha1.Verrazzano
		ingNames []types.NamespacedName
	}{
		// GIVEN a call to GetCertificateNames
		// WHEN all components are disabled
		// THEN no certificates are returned
		{
			name: "TestGetIngress when all disabled",
			vz: v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						AuthProxy: &v1alpha1.AuthProxyComponent{Enabled: &disabled},
						Ingress:   &v1alpha1.IngressNginxComponent{Enabled: &disabled},
						Thanos:    &v1alpha1.ThanosComponent{Enabled: &disabled},
						DNS: &v1alpha1.DNSComponent{
							OCI: &v1alpha1.OCI{
								DNSZoneName: "mydomain.com",
							},
						},
					},
				},
			},
		},
		// GIVEN a call to GetCertificateNames
		// WHEN all Thanos is disabled
		// THEN no certificates are returned
		{
			name: "TestGetIngress when Thanos disabled",
			vz: v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						AuthProxy: &v1alpha1.AuthProxyComponent{Enabled: &enabled},
						Ingress:   &v1alpha1.IngressNginxComponent{Enabled: &enabled},
						Thanos:    &v1alpha1.ThanosComponent{Enabled: &disabled},
						DNS: &v1alpha1.DNSComponent{
							OCI: &v1alpha1.OCI{
								DNSZoneName: "mydomain.com",
							},
						},
					},
				},
			},
		},
		// GIVEN a call to GetCertificateNames
		// WHEN all NGINX is disabled
		// THEN no certificates are returned
		{
			name: "TestGetIngress when NGINX disabled",
			vz: v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						AuthProxy: &v1alpha1.AuthProxyComponent{Enabled: &enabled},
						Ingress:   &v1alpha1.IngressNginxComponent{Enabled: &disabled},
						Thanos:    &v1alpha1.ThanosComponent{Enabled: &enabled},
						DNS: &v1alpha1.DNSComponent{
							OCI: &v1alpha1.OCI{
								DNSZoneName: "mydomain.com",
							},
						},
					},
				},
			},
		},
		// GIVEN a call to GetCertificateNames
		// WHEN the authproxy is disabled
		// THEN and certificate in the verrazzano-system namespace is returned
		{
			name: "TestGetIngress when Authproxy disabled",
			vz: v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						AuthProxy: &v1alpha1.AuthProxyComponent{Enabled: &disabled},
						Ingress:   &v1alpha1.IngressNginxComponent{Enabled: &enabled},
						Thanos:    &v1alpha1.ThanosComponent{Enabled: &enabled},
						DNS: &v1alpha1.DNSComponent{
							OCI: &v1alpha1.OCI{
								DNSZoneName: "mydomain.com",
							},
						},
					},
				},
			},
			ingNames: []types.NamespacedName{
				{Namespace: constants.VerrazzanoSystemNamespace, Name: queryCertificateName},
				{Namespace: constants.VerrazzanoSystemNamespace, Name: queryStoreCertificateName},
			},
		},
		// GIVEN a call to GetCertificateNames
		// WHEN all components are enabled
		// THEN an certificate in the authproxy namespace is returned
		{
			name: "TestGetIngress when all enabled",
			vz: v1alpha1.Verrazzano{
				Spec: v1alpha1.VerrazzanoSpec{
					Components: v1alpha1.ComponentSpec{
						AuthProxy: &v1alpha1.AuthProxyComponent{Enabled: &enabled},
						Ingress:   &v1alpha1.IngressNginxComponent{Enabled: &enabled},
						Thanos:    &v1alpha1.ThanosComponent{Enabled: &enabled},
						DNS: &v1alpha1.DNSComponent{
							OCI: &v1alpha1.OCI{
								DNSZoneName: "mydomain.com",
							},
						},
					},
				},
			},
			ingNames: []types.NamespacedName{
				{Namespace: authproxy.ComponentNamespace, Name: queryCertificateName},
				{Namespace: constants.VerrazzanoSystemNamespace, Name: queryStoreCertificateName},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := fake.NewClientBuilder().WithScheme(scheme).Build()
			ctx := spi.NewFakeContext(client, &test.vz, nil, false)
			nsn := NewComponent().GetCertificateNames(ctx)
			assert.Equal(t, test.ingNames, nsn)
		})
	}
}

type readinessTestType struct {
	name                  string
	queryDeployed         bool
	queryFrontendDeployed bool
	storegatewayDeployed  bool
	queryReady            bool
	queryFrontendReady    bool
	storegatewayReady     bool
	expectReady           bool
}

// readinessTests are test cases used by both TestIsReady and TestIsAvailable
var readinessTests = []readinessTestType{
	// The Thanos Query and Thanos Query frontend deployments exists and the pods are ready
	{
		name:                  "Thanos Query and Query Frontend are deployed and ready",
		queryDeployed:         true,
		queryFrontendDeployed: true,
		storegatewayDeployed:  false,
		queryReady:            true,
		queryFrontendReady:    true,
		storegatewayReady:     false,
		expectReady:           true,
	},
	// The Thanos Query and Thanos Query frontend deployments exists but some of the pods are not ready
	{
		name:                  "Thanos Query and Query Frontend are deployed but one is not ready",
		queryDeployed:         true,
		queryFrontendDeployed: true,
		storegatewayDeployed:  false,
		queryReady:            true,
		queryFrontendReady:    false,
		storegatewayReady:     false,
		expectReady:           false,
	},
	// The Thanos Query and Thanos Query frontend deployments exists and are available, Storegateway
	// exists but pod is not ready
	{
		name:                  "Thanos Storegateway is also deployed but not ready",
		queryDeployed:         true,
		queryFrontendDeployed: true,
		storegatewayDeployed:  true,
		queryReady:            true,
		queryFrontendReady:    true,
		storegatewayReady:     false,
		expectReady:           false,
	},
	// The Thanos Query and Thanos Query frontend deployments and Storegateway statefulset exist
	// and are available
	{
		name:                  "Thanos all components are deployed and ready",
		queryDeployed:         true,
		queryFrontendDeployed: true,
		storegatewayDeployed:  true,
		queryReady:            true,
		queryFrontendReady:    true,
		storegatewayReady:     true,
		expectReady:           true,
	},
}

// TestIsReady tests IsReady for the Thanos component
func TestIsReady(t *testing.T) {
	scheme := k8scheme.Scheme

	for _, test := range readinessTests {
		t.Run(test.name, func(t *testing.T) {
			objectsToCreate := getReadinessTestObjects(test)
			client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objectsToCreate...).Build()
			ctx := spi.NewFakeContext(client, &v1alpha1.Verrazzano{}, nil, true)
			assert.Equal(t, test.expectReady, NewComponent().IsReady(ctx))
		})
	}
}

// TestIsAvailable tests IsAvailable for the Thanos component
func TestIsAvailable(t *testing.T) {
	scheme := k8scheme.Scheme

	for _, test := range readinessTests {
		t.Run(test.name, func(t *testing.T) {
			objectsToCreate := getReadinessTestObjects(test)
			client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objectsToCreate...).Build()
			ctx := spi.NewFakeContext(client, &v1alpha1.Verrazzano{}, nil, true)
			_, availability := NewComponent().IsAvailable(ctx)
			if test.expectReady {
				assert.Equal(t, v1alpha1.ComponentAvailability(v1alpha1.ComponentAvailable), availability)
			} else {
				assert.Equal(t, v1alpha1.ComponentAvailability(v1alpha1.ComponentUnavailable), availability)
			}
		})
	}
}

func getReadinessTestObjects(test readinessTestType) []cliruntime.Object {
	var objectsToCreate []cliruntime.Object
	revisionHash := "12345"
	controllerRevisionName := "crevname"

	if test.queryDeployed {
		objectsToCreate = []cliruntime.Object{makeDeployment(queryDeployment, test.queryReady)}
		objectsToCreate = append(objectsToCreate, makeReplicaset(queryDeployment, revisionHash))
	}
	if test.queryFrontendDeployed {
		objectsToCreate = append(objectsToCreate, makeDeployment(frontendDeployment, test.queryFrontendReady))
		objectsToCreate = append(objectsToCreate, makeReplicaset(frontendDeployment, revisionHash))
	}
	if test.storegatewayDeployed {
		objectsToCreate = append(objectsToCreate, makeStatefulset(storeGatewayStatefulset, test.storegatewayReady))
		objectsToCreate = append(objectsToCreate, makeControllerRevision(controllerRevisionName))
	}
	if test.queryReady {
		objectsToCreate = append(objectsToCreate, makePod(queryDeployment, map[string]string{
			"app":               queryDeployment,
			"pod-template-hash": revisionHash,
		}))
	}
	if test.queryFrontendReady {
		objectsToCreate = append(objectsToCreate, makePod(frontendDeployment, map[string]string{
			"app":               frontendDeployment,
			"pod-template-hash": revisionHash,
		}))
	}
	if test.storegatewayReady {
		objectsToCreate = append(objectsToCreate, makePod(storeGatewayStatefulset, map[string]string{
			"app":                      storeGatewayStatefulset,
			"controller-revision-hash": controllerRevisionName,
		}))
	}
	return objectsToCreate
}

func makeControllerRevision(name string) cliruntime.Object {
	return &appsv1.ControllerRevision{
		ObjectMeta: v1.ObjectMeta{
			Namespace: ComponentNamespace,
			Name:      name,
		},
		Revision: 1,
	}
}

func makeStatefulset(name string, ready bool) *appsv1.StatefulSet {
	var readyReplicas int32
	if ready {
		readyReplicas = 1
	}
	return &appsv1.StatefulSet{
		ObjectMeta: v1.ObjectMeta{Name: name, Namespace: ComponentNamespace},
		Spec:       appsv1.StatefulSetSpec{Selector: &v1.LabelSelector{MatchLabels: map[string]string{"app": name}}},
		Status:     appsv1.StatefulSetStatus{Replicas: 1, AvailableReplicas: 1, UpdatedReplicas: 1, ReadyReplicas: readyReplicas},
	}
}

func makeDeployment(name string, ready bool) *appsv1.Deployment {
	var readyReplicas int32
	if ready {
		readyReplicas = 1
	}
	return &appsv1.Deployment{
		ObjectMeta: v1.ObjectMeta{Name: name, Namespace: ComponentNamespace},
		Spec:       appsv1.DeploymentSpec{Selector: &v1.LabelSelector{MatchLabels: map[string]string{"app": name}}},
		Status:     appsv1.DeploymentStatus{Replicas: 1, AvailableReplicas: 1, UpdatedReplicas: 1, ReadyReplicas: readyReplicas},
	}
}

func makeReplicaset(name string, suffix string) cliruntime.Object {
	return &appsv1.ReplicaSet{
		ObjectMeta: v1.ObjectMeta{
			Namespace:   ComponentNamespace,
			Name:        fmt.Sprintf("%s-%s", name, suffix),
			Annotations: map[string]string{"deployment.kubernetes.io/revision": "1"},
		},
	}
}

func makePod(baseName string, labels map[string]string) cliruntime.Object {
	return &corev1.Pod{
		ObjectMeta: v1.ObjectMeta{
			Name:      fmt.Sprintf("%s-1234", baseName),
			Namespace: ComponentNamespace,
			Labels:    labels,
		},
		Status: corev1.PodStatus{
			Conditions: []corev1.PodCondition{
				{
					Type:   corev1.PodReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}
}
