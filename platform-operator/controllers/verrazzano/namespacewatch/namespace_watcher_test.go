// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package namespacewatch

import (
	"context"
	asserts "github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
	"time"
)

const reldir = "../../../manifests/profiles"

var period = time.Duration(10) * time.Second
var testScheme = runtime.NewScheme()

func init() {
	_ = k8scheme.AddToScheme(testScheme)
	_ = v1alpha1.AddToScheme(testScheme)
}

func TestStart(t *testing.T) {
	client := fake.NewClientBuilder().WithScheme(testScheme).WithObjects().Build()
	ctx := spi.NewFakeContext(client, nil, nil, false)
	namespaceWatcher = NewNamespaceWatcher(ctx.Client(), period)
	asserts.Nil(t, namespaceWatcher.shutdown)
	namespaceWatcher.Start()
	asserts.NotNil(t, namespaceWatcher.shutdown)
	namespaceWatcher.Start()
	asserts.NotNil(t, namespaceWatcher.shutdown)
	namespaceWatcher.Pause()
	asserts.Nil(t, namespaceWatcher.shutdown)
	namespaceWatcher.Pause()
	asserts.Nil(t, namespaceWatcher.shutdown)
}

// TestMoveSystemNamespaces tests the following cases
// GIVEN that rancher component is enabled and in not ready state in Verrazzano installation
// OR when subcomponents are not ready
// THEN no operation takes place
func TestNotToMoveSystemNamespacesWhenRancherNotReady(t *testing.T) {
	namespace1 := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "verrazzano-system",
			Labels: map[string]string{
				constants.VerrazzanoManagedKey: "verrazzano-system",
			},
		},
	}
	enabled := true
	var availability v1alpha1.ComponentAvailability
	availability = "Available"
	vzCR := &v1alpha1.Verrazzano{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "example",
		},
		Spec: v1alpha1.VerrazzanoSpec{
			Components: v1alpha1.ComponentSpec{
				Rancher: &v1alpha1.RancherComponent{
					Enabled: &enabled,
				},
			},
		},
		Status: v1alpha1.VerrazzanoStatus{
			Components: v1alpha1.ComponentStatusMap{
				"rancher": &v1alpha1.ComponentStatusDetails{
					Name:      "rancher",
					State:     "Ready",
					Available: &availability,
				},
			},
		},
	}

	testScheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "install.verrazzano.io/v1alpha1",
		Kind:    "Verrazzano",
		Version: "v1alpha1",
	}, &v1alpha1.Verrazzano{})
	config.TestProfilesDir = reldir
	defer func() { config.TestProfilesDir = "" }()
	client := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(vzCR,
		newReplicaSet("cattle-system", "rancher"),
		newReplicaSet("cattle-system", "rancher-webhook"),
		newReplicaSet("cattle-fleet-system", "gitjob"),
		newReplicaSet("cattle-fleet-system", "fleet-controller"),
		newReplicaSet("cattle-fleet-local-system", "fleet-agent"),
		newPod("cattle-system", "rancher"), namespace1).Build()
	ctx := spi.NewFakeContext(client, vzCR, nil, false)
	namespaceWatcher = NewNamespaceWatcher(ctx.Client(), period)
	projectID := "p-47cnm"
	err := namespaceWatcher.MoveSystemNamespacesToRancherSystemProject(projectID)
	asserts.NoError(t, err)
	ns := v1.Namespace{}
	asserts.NoError(t, client.Get(context.Background(), types.NamespacedName{Name: "verrazzano-system"}, &ns))
	asserts.Equal(t, ns.Annotations[RancherProjectIDLabelKey], "")
}

// TestToNotMoveSystemNamespaces tests the following cases
// GIVEN that rancher component is enabled and in ready state in Verrazzano installation
// When namespaces on the cluster does not have label "verrazzano.io/namespace"
// THEN the namespace  is ignored
func TestToNotMoveSystemNamespacesWhenNoSystemNSLabel(t *testing.T) {
	namespace1 := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "verrazzano-system",
		},
	}
	enabled := true
	var availability v1alpha1.ComponentAvailability
	availability = "Available"
	vzCR := &v1alpha1.Verrazzano{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "example",
		},
		Spec: v1alpha1.VerrazzanoSpec{
			Components: v1alpha1.ComponentSpec{
				Rancher: &v1alpha1.RancherComponent{
					Enabled: &enabled,
				},
			},
		},
		Status: v1alpha1.VerrazzanoStatus{
			Components: v1alpha1.ComponentStatusMap{
				"rancher": &v1alpha1.ComponentStatusDetails{
					Name:      "rancher",
					State:     "Ready",
					Available: &availability,
				},
			},
		},
	}

	testScheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "install.verrazzano.io/v1alpha1",
		Kind:    "Verrazzano",
		Version: "v1alpha1",
	}, &v1alpha1.Verrazzano{})
	config.TestProfilesDir = reldir
	defer func() { config.TestProfilesDir = "" }()
	client := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(vzCR,
		newReplicaSet("cattle-system", "rancher"),
		newReplicaSet("cattle-system", "rancher-webhook"),
		newReplicaSet("cattle-fleet-system", "gitjob"),
		newReplicaSet("cattle-fleet-system", "fleet-controller"),
		newReplicaSet("cattle-fleet-local-system", "fleet-agent"),
		newPod("cattle-system", "rancher"),
		newPod("cattle-system", "rancher-webhook"),
		newPod("cattle-fleet-system", "gitjob"),
		newPod("cattle-fleet-system", "fleet-controller"),
		newPod("cattle-fleet-local-system", "fleet-agent"),
		newReadyDeployment("cattle-system", "rancher"),
		newReadyDeployment("cattle-system", "rancher-webhook"),
		newReadyDeployment("cattle-fleet-system", "gitjob"),
		newReadyDeployment("cattle-fleet-system", "fleet-controller"),
		newReadyDeployment("cattle-fleet-local-system", "fleet-agent"), namespace1).Build()
	ctx := spi.NewFakeContext(client, vzCR, nil, false)
	namespaceWatcher = NewNamespaceWatcher(ctx.Client(), period)
	projectID := "p-47cnm"
	err := namespaceWatcher.MoveSystemNamespacesToRancherSystemProject(projectID)
	asserts.NoError(t, err)
	ns := v1.Namespace{}
	asserts.NoError(t, client.Get(context.Background(), types.NamespacedName{Name: "verrazzano-system"}, &ns))
	asserts.NotEqual(t, ns.Annotations[RancherProjectIDLabelKey], "local:"+projectID)
}

