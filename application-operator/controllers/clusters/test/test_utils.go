// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusterstest

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"

	"github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
	"github.com/verrazzano/verrazzano/application-operator/mocks"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/yaml"
)

// UnitTestClusterName - cluster name used in unit tests
const UnitTestClusterName = "cluster1"

// NewRequest creates a new reconciler request for testing
// namespace - The namespace to use in the request
// name - The name to use in the request
func NewRequest(namespace string, name string) ctrl.Request {
	return ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		},
	}
}

// ReadYaml2Json reads the testdata YAML file at the given path, converts it to JSON and returns
// a byte slice containing the JSON
func ReadYaml2Json(filename string) ([]byte, error) {
	yamlBytes, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read test data file %s: %s", filename, err.Error())
	}
	jsonBytes, err := yaml.YAMLToJSON(yamlBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshall YAML to JSON in file %s: %s", filename, err.Error())
	}
	return jsonBytes, nil
}

// ReadContainerizedWorkload reads the raw workload (typically from an OAM component) into
// a ContainerizedWorkload object
func ReadContainerizedWorkload(rawWorkload runtime.RawExtension) (v1alpha2.ContainerizedWorkload, error) {
	ctrWorkload := v1alpha2.ContainerizedWorkload{}
	workloadBytes, err := rawWorkload.MarshalJSON()
	if err != nil {
		return ctrWorkload, err
	}
	err = json.Unmarshal(workloadBytes, &ctrWorkload)
	return ctrWorkload, err
}

// DoExpectGetMCRegistrationSecret adds an expectation to the given MockClient to expect a Get
// call for the managed cluster registration secret, and populate it with the cluster-name
func DoExpectGetMCRegistrationSecret(cli *mocks.MockClient) {
	// expect a call to fetch the MCRegistrationSecret and return a fake one with a specific cluster name
	mockRegistrationSecretData := map[string][]byte{constants.ClusterNameData: []byte(UnitTestClusterName)}
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{
			Namespace: clusters.MCRegistrationSecretFullName.Namespace,
			Name:      clusters.MCRegistrationSecretFullName.Name},
			gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, secret *v1.Secret) error {
			secret.Data = mockRegistrationSecretData
			secret.ObjectMeta.Namespace = clusters.MCRegistrationSecretFullName.Namespace
			secret.ObjectMeta.Name = clusters.MCRegistrationSecretFullName.Name
			return nil
		})
}

// AssertMultiClusterResourceStatus asserts that the status, conditions and cluster level status
// on the MultiClusterResourceStatus are as expected
func AssertMultiClusterResourceStatus(assert *assert.Assertions,
	actualStatus v1alpha1.MultiClusterResourceStatus,
	expectedState v1alpha1.StateType,
	expectedConditionType v1alpha1.ConditionType,
	expectedConditionStatus v1.ConditionStatus) {

	assert.Equal(expectedState, actualStatus.State)
	assert.Equal(1, len(actualStatus.Conditions))
	assert.Equal(expectedConditionStatus, actualStatus.Conditions[0].Status)
	assert.Equal(expectedConditionType, actualStatus.Conditions[0].Type)

	assert.Equal(1, len(actualStatus.Clusters))
	assert.Equal(UnitTestClusterName, actualStatus.Clusters[0].Name)
	assert.Equal(expectedState, actualStatus.Clusters[0].State)
}

// DoExpectUpdateState checks that the status writer was retrieved and an update of the
// resource's state to the given value was called
func DoExpectUpdateState(t *testing.T, cli *mocks.MockClient, statusWriter *mocks.MockStatusWriter, mcAppConfig clusters.MultiClusterResource, expectedState v1alpha1.StateType) {
	// expect a call to update the status of the MultiClusterApplicationConfiguration
	cli.EXPECT().Status().Return(statusWriter)

	// the status update should be to success status/conditions on the MultiClusterApplicationConfiguration
	statusWriter.EXPECT().
		Update(gomock.Any(), gomock.AssignableToTypeOf(mcAppConfig), gomock.Any()).
		DoAndReturn(func(ctx context.Context, mcAppConfig clusters.MultiClusterResource, opts ...client.UpdateOption) error {
			assert.Equal(t, v1alpha1.Pending, mcAppConfig.GetStatus().State)
			return nil
		})
}

// ExpectDeleteAssociatedResource adds an expectation to the given MockClient to expect a Get
// call for the runtime resource and a subsequent request to delete it
func ExpectDeleteAssociatedResource(cli *mocks.MockClient, resource runtime.Object, name types.NamespacedName) {
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: name.Namespace, Name: name.Name}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, obj runtime.Object) error {
			return nil
		})

	cli.EXPECT().
		Delete(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, resource runtime.Object, opts ...client.DeleteOption) error {
			return nil
		})

}
