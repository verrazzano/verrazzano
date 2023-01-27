// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vzchecks

import (
	k8score "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// ProfileType is the type of installation profile.
type ProfileType string

type ResourceType string

type VZRequirement struct {
	nodeCount        int
	cpu              *resourceInfo
	memory           *resourceInfo
	ephemeralStorage *resourceInfo
}

type resourceInfo struct {
	resourceType ResourceType
	allocatable  resource.Quantity
}

const (
	// Dev identifies the development install profile
	Dev ProfileType = "dev"
	// Prod identifies the production install profile
	Prod ProfileType = "prod"
	// ManagedCluster identifies the production managed-cluster install profile
	ManagedCluster ProfileType = "managed-cluster"
)

const (
	CPU              = ResourceType(k8score.ResourceCPU)
	Memory           = ResourceType(k8score.ResourceMemory)
	EphemeralStorage = ResourceType(k8score.ResourceEphemeralStorage)
)

const (
	//nodeCountReqMsg = "minimum required number of worker nodes is %d but the available number of worker nodes is %d"
	cpuReqMsg     = "minimum required CPUs is %v but the CPUs on node %s is %v"
	memoryReqMsg  = "minimum required memory is %sG but the memory on node %s is %sG"
	storageReqMsg = "minimum required ephemeral storage is %sG but the ephemeral storage on node %s is %sG"
)

var (
	DevReq = VZRequirement{
		nodeCount:        1,
		cpu:              &resourceInfo{resourceType: CPU, allocatable: resource.MustParse("2")},
		memory:           &resourceInfo{resourceType: Memory, allocatable: resource.MustParse("16G")},
		ephemeralStorage: &resourceInfo{resourceType: EphemeralStorage, allocatable: resource.MustParse("100G")},
	}
	ProdReq = VZRequirement{
		nodeCount:        3,
		cpu:              &resourceInfo{resourceType: CPU, allocatable: resource.MustParse("4")},
		memory:           &resourceInfo{resourceType: Memory, allocatable: resource.MustParse("32G")},
		ephemeralStorage: &resourceInfo{resourceType: EphemeralStorage, allocatable: resource.MustParse("100G")},
	}
	ManagedReq = VZRequirement{
		nodeCount:        1,
		cpu:              &resourceInfo{resourceType: CPU, allocatable: resource.MustParse("4")},
		memory:           &resourceInfo{resourceType: Memory, allocatable: resource.MustParse("32G")},
		ephemeralStorage: &resourceInfo{resourceType: EphemeralStorage, allocatable: resource.MustParse("100G")},
	}
	profileMap = map[ProfileType]VZRequirement{
		Dev:            DevReq,
		Prod:           ProdReq,
		ManagedCluster: ManagedReq,
	}
)

// getVZRequirement gets the VZRequirement based on the profile passed
func getVZRequirement(requestedProfile ProfileType) VZRequirement {
	if len(requestedProfile) == 0 {
		// Default profile is Prod
		requestedProfile = Prod
	}
	if val, ok := profileMap[requestedProfile]; ok {
		return val
	}
	return VZRequirement{}
}
