// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"testing"

	asserts "github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestPostUninstall tests the post uninstall process for Rancher
// GIVEN a call to postUninstall
// WHEN the objects exist in the cluster
// THEN no error is returned and all objects are deleted
func TestPostUninstall(t *testing.T) {
	assert := asserts.New(t)
	vz := v1alpha1.Verrazzano{}

	nonRanNSName := "not-rancher"
	rancherNSName := "cattle-system"
	rancherNSName2 := "fleet-rancher"

	nonRancherNs := v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: nonRanNSName,
		},
	}
	rancherNs := v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: rancherNSName,
		},
	}
	rancherNs2 := v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: rancherNSName2,
		},
	}

	tests := []struct {
		name    string
		objects []clipkg.Object
	}{
		{
			name: "test empty cluster",
		},
		{
			name: "test non Rancher ns",
			objects: []clipkg.Object{
				&nonRancherNs,
			},
		},
		{
			name: "test Rancher ns",
			objects: []clipkg.Object{
				&nonRancherNs,
				&rancherNs,
			},
		},
		{
			name: "test multiple Rancher ns",
			objects: []clipkg.Object{
				&nonRancherNs,
				&rancherNs,
				&rancherNs2,
			},
		},
	}
	setRancherSystemTool("echo")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().WithScheme(getScheme()).WithObjects(tt.objects...).Build()
			ctx := spi.NewFakeContext(c, &vz, false)
			err := postUninstall(ctx)
			assert.NoError(err)
		})
	}
}
