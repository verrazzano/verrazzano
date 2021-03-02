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

	esUrl := "https://someEsUrl"
	esUser := "someEsUser"
	esPwd := "xyzabc"
	esSecret1 := v1.Secret{
		Data: map[string][]byte{
			constants.ElasticsearchURLData:      []byte(esUrl),
			constants.ElasticsearchUsernameData: []byte(esUser),
			constants.ElasticsearchPasswordData: []byte(esPwd)},
	}
	esSecret1.Name = MCElasticsearchSecretFullName.Name
	esSecret1.Namespace = MCElasticsearchSecretFullName.Namespace

	cli.EXPECT().
		Get(gomock.Any(), MCElasticsearchSecretFullName, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *v1.Secret) error {
			secret.Name = esSecret1.Name
			secret.Namespace = esSecret1.Namespace
			secret.Data = esSecret1.Data
			return nil
		})

	esDetails := FetchManagedClusterElasticSearchDetails(context.TODO(), cli)

	asserts.Equal(t, esUrl, esDetails.URL)
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
		Get(gomock.Any(), MCElasticsearchSecretFullName, gomock.Not(gomock.Nil())).
		Return(kerr.NewNotFound(schema.ParseGroupResource("Secret"), MCElasticsearchSecretFullName.Name))

	esDetails := FetchManagedClusterElasticSearchDetails(context.TODO(), cli)

	asserts.Equal(t, "", esDetails.URL)
	asserts.Equal(t, "", esDetails.SecretName)
	mocker.Finish()
}

// TestGetClusterName tests fetching the cluster name from the managed cluster registration secret
// GIVEN The managed cluster registration secret exists
// WHEN GetClusterName function is called
// THEN expect the managed-cluster-name from the secret to be returned
// GIVEN the managed cluster secret does not exist
// WHEN GetClusterName function is called
// THEN expect that empty string is returned
func TestGetClusterName(t *testing.T) {
	mocker := gomock.NewController(t)
	cli := mocks.NewMockClient(mocker)

	expectMCRegistrationSecret(cli, "mycluster1", 1)

	name := GetClusterName(context.TODO(), cli)
	asserts.Equal(t, "mycluster1", name)

	// Repeat test for registration secret not found
	cli.EXPECT().
		Get(gomock.Any(), MCRegistrationSecretFullName, gomock.Not(gomock.Nil())).
		Return(kerr.NewNotFound(schema.ParseGroupResource("Secret"), MCRegistrationSecretFullName.Name))

	name = GetClusterName(context.TODO(), cli)
	asserts.Equal(t, "", name)

	mocker.Finish()
}

// TestGetConditionAndStateFromResult tests the GetConditionAndStateFromResult function
// GIVEN a nil or non-nil err
// WHEN GetConditionAndStateFromResult is called
// The returned condition and state show success or failure depending on err == nil or not,
// and the message and cluster name are correctly populated
func TestGetConditionAndStateFromResult(t *testing.T) {
	// GIVEN err == nil
	condition, stateType := GetConditionAndStateFromResult(nil, controllerutil.OperationResultCreated, "myresource type", "mytestcluster")
	asserts.Equal(t, clustersv1alpha1.Ready, stateType)
	asserts.Equal(t, clustersv1alpha1.DeployComplete, condition.Type)
	asserts.Equal(t, v1.ConditionTrue, condition.Status)
	asserts.Equal(t, "mytestcluster", condition.ClusterName)
	asserts.Contains(t, condition.Message, "myresource type")
	asserts.Contains(t, condition.Message, controllerutil.OperationResultCreated)

	// GIVEN err != nil
	someerr := errors.New("some error msg")
	condition, stateType = GetConditionAndStateFromResult(someerr, controllerutil.OperationResultCreated, "myresource type", "mytestcluster")
	asserts.Equal(t, clustersv1alpha1.Failed, stateType)
	asserts.Equal(t, clustersv1alpha1.DeployFailed, condition.Type)
	asserts.Equal(t, v1.ConditionTrue, condition.Status)
	asserts.Equal(t, "mytestcluster", condition.ClusterName)
	asserts.Contains(t, condition.Message, someerr.Error())
}

// TestGetManagedClusterElasticsearchSecretKey tests that GetManagedClusterElasticsearchSecretKey
// returns the correct value
func TestGetManagedClusterElasticsearchSecretKey(t *testing.T) {
	key := GetManagedClusterElasticsearchSecretKey()
	asserts.Equal(t, MCElasticsearchSecretFullName, key)
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

	expectMCRegistrationSecret(cli, "mycluster", 3)

	asserts.True(t, IsPlacedInThisCluster(context.TODO(), cli, placementOnlyMyCluster))
	asserts.True(t, IsPlacedInThisCluster(context.TODO(), cli, placementWithMyCluster))
	asserts.False(t, IsPlacedInThisCluster(context.TODO(), cli, placementNotMyCluster))

	mocker.Finish()
}

