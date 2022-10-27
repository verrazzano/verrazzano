// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ready

import (
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestIsComponentAvailable(t *testing.T) {

	const (
		zeroReplicas = 0
		oneReplica   = 1
		name         = "foo"
		namespace    = "bar"
	)
	emptyClient := fake.NewClientBuilder().WithScheme(getScheme()).Build()
	selectors := []clipkg.ListOption{
		clipkg.InNamespace(namespace),
	}
	nsn := types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
	objectMeta := metav1.ObjectMeta{
		Name:      name,
		Namespace: namespace,
	}
	readyAndAvailableClient := fake.NewClientBuilder().WithScheme(getScheme()).
		WithObjects(&appsv1.Deployment{
			ObjectMeta: objectMeta,
			Status: appsv1.DeploymentStatus{
				Replicas:      oneReplica,
				ReadyReplicas: oneReplica,
			},
		}, &appsv1.StatefulSet{
			ObjectMeta: objectMeta,
			Status: appsv1.StatefulSetStatus{
				Replicas:      oneReplica,
				ReadyReplicas: oneReplica,
			},
		}, &appsv1.DaemonSet{
			ObjectMeta: objectMeta,
			Status: appsv1.DaemonSetStatus{
				NumberReady:            oneReplica,
				DesiredNumberScheduled: oneReplica,
			},
		}).Build()
	unreadyClient := fake.NewClientBuilder().WithScheme(getScheme()).
		WithObjects(&appsv1.Deployment{
			ObjectMeta: objectMeta,
			Status: appsv1.DeploymentStatus{
				Replicas:      oneReplica,
				ReadyReplicas: zeroReplicas,
			},
		}, &appsv1.StatefulSet{
			ObjectMeta: objectMeta,
			Status: appsv1.StatefulSetStatus{
				Replicas:      oneReplica,
				ReadyReplicas: zeroReplicas,
			},
		}, &appsv1.DaemonSet{
			ObjectMeta: objectMeta,
			Status: appsv1.DaemonSetStatus{
				DesiredNumberScheduled: oneReplica,
				NumberReady:            zeroReplicas,
			},
		}).Build()
	var tests = []struct {
		name      string
		ao        *AvailabilityObjects
		client    clipkg.Client
		available bool
	}{
		{
			"available when no objects",
			&AvailabilityObjects{},
			emptyClient,
			true,
		},
		{
			"unavailable when deploy not present",
			&AvailabilityObjects{
				DeploymentNames: []types.NamespacedName{nsn},
			},
			emptyClient,
			false,
		},
		{
			"unavailable when sts not present",
			&AvailabilityObjects{
				StatefulsetNames: []types.NamespacedName{nsn},
			},
			emptyClient,
			false,
		},
		{
			"unavailable when ds not present",
			&AvailabilityObjects{
				DaemonsetNames: []types.NamespacedName{nsn},
			},
			emptyClient,
			false,
		},
		{
			"unavailable when deploy replicas not ready",
			&AvailabilityObjects{
				DeploymentNames: []types.NamespacedName{nsn},
			},
			unreadyClient,
			false,
		},
		{
			"unavailable when sts replicas not ready",
			&AvailabilityObjects{
				StatefulsetNames: []types.NamespacedName{nsn},
			},
			unreadyClient,
			false,
		},
		{
			"unavailable when ds replicas not ready",
			&AvailabilityObjects{
				DaemonsetNames: []types.NamespacedName{nsn},
			},
			unreadyClient,
			false,
		},
		{
			"available when all objects present",
			&AvailabilityObjects{
				DeploymentNames:  []types.NamespacedName{nsn},
				StatefulsetNames: []types.NamespacedName{nsn},
				DaemonsetNames:   []types.NamespacedName{nsn},
			},
			readyAndAvailableClient,
			true,
		},
		{
			"(selectors) unavailable when deployment not ready",
			&AvailabilityObjects{
				DeploymentSelectors: selectors,
			},
			unreadyClient,
			false,
		},
		{
			"(selectors) unavailable when deployment not found",
			&AvailabilityObjects{
				DeploymentSelectors: selectors,
			},
			emptyClient,
			false,
		},
		{
			"(selectors) available when all objects present",
			&AvailabilityObjects{
				DeploymentSelectors: selectors,
			},
			readyAndAvailableClient,
			true,
		},
	}

	log := vzlog.DefaultLogger()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, available := tt.ao.IsAvailable(log, tt.client)
			assert.Equal(t, tt.available, available)
		})
	}
}
