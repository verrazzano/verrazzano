// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vmc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/thanos"
	istionet "istio.io/api/networking/v1beta1"
	istioclinet "istio.io/client-go/pkg/apis/networking/v1beta1"
	v1 "k8s.io/api/core/v1"
	k8sapiext "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
			cli := fake.NewClientBuilder().WithScheme(makeThanosTestScheme()).WithRuntimeObjects(
				makeThanosConfigMapWithExistingHosts(t, tt.existingHosts, tt.useValidCM),
				makeThanosEnabledVerrazzano(),
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
				assertThanosEndpointsConfigMap(ctx, t, cli, tt.expectNumHosts, tt.host, hostShouldExist)
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
			cli := fake.NewClientBuilder().WithScheme(makeThanosTestScheme()).WithRuntimeObjects(
				makeThanosConfigMapWithExistingHosts(t, tt.existingHosts, tt.useValidCM),
				makeThanosEnabledVerrazzano(),
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
				assertThanosEndpointsConfigMap(ctx, t, cli, tt.expectNumHosts, tt.host, hostShouldExist)
			}
		})
	}
}

// TestSyncThanosQuery tests the syncThanosQuery function which is the top level entry point
func TestSyncThanosQuery(t *testing.T) {
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
				ObjectMeta: metav1.ObjectMeta{Name: "somename", Namespace: constants.VerrazzanoMultiClusterNamespace},
				Status:     vmcStatus,
			}
			cli := fake.NewClientBuilder().WithScheme(makeThanosTestScheme()).WithRuntimeObjects(
				makeThanosConfigMapWithExistingHosts(t, tt.configMapExistingHosts, true),
				makeThanosEnabledVerrazzano(),
				&k8sapiext.CustomResourceDefinition{ObjectMeta: metav1.ObjectMeta{Name: serviceEntryCRDName}},
				&k8sapiext.CustomResourceDefinition{ObjectMeta: metav1.ObjectMeta{Name: destinationRuleCRDName}},
			).Build()
			r := &VerrazzanoManagedClusterReconciler{
				Client: cli,
				log:    log,
			}
			err := r.syncThanosQuery(ctx, vmc)
			assert.NoError(t, err)
			assertThanosEndpointsConfigMap(ctx, t, cli, tt.expectedConfigMapHosts, tt.hostname, tt.hostShouldExistInCM)
			if tt.hostShouldExistInCM {
				assertThanosServiceEntry(t, r, vmc.Name, tt.hostname, thanosGrpcIngressPort)
				assertThanosDestinationRule(t, r, vmc.Name, tt.hostname, thanosGrpcIngressPort)
			}
		})
	}
}

func makeThanosTestScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = v1beta1.AddToScheme(scheme)
	_ = v1.AddToScheme(scheme)
	istioclinet.AddToScheme(scheme)
	k8sapiext.AddToScheme(scheme)
	return scheme
}

func makeThanosEnabledVerrazzano() *v1beta1.Verrazzano {
	trueVal := true
	return &v1beta1.Verrazzano{
		Spec: v1beta1.VerrazzanoSpec{
			Components: v1beta1.ComponentSpec{
				Thanos: &v1beta1.ThanosComponent{Enabled: &trueVal},
			},
		},
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
		ObjectMeta: metav1.ObjectMeta{Namespace: thanos.ComponentNamespace, Name: ThanosManagedClusterEndpointsConfigMap},
		Data: map[string]string{
			serviceDiscoveryKey: string(yamlExistingHostInfo),
		},
	}
}

