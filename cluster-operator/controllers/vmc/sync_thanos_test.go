// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vmc

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/metricsutils"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/thanos"
	istionet "istio.io/api/networking/v1beta1"
	istioclinet "istio.io/client-go/pkg/apis/networking/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
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
	name              string
	clusterNumToCheck int
	numClusters       int
	expectError       bool
	expectNumHosts    int
	changedHost       *string
	useValidCM        bool
}

func TestAddThanosHostIfNotPresent(t *testing.T) {
	vmcPrefix := "cluster"
	host := "test-host"
	newHost := "altered-host"
	tests := []addRemoveSyncThanosTestType{
		{"no existing VMC", 1, 0, false, 1, nil, true},
		{"VMC already exists", 2, 2, false, 2, nil, true},
		{"VMC already exists host changed", 2, 2, false, 2, &newHost, true},
		{"VMC does not exist", 3, 2, false, 3, nil, true},
		{"existing ConfigMap is malformed", 1, 0, false, 1, nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := vzlog.DefaultLogger()
			ctx := context.TODO()
			effectiveHost := host
			cli := fake.NewClientBuilder().WithScheme(makeThanosTestScheme()).WithRuntimeObjects(
				makeThanosConfigMapWithExistingHosts(t, tt.useValidCM, tt.numClusters, toGrpcTarget(effectiveHost), vmcPrefix),
				makeThanosEnabledVerrazzano(),
			).Build()
			r := &VerrazzanoManagedClusterReconciler{
				Client: cli,
				log:    log,
			}
			vmcName := fmt.Sprintf("%s%d", vmcPrefix, tt.clusterNumToCheck)
			if tt.changedHost != nil {
				effectiveHost = *tt.changedHost
			}
			err := r.addThanosHostIfNotPresent(ctx, effectiveHost, vmcName)
			if tt.expectError {
				assert.Error(t, err, "Expected error")
			} else {
				clusterShouldExist := true
				assertThanosEndpointsConfigMap(ctx, t, cli, tt.expectNumHosts, toGrpcTarget(effectiveHost), vmcName, clusterShouldExist)
			}
		})
	}
}

func TestRemoveThanosHostFromConfigMap(t *testing.T) {
	vmcPrefix := "cluster"
	hostName := toGrpcTarget("test-host")
	tests := []addRemoveSyncThanosTestType{
		{"no existing hosts", 0, 0, false, 0, nil, true},
		{"host already exists", 2, 2, false, 1, nil, true},
		{"host does not exist", 3, 2, false, 2, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := vzlog.DefaultLogger()
			ctx := context.TODO()
			cli := fake.NewClientBuilder().WithScheme(makeThanosTestScheme()).WithRuntimeObjects(
				makeThanosConfigMapWithExistingHosts(t, tt.useValidCM, tt.numClusters, hostName, vmcPrefix),
				makeThanosEnabledVerrazzano(),
			).Build()
			r := &VerrazzanoManagedClusterReconciler{
				Client: cli,
				log:    log,
			}
			vmcName := fmt.Sprintf("%s%d", vmcPrefix, tt.clusterNumToCheck)
			err := r.removeThanosHostFromConfigMap(ctx, vmcName, log)
			if tt.expectError {
				assert.Error(t, err, "Expected error")
			} else {
				clusterShouldExist := false
				assertThanosEndpointsConfigMap(ctx, t, cli, tt.expectNumHosts, hostName, vmcName, clusterShouldExist)
			}
		})
	}
}

