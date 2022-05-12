// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package utils_test

import (
	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/verrazzano-backup/lib/constants"
	"github.com/verrazzano/verrazzano/verrazzano-backup/lib/utils"
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
				v1.PodCondition{
					Type:   "Ready",
					Status: "True",
				},
				v1.PodCondition{
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
	spew.Dump(conData)
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
	//_, err := k8s.GetBackup(dclient, "system", "foo", log)
	//assert.NotNil(t, err)
	wg.Add(1)
	err = k8s.CheckPodStatus(fc, "foo", "foo", "down", "1s", log, &wg)
	log.Infof("%v", err)
	assert.NotNil(t, err)
	wg.Wait()
}

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
				v1.PodCondition{
					Type:   "Ready",
					Status: "True",
				},
				v1.PodCondition{
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
				v1.PodCondition{
					Type:   "Ready",
					Status: "True",
				},
				v1.PodCondition{
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