func assertThanosEndpointsConfigMap(ctx context.Context, t *testing.T, cli client.WithWatch, expectNumHosts int, host string, hostShoudExist bool) {
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

// TestCreateServiceEntry tests ServiceEntry creation scenarios.
func TestCreateServiceEntry(t *testing.T) {
	const managedClusterName = "managed-cluster"
	const host = "thanos-query.example.com"
	const port = uint32(443)

	log := vzlog.DefaultLogger()

	// GIVEN the CRD for Istio ServiceEntry does not exist in the cluster
	// WHEN  the createOrUpdateServiceEntry function is called
	// THEN  the call does not return an error and no ServiceEntry is created
	cli := fake.NewClientBuilder().WithScheme(makeThanosTestScheme()).WithRuntimeObjects().Build()
	r := &VerrazzanoManagedClusterReconciler{
		Client: cli,
		log:    log,
	}

	err := r.createOrUpdateServiceEntry(managedClusterName, host, port)
	assert.NoError(t, err)

	se := &istioclinet.ServiceEntry{}
	err = r.Client.Get(context.TODO(), client.ObjectKey{Namespace: constants.VerrazzanoMonitoringNamespace, Name: managedClusterName}, se)
	assert.True(t, k8serrors.IsNotFound(err))

	// GIVEN the CRD for Istio ServiceEntry exists in the cluster
	// AND   the ServiceEntry does not exist
	// WHEN  the createOrUpdateServiceEntry function is called
	// THEN  the call does not return an error and a ServiceEntry is created
	cli = fake.NewClientBuilder().WithScheme(makeThanosTestScheme()).WithRuntimeObjects(
		&k8sapiext.CustomResourceDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: serviceEntryCRDName,
			},
		},
	).Build()
	r = &VerrazzanoManagedClusterReconciler{
		Client: cli,
		log:    log,
	}

	err = r.createOrUpdateServiceEntry(managedClusterName, host, port)
	assert.NoError(t, err)
	assertThanosServiceEntry(t, r, managedClusterName, host, port)
}

// TestUpdateServiceEntry tests ServiceEntry update scenarios.
func TestUpdateServiceEntry(t *testing.T) {
	const managedClusterName = "managed-cluster"
	const host = "thanos-query.example.com"
	const port = uint32(443)

	log := vzlog.DefaultLogger()

	// GIVEN the ServiceEntry exists
	// WHEN  the createOrUpdateServiceEntry function is called
	// THEN  the call does not return an error and the ServiceEntry is updated
	se := &istioclinet.ServiceEntry{ObjectMeta: metav1.ObjectMeta{Name: managedClusterName, Namespace: constants.VerrazzanoMonitoringNamespace}}
	populateServiceEntry(se, "bad-bad-host", port)

	cli := fake.NewClientBuilder().WithScheme(makeThanosTestScheme()).WithRuntimeObjects(
		&k8sapiext.CustomResourceDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: serviceEntryCRDName,
			},
		},
		se,
	).Build()
	r := &VerrazzanoManagedClusterReconciler{
		Client: cli,
		log:    log,
	}

	err := r.createOrUpdateServiceEntry(managedClusterName, host, port)
	assert.NoError(t, err)

	assertThanosServiceEntry(t, r, managedClusterName, host, port)
	err = r.Client.Get(context.TODO(), client.ObjectKey{Namespace: constants.VerrazzanoMonitoringNamespace, Name: managedClusterName}, se)
	assert.NoError(t, err)
	assert.Contains(t, se.Spec.Hosts, host)
	assert.Equal(t, se.Spec.Resolution, istionet.ServiceEntry_DNS)
	assert.Equal(t, se.Spec.Ports[0].Number, port)
	assert.Equal(t, se.Spec.Ports[0].TargetPort, port)
	assert.Equal(t, se.Spec.Ports[0].Protocol, "GRPC")
}

