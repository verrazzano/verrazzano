// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package status

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestStatefulsetReady(t *testing.T) {
	notEnoughReplicas := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "foobar",
			Name:      "foo",
		},
		Status: appsv1.StatefulSetStatus{
			ReadyReplicas: 0,
		},
	}
	enoughReplicas := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "foobar",
			Name:      "foo",
		},
		Status: appsv1.StatefulSetStatus{
			ReadyReplicas: 1,
		},
	}
	oneName := []types.NamespacedName{
		{
			Name:      "foo",
			Namespace: "foobar",
		},
	}
	var noName []types.NamespacedName = nil
	var tests = []struct {
		name  string
		c     client.Client
		n     []types.NamespacedName
		ready bool
	}{
		{
			"should be not ready when statefulset not found",
			fake.NewFakeClientWithScheme(k8scheme.Scheme),
			oneName,
			false,
		},
		{
			"should be not ready when statefulset doesn't have enough ready replicas",
			fake.NewFakeClientWithScheme(k8scheme.Scheme, notEnoughReplicas),
			oneName,
			false,
		},
		{
			"should be ready when no statefulset provided",
			fake.NewFakeClientWithScheme(k8scheme.Scheme),
			noName,
			true,
		},
		{
			"should be ready statefulset has enough ready replicas",
			fake.NewFakeClientWithScheme(k8scheme.Scheme, enoughReplicas),
			oneName,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.ready, StatefulsetReady(vzlog.DefaultLogger(), tt.c, tt.n, 1, ""))
		})
	}
}
