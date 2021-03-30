// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusters

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	asserts "github.com/stretchr/testify/assert"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/application-operator/mocks"
	v1 "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// TestFetchManagedClusterElasticSearchDetails_Exists tests fetching Elasticsearch details
// when the ES secret exists
// GIVEN the Elasticsearch secret exists
// WHEN FetchManagedClusterElasticSearchDetails function is called
// THEN expect that the returned ElasticsearchDetails has the correct URL and secret name fields
func TestFetchManagedClusterElasticSearchDetails_Exists(t *testing.T) {
	mocker := gomock.NewController(t)
	cli := mocks.NewMockClient(mocker)

	esURL := "https://someEsUrl"
	esUser := "someEsUser"
	esPwd := "xyzabc"
	esSecret1 := v1.Secret{
		Data: map[string][]byte{
			constants.ElasticsearchURLData:      []byte(esURL),
			constants.ElasticsearchUsernameData: []byte(esUser),
			constants.ElasticsearchPasswordData: []byte(esPwd)},
	}
	esSecret1.Name = MCRegistrationSecretFullName.Name
	esSecret1.Namespace = MCRegistrationSecretFullName.Namespace

	cli.EXPECT().
		Get(gomock.Any(), MCRegistrationSecretFullName, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *v1.Secret) error {
			secret.Name = esSecret1.Name
			secret.Namespace = esSecret1.Namespace
			secret.Data = esSecret1.Data
			return nil
		})

	esDetails := FetchManagedClusterElasticSearchDetails(context.TODO(), cli)

	asserts.Equal(t, esURL, esDetails.URL)
	asserts.Equal(t, esSecret1.Name, esDetails.SecretName)
	mocker.Finish()
}

// TestFetchManagedClusterElasticSearchDetails_DoesNotExist tests fetching Elasticsearch details
// when the ES secret does not exist
// GIVEN the Elasticsearch secret does not exist
// WHEN FetchManagedClusterElasticSearchDetails function is called
// THEN expect that the returned ElasticsearchDetails has empty values
func TestFetchManagedClusterElasticSearchDetails_DoesNotExist(t *testing.T) {
	mocker := gomock.NewController(t)
	cli := mocks.NewMockClient(mocker)

	cli.EXPECT().
		Get(gomock.Any(), MCRegistrationSecretFullName, gomock.Not(gomock.Nil())).
		Return(kerr.NewNotFound(schema.ParseGroupResource("Secret"), MCRegistrationSecretFullName.Name))

	esDetails := FetchManagedClusterElasticSearchDetails(context.TODO(), cli)

	asserts.Equal(t, "", esDetails.URL)
	asserts.Equal(t, "", esDetails.SecretName)
	mocker.Finish()
}

// TestGetClusterName tests fetching the cluster name from the managed cluster registration secret
// GIVEN The managed cluster registration secret exists
// WHEN GetClusterName function is called
// THEN expect the managed-cluster-name from the secret to be returned
// GIVEN the managed cluster secret does not exist but the local resgistration secret does
// WHEN GetClusterName function is called
// THEN expect that managed-cluster-name from the local secret to be returned
// GIVEN the managed cluster secret and the local do not exist
// WHEN GetClusterName function is called
// THEN expect that empty string is returned
func TestGetClusterName(t *testing.T) {
	mocker := gomock.NewController(t)
	cli := mocks.NewMockClient(mocker)

	expectMCRegistrationSecret(cli, "mycluster1", MCRegistrationSecretFullName, 1)

	name := GetClusterName(context.TODO(), cli)
	asserts.Equal(t, "mycluster1", name)

	// Repeat test for registration secret not found, then local registration secret should be used
	cli.EXPECT().
		Get(gomock.Any(), MCRegistrationSecretFullName, gomock.Not(gomock.Nil())).
		Return(kerr.NewNotFound(schema.ParseGroupResource("Secret"), MCRegistrationSecretFullName.Name))

	expectMCRegistrationSecret(cli, "mycluster1", MCLocalRegistrationSecretFullName, 1)

	name = GetClusterName(context.TODO(), cli)
	asserts.Equal(t, "mycluster1", name)

	// Repeat test for registration secret and local secret not found
	cli.EXPECT().
		Get(gomock.Any(), MCRegistrationSecretFullName, gomock.Not(gomock.Nil())).
		Return(kerr.NewNotFound(schema.ParseGroupResource("Secret"), MCRegistrationSecretFullName.Name))

	cli.EXPECT().
		Get(gomock.Any(), MCLocalRegistrationSecretFullName, gomock.Not(gomock.Nil())).
		Return(kerr.NewNotFound(schema.ParseGroupResource("Secret"), MCLocalRegistrationSecretFullName.Name))

	name = GetClusterName(context.TODO(), cli)
	asserts.Equal(t, "", name)

	mocker.Finish()
}

