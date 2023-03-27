// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vmc

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/thanos"
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"
)

type syncThanosTestType struct {
	name           string
	host           string
	existingHosts  []string
	expectError    bool
	expectNumHosts int
	hostShoudExist bool
}

func TestVerrazzanoManagedClusterReconciler_addThanosHostIfNotPresent(t *testing.T) {
	tests := []syncThanosTestType{
		// TODO: Add test cases.
		{"no existing hosts", "newhostname", []string{}, false, 1, true},
		{"host already exists", "newhostname", []string{"otherhost", "newhostname"}, false, 2, true},
		{"host does not exist", "newhostname", []string{"otherhost", "yetanotherhost"}, false, 3, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := vzlog.DefaultLogger()
			ctx := context.TODO()
			// existingHostInfo := thanosServiceDiscovery{targets: tt.existingHosts}
			existingHostInfo := map[string]interface{}{"targets": []string{}}
			yamlExistingHostInfo, err := yaml.Marshal(existingHostInfo)
			assert.NoError(t, err)
			cli := fake.NewClientBuilder().WithRuntimeObjects(
				&v1.ConfigMap{
					ObjectMeta: v12.ObjectMeta{Namespace: thanos.ComponentNamespace, Name: ThanosManagedClusterEndpointsConfigMap},
					Data: map[string]string{
						serviceDiscoveryKey: string(yamlExistingHostInfo),
					},
				},
			).Build()
			r := &VerrazzanoManagedClusterReconciler{
				Client: cli,
				log:    log,
			}
			err = r.addThanosHostIfNotPresent(ctx, tt.host, log)
			if tt.expectError {
				assert.Error(t, err, "Expected error")
			} else {
				modifiedConfigMap := &v1.ConfigMap{}
				err = cli.Get(ctx, types.NamespacedName{Namespace: thanos.ComponentNamespace, Name: ThanosManagedClusterEndpointsConfigMap}, modifiedConfigMap)
				assert.NoError(t, err)
				modifiedContent := thanosServiceDiscovery{}
				err = yaml.Unmarshal([]byte(modifiedConfigMap.Data[serviceDiscoveryKey]), &modifiedContent)
				assert.NoError(t, err)
				assert.Equalf(t, tt.expectNumHosts, len(modifiedContent.targets), "Expected %d hosts in modified config map", tt.expectNumHosts)
				if tt.hostShoudExist {
					assert.Contains(t, modifiedContent.targets, toGrpcTarget(tt.host))
				}
			}
		})
	}
}

func TestVerrazzanoManagedClusterReconciler_getThanosEndpointsConfigMap(t *testing.T) {
	type fields struct {
		Client client.Client
		Scheme *runtime.Scheme
		log    vzlog.VerrazzanoLogger
	}
	type args struct {
		ctx context.Context
		log vzlog.VerrazzanoLogger
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *v1.ConfigMap
		wantErr assert.ErrorAssertionFunc
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &VerrazzanoManagedClusterReconciler{
				Client: tt.fields.Client,
				Scheme: tt.fields.Scheme,
				log:    tt.fields.log,
			}
			got, err := r.getThanosEndpointsConfigMap(tt.args.ctx, tt.args.log)
			if !tt.wantErr(t, err, fmt.Sprintf("getThanosEndpointsConfigMap(%v, %v)", tt.args.ctx, tt.args.log)) {
				return
			}
			assert.Equalf(t, tt.want, got, "getThanosEndpointsConfigMap(%v, %v)", tt.args.ctx, tt.args.log)
		})
	}
}

func TestVerrazzanoManagedClusterReconciler_removeThanosHostFromConfigMap(t *testing.T) {
	type fields struct {
		Client client.Client
		Scheme *runtime.Scheme
		log    vzlog.VerrazzanoLogger
	}
	type args struct {
		ctx  context.Context
		host string
		log  vzlog.VerrazzanoLogger
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &VerrazzanoManagedClusterReconciler{
				Client: tt.fields.Client,
				Scheme: tt.fields.Scheme,
				log:    tt.fields.log,
			}
			tt.wantErr(t, r.removeThanosHostFromConfigMap(tt.args.ctx, tt.args.host, tt.args.log), fmt.Sprintf("removeThanosHostFromConfigMap(%v, %v, %v)", tt.args.ctx, tt.args.host, tt.args.log))
		})
	}
}

func TestVerrazzanoManagedClusterReconciler_syncThanosQueryEndpoint(t *testing.T) {
	type fields struct {
		Client client.Client
		Scheme *runtime.Scheme
		log    vzlog.VerrazzanoLogger
	}
	type args struct {
		ctx context.Context
		vmc *clustersv1alpha1.VerrazzanoManagedCluster
		log vzlog.VerrazzanoLogger
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &VerrazzanoManagedClusterReconciler{
				Client: tt.fields.Client,
				Scheme: tt.fields.Scheme,
				log:    tt.fields.log,
			}
			tt.wantErr(t, r.syncThanosQueryEndpoint(tt.args.ctx, tt.args.vmc, tt.args.log), fmt.Sprintf("syncThanosQueryEndpoint(%v, %v, %v)", tt.args.ctx, tt.args.vmc, tt.args.log))
		})
	}
}

func Test_findHost(t *testing.T) {
	type args struct {
		serviceDiscovery *thanosServiceDiscovery
		host             string
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, findHost(tt.args.serviceDiscovery, tt.args.host), "findHost(%v, %v)", tt.args.serviceDiscovery, tt.args.host)
		})
	}
}

func Test_parseThanosEndpointsConfigMap(t *testing.T) {
	type args struct {
		configMap *v1.ConfigMap
		log       vzlog.VerrazzanoLogger
	}
	tests := []struct {
		name    string
		args    args
		want    *thanosServiceDiscovery
		wantErr assert.ErrorAssertionFunc
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseThanosEndpointsConfigMap(tt.args.configMap, tt.args.log)
			if !tt.wantErr(t, err, fmt.Sprintf("parseThanosEndpointsConfigMap(%v, %v)", tt.args.configMap, tt.args.log)) {
				return
			}
			assert.Equalf(t, tt.want, got, "parseThanosEndpointsConfigMap(%v, %v)", tt.args.configMap, tt.args.log)
		})
	}
}

func Test_toGrpcTarget(t *testing.T) {
	type args struct {
		hostname string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, toGrpcTarget(tt.args.hostname), "toGrpcTarget(%v)", tt.args.hostname)
		})
	}
}
