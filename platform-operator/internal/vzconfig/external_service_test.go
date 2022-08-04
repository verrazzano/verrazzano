// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vzconfig

import (
	asserts "github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	fakes "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestGetExternalIP(t *testing.T) {
	assert := asserts.New(t)

	svcName := "foo"
	svcNamespace := "bar"
	ipaddr := "0.0.0.0"
	svcNoIP := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      svcName,
			Namespace: svcNamespace,
		},
	}

	svcIPExt := svcNoIP.DeepCopy()
	svcIPExt.Spec.ExternalIPs = []string{ipaddr}

	svcIPStatus := svcNoIP.DeepCopy()
	svcIPStatus.Status.LoadBalancer.Ingress = []v1.LoadBalancerIngress{
		{
			IP: ipaddr,
		},
	}

	tests := []struct {
		name         string
		ingType      vzapi.IngressType
		svcName      string
		svcNamespace string
		service      *v1.Service
		expectError  bool
	}{
		{
			name:         "test empty",
			ingType:      vzapi.LoadBalancer,
			svcName:      svcName,
			svcNamespace: svcNamespace,
			service:      &v1.Service{},
			expectError:  true,
		},
		{
			name:         "test standard service",
			ingType:      vzapi.LoadBalancer,
			svcName:      svcName,
			svcNamespace: svcNamespace,
			service:      svcNoIP,
			expectError:  true,
		},
		{
			name:         "test service external IP",
			ingType:      vzapi.LoadBalancer,
			svcName:      svcName,
			svcNamespace: svcNamespace,
			service:      svcIPExt,
			expectError:  false,
		},
		{
			name:         "test service status IP",
			ingType:      vzapi.NodePort,
			svcName:      svcName,
			svcNamespace: svcNamespace,
			service:      svcIPStatus,
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			cli := fakes.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(tt.service).Build()
			ip, err := GetExternalIP(cli, tt.ingType, tt.svcName, tt.svcNamespace)
			if tt.expectError {
				assert.Error(err)
				return
			}
			assert.NoError(err)
			assert.Equal(ip, ipaddr)
		})
	}
}