// TestSyncThanosQuery tests the syncThanosQuery function which is the top level entry point
func TestSyncThanosQuery(t *testing.T) {
	hostName := "test-host"
	vmcPrefix := "cluster"
	tests := []struct {
		name                   string
		vmcStatus              *clustersv1alpha1.VerrazzanoManagedClusterStatus
		expectedConfigMapHosts int
		numClusters            int
		clusterToSync          int
		clusterShouldExistInCM bool
		prometheusConfig       *corev1.Secret
	}{
		{"VMC status empty", nil, 1, 1, 1, false, nil},
		{"VMC status has no Thanos host",
			&clustersv1alpha1.VerrazzanoManagedClusterStatus{APIUrl: "someurl"},
			1,
			1,
			1,
			false,
			nil,
		},
		{"VMC status has existing VMC",
			&clustersv1alpha1.VerrazzanoManagedClusterStatus{APIUrl: "someurl", ThanosQueryStore: hostName},
			2,
			2,
			1,
			true, // new host already exists in query endpoints configmap, should still exist
			nil,
		},
		{"VMC status has non-existing Thanos host",
			&clustersv1alpha1.VerrazzanoManagedClusterStatus{APIUrl: "someurl", ThanosQueryStore: hostName},
			3,
			2,
			3,
			true, // new host should be added to query endpoints configmap
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := vzlog.DefaultLogger()
			ctx := context.TODO()
			var vmcStatus clustersv1alpha1.VerrazzanoManagedClusterStatus
			thanosHost := ""
			if tt.vmcStatus != nil {
				vmcStatus = *tt.vmcStatus
				thanosHost = vmcStatus.ThanosQueryStore
			}
			vmc := &clustersv1alpha1.VerrazzanoManagedCluster{
				ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s%d", vmcPrefix, tt.clusterToSync), Namespace: constants.VerrazzanoMultiClusterNamespace},
				Status:     vmcStatus,
			}
			cliBuilder := fake.NewClientBuilder().WithScheme(makeThanosTestScheme()).WithRuntimeObjects(
				makeThanosConfigMapWithExistingHosts(t, true, tt.numClusters, thanosHost, vmcPrefix),
				makeThanosEnabledVerrazzano(),
				&k8sapiext.CustomResourceDefinition{ObjectMeta: metav1.ObjectMeta{Name: serviceEntryCRDName}},
				&k8sapiext.CustomResourceDefinition{ObjectMeta: metav1.ObjectMeta{Name: destinationRuleCRDName}},
			)
			if tt.prometheusConfig != nil {
				cliBuilder = cliBuilder.WithObjects(tt.prometheusConfig)
			}
			cli := cliBuilder.Build()
			r := &VerrazzanoManagedClusterReconciler{
				Client: cli,
				log:    log,
			}
			err := r.syncThanosQuery(ctx, vmc)
			assert.NoError(t, err)
			assertThanosEndpointsConfigMap(ctx, t, cli, tt.expectedConfigMapHosts, toGrpcTarget(thanosHost), vmc.Name, tt.clusterShouldExistInCM)
			if tt.clusterShouldExistInCM {
				assertThanosServiceEntry(t, r, vmc.Name, hostName, thanosGrpcIngressPort)
				assertThanosDestinationRule(t, r, vmc.Name, hostName, thanosGrpcIngressPort)
			}
			if tt.prometheusConfig != nil {
				assertAdditionalScrapeConfigRemoved(t, r, vmc.Name)
			}
		})
	}
}

// TestDeleteClusterThanosEndpoint tests the deleteClusterThanosEndpoint function.
func TestDeleteClusterThanosEndpoint(t *testing.T) {
	vmcPrefix := "managed"
	const managedClusterName = "managed1"
	const hostName = "thanos-query.example.com"
	host := toGrpcTarget(hostName)

	vmcStatus := clustersv1alpha1.VerrazzanoManagedClusterStatus{APIUrl: "someurl", ThanosQueryStore: hostName}
	vmc := &clustersv1alpha1.VerrazzanoManagedCluster{
		ObjectMeta: metav1.ObjectMeta{Name: managedClusterName, Namespace: constants.VerrazzanoMultiClusterNamespace},
		Status:     vmcStatus,
	}
	cli := fake.NewClientBuilder().WithScheme(makeThanosTestScheme()).WithRuntimeObjects(
		makeThanosConfigMapWithExistingHosts(t, true, 1, host, vmcPrefix),
		makeThanosEnabledVerrazzano(),
		&k8sapiext.CustomResourceDefinition{ObjectMeta: metav1.ObjectMeta{Name: serviceEntryCRDName}},
		&k8sapiext.CustomResourceDefinition{ObjectMeta: metav1.ObjectMeta{Name: destinationRuleCRDName}},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      constants.PromManagedClusterCACertsSecretName,
				Namespace: constants.VerrazzanoMonitoringNamespace,
			},
			Data: map[string][]byte{
				"ca-" + managedClusterName:      []byte("ca-cert-1"),
				"ca-some-other-managed-cluster": []byte("ca-cert-2"),
			},
		},
	).Build()

	r := &VerrazzanoManagedClusterReconciler{
		Client: cli,
		log:    vzlog.DefaultLogger(),
	}

	// first sync to update endpoint configmap, add CA cert volume and volume mount, create ServiceEntry and
	// DestinationRule
	err := r.syncThanosQuery(context.TODO(), vmc)
	assert.NoError(t, err)

	assertThanosServiceEntry(t, r, vmc.Name, hostName, thanosGrpcIngressPort)
	assertThanosDestinationRule(t, r, vmc.Name, hostName, thanosGrpcIngressPort)

	// make sure the volume annotations have been added to the deployment
	queryDeploy := &appsv1.Deployment{}
	err = cli.Get(context.TODO(), client.ObjectKey{Namespace: constants.VerrazzanoMonitoringNamespace, Name: thanosQueryDeployName}, queryDeploy)
	assert.NoError(t, err)

	assert.Contains(t, queryDeploy.Spec.Template.ObjectMeta.Annotations, istioVolumeAnnotation)
	assert.Contains(t, queryDeploy.Spec.Template.ObjectMeta.Annotations, istioVolumeMountAnnotation)

	// GIVEN we have sync'ed a managed cluster Thanos endpoint
	// WHEN we call deleteClusterThanosEndpoint
	// THEN the resources we created during sync are cleaned up
	err = r.syncThanosQueryEndpointDelete(context.TODO(), vmc)
	assert.NoError(t, err)

	// ServiceEntry and DestinationRule should be gone
	se := &istioclinet.ServiceEntry{}
	err = r.Client.Get(context.TODO(), client.ObjectKey{Namespace: constants.VerrazzanoMonitoringNamespace, Name: managedClusterName}, se)
	assert.True(t, k8serrors.IsNotFound(err))

	dr := &istioclinet.DestinationRule{}
	err = r.Client.Get(context.TODO(), client.ObjectKey{Namespace: constants.VerrazzanoMonitoringNamespace, Name: managedClusterName}, dr)
	assert.True(t, k8serrors.IsNotFound(err))
}

func makeThanosTestScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	v1beta1.AddToScheme(scheme)
	corev1.AddToScheme(scheme)
	appsv1.AddToScheme(scheme)
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

func makeThanosConfigMapWithExistingHosts(t *testing.T, useValidConfigMap bool, numClusters int, host, vmcPrefix string) *corev1.ConfigMap {
	var yamlExistingHostInfo []byte
	var err error
	if useValidConfigMap {
		existingHostInfo := []*thanosServiceDiscovery{}
		for i := 1; i <= numClusters; i++ {
			existingHostInfo = append(existingHostInfo, &thanosServiceDiscovery{
				Targets: []string{host},
				Labels: map[string]string{
					verrazzanoManagedLabel: fmt.Sprintf("%s%d", vmcPrefix, i),
				},
			})
		}
		yamlExistingHostInfo, err = yaml.Marshal(existingHostInfo)
		assert.NoError(t, err)
	} else {
		yamlExistingHostInfo = []byte("- targets: garbledTextHere")
	}
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Namespace: thanos.ComponentNamespace, Name: ThanosManagedClusterEndpointsConfigMap},
		Data: map[string]string{
			serviceDiscoveryKey: string(yamlExistingHostInfo),
		},
	}
}

func assertThanosEndpointsConfigMap(ctx context.Context, t *testing.T, cli client.WithWatch, expectNumHosts int, host, vmcName string, vmcShoudExist bool) {
	modifiedConfigMap := &corev1.ConfigMap{}
	err := cli.Get(ctx, types.NamespacedName{Namespace: thanos.ComponentNamespace, Name: ThanosManagedClusterEndpointsConfigMap}, modifiedConfigMap)
	assert.NoError(t, err)
	var modifiedContent []*thanosServiceDiscovery
	err = yaml.Unmarshal([]byte(modifiedConfigMap.Data[serviceDiscoveryKey]), &modifiedContent)
	assert.NoError(t, err)
	assert.Len(t, modifiedContent, expectNumHosts, "Expected %d service discovery entries", expectNumHosts)
	if vmcShoudExist {
		for _, sd := range modifiedContent {
			if val, ok := sd.Labels[verrazzanoManagedLabel]; ok && val == vmcName {
				assert.Equal(t, host, sd.Targets[0])
				return
			}
		}
		assert.Fail(t, fmt.Sprintf("Failed to find Service Discovery for VMC %s", vmcName))
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
	vmc := &clustersv1alpha1.VerrazzanoManagedCluster{ObjectMeta: metav1.ObjectMeta{Name: managedClusterName}}

	err := r.createOrUpdateDestinationRule(vmc, host, port)
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

	err = r.createOrUpdateDestinationRule(vmc, host, port)
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
	vmc := &clustersv1alpha1.VerrazzanoManagedCluster{ObjectMeta: metav1.ObjectMeta{Name: managedClusterName}}
	dr := &istioclinet.DestinationRule{ObjectMeta: metav1.ObjectMeta{Name: managedClusterName, Namespace: constants.VerrazzanoMonitoringNamespace}}
	populateDestinationRule(dr, "bad-bad-host", port, vmc)

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

	err := r.createOrUpdateDestinationRule(vmc, host, port)
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
	vmc := &clustersv1alpha1.VerrazzanoManagedCluster{ObjectMeta: metav1.ObjectMeta{Name: managedClusterName}}
	dr := &istioclinet.DestinationRule{ObjectMeta: metav1.ObjectMeta{Name: managedClusterName, Namespace: constants.VerrazzanoMonitoringNamespace}}
	populateDestinationRule(dr, host, port, vmc)

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
	assert.Equal(t, dr.Spec.TrafficPolicy.PortLevelSettings[0].Tls.Sni, hostName)
}

func assertAdditionalScrapeConfigRemoved(t *testing.T, r *VerrazzanoManagedClusterReconciler, vmcName string) {
	sec := &corev1.Secret{}
	err := r.Client.Get(context.TODO(), client.ObjectKey{Namespace: constants.VerrazzanoMonitoringNamespace, Name: vzconst.PromAdditionalScrapeConfigsSecretName}, sec)
	assert.NoError(t, err)
	data, ok := sec.Data[vzconst.PromAdditionalScrapeConfigsSecretKey]
	assert.True(t, ok, "Additional scrape configs key not found in secret")
	assert.NotEmpty(t, data)
	scrapeConfigContainer, err := metricsutils.ParseScrapeConfig(string(data))
	assert.NoError(t, err)
	assert.Negative(t, metricsutils.FindScrapeJob(scrapeConfigContainer, vmcName))
}
