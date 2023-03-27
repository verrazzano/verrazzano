// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vmc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/thanos"
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"
)

type addRemoveSyncThanosTestType struct {
	name           string
	host           string
	existingHosts  []string
	expectError    bool
	expectNumHosts int
	useValidCM     bool
}

func TestAddThanosHostIfNotPresent(t *testing.T) {
	newHostName := "newhostname"
	otherHost1 := toGrpcTarget("otherhost1")
	otherHost2 := toGrpcTarget("otherhost2")
	newHost := toGrpcTarget(newHostName)
	tests := []addRemoveSyncThanosTestType{
		{"no existing hosts", newHostName, []string{}, false, 1, true},
		{"host already exists", newHostName, []string{otherHost1, newHost}, false, 2, true},
		{"host does not exist", newHostName, []string{otherHost1, otherHost2}, false, 3, true},
		{"existing ConfigMap is malformed", newHostName, []string{}, false, 1, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := vzlog.DefaultLogger()
			ctx := context.TODO()
			cli := fake.NewClientBuilder().WithRuntimeObjects(
				makeThanosConfigMapWithExistingHosts(t, tt.existingHosts, tt.useValidCM),
			).Build()
			r := &VerrazzanoManagedClusterReconciler{
				Client: cli,
				log:    log,
			}
			err := r.addThanosHostIfNotPresent(ctx, tt.host)
			if tt.expectError {
				assert.Error(t, err, "Expected error")
			} else {
				hostShouldExist := true
				assertThanosEndpointsConfigMap(t, cli, ctx, tt.expectNumHosts, tt.host, hostShouldExist)
			}
		})
	}
}

func TestRemoveThanosHostFromConfigMap(t *testing.T) {
	newHostName := "newhostname"
	otherHost1 := toGrpcTarget("otherhost1")
	otherHost2 := toGrpcTarget("otherhost2")
	newHost := toGrpcTarget(newHostName)
	tests := []addRemoveSyncThanosTestType{
		{"no existing hosts", newHostName, []string{}, false, 0, true},
		{"host already exists", newHostName, []string{otherHost1, newHost}, false, 1, true},
		{"host does not exist", newHostName, []string{otherHost1, otherHost2}, false, 2, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := vzlog.DefaultLogger()
			ctx := context.TODO()
			cli := fake.NewClientBuilder().WithRuntimeObjects(
				makeThanosConfigMapWithExistingHosts(t, tt.existingHosts, false),
			).Build()
			r := &VerrazzanoManagedClusterReconciler{
				Client: cli,
				log:    log,
			}
			err := r.removeThanosHostFromConfigMap(ctx, tt.host, log)
			if tt.expectError {
				assert.Error(t, err, "Expected error")
			} else {
				hostShouldExist := false
				assertThanosEndpointsConfigMap(t, cli, ctx, tt.expectNumHosts, tt.host, hostShouldExist)
			}
		})
	}
}

func TestSyncThanosQueryEndpoint(t *testing.T) {
	newHostName := "newhostname"
	otherHost1 := toGrpcTarget("otherhost1")
	otherHost2 := toGrpcTarget("otherhost2")
	newHost := toGrpcTarget(newHostName)
	tests := []struct {
		name                   string
		vmcStatus              *clustersv1alpha1.VerrazzanoManagedClusterStatus
		expectedConfigMapHosts int
		hostname               string
		configMapExistingHosts []string
		hostShouldExistInCM    bool
	}{
		{"VMC status empty", nil, 1, "", []string{otherHost1}, false},
		{"VMC status has no Thanos host",
			&clustersv1alpha1.VerrazzanoManagedClusterStatus{APIUrl: "someurl"},
			1,
			"",
			[]string{otherHost1},
			false,
		},
		{"VMC status has existing Thanos host",
			&clustersv1alpha1.VerrazzanoManagedClusterStatus{APIUrl: "someurl", ThanosHost: newHostName},
			2,
			newHostName,
			[]string{newHost, otherHost1},
			true, // new host already exists in query endpoints configmap, should still exist
		},
		{"VMC status has non-existing Thanos host",
			&clustersv1alpha1.VerrazzanoManagedClusterStatus{APIUrl: "someurl", ThanosHost: newHostName},
			3,
			newHostName,
			[]string{otherHost1, otherHost2},
			true, // new host should be added to query endpoints configmap
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := vzlog.DefaultLogger()
			ctx := context.TODO()
			var vmcStatus clustersv1alpha1.VerrazzanoManagedClusterStatus
			if tt.vmcStatus != nil {
				vmcStatus = *tt.vmcStatus
			}
			vmc := &clustersv1alpha1.VerrazzanoManagedCluster{
				ObjectMeta: v12.ObjectMeta{Name: "somename", Namespace: constants.VerrazzanoMultiClusterNamespace},
				Status:     vmcStatus,
			}
			cli := fake.NewClientBuilder().WithRuntimeObjects(
				makeThanosConfigMapWithExistingHosts(t, tt.configMapExistingHosts, false),
			).Build()
			r := &VerrazzanoManagedClusterReconciler{
				Client: cli,
				log:    log,
			}
			err := r.syncThanosQueryEndpoint(ctx, vmc)
			assert.NoError(t, err)
			assertThanosEndpointsConfigMap(t, cli, ctx, tt.expectedConfigMapHosts, tt.hostname, tt.hostShouldExistInCM)
		})
	}
}

func makeThanosConfigMapWithExistingHosts(t *testing.T, hosts []string, useValidConfigMap bool) *v1.ConfigMap {
	var yamlExistingHostInfo []byte
	var err error
	if useValidConfigMap {
		existingHostInfo := []*thanosServiceDiscovery{
			{
				Targets: hosts,
			},
		}
		yamlExistingHostInfo, err = yaml.Marshal(existingHostInfo)
		assert.NoError(t, err)
	} else {
		yamlExistingHostInfo = []byte("- targets: garbledTextHere")
	}
	return &v1.ConfigMap{
		ObjectMeta: v12.ObjectMeta{Namespace: thanos.ComponentNamespace, Name: ThanosManagedClusterEndpointsConfigMap},
		Data: map[string]string{
			serviceDiscoveryKey: string(yamlExistingHostInfo),
		},
	}
}

func assertThanosEndpointsConfigMap(t *testing.T, cli client.WithWatch, ctx context.Context, expectNumHosts int, host string, hostShoudExist bool) {
	modifiedConfigMap := &v1.ConfigMap{}
	err := cli.Get(ctx, types.NamespacedName{Namespace: thanos.ComponentNamespace, Name: ThanosManagedClusterEndpointsConfigMap}, modifiedConfigMap)
	assert.NoError(t, err)
	// make sure "targets" element is serialized in lower case in the config map
	assert.Contains(t, modifiedConfigMap.Data[serviceDiscoveryKey], "targets")
	modifiedContent := []*thanosServiceDiscovery{}
	err = yaml.Unmarshal([]byte(modifiedConfigMap.Data[serviceDiscoveryKey]), &modifiedContent)
	assert.NoError(t, err)
	// for now we are only testing with a single service discovery entry with zero or more Targets
	assert.Len(t, modifiedContent, 1)
	assert.Equalf(t, expectNumHosts, len(modifiedContent[0].Targets), "Expected %d hosts in modified config map", expectNumHosts)
	if hostShoudExist {
		assert.Contains(t, modifiedContent[0].Targets, toGrpcTarget(host))
	}
}