// TestGetConditionFromResult tests the GetConditionFromResult function
// GIVEN a nil or non-nil err
// WHEN GetConditionFromResult is called
// The returned condition and state show success or failure depending on err == nil or not,
// and the message is correctly populated
func TestGetConditionFromResult(t *testing.T) {
	// GIVEN err == nil
	condition := GetConditionFromResult(nil, controllerutil.OperationResultCreated, "myresource type")
	// TODO Mar 3 asserts.Equal(t, clustersv1alpha1.Succeeded, stateType)
	asserts.Equal(t, clustersv1alpha1.DeployComplete, condition.Type)
	asserts.Equal(t, v1.ConditionTrue, condition.Status)
	asserts.Contains(t, condition.Message, "myresource type")
	asserts.Contains(t, condition.Message, controllerutil.OperationResultCreated)

	// GIVEN err != nil
	someerr := errors.New("some error msg")
	condition = GetConditionFromResult(someerr, controllerutil.OperationResultCreated, "myresource type")
	// TODO Mar 3 asserts.Equal(t, clustersv1alpha1.Failed, stateType)
	asserts.Equal(t, clustersv1alpha1.DeployFailed, condition.Type)
	asserts.Equal(t, v1.ConditionTrue, condition.Status)
	asserts.Contains(t, condition.Message, someerr.Error())
}

// TestIgnoreNotFoundWithLog tests the IgnoreNotFoundWithLog function
// GIVEN a K8S NotFound error
// WHEN IgnoreNotFoundWithLog is called
// THEN a nil error is returned
// GIVEN any other type of error
// WHEN IgnoreNotFoundWithLog is called
// THEN the error is returned
func TestIgnoreNotFoundWithLog(t *testing.T) {
	logger := controllerruntime.Log.WithName("somelogger")
	err := IgnoreNotFoundWithLog("myResourceType", kerr.NewNotFound(controllerruntime.GroupResource{}, ""), logger)
	asserts.Nil(t, err)

	otherErr := kerr.NewBadRequest("some other error")
	err = IgnoreNotFoundWithLog("myResourceType", otherErr, logger)
	asserts.Equal(t, otherErr, err)
}

// TestIsPlacedInThisCluster tests the IsPlacedInThisCluster function
// GIVEN a placement object
// WHEN IsPlacedInThisCluster is called
// THEN it returns true if the placement includes this cluster's name, false otherwise
func TestIsPlacedInThisCluster(t *testing.T) {
	mocker := gomock.NewController(t)
	cli := mocks.NewMockClient(mocker)
	placementOnlyMyCluster := clustersv1alpha1.Placement{Clusters: []clustersv1alpha1.Cluster{{Name: "mycluster"}}}
	placementWithMyCluster := clustersv1alpha1.Placement{Clusters: []clustersv1alpha1.Cluster{{Name: "othercluster"}, {Name: "mycluster"}}}
	placementNotMyCluster := clustersv1alpha1.Placement{Clusters: []clustersv1alpha1.Cluster{{Name: "othercluster"}, {Name: "NOTmycluster"}}}

	expectMCRegistrationSecret(cli, "mycluster", MCRegistrationSecretFullName, 3)

	asserts.True(t, IsPlacedInThisCluster(context.TODO(), cli, placementOnlyMyCluster))
	asserts.True(t, IsPlacedInThisCluster(context.TODO(), cli, placementWithMyCluster))
	asserts.False(t, IsPlacedInThisCluster(context.TODO(), cli, placementNotMyCluster))

	mocker.Finish()
}