// TestDeleteServiceEntry tests ServiceEntry deletion scenarios.
func TestDeleteServiceEntry(t *testing.T) {
	const managedClusterName = "managed-cluster"
	const host = "thanos-query.example.com"
	const port = uint32(443)

	log := vzlog.DefaultLogger()

	// GIVEN the CRD for Istio ServiceEntry does not exist in the cluster
	// WHEN  the deleteServiceEntry function is called
	// THEN  the call does not return an error
	cli := fake.NewClientBuilder().WithScheme(makeThanosTestScheme()).WithRuntimeObjects().Build()
	r := &VerrazzanoManagedClusterReconciler{
		Client: cli,
		log:    log,
	}

	err := r.deleteServiceEntry(managedClusterName)
	assert.NoError(t, err)

	// GIVEN the ServiceEntry exists
	// WHEN  the deleteServiceEntry function is called
	// THEN  the call does not return an error and the ServiceEntry is deleted
	se := &istioclinet.ServiceEntry{ObjectMeta: metav1.ObjectMeta{Name: managedClusterName, Namespace: constants.VerrazzanoMonitoringNamespace}}
	populateServiceEntry(se, host, port)

	cli = fake.NewClientBuilder().WithScheme(makeThanosTestScheme()).WithRuntimeObjects(
		&k8sapiext.CustomResourceDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: serviceEntryCRDName,
			},
		},
		se,
	).Build()
	r = &VerrazzanoManagedClusterReconciler{
		Client: cli,
		log:    log,
	}

	err = r.deleteServiceEntry(managedClusterName)
	assert.NoError(t, err)

	err = r.Client.Get(context.TODO(), client.ObjectKey{Namespace: constants.VerrazzanoMonitoringNamespace, Name: managedClusterName}, se)
	assert.True(t, k8serrors.IsNotFound(err))

	// GIVEN the ServiceEntry does not exist
	// WHEN  the deleteServiceEntry function is called
	// THEN  the call does not return an error
	err = r.deleteServiceEntry(managedClusterName)
	assert.NoError(t, err)
}

// TestCreateDestinationRule tests DestinationRule creation scenarios.
func TestCreateDestinationRule(t *testing.T) {
	const managedClusterName = "managed-cluster"
	const host = "thanos-query.example.com"
	const port = uint32(443)

	log := vzlog.DefaultLogger()

	// GIVEN the CRD for Istio DestinationRule does not exist in the cluster
	// WHEN  the createOrUpdateDestinationRule function is called
	// THEN  the call does not return an error and no DestinationRule is created
	cli := fake.NewClientBuilder().WithScheme(makeThanosTestScheme()).WithRuntimeObjects().Build()
	r := &VerrazzanoManagedClusterReconciler{
		Client: cli,
		log:    log,
	}

	err := r.createOrUpdateDestinationRule(managedClusterName, host, port)
	assert.NoError(t, err)

	dr := &istioclinet.DestinationRule{}
	err = r.Client.Get(context.TODO(), client.ObjectKey{Namespace: constants.VerrazzanoMonitoringNamespace, Name: managedClusterName}, dr)
	assert.True(t, k8serrors.IsNotFound(err))

	// GIVEN the CRD for Istio DestinationRule exists in the cluster
	// AND   the DestinationRule does not exist
	// WHEN  the createOrUpdateDestinationRule function is called
	// THEN  the call does not return an error and a DestinationRule is created
	cli = fake.NewClientBuilder().WithScheme(makeThanosTestScheme()).WithRuntimeObjects(
		&k8sapiext.CustomResourceDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: destinationRuleCRDName,
			},
		},
	).Build()
	r = &VerrazzanoManagedClusterReconciler{
		Client: cli,
		log:    log,
	}

	err = r.createOrUpdateDestinationRule(managedClusterName, host, port)
	assert.NoError(t, err)

	assertThanosDestinationRule(t, r, managedClusterName, host, port)
}

// TestUpdateDestinationRule tests DestinationRule update scenarios.
func TestUpdateDestinationRule(t *testing.T) {
	const managedClusterName = "managed-cluster"
	const host = "thanos-query.example.com"
	const port = uint32(thanosGrpcIngressPort)

	log := vzlog.DefaultLogger()

	// GIVEN the DestinationRule exists
	// WHEN  the createOrUpdateDestinationRule function is called
	// THEN  the call does not return an error and the DestinationRule is updated
	dr := &istioclinet.DestinationRule{ObjectMeta: metav1.ObjectMeta{Name: managedClusterName, Namespace: constants.VerrazzanoMonitoringNamespace}}
	populateDestinationRule(dr, "bad-bad-host", port)

	cli := fake.NewClientBuilder().WithScheme(makeThanosTestScheme()).WithRuntimeObjects(
		&k8sapiext.CustomResourceDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: destinationRuleCRDName,
			},
		},
		dr,
	).Build()
	r := &VerrazzanoManagedClusterReconciler{
		Client: cli,
		log:    log,
	}

	err := r.createOrUpdateDestinationRule(managedClusterName, host, port)
	assert.NoError(t, err)

	assertThanosDestinationRule(t, r, managedClusterName, host, port)
}

