// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestCreateAdminSecretIfNotExists(t *testing.T) {

}

func TestGetRancherIP(t *testing.T) {
	log := getLogger()
	ingresses := []v1.LoadBalancerIngress{
		{"ip", "hostname"},
	}
	objectMeta := metav1.ObjectMeta{
		Name:      ComponentName,
		Namespace: ComponentNamespace,
	}
	in := networking.Ingress{
		ObjectMeta: objectMeta,
		Status: networking.IngressStatus{
			LoadBalancer: v1.LoadBalancerStatus{
				Ingress: ingresses,
			},
		},
	}
	inNoIp := networking.Ingress{
		ObjectMeta: objectMeta,
	}

	var tests = []struct {
		testName string
		c        client.Client
		isErr    bool
	}{
		{
			"should be able to get an ingress ip",
			fake.NewFakeClientWithScheme(getScheme(), &in),
			false,
		},
		{
			"ingress should not be found",
			fake.NewFakeClientWithScheme(getScheme()),
			true,
		},
		{
			"ingress ip should not be found",
			fake.NewFakeClientWithScheme(getScheme(), &inNoIp),
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			ip, err := getRancherIngressIP(log, tt.c)
			if tt.isErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, "ip", ip)
			}
		})
	}
}