// TestStatusNeedsUpdate tests various combinations of input to the StatusNeedsUpdate function
// GIVEN a current status present on the resource
// WHEN StatusNeedsUpdate is called with a new condition, state and cluster level status
// THEN it returns false if the new condition and cluster level status are already present
// in the status, AND the new state matches the current state, true otherwise
func TestStatusNeedsUpdate(t *testing.T) {
	conditionTimestamp := time.Now()
	formattedConditionTimestamp := conditionTimestamp.Format(time.RFC3339)
	curConditions := []clustersv1alpha1.Condition{
		{Type: clustersv1alpha1.DeployComplete, Status: v1.ConditionTrue, LastTransitionTime: formattedConditionTimestamp},
	}
	curState := clustersv1alpha1.Failed
	curCluster1Status := clustersv1alpha1.ClusterLevelStatus{Name: "cluster1", State: clustersv1alpha1.Succeeded, LastUpdateTime: formattedConditionTimestamp}
	curCluster2Status := clustersv1alpha1.ClusterLevelStatus{Name: "cluster2", State: clustersv1alpha1.Failed, LastUpdateTime: formattedConditionTimestamp}

	curStatus := clustersv1alpha1.MultiClusterResourceStatus{
		Conditions: curConditions,
		State:      curState,
		Clusters:   []clustersv1alpha1.ClusterLevelStatus{curCluster1Status, curCluster2Status},
	}

	otherTimestamp := conditionTimestamp.AddDate(0, 0, 1).Format(time.RFC3339)
	newCond := clustersv1alpha1.Condition{Type: clustersv1alpha1.DeployFailed, Status: v1.ConditionTrue}
	existingCond := curConditions[0]
	newCluster1Status := clustersv1alpha1.ClusterLevelStatus{
		Name:           curCluster1Status.Name,
		State:          clustersv1alpha1.Failed,
		LastUpdateTime: formattedConditionTimestamp}
	newCluster2Status := clustersv1alpha1.ClusterLevelStatus{
		Name:           curCluster2Status.Name,
		State:          clustersv1alpha1.Succeeded,
		LastUpdateTime: formattedConditionTimestamp}

	existingCondDiffTimestampCluster1 := clustersv1alpha1.Condition{
		Type: curConditions[0].Type, Status: curConditions[0].Status, LastTransitionTime: otherTimestamp}

	cluster1StatusDiffTimestamp := clustersv1alpha1.ClusterLevelStatus{
		Name:           curCluster1Status.Name,
		State:          curCluster1Status.State,
		LastUpdateTime: otherTimestamp}

	newClusterStatus := clustersv1alpha1.ClusterLevelStatus{
		Name:           "newCluster",
		State:          clustersv1alpha1.Succeeded,
		LastUpdateTime: otherTimestamp}

	// Asserts new condition, same cluster status for each cluster- needs update
	asserts.True(t, StatusNeedsUpdate(curStatus, newCond, curCluster1Status))
	asserts.True(t, StatusNeedsUpdate(curStatus, newCond, curCluster2Status))

	// new condition, same cluster status for each cluster - needs update
	asserts.True(t, StatusNeedsUpdate(curStatus, newCond, curCluster1Status))
	asserts.True(t, StatusNeedsUpdate(curStatus, newCond, curCluster2Status))

	// same condition, same cluster status for each clusters - does not need update
	asserts.False(t, StatusNeedsUpdate(curStatus, existingCond, curCluster1Status))
	asserts.False(t, StatusNeedsUpdate(curStatus, existingCond, curCluster2Status))

	// same condition, different cluster status for each cluster - needs update
	asserts.True(t, StatusNeedsUpdate(curStatus, existingCond, newCluster1Status))
	asserts.True(t, StatusNeedsUpdate(curStatus, existingCond, newCluster2Status))

	// same condition, differing in condition timestamp - does not need update
	asserts.False(t, StatusNeedsUpdate(curStatus, existingCondDiffTimestampCluster1, curCluster1Status))

	// same condition, differing in cluster status timestamp - does not need update
	asserts.False(t, StatusNeedsUpdate(curStatus, existingCondDiffTimestampCluster1, cluster1StatusDiffTimestamp))

	// same condition, new cluster not present in conditions - needs update
	asserts.True(t, StatusNeedsUpdate(curStatus, existingCond, newClusterStatus))
}

