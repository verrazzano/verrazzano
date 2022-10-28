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
	allNotMet              = []string{nodeCountReqMsg, cpuReqMsg, memoryReqMsg, storageReqMsg}
	onlyStorageMet         = []string{nodeCountReqMsg, cpuReqMsg, memoryReqMsg}
	onlyMemoryMet          = []string{nodeCountReqMsg, cpuReqMsg, storageReqMsg}
	onlyCPUMet             = []string{nodeCountReqMsg, memoryReqMsg, storageReqMsg}
	onlyNodeCountMet       = []string{cpuReqMsg, memoryReqMsg, storageReqMsg}
	storageAndMemoryMet    = []string{nodeCountReqMsg, cpuReqMsg}
	storageAndCPUMet       = []string{nodeCountReqMsg, memoryReqMsg}
	storageAndNodeCountMet = []string{memoryReqMsg, cpuReqMsg}
	memoryAndCPUMet        = []string{nodeCountReqMsg, storageReqMsg}
	memoryAndNodeCountMet  = []string{cpuReqMsg, storageReqMsg}
	NodeCountAndCPUMet     = []string{memoryReqMsg, storageReqMsg}
	allMet                 = []string{}
)

// TestPrerequisiteCheck tests prerequisite checks for various profiles
func TestPrerequisiteCheck(t *testing.T) {
	var tests = []struct {
		profile   ProfileType
		nodeCount int
		cpu       string
		memory    string
		storage   string
		hasError  bool
		errMsgs   []string
	}{
		{Prod, 2, "1", "423Ki", "50G", true, allNotMet},
		{Prod, 3, "1", "12G", "50G", true, onlyNodeCountMet},
		{Prod, 2, "1", "12G", "100G", true, onlyStorageMet},
		{Prod, 2, "4", "12G", "50G", true, onlyCPUMet},
		{Prod, 2, "1", "32G", "50G", true, onlyMemoryMet},
		{Prod, 1, "6", "35G", "50G", true, memoryAndCPUMet},
		{Prod, 2, "5", "12G", "100G", true, storageAndCPUMet},
		{Prod, 2, "3", "32G", "100G", true, storageAndMemoryMet},
		{Prod, 5, "6", "32G", "500G", false, allMet},
		{Dev, 0, "1", "10G", "50G", true, allNotMet},
		{Dev, 1, "1", "12G", "50G", true, onlyNodeCountMet},
		{ManagedCluster, 2, "1", "12G", "100G", true, onlyStorageMet},
		{ManagedCluster, 2, "4", "12G", "50G", true, onlyCPUMet},
		{Dev, 2, "2", "2G", "50G", true, NodeCountAndCPUMet},
		{Dev, 1, "1", "5G", "100G", true, storageAndNodeCountMet},
		{ManagedCluster, 2, "5", "12G", "100G", true, storageAndCPUMet},
		{Dev, 2, "1", "32G", "10G", true, memoryAndNodeCountMet},
		{Dev, 3, "3", "20G", "500G", false, allMet},
	}

	for _, tt := range tests {
		t.Run(string(tt.profile), func(t *testing.T) {
			var nodes []client2.Object
			for i := 1; i <= tt.nodeCount; i++ {
				nodes = append(nodes, node(fmt.Sprintf("node%d", i), tt.cpu, tt.memory, tt.storage))
			}
			client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
				nodes...).Build()
			errs := PrerequisiteCheck(client, tt.profile)
			if tt.hasError {
				assert.Greater(t, len(errs), 0)
				errCount := 0
				vzReq := getVZRequirement(tt.profile)
				for _, errMsg := range tt.errMsgs {
					expectedMsg := ""
					if errMsg == nodeCountReqMsg {
						expectedMsg = fmt.Sprintf(errMsg, tt.profile, vzReq.nodeCount, tt.nodeCount)
						errPresent := ErrorsContainsMessage(errs, expectedMsg)
						assert.Equal(t, true, errPresent, "Expected error message \"%s\" not found", expectedMsg)
						errCount++
						continue
					}
					for i := 1; i <= tt.nodeCount; i++ {
						switch errMsg {
						case cpuReqMsg:
							expectedMsg = fmt.Sprintf(errMsg, tt.profile, vzReq.cpu.allocatable.Value(),
								fmt.Sprintf("node%d", i), tt.cpu)
						case memoryReqMsg:
							expectedMsg = fmt.Sprintf(errMsg, tt.profile, convertQuantityToString(vzReq.memory.allocatable),
								fmt.Sprintf("node%d", i), convertQuantityToString(resource.MustParse(tt.memory)))
						case storageReqMsg:
							expectedMsg = fmt.Sprintf(errMsg, tt.profile, convertQuantityToString(vzReq.ephemeralStorage.allocatable),
								fmt.Sprintf("node%d", i), convertQuantityToString(resource.MustParse(tt.storage)))
						}
						errPresent := ErrorsContainsMessage(errs, expectedMsg)
						assert.Equal(t, true, errPresent, "Expected error message \"%s\" not found", expectedMsg)
						errCount++
					}
				}
				assert.Equal(t, errCount, len(errs))
			} else {
				assert.Equal(t, len(errs), 0)
			}
		})
	}
}

func node(name string, cpu string, memory string, ephemeralStorage string) *v1.Node {
	return &v1.Node{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Node",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Status: v1.NodeStatus{
			Allocatable: v1.ResourceList{
				"cpu":               resource.MustParse(cpu),
				"memory":            resource.MustParse(memory),
				"ephemeral-storage": resource.MustParse(ephemeralStorage),
			},
		},
	}
}

// ErrorsContainsMessage checks for a error message in a slice of errors
// errs is the error slice to search. May be nil.
// msg is the string to search for in the errs.
// Returns true if the string is found in the slice and false otherwise.
func ErrorsContainsMessage(errs []error, msg string) bool {
	for _, err := range errs {
		if err.Error() == msg {
			return true
		}
	}
	return false
}