// TestDeleteDestinationRule tests DestinationRule deletion scenarios.
func TestDeleteDestinationRule(t *testing.T) {
	const managedClusterName = "managed-cluster"
	const host = "thanos-query.example.com"
	const port = uint32(443)

	log := vzlog.DefaultLogger()

	// GIVEN the CRD for Istio DestinationRule does not exist in the cluster
	// WHEN  the deleteDestinationRule function is called
	// THEN  the call does not return an error
	cli := fake.NewClientBuilder().WithScheme(makeThanosTestScheme()).WithRuntimeObjects().Build()
	r := &VerrazzanoManagedClusterReconciler{
		Client: cli,
		log:    log,
	}

	err := r.deleteDestinationRule(managedClusterName)
	assert.NoError(t, err)

	// GIVEN the DestinationRule exists
	// WHEN  the deleteDestinationRule function is called
	// THEN  the call does not return an error and the DestinationRule is deleted
	dr := &istioclinet.DestinationRule{ObjectMeta: metav1.ObjectMeta{Name: managedClusterName, Namespace: constants.VerrazzanoMonitoringNamespace}}
	populateDestinationRule(dr, host, port)

	cli = fake.NewClientBuilder().WithScheme(makeThanosTestScheme()).WithRuntimeObjects(
		&k8sapiext.CustomResourceDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: destinationRuleCRDName,
			},
		},
		dr,
	).Build()
	r = &VerrazzanoManagedClusterReconciler{
		Client: cli,
		log:    log,
	}

	err = r.deleteDestinationRule(managedClusterName)
	assert.NoError(t, err)

	err = r.Client.Get(context.TODO(), client.ObjectKey{Namespace: constants.VerrazzanoMonitoringNamespace, Name: managedClusterName}, dr)
	assert.True(t, k8serrors.IsNotFound(err))

	// GIVEN the DestinationRule does not exist
	// WHEN  the deleteDestinationRule function is called
	// THEN  the call does not return an error
	err = r.deleteDestinationRule(managedClusterName)
	assert.NoError(t, err)
}

func assertThanosServiceEntry(t *testing.T, r *VerrazzanoManagedClusterReconciler, managedClusterName string, host string, port uint32) {
	se := &istioclinet.ServiceEntry{}
	err := r.Client.Get(context.TODO(), client.ObjectKey{Namespace: constants.VerrazzanoMonitoringNamespace, Name: managedClusterName}, se)
	assert.NoError(t, err)
	assert.Contains(t, se.Spec.Hosts, host)
	assert.Equal(t, se.Spec.Resolution, istionet.ServiceEntry_DNS)
	assert.Equal(t, se.Spec.Ports[0].Number, port)
	assert.Equal(t, se.Spec.Ports[0].TargetPort, port)
	assert.Equal(t, se.Spec.Ports[0].Protocol, "GRPC")
}

func assertThanosDestinationRule(t *testing.T, r *VerrazzanoManagedClusterReconciler, clusterName string, hostName string, portNum uint32) {
	dr := &istioclinet.DestinationRule{}
	err := r.Client.Get(context.TODO(), client.ObjectKey{Namespace: constants.VerrazzanoMonitoringNamespace, Name: clusterName}, dr)
	assert.NoError(t, err)
	assert.Equal(t, dr.Spec.Host, hostName)
	assert.Equal(t, dr.Spec.TrafficPolicy.PortLevelSettings[0].Port.Number, portNum)
	assert.Equal(t, dr.Spec.TrafficPolicy.PortLevelSettings[0].Tls.Mode, istionet.ClientTLSSettings_SIMPLE)
}
