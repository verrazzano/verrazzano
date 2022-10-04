// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package status

import (
	"testing"

	"github.com/verrazzano/verrazzano/pkg/log/vzlog"

	v1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestIngressesPresent(t *testing.T) {
	ingress := v1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "foobar",
		},
	}
	oneName := []types.NamespacedName{
		{
			Name:      "foo",
			Namespace: "foobar",
		},
	}
	multipleNames := append(oneName, types.NamespacedName{Name: "anotherIng", Namespace: "foobar"})

	var noName []types.NamespacedName
	var tests = []struct {
		name    string
		c       client.Client
		n       []types.NamespacedName
		present bool
	}{
		{
			"should be false when ingress not found",
			fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build(),
			oneName,
			false,
		},
		{
			"should be false when only some ingresses are found",
			fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(&ingress).Build(),
			multipleNames,
			false,
		},
		{
			"should be true when ingress exists",
			fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(&ingress).Build(),
			oneName,
			true,
		},
		{
			"should be true when no ingress names provided",
			fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build(),
			noName,
			true,
		},
		{
			"should be present when ingress exists",
			fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(&ingress).Build(),
			oneName,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if present := IngressesPresent(vzlog.DefaultLogger(), tt.c, tt.n, ""); present != tt.present {
				t.Errorf("IngressesPresent() = %v, want %v", present, tt.present)
			}
		})
	}
}
