// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
	"github.com/verrazzano/verrazzano/application-operator/mocks"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// TestSyncer_AgentReadyToSync tests the AgentReadyToSync method of Syncer
func TestSyncer_AgentReadyToSync(t *testing.T) {
	type fields struct {
		AgentSecretFound bool
		AgentSecretValid bool
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{"agent secret found not valid", fields{AgentSecretFound: true, AgentSecretValid: false}, false},
		{"agent secret not found", fields{AgentSecretFound: false, AgentSecretValid: false}, false},
		{"agent secret found and valid", fields{AgentSecretFound: true, AgentSecretValid: true}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Syncer{
				AgentSecretFound: tt.fields.AgentSecretFound,
				AgentSecretValid: tt.fields.AgentSecretValid,
			}
			if got := s.AgentReadyToSync(); got != tt.want {
				t.Errorf("AgentReadyToSync() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestSyncer_isThisCluster tests the isThisCluster method of Syncer
func TestSyncer_isThisCluster(t *testing.T) {
	tests := []struct {
		name               string
		managedClusterName string
		placement          v1alpha1.Placement
		want               bool
	}{
		{"same cluster single placement", "mycluster1", v1alpha1.Placement{Clusters: []v1alpha1.Cluster{{Name: "mycluster1"}}}, true},
		{"same cluster multi-placement", "mycluster1", v1alpha1.Placement{Clusters: []v1alpha1.Cluster{{Name: "othercluster"}, {Name: "mycluster1"}}}, true},
		{"different cluster single placement", "mycluster1", v1alpha1.Placement{Clusters: []v1alpha1.Cluster{{Name: "othercluster"}}}, false},
		{"different cluster multi-placement", "mycluster1", v1alpha1.Placement{Clusters: []v1alpha1.Cluster{{Name: "othercluster"}, {Name: "mycluster2"}}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Syncer{
				ManagedClusterName: tt.managedClusterName,
			}
			if got := s.isThisCluster(tt.placement); got != tt.want {
				t.Errorf("isThisCluster() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestSyncer_processStatusUpdates tests the processStatusUpdates method of Syncer
// GIVEN a syncer object created with a status updates channel
// WHEN processStatusUpdates is called
// THEN for every message written to the channel, a corresponding status update to admin cluster
// is generated
func TestSyncer_processStatusUpdates(t *testing.T) {
	// Admin cluster mocks
	adminMocker := gomock.NewController(t)
	adminMock := mocks.NewMockClient(adminMocker)
	statusMocker := gomock.NewController(t)
	statusMock := mocks.NewMockClient(statusMocker)

	statusUpdatesChan := make(chan clusters.StatusUpdateMessage, 5)

	// write some messages to the status update channel for the agent to make sure
	// they get discarded when there is no admin cluster to connect to
	// write some messages to the status update channel for the agent to make sure
	// they get discarded when there is no admin cluster to connect to
	statusUpdates := makeStatusUpdateMessages()
	for _, update := range statusUpdates {
		statusUpdatesChan <- update
	}

	// Expect every status update that is in the statusUpdates array to be sent
	// to the admin cluster
	adminMock.EXPECT().Status().Times(len(statusUpdates)).Return(statusMock)
	for _, updateMsg := range statusUpdates {
		statusMock.EXPECT().Update(gomock.Any(), updateMsg.Resource)
	}

	// Make the request
	s := &Syncer{
		AdminClient:         adminMock,
		ManagedClusterName:  "mycluster1",
		StatusUpdateChannel: statusUpdatesChan,
		Log:                 ctrl.Log.WithName("statusUpdateUnitTest"),
	}
	s.processStatusUpdates()

	statusMocker.Finish()
	adminMocker.Finish()
}

func makeStatusUpdateMessages() []clusters.StatusUpdateMessage {
	secret := v1alpha1.MultiClusterSecret{}
	secret.Name = "somesecret"
	secret.Namespace = "somens"

	appConfig := v1alpha1.MultiClusterApplicationConfiguration{}
	appConfig.Name = "someappconf"
	appConfig.Namespace = "appconfns"
	msg1 := clusters.StatusUpdateMessage{
		NewCondition:     v1alpha1.Condition{Type: v1alpha1.DeployFailed, Status: corev1.ConditionTrue, Message: "my msg 1"},
		NewClusterStatus: v1alpha1.ClusterLevelStatus{Name: "cluster1", State: v1alpha1.Failed},
		Resource:         &secret,
	}
	msg2 := clusters.StatusUpdateMessage{
		NewCondition:     v1alpha1.Condition{Type: v1alpha1.DeployComplete, Status: corev1.ConditionTrue, Message: "my msg 2"},
		NewClusterStatus: v1alpha1.ClusterLevelStatus{Name: "cluster1", State: v1alpha1.Succeeded},
		Resource:         &appConfig,
	}
	return []clusters.StatusUpdateMessage{msg1, msg2}
}