// TestMoveSystemNamespaces tests the following cases
// GIVEN that rancher component is enabled and in ready state in Verrazzano installation
// When namespaces on the cluster has a label "verrazzano.io/namespace"
// And when namespaces on the cluster does not have a label management.cattle.io/system-namespace
// THEN the method retrieves the System project ID from the rancher
// And updates the namespace annotation and label with the Project ID.
func TestMoveSystemNamespaces(t *testing.T) {
	namespace1 := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "verrazzano-system",
			Labels: map[string]string{
				constants.VerrazzanoManagedKey: "verrazzano-system",
			},
		},
	}
	enabled := true
	var availability v1alpha1.ComponentAvailability
	availability = "Available"
	vzCR := &v1alpha1.Verrazzano{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "example",
		},
		Spec: v1alpha1.VerrazzanoSpec{
			Components: v1alpha1.ComponentSpec{
				Rancher: &v1alpha1.RancherComponent{
					Enabled: &enabled,
				},
			},
		},
		Status: v1alpha1.VerrazzanoStatus{
			Components: v1alpha1.ComponentStatusMap{
				"rancher": &v1alpha1.ComponentStatusDetails{
					Name:      "rancher",
					State:     "Ready",
					Available: &availability,
				},
			},
		},
	}

	testScheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "install.verrazzano.io/v1alpha1",
		Kind:    "Verrazzano",
		Version: "v1alpha1",
	}, &v1alpha1.Verrazzano{})
	config.TestProfilesDir = reldir
	defer func() { config.TestProfilesDir = "" }()
	client := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(vzCR,
		newReplicaSet("cattle-system", "rancher"),
		newReplicaSet("cattle-system", "rancher-webhook"),
		newReplicaSet("cattle-fleet-system", "gitjob"),
		newReplicaSet("cattle-fleet-system", "fleet-controller"),
		newReplicaSet("cattle-fleet-local-system", "fleet-agent"),
		newPod("cattle-system", "rancher"),
		newPod("cattle-system", "rancher-webhook"),
		newPod("cattle-fleet-system", "gitjob"),
		newPod("cattle-fleet-system", "fleet-controller"),
		newPod("cattle-fleet-local-system", "fleet-agent"),
		newReadyDeployment("cattle-system", "rancher"),
		newReadyDeployment("cattle-system", "rancher-webhook"),
		newReadyDeployment("cattle-fleet-system", "gitjob"),
		newReadyDeployment("cattle-fleet-system", "fleet-controller"),
		newReadyDeployment("cattle-fleet-local-system", "fleet-agent"), namespace1).Build()
	ctx := spi.NewFakeContext(client, vzCR, nil, false)
	namespaceWatcher = NewNamespaceWatcher(ctx.Client(), period)
	projectID := "p-47cnm"
	err := namespaceWatcher.MoveSystemNamespacesToRancherSystemProject(projectID)
	asserts.NoError(t, err)
	ns := v1.Namespace{}
	asserts.NoError(t, client.Get(context.Background(), types.NamespacedName{Name: "verrazzano-system"}, &ns))
	asserts.Equal(t, ns.Annotations[RancherProjectIDLabelKey], "local:"+projectID)
}

func newPod(namespace string, name string) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			Labels: map[string]string{
				"app":               name,
				"pod-template-hash": "95d8c5d97",
			},
		},
	}
}

func newReplicaSet(namespace string, name string) *appsv1.ReplicaSet {
	return &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   namespace,
			Name:        name + "-95d8c5d97",
			Annotations: map[string]string{"deployment.kubernetes.io/revision": "1"},
		},
	}
}

// Create a new deployment object for testing
func newReadyDeployment(namespace string, name string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			Labels:    map[string]string{"app": name},
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": name},
			},
		},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: 1,
			Replicas:          1,
			UpdatedReplicas:   1,
		},
	}
}