// TestStatusNeedsUpdate tests the StatusNeedsUpdate function
// GIVEN a current state and a list of current conditions present on the resource
// WHEN StatusNeedsUpdate is called with a new condition and state
// THEN it returns true if the new condition's content (including cluster name) are already present
// in the list of existing conditions AND the new state matches the current state, false otherwise
func TestStatusNeedsUpdate(t *testing.T) {
	conditionTimestamp := time.Now()
	formattedConditionTimestamp := conditionTimestamp.Format(time.RFC3339)
	curConditions := []clustersv1alpha1.Condition{
		{Type: clustersv1alpha1.DeployComplete, Status: v1.ConditionTrue, ClusterName: "cluster1", LastTransitionTime: formattedConditionTimestamp},
		{Type: clustersv1alpha1.DeployFailed, Status: v1.ConditionTrue, ClusterName: "cluster2", LastTransitionTime: formattedConditionTimestamp},
	}
	curState := clustersv1alpha1.Ready

	otherState := clustersv1alpha1.Failed
	otherTimestamp := conditionTimestamp.AddDate(0, 0, 1).Format(time.RFC3339)
	newCondCluster1 := clustersv1alpha1.Condition{Type: clustersv1alpha1.DeployFailed, Status: v1.ConditionTrue, ClusterName: "cluster1"}
	existingCondCluster1 := curConditions[0]
	existingCondNewCluster := clustersv1alpha1.Condition{Type: clustersv1alpha1.DeployComplete, Status: v1.ConditionTrue, ClusterName: "newCluster"}
	newCondCluster2 := clustersv1alpha1.Condition{Type: clustersv1alpha1.DeployFailed, Status: v1.ConditionFalse, ClusterName: "cluster2"}
	existingCondCluster2 := curConditions[1]

	existingCondDiffTimestampCluster1 := clustersv1alpha1.Condition{
		Type: curConditions[0].Type, Status: curConditions[0].Status, ClusterName: curConditions[0].ClusterName, LastTransitionTime: otherTimestamp}
	existingCondDiffTimestampCluster2 := clustersv1alpha1.Condition{
		Type: curConditions[1].Type, Status: curConditions[1].Status, ClusterName: curConditions[1].ClusterName, LastTransitionTime: otherTimestamp}

	// Asserts for BOTH of the clusters already present in conditions (cluster1 and cluster2)
	// new condition, same state - needs update
	asserts.True(t, StatusNeedsUpdate(curConditions, curState, newCondCluster1, curState))
	asserts.True(t, StatusNeedsUpdate(curConditions, curState, newCondCluster2, curState))

	// new condition, different state - needs update
	asserts.True(t, StatusNeedsUpdate(curConditions, curState, newCondCluster1, otherState))
	asserts.True(t, StatusNeedsUpdate(curConditions, curState, newCondCluster2, otherState))

	// same condition, different state - needs update
	asserts.True(t, StatusNeedsUpdate(curConditions, curState, existingCondCluster1, otherState))
	asserts.True(t, StatusNeedsUpdate(curConditions, curState, existingCondCluster2, otherState))

	// same condition, same state - does not need update
	asserts.False(t, StatusNeedsUpdate(curConditions, curState, existingCondCluster1, curState))
	asserts.False(t, StatusNeedsUpdate(curConditions, curState, existingCondCluster2, curState))

	// same condition and state, differing in timestamp - does not need update
	asserts.False(t, StatusNeedsUpdate(curConditions, curState, existingCondDiffTimestampCluster1, curState))
	asserts.False(t, StatusNeedsUpdate(curConditions, curState, existingCondDiffTimestampCluster2, curState))

	// same condition, same state, new cluster not present in conditions - needs update
	asserts.True(t, StatusNeedsUpdate(curConditions, curState, existingCondNewCluster, curState))
}

func expectMCRegistrationSecret(cli *mocks.MockClient, clusterName string, times int) {
	regSecretData := map[string][]byte{constants.ClusterNameData: []byte(clusterName)}
	cli.EXPECT().
		Get(gomock.Any(), MCRegistrationSecretFullName, gomock.Not(gomock.Nil())).
		Times(times).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *v1.Secret) error {
			secret.Name = MCRegistrationSecretFullName.Name
			secret.Namespace = MCRegistrationSecretFullName.Namespace
			secret.Data = regSecretData
			return nil
		})
}
