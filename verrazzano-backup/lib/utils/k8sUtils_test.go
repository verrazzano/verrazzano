// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package utils_test

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/verrazzano-backup/lib/constants"
	"github.com/verrazzano/verrazzano/verrazzano-backup/lib/utils"
	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sync"
	"testing"
)

var (
	TestPod = v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "foo",
			Namespace:   "foo",
			Annotations: map[string]string{},
		},
		Status: v1.PodStatus{
			Phase: "Running",
			Conditions: []v1.PodCondition{
				{
					Type:   "Ready",
					Status: "True",
				},
				{
					Type:   "NotReady",
					Status: "True",
				},
			},
		},
	}
)

// TestPopulateConnData tests the PopulateConnData method for the following use case.
// GIVEN a velero backup name
// WHEN velero backup is in progress
// THEN fetches the secret associate with velero backup
func TestPopulateConnData(t *testing.T) {
	t.Parallel()
	log, f := logHelper()
	defer os.Remove(f)
	var clientk client.Client
	k8s := utils.K8s(&utils.K8sImpl{})
	dclient := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme())
	conData, err := k8s.PopulateConnData(dclient, clientk, constants.VeleroNameSpace, "Foo", log)
	assert.Nil(t, conData)
	assert.NotNil(t, err)
}

// TestGetBackupStorageLocation tests the GetBackupStorageLocation method for the following use case.
// GIVEN a velero backup storage location name
// WHEN invoked
// THEN fetches backup storage location object
func TestGetBackupStorageLocation(t *testing.T) {
	t.Parallel()
	log, f := logHelper()
	defer os.Remove(f)
	k8s := utils.K8s(&utils.K8sImpl{})
	dclient := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme())
	_, err := k8s.GetBackupStorageLocation(dclient, "system", "fsl", log)
	assert.NotNil(t, err)
}

// TestGetBackup tests the GetBackup method for the following use case.
// GIVEN a velero backup name
// WHEN invoked
// THEN fetches backup object
func TestGetBackup(t *testing.T) {
	t.Parallel()
	log, f := logHelper()
	defer os.Remove(f)
	k8s := utils.K8s(&utils.K8sImpl{})
	dclient := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme())
	_, err := k8s.GetBackup(dclient, "system", "foo", log)
	assert.NotNil(t, err)
}

// TestCheckPodStatus tests the CheckPodStatus method for the following use case.
// GIVEN a pod name
// WHEN invoked
// THEN fetches pod status and monitors it depending on the checkFlag
func TestCheckPodStatus(t *testing.T) {
	t.Parallel()
	log, f := logHelper()
	defer os.Remove(f)
	k8s := utils.K8s(&utils.K8sImpl{})
	var wg sync.WaitGroup

	fc := fake.NewSimpleClientset()
	wg.Add(1)
	err := k8s.CheckPodStatus(fc, "foo", "foo", "up", "10m", log, &wg)
	log.Infof("%v", err)
	assert.NotNil(t, err)
	wg.Wait()

	fc = fake.NewSimpleClientset(&TestPod)
	wg.Add(1)
	err = k8s.CheckPodStatus(fc, "foo", "foo", "up", "10m", log, &wg)
	log.Infof("%v", err)
	assert.Nil(t, err)
	wg.Wait()

	fc = fake.NewSimpleClientset(&TestPod)
	wg.Add(1)
	err = k8s.CheckPodStatus(fc, "foo", "foo", "down", "1s", log, &wg)
	log.Infof("%v", err)
	assert.NotNil(t, err)
	wg.Wait()
}

// TestCheckAllPodsAfterRestore tests the CheckAllPodsAfterRestore method for the following use case.
// GIVEN k8s client
// WHEN restore is complete
// THEN checks kibana and ingest pods are Ready after reboot
func TestCheckAllPodsAfterRestore(t *testing.T) {
	t.Parallel()
	IngestLabel := make(map[string]string)
	KibanaLabel := make(map[string]string)
	IngestLabel["app"] = "system-es-ingest"
	KibanaLabel["app"] = "system-kibana"
	IngestPod := v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "foo",
			Namespace:   constants.VerrazzanoNameSpaceName,
			Annotations: map[string]string{},
			Labels:      IngestLabel,
		},
		Status: v1.PodStatus{
			Phase: "Running",
			Conditions: []v1.PodCondition{
				{
					Type:   "Ready",
					Status: "True",
				},
				{
					Type:   "NotReady",
					Status: "True",
				},
			},
		},
	}

	KibanaPod := v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "bar",
			Namespace:   constants.VerrazzanoNameSpaceName,
			Annotations: map[string]string{},
			Labels:      KibanaLabel,
		},
		Status: v1.PodStatus{
			Phase: "Running",
			Conditions: []v1.PodCondition{
				{
					Type:   "Ready",
					Status: "True",
				},
				{
					Type:   "NotReady",
					Status: "True",
				},
			},
		},
	}

	log, f := logHelper()
	defer os.Remove(f)
	k8s := utils.K8s(&utils.K8sImpl{})
	os.Setenv(constants.OpenSearchHealthCheckTimeoutKey, "1s")

	fc := fake.NewSimpleClientset()
	err := k8s.CheckAllPodsAfterRestore(fc, log)
	log.Infof("%v", err)
	assert.Nil(t, err)

	fc = fake.NewSimpleClientset(&IngestPod, &KibanaPod)
	err = k8s.CheckAllPodsAfterRestore(fc, log)
	log.Infof("%v", err)
	assert.Nil(t, err)

}