// TestCreateClusterLevelStatus tests the CreateClusterLevelStatus function
// GIVEN a condition and state
// WHEN CreateClusterLevelStatus is called
// THEN it returns a cluster state correctly populated
func TestCreateClusterLevelStatus(t *testing.T) {
	formattedConditionTimestamp := time.Now().Format(time.RFC3339)
	condition1 := clustersv1alpha1.Condition{
		Type: clustersv1alpha1.DeployComplete, Status: v1.ConditionTrue, LastTransitionTime: formattedConditionTimestamp,
	}
	condition2 := clustersv1alpha1.Condition{
		Type: clustersv1alpha1.DeployFailed, Status: v1.ConditionTrue, LastTransitionTime: formattedConditionTimestamp,
	}
	clusterState1 := CreateClusterLevelStatus(condition1, "cluster1")
	asserts.Equal(t, "cluster1", clusterState1.Name)
	asserts.Equal(t, clustersv1alpha1.Succeeded, clusterState1.State)
	asserts.Equal(t, formattedConditionTimestamp, clusterState1.LastUpdateTime)

	clusterState2 := CreateClusterLevelStatus(condition2, "somecluster")
	asserts.Equal(t, "somecluster", clusterState2.Name)
	asserts.Equal(t, clustersv1alpha1.Failed, clusterState2.State)
	asserts.Equal(t, formattedConditionTimestamp, clusterState2.LastUpdateTime)
}

// TestComputeEffectiveState tests the ComputeEffectiveState function
// GIVEN a multi cluster resource status and its placements
// WHEN ComputeEffectiveState is called
// THEN it returns a cluster state that rolls up the states of individual cluster placements
func TestComputeEffectiveState(t *testing.T) {
	placement := clustersv1alpha1.Placement{
		Clusters: []clustersv1alpha1.Cluster{
			{Name: "cluster1"},
			{Name: "cluster2"},
		},
	}
	allSucceededStatus := clustersv1alpha1.MultiClusterResourceStatus{
		Clusters: []clustersv1alpha1.ClusterLevelStatus{
			{Name: "cluster1", State: clustersv1alpha1.Succeeded},
			{Name: "cluster2", State: clustersv1alpha1.Succeeded},
		},
	}
	somePendingStatus := clustersv1alpha1.MultiClusterResourceStatus{
		Clusters: []clustersv1alpha1.ClusterLevelStatus{
			{Name: "cluster1", State: clustersv1alpha1.Succeeded},
			{Name: "cluster2", State: clustersv1alpha1.Pending},
		},
	}
	someFailedSomePendingStatus := clustersv1alpha1.MultiClusterResourceStatus{
		Clusters: []clustersv1alpha1.ClusterLevelStatus{
			{Name: "cluster1", State: clustersv1alpha1.Failed},
			{Name: "cluster2", State: clustersv1alpha1.Pending},
		},
	}
	someFailedSomeSucceededStatus := clustersv1alpha1.MultiClusterResourceStatus{
		Clusters: []clustersv1alpha1.ClusterLevelStatus{
			{Name: "cluster1", State: clustersv1alpha1.Succeeded},
			{Name: "cluster2", State: clustersv1alpha1.Failed},
		},
	}
	failedWithUnknownClusterSucceededStatus := clustersv1alpha1.MultiClusterResourceStatus{
		Clusters: []clustersv1alpha1.ClusterLevelStatus{
			{Name: "cluster3", State: clustersv1alpha1.Succeeded},
			{Name: "cluster2", State: clustersv1alpha1.Failed},
		},
	}
	pendingWithUnknownClusterSucceededStatus := clustersv1alpha1.MultiClusterResourceStatus{
		Clusters: []clustersv1alpha1.ClusterLevelStatus{
			{Name: "cluster3", State: clustersv1alpha1.Succeeded},
			{Name: "cluster2", State: clustersv1alpha1.Pending},
		},
	}
	asserts.Equal(t, clustersv1alpha1.Succeeded, ComputeEffectiveState(allSucceededStatus, placement))
	asserts.Equal(t, clustersv1alpha1.Pending, ComputeEffectiveState(somePendingStatus, placement))
	asserts.Equal(t, clustersv1alpha1.Failed, ComputeEffectiveState(someFailedSomePendingStatus, placement))
	asserts.Equal(t, clustersv1alpha1.Failed, ComputeEffectiveState(someFailedSomeSucceededStatus, placement))
	asserts.Equal(t, clustersv1alpha1.Failed, ComputeEffectiveState(failedWithUnknownClusterSucceededStatus, placement))
	asserts.Equal(t, clustersv1alpha1.Pending, ComputeEffectiveState(pendingWithUnknownClusterSucceededStatus, placement))
}

