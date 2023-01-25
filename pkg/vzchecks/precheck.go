// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vzchecks

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8s/node"
	k8score "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
)

// PrerequisiteCheck checks the prerequisites before applying the Verrazzano CR
func PrerequisiteCheck(client clipkg.Client, profile ProfileType) []error {
	return preCheck(client, profile)
}

func preCheck(client clipkg.Client, profile ProfileType) []error {
	var errs []error
	vzReq := getVZRequirement(profile)
	if vzReq == (VZRequirement{}) {
		return errs
	}
	nodeList, err := node.GetK8sNodeList(client)
	if err != nil {
		return []error{err}
	}
	//if len(nodeList.Items) < vzReq.nodeCount {
	//	errs = append(errs, fmt.Errorf(nodeCountReqMsg, vzReq.nodeCount, len(nodeList.Items)))
	//}

	for _, node := range nodeList.Items {
		var cpuAllocatable = node.Status.Allocatable[k8score.ResourceCPU]
		var memoryAllocatable = node.Status.Allocatable[k8score.ResourceMemory]
		var storageAllocatable = node.Status.Allocatable[k8score.ResourceEphemeralStorage]
		if cpuAllocatable.MilliValue() < vzReq.cpu.allocatable.MilliValue() {
			errs = append(errs, fmt.Errorf(cpuReqMsg, vzReq.cpu.allocatable.Value(),
				node.Name, cpuAllocatable.Value()))
		}
		if memoryAllocatable.MilliValue() < vzReq.memory.allocatable.MilliValue() {
			errs = append(errs, fmt.Errorf(memoryReqMsg, convertQuantityToString(vzReq.memory.allocatable),
				node.Name, convertQuantityToString(memoryAllocatable)))
		}
		if storageAllocatable.MilliValue() < vzReq.ephemeralStorage.allocatable.MilliValue() {
			errs = append(errs, fmt.Errorf(storageReqMsg, convertQuantityToString(vzReq.ephemeralStorage.allocatable),
				node.Name, convertQuantityToString(storageAllocatable)))
		}
	}
	return errs
}

// convertQuantityToString converts the given quantity value to a Gigabyte value
func convertQuantityToString(quantity resource.Quantity) string {
	gigabyteFormat := resource.MustParse("1G")
	m := float64(quantity.Value()) / float64(gigabyteFormat.Value())
	return strconv.FormatFloat(m, 'g', -1, 64)
}