// TestCheckDeployment tests the CheckDeployment method for the following use case.
// GIVEN k8s client
// WHEN restore is complete
// THEN checks kibana deployment is present on system
func TestCheckDeployment(t *testing.T) {
	t.Parallel()
	KibanaLabel := make(map[string]string)
	KibanaLabel["verrazzano-component"] = "kibana"
	PrimaryDeploy := apps.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "foo",
			Namespace:   constants.VerrazzanoNameSpaceName,
			Annotations: map[string]string{},
			Labels:      KibanaLabel,
		},
	}

	SecondaryDeploy := apps.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "bar",
			Namespace:   constants.VerrazzanoNameSpaceName,
			Annotations: map[string]string{},
			Labels:      KibanaLabel,
		},
	}

	log, f := logHelper()
	defer os.Remove(f)
	k8s := utils.K8s(&utils.K8sImpl{})
	os.Setenv(constants.OpenSearchHealthCheckTimeoutKey, "1s")

	fmt.Println("Deployment not found")
	fc := fake.NewSimpleClientset()
	ok, err := k8s.CheckDeployment(fc, constants.KibanaDeploymentLabelSelector, constants.VerrazzanoNameSpaceName, log)
	log.Infof("%v", err)
	assert.Nil(t, err)
	assert.Equal(t, ok, false)

	fmt.Println("Deployment found")
	fc = fake.NewSimpleClientset(&PrimaryDeploy)
	ok, err = k8s.CheckDeployment(fc, constants.KibanaDeploymentLabelSelector, constants.VerrazzanoNameSpaceName, log)
	log.Infof("%v", err)
	assert.Nil(t, err)
	assert.Equal(t, ok, true)

	fmt.Println("Multiple Deployments found")
	fc = fake.NewSimpleClientset(&PrimaryDeploy, &SecondaryDeploy)
	ok, err = k8s.CheckDeployment(fc, constants.KibanaDeploymentLabelSelector, constants.VerrazzanoNameSpaceName, log)
	log.Infof("%v", err)
	assert.Nil(t, err)
	assert.Equal(t, ok, false)
}

// TestIsPodReady tests the IsPodReady method for the following use case.
// GIVEN k8s client
// WHEN restore is complete
// THEN checks is pod is in ready state
func TestIsPodReady(t *testing.T) {
	t.Parallel()

	ReadyPod := v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "foo",
			Namespace:   constants.VerrazzanoNameSpaceName,
			Annotations: map[string]string{},
		},
		Status: v1.PodStatus{
			Phase: "Running",
			Conditions: []v1.PodCondition{
				{
					Type:   "Initialized",
					Status: "True",
				},
				{
					Type:   "Ready",
					Status: "True",
				},
				{
					Type:   "ContainersReady",
					Status: "True",
				},
				{
					Type:   "PodScheduled",
					Status: "True",
				},
			},
		},
	}

	NotReadyPod := v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "foo",
			Namespace:   constants.VerrazzanoNameSpaceName,
			Annotations: map[string]string{},
		},
		Status: v1.PodStatus{
			Phase: "Running",
			Conditions: []v1.PodCondition{
				{
					Type:   "Initialized",
					Status: "True",
				},
				{
					Type:   "ContainersReady",
					Status: "True",
				},
				{
					Type:   "PodScheduled",
					Status: "True",
				},
			},
		},
	}

	log, f := logHelper()
	defer os.Remove(f)
	k8s := utils.K8s(&utils.K8sImpl{})
	os.Setenv(constants.OpenSearchHealthCheckTimeoutKey, "1s")

	ok, err := k8s.IsPodReady(&ReadyPod, log)
	log.Infof("%v", err)
	assert.Nil(t, err)
	assert.Equal(t, ok, true)

	ok, err = k8s.IsPodReady(&NotReadyPod, log)
	log.Infof("%v", err)
	assert.Nil(t, err)
	assert.Equal(t, ok, false)

}
