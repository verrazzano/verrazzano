// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vzchecks

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	client2 "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

var (
	allNotMet      = []string{cpuReqMsg, memoryReqMsg, storageReqMsg}
	onlyStorageMet = []string{cpuReqMsg, memoryReqMsg}
	onlyMemoryMet  = []string{cpuReqMsg, storageReqMsg}
	onlyCPUMet     = []string{memoryReqMsg, storageReqMsg}
	//onlyNodeCountMet       = []string{cpuReqMsg, memoryReqMsg, storageReqMsg}
	storageAndMemoryMet = []string{cpuReqMsg}
	storageAndCPUMet    = []string{memoryReqMsg}
	//storageAndNodeCountMet = []string{memoryReqMsg, cpuReqMsg}
	memoryAndCPUMet = []string{storageReqMsg}
	//memoryAndNodeCountMet  = []string{cpuReqMsg, storageReqMsg}
	nodeCountAndCPUMet = []string{memoryReqMsg, storageReqMsg}
	memoryNotMet       = []string{memoryReqMsg}
	//nodeNotMet             = []string{nodeCountReqMsg}
	allMet = []string{}
)

type testData = struct {
	profile   ProfileType
	nodeCount int
	cpu       string
	memory    string
	storage   string
}

// TestPrerequisiteCheck tests prerequisite checks for various profiles
func TestPrerequisiteCheck(t *testing.T) {
	var tests = []struct {
		data     testData
		errCount int
		errTypes []string
	}{
		{testData{Prod, 2, "1", "423Ki", "50G"}, 7, allNotMet},
		//{testData{Prod, 3, "1", "12G", "50G"}, 9, onlyNodeCountMet},
		{testData{Prod, 2, "1", "12G", "100G"}, 5, onlyStorageMet},
		{testData{Prod, 2, "4", "12G", "50G"}, 5, onlyCPUMet},
		{testData{Prod, 2, "1", "32G", "50G"}, 5, onlyMemoryMet},
		{testData{Prod, 1, "6", "35G", "50G"}, 2, memoryAndCPUMet},
		{testData{Prod, 2, "5", "12G", "100G"}, 3, storageAndCPUMet},
		{testData{Prod, 2, "3", "32G", "100G"}, 3, storageAndMemoryMet},
		{testData{Prod, 5, "6", "32G", "500G"}, 0, allMet},
		{testData{Dev, 0, "1", "10G", "50G"}, 1, allNotMet},
		//{testData{Dev, 1, "1", "12G", "50G"}, 3, onlyNodeCountMet},
		//{testData{ManagedCluster, 2, "1", "12G", "100G"}, 4, storageAndNodeCountMet},
		{testData{ManagedCluster, 2, "4", "12G", "50G"}, 4, nodeCountAndCPUMet},
		{testData{Dev, 2, "2", "2G", "50G"}, 4, nodeCountAndCPUMet},
		//{testData{Dev, 1, "1", "5G", "100G"}, 2, storageAndNodeCountMet},
		{testData{ManagedCluster, 2, "5", "12G", "100G"}, 2, memoryNotMet},
		//{testData{Dev, 1, "1", "32G", "10G"}, 2, memoryAndNodeCountMet},
		//{testData{Dev, 0, "", "", ""}, 1, nodeNotMet},
		{testData{"unspecified", 1, "2", "24G", "50G"}, 0, allMet},
	}

	for _, tt := range tests {
		t.Run(string(tt.data.profile), func(t *testing.T) {
			client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
				getNodes(tt.data)...).Build()
			errs := PrerequisiteCheck(client, tt.data.profile)
			assert.Equal(t, tt.errCount, len(errs))
			vzReq := getVZRequirement(tt.data.profile)
			for _, errMsg := range tt.errTypes {
				for i := 1; i <= tt.data.nodeCount; i++ {
					nodeName := fmt.Sprintf("node%d", i)
					expectedMsg := getExpectedMessage(errMsg, nodeName, vzReq, tt.data)
					assert.Equal(t, true, errorSliceContainsMessage(errs, expectedMsg),
						"Expected error message \"%s\" not found", expectedMsg)
				}
			}
		})
	}
}

// errorSliceContainsMessage checks for a error message in a slice of errors
// errs is the error slice to search. May be nil.
// msg is the string to search for in the errs.
// Returns true if the string is found in the slice and false otherwise.
func errorSliceContainsMessage(errs []error, msg string) bool {
	for _, err := range errs {
		if err.Error() == msg {
			return true
		}
	}
	return false
}

func getNodes(data testData) []client2.Object {
	var nodes []client2.Object
	for i := 1; i <= data.nodeCount; i++ {
		nodes = append(nodes, &v1.Node{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Node",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("node%d", i),
			},
			Status: v1.NodeStatus{
				Allocatable: v1.ResourceList{
					"cpu":               resource.MustParse(data.cpu),
					"memory":            resource.MustParse(data.memory),
					"ephemeral-storage": resource.MustParse(data.storage),
				},
			},
		})
	}
	return nodes
}

func getExpectedMessage(errMsg string, nodeName string, vzReq VZRequirement, data testData) string {
	expectedMsg := ""
	switch errMsg {
	//case nodeCountReqMsg:
	//	expectedMsg = fmt.Sprintf(errMsg, vzReq.nodeCount, data.nodeCount)
	case cpuReqMsg:
		expectedMsg = fmt.Sprintf(errMsg, vzReq.cpu.allocatable.Value(),
			nodeName, data.cpu)
	case memoryReqMsg:
		expectedMsg = fmt.Sprintf(errMsg, convertQuantityToString(vzReq.memory.allocatable),
			nodeName, convertQuantityToString(resource.MustParse(data.memory)))
	case storageReqMsg:
		expectedMsg = fmt.Sprintf(errMsg, convertQuantityToString(vzReq.ephemeralStorage.allocatable),
			nodeName, convertQuantityToString(resource.MustParse(data.storage)))
	}
	return expectedMsg
}
