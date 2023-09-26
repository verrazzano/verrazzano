// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusters

import (
	"context"
	"errors"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"testing"
	"time"

	"github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/golang/mock/gomock"
	asserts "github.com/stretchr/testify/assert"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/application-operator/mocks"
	kerr "github.com/verrazzano/verrazzano/pkg/k8s/errors"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

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
		Get(gomock.Any(), MCRegistrationSecretFullName, gomock.Not(gomock.Nil()), gomock.Any()).
		Return(kerr.NewNotFound(schema.ParseGroupResource("Secret"), MCRegistrationSecretFullName.Name))

	expectMCRegistrationSecret(cli, "mycluster1", MCLocalRegistrationSecretFullName, 1)

	name = GetClusterName(context.TODO(), cli)
	asserts.Equal(t, "mycluster1", name)

	// Repeat test for registration secret and local secret not found
	cli.EXPECT().
		Get(gomock.Any(), MCRegistrationSecretFullName, gomock.Not(gomock.Nil()), gomock.Any()).
		Return(kerr.NewNotFound(schema.ParseGroupResource("Secret"), MCRegistrationSecretFullName.Name))

	cli.EXPECT().
		Get(gomock.Any(), MCLocalRegistrationSecretFullName, gomock.Not(gomock.Nil()), gomock.Any()).
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
	log := zap.S().With("somelogger")
	result, err := IgnoreNotFoundWithLog(kerr.NewNotFound(controllerruntime.GroupResource{}, ""), log)
	asserts.Nil(t, err)
	asserts.False(t, result.Requeue)

	otherErr := kerr.NewBadRequest("some other error")
	result, err = IgnoreNotFoundWithLog(otherErr, log)
	asserts.Nil(t, err)
	asserts.True(t, result.Requeue)
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
	curCluster1Status := clustersv1alpha1.ClusterLevelStatus{Name: "cluster1", State: clustersv1alpha1.Succeeded, Message: "success msg", LastUpdateTime: formattedConditionTimestamp}
	curCluster2Status := clustersv1alpha1.ClusterLevelStatus{Name: "cluster2", State: clustersv1alpha1.Failed, Message: "failure msg", LastUpdateTime: formattedConditionTimestamp}

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
		Message:        "cluster failed",
		State:          clustersv1alpha1.Failed,
		LastUpdateTime: formattedConditionTimestamp}
	newCluster2Status := clustersv1alpha1.ClusterLevelStatus{
		Name:           curCluster2Status.Name,
		Message:        "cluster succeeded",
		State:          clustersv1alpha1.Succeeded,
		LastUpdateTime: formattedConditionTimestamp}

	existingCondDiffTimestampCluster1 := clustersv1alpha1.Condition{
		Type: curConditions[0].Type, Status: curConditions[0].Status,
		Message: curConditions[0].Message, LastTransitionTime: otherTimestamp}

	existingCondDiffMessageCluster1 := clustersv1alpha1.Condition{
		Type: curConditions[0].Type, Status: curConditions[0].Status,
		Message: "Some other different message", LastTransitionTime: curConditions[0].LastTransitionTime}

	cluster1StatusDiffTimestamp := clustersv1alpha1.ClusterLevelStatus{
		Name:           curCluster1Status.Name,
		Message:        curCluster1Status.Message,
		State:          curCluster1Status.State,
		LastUpdateTime: otherTimestamp}

	newClusterStatus := clustersv1alpha1.ClusterLevelStatus{
		Name:           "newCluster",
		Message:        "cluster succeeded",
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

	// same condition, differing in condition message - needs update
	asserts.True(t, StatusNeedsUpdate(curStatus, existingCondDiffMessageCluster1, curCluster1Status))

	// same condition, differing in condition message - needs update
	asserts.True(t, StatusNeedsUpdate(curStatus, existingCondDiffMessageCluster1, cluster1StatusDiffTimestamp))

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
		Type: clustersv1alpha1.DeployComplete, Status: v1.ConditionTrue, Message: "cond1 msg", LastTransitionTime: formattedConditionTimestamp,
	}
	condition2 := clustersv1alpha1.Condition{
		Type: clustersv1alpha1.DeployFailed, Status: v1.ConditionTrue, Message: "cond2 msg", LastTransitionTime: formattedConditionTimestamp,
	}
	clusterState1 := CreateClusterLevelStatus(condition1, "cluster1")
	asserts.Equal(t, "cluster1", clusterState1.Name)
	asserts.Equal(t, clustersv1alpha1.Succeeded, clusterState1.State)
	asserts.Equal(t, formattedConditionTimestamp, clusterState1.LastUpdateTime)
	asserts.Equal(t, condition1.Message, clusterState1.Message)

	clusterState2 := CreateClusterLevelStatus(condition2, "somecluster")
	asserts.Equal(t, "somecluster", clusterState2.Name)
	asserts.Equal(t, clustersv1alpha1.Failed, clusterState2.State)
	asserts.Equal(t, formattedConditionTimestamp, clusterState2.LastUpdateTime)
	asserts.Equal(t, condition2.Message, clusterState2.Message)
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
		Get(gomock.Any(), secreteNameFullName, gomock.Not(gomock.Nil()), gomock.Any()).
		Times(times).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *v1.Secret, opts ...client.GetOption) error {
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

// TestDeleteAssociatedResource tests that if DeleteAssociatedResource is called
// the given resourceToDelete is deleted and the finalizer on the mcResource is removed
// GIVEN a MultiCluster resource and a resourceToDelete,
// WHEN TestDeleteAssociatedResource is called and the resourceToDelete is successfully deleted
// THEN the finalizer should be removed
// GIVEN a MultiCluster resource and a resourceToDelete,
// WHEN TestDeleteAssociatedResource is called and it fails to delete the resourceToDelete
// THEN the finalizer should NOT be removed
func TestDeleteAssociatedResource(t *testing.T) {
	mocker := gomock.NewController(t)
	cli := mocks.NewMockClient(mocker)

	mcResource := clustersv1alpha1.MultiClusterApplicationConfiguration{
		Spec: clustersv1alpha1.MultiClusterApplicationConfigurationSpec{
			Placement: clustersv1alpha1.Placement{
				Clusters: []clustersv1alpha1.Cluster{{Name: "mycluster"}},
			},
		},
	}
	mcResource.Name = "mymcappconfig"
	mcResource.Namespace = "myns"

	resourceToDeleteName := types.NamespacedName{Name: "myappconfig", Namespace: "myns"}
	finalizerToDelete := "thisfinalizergoes"
	finalizerNotDelete := "thisfinalizerstays"

	mcResource.SetFinalizers([]string{finalizerNotDelete, finalizerToDelete})

	// GIVEN that the deletion succeeds
	// THEN the finalizer should be removed

	// expect get and delete the app config with name resourceToDeleteName, mocking successful deletion
	expectGetAndDeleteAppConfig(t, cli, resourceToDeleteName, nil)

	// The finalizer should be removed
	cli.EXPECT().
		Update(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, mcApp *clustersv1alpha1.MultiClusterApplicationConfiguration, opts ...client.UpdateOption) error {
			asserts.NotContains(t, mcApp.GetFinalizers(), finalizerToDelete)
			asserts.Contains(t, mcApp.GetFinalizers(), finalizerNotDelete)
			return nil
		})

	err := DeleteAssociatedResource(context.TODO(), cli, &mcResource, finalizerToDelete, &v1alpha2.ApplicationConfiguration{}, resourceToDeleteName)
	asserts.Nil(t, err)

	// GIVEN that the deletion fails
	// THEN the finalizer should NOT be removed

	// expect get and delete the app config with name resourceToDeleteName, mocking FAILED deletion
	expectGetAndDeleteAppConfig(t, cli, resourceToDeleteName, errors.New("I will not delete you, resource"))

	// There should be no more interactions i.e. the finalizer should not be removed

	err = DeleteAssociatedResource(context.TODO(), cli, &mcResource, finalizerToDelete, &v1alpha2.ApplicationConfiguration{}, resourceToDeleteName)
	asserts.NotNil(t, err)

	mocker.Finish()
}

