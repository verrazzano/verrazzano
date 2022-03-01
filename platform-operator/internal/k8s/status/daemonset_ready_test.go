// Copyright (c) 2022, Oracle and/or its affiliates.
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

func TestDaemonSetsReady(t *testing.T) {
	notEnoughNodes := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "foobar",
			Name:      "foo",
		},
		Status: appsv1.DaemonSetStatus{
			NumberAvailable: 0,
		},
	}
	enoughNodes := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "foobar",
			Name:      "foo",
		},
		Status: appsv1.DaemonSetStatus{
			NumberAvailable: 1,
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
			"should be not ready when daemonset not found",
			fake.NewFakeClientWithScheme(k8scheme.Scheme),
			oneName,
			false,
		},
		{
			"should be not ready when daemonset doesn't have enough ready nodes",
			fake.NewFakeClientWithScheme(k8scheme.Scheme, notEnoughNodes),
			oneName,
			false,
		},
		{
			"should be ready when no daemonset provided",
			fake.NewFakeClientWithScheme(k8scheme.Scheme),
			noName,
			true,
		},
		{
			"should be ready daemonset has enough ready nodes",
			fake.NewFakeClientWithScheme(k8scheme.Scheme, enoughNodes),
			oneName,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.ready, DaemonSetsReady(vzlog.DefaultLogger(), tt.c, tt.n, 1, ""))
		})
	}
}
