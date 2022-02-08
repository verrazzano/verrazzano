// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package authproxy

import (
	"testing"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"

	globalconst "github.com/verrazzano/verrazzano/pkg/constants"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
)

// TestIsReady tests the AuthProxy IsReady call
// GIVEN a AuthProxy component
//  WHEN I call IsReady when all requirements are met
//  THEN true or false is returned
func TestIsReady(t *testing.T) {
	tests := []struct {
		name      string
		client    client.Client
		wantError bool
	}{
		{
			name: "Test IsReady when AuthProxy is successfully deployed",
			client: fake.NewFakeClientWithScheme(testScheme,
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: globalconst.VerrazzanoSystemNamespace,
						Name:      "verrazzano-authproxy",
					},
					Status: appsv1.DeploymentStatus{
						Replicas:            1,
						ReadyReplicas:       1,
						AvailableReplicas:   1,
						UnavailableReplicas: 0,
					},
				}),
			wantError: false,
		},
		{
			name: "Test IsReady when AuthProxy deployment is not ready",
			client: fake.NewFakeClientWithScheme(testScheme,
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: globalconst.VerrazzanoSystemNamespace,
						Name:      "verrazzano-authproxy",
					},
					Status: appsv1.DeploymentStatus{
						Replicas:            1,
						ReadyReplicas:       1,
						AvailableReplicas:   0,
						UnavailableReplicas: 1,
					},
				}),
			wantError: true,
		},
		{
			name:      "Test IsReady when AuthProxy deployment does not exist",
			client:    fake.NewFakeClientWithScheme(testScheme),
			wantError: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := spi.NewFakeContext(tt.client, &vzapi.Verrazzano{}, false)
			if tt.wantError {
				assert.False(t, NewComponent().IsReady(ctx))
			} else {
				assert.True(t, NewComponent().IsReady(ctx))
			}
		})
	}
}
