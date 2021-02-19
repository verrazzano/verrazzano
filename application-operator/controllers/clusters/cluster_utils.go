// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusters

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"
)

// StatusNeedsUpdate determines based on the current state and conditions of a MultiCluster
// resource, as well as the new state and condition to be set, whether the status update
// needs to be done
func StatusNeedsUpdate(curConditions []clustersv1alpha1.Condition, curState clustersv1alpha1.StateType,
	newCondition clustersv1alpha1.Condition, newState clustersv1alpha1.StateType) bool {
	if newState == clustersv1alpha1.Failed {
		return true
	}
	if newState != curState {
		return true
	}
	foundStatus := false
	for _, existingCond := range curConditions {
		if existingCond.Status == newCondition.Status &&
			existingCond.Message == newCondition.Message &&
			existingCond.Type == newCondition.Type {
			foundStatus = true
		}
	}
	return !foundStatus
}

// GetConditionAndStateFromResult - Based on the result of a create/update operation on the
// embedded resource, returns the Condition and State that must be set on a MultiCluster
// resource's Status
func GetConditionAndStateFromResult(err error, opResult controllerutil.OperationResult, msgPrefix string) (clustersv1alpha1.Condition, clustersv1alpha1.StateType) {
	var condition clustersv1alpha1.Condition
	var state clustersv1alpha1.StateType
	if err != nil {
		condition = clustersv1alpha1.Condition{
			Type:               clustersv1alpha1.DeployFailed,
			Status:             corev1.ConditionTrue,
			Message:            err.Error(),
			LastTransitionTime: time.Now().Format(time.RFC3339),
		}
		state = clustersv1alpha1.Failed
	} else {
		msg := fmt.Sprintf("%v %v", msgPrefix, opResult)
		condition = clustersv1alpha1.Condition{
			Type:               clustersv1alpha1.DeployComplete,
			Status:             corev1.ConditionTrue,
			Message:            msg,
			LastTransitionTime: time.Now().Format(time.RFC3339),
		}
		state = clustersv1alpha1.Ready
	}

	return condition, state
}

// NewScheme creates a new scheme that includes this package's object to use for testing
func NewScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	clustersv1alpha1.AddToScheme(scheme)
	return scheme
}

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
	yamlBytes, err := ioutil.ReadFile(filename)
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