// TestRequeueWithDelay tests that when a result is requested it has requeue set to true and a requeue after greater
// than 2 seconds
// GIVEN a need for a requeue result
// WHEN NewRequeueWithDelay is called
// THEN the returned result indicates a requeue with a requeueAfter time greaater than or equal to 2 seconds
func TestRequeueWithDelay(t *testing.T) {
	result := NewRequeueWithDelay()
	asserts.True(t, result.Requeue)
	asserts.GreaterOrEqual(t, result.RequeueAfter.Seconds(), time.Duration(2).Seconds())
}

// GIVEN a need for a testing a result for requeueing
// WHEN ShouldRequeue is called
// THEN a value of true is returned if Requeue is set to true or the RequeueAfter time is greater than 0, false otherwise
func TestShouldRequeue(t *testing.T) {
	val := ShouldRequeue(reconcile.Result{Requeue: true})
	asserts.True(t, val)
	val = ShouldRequeue(reconcile.Result{Requeue: false})
	asserts.False(t, val)
	val = ShouldRequeue(reconcile.Result{RequeueAfter: time.Duration(2)})
	asserts.True(t, val)
	val = ShouldRequeue(reconcile.Result{RequeueAfter: time.Duration(0)})
	asserts.False(t, val)
}

func expectGetAndDeleteAppConfig(t *testing.T, cli *mocks.MockClient, name types.NamespacedName, deleteErr error) {
	cli.EXPECT().
		Get(gomock.Any(), name, gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, appConfig *v1alpha2.ApplicationConfiguration, opts ...client.GetOption) error {
			appConfig.Name = name.Name
			appConfig.Namespace = name.Namespace
			return nil
		})

	cli.EXPECT().
		Delete(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, appConfig *v1alpha2.ApplicationConfiguration, opt ...client.DeleteOption) error {
			asserts.Equal(t, name.Name, appConfig.Name)
			asserts.Equal(t, name.Namespace, appConfig.Namespace)
			return deleteErr
		})
}