func TestUpdateClusterLevelStatus(t *testing.T) {
	resourceStatus := clustersv1alpha1.MultiClusterResourceStatus{
		Clusters: []clustersv1alpha1.ClusterLevelStatus{
			{Name: "cluster1", State: clustersv1alpha1.Failed},
			{Name: "cluster2", State: clustersv1alpha1.Pending},
		},
	}

	cluster1Succeeded := clustersv1alpha1.ClusterLevelStatus{
		Name: "cluster1", State: clustersv1alpha1.Succeeded,
	}

	cluster2Failed := clustersv1alpha1.ClusterLevelStatus{
		Name: "cluster2", State: clustersv1alpha1.Failed,
	}

	newClusterPending := clustersv1alpha1.ClusterLevelStatus{
		Name: "newCluster", State: clustersv1alpha1.Pending,
	}

	// existing cluster cluster1 should be updated
	SetClusterLevelStatus(&resourceStatus, cluster1Succeeded)
	asserts.Equal(t, 2, len(resourceStatus.Clusters))
	asserts.Equal(t, clustersv1alpha1.Succeeded, resourceStatus.Clusters[0].State)

	// existing cluster cluster2 should be updated
	SetClusterLevelStatus(&resourceStatus, cluster2Failed)
	asserts.Equal(t, 2, len(resourceStatus.Clusters))
	asserts.Equal(t, clustersv1alpha1.Failed, resourceStatus.Clusters[1].State)

	// hitherto unseen cluster should be added to the cluster level statuses list
	SetClusterLevelStatus(&resourceStatus, newClusterPending)
	asserts.Equal(t, 3, len(resourceStatus.Clusters))
	asserts.Equal(t, clustersv1alpha1.Pending, resourceStatus.Clusters[2].State)

}

func expectMCRegistrationSecret(cli *mocks.MockClient, clusterName string, secreteNameFullName types.NamespacedName, times int) {
	regSecretData := map[string][]byte{constants.ClusterNameData: []byte(clusterName)}
	cli.EXPECT().
		Get(gomock.Any(), secreteNameFullName, gomock.Not(gomock.Nil())).
		Times(times).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *v1.Secret) error {
			secret.Name = secreteNameFullName.Name
			secret.Namespace = secreteNameFullName.Namespace
			secret.Data = regSecretData
			return nil
		})
}

// TestSetEffectiveStateIfChanged tests that if the effective state of a resource has changed, it's
// state is changed
// GIVEN a MultiCluster resource whose effective state is unchanged
// WHEN SetEffectiveStateIfChanged is called
// THEN the state should not be updated
// GIVEN a MultiCluster resource whose effective state has changed
// WHEN SetEffectiveStateIfChanged is called
// THEN the state should be updated
func TestSetEffectiveStateIfChanged(t *testing.T) {
	placement := clustersv1alpha1.Placement{
		Clusters: []clustersv1alpha1.Cluster{
			{Name: "cluster1"},
			{Name: "cluster2"},
		},
	}
	secret := clustersv1alpha1.MultiClusterSecret{
		Spec: clustersv1alpha1.MultiClusterSecretSpec{
			Placement: placement,
		},
		Status: clustersv1alpha1.MultiClusterResourceStatus{State: clustersv1alpha1.Pending},
	}
	secret.Name = "mysecret"
	secret.Namespace = "myns"

	// Make a call with the effective state of the resource unchanged, and no change should occur
	SetEffectiveStateIfChanged(placement, &secret.Status)
	asserts.Equal(t, clustersv1alpha1.Pending, secret.Status.State)

	// add cluster level status info to the secret's status, and make a call again - this time
	// it should update the status of the resource since the effective state changes
	secret.Status.Clusters = []clustersv1alpha1.ClusterLevelStatus{
		{Name: "cluster1", State: clustersv1alpha1.Failed},
	}

	SetEffectiveStateIfChanged(placement, &secret.Status)
	asserts.Equal(t, clustersv1alpha1.Failed, secret.Status.State)
}
