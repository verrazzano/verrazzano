// Copyright (c) 2022, Oracle and/or its affiliates.
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
	nodeCountReqMsg = "minimum required number of worker nodes for %s profile is %d but the available number of worker nodes is %d"
	cpuReqMsg       = "minimum required CPUs for %s profile is %v but the CPUs on node %s is %v"
	memoryReqMsg    = "minimum required memory for %s profile is %sG but the memory on node %s is %sG"
	storageReqMsg   = "minimum required ephemeral storage for %s profile is %sG but the ephemeral storage on node %s is %sG"
)

var (
	DevProfile = VZRequirement{
		nodeCount:        1,
		cpu:              &resourceInfo{resourceType: CPU, allocatable: resource.MustParse("2")},
		memory:           &resourceInfo{resourceType: Memory, allocatable: resource.MustParse("16G")},
		ephemeralStorage: &resourceInfo{resourceType: EphemeralStorage, allocatable: resource.MustParse("100G")},
	}
	ProdProfile = VZRequirement{
		nodeCount:        3,
		cpu:              &resourceInfo{resourceType: CPU, allocatable: resource.MustParse("4")},
		memory:           &resourceInfo{resourceType: Memory, allocatable: resource.MustParse("32G")},
		ephemeralStorage: &resourceInfo{resourceType: EphemeralStorage, allocatable: resource.MustParse("100G")},
	}
	ManagedProfile = VZRequirement{
		nodeCount:        3,
		cpu:              &resourceInfo{resourceType: CPU, allocatable: resource.MustParse("4")},
		memory:           &resourceInfo{resourceType: Memory, allocatable: resource.MustParse("32G")},
		ephemeralStorage: &resourceInfo{resourceType: EphemeralStorage, allocatable: resource.MustParse("100G")},
	}
	profileMap = map[ProfileType]VZRequirement{
		Dev:            DevProfile,
		Prod:           ProdProfile,
		ManagedCluster: ManagedProfile,
	}
)

// getVZRequirement gets the VZRequirement based on the profile passed
func getVZRequirement(requestedProfile ProfileType) VZRequirement {
	if val, ok := profileMap[requestedProfile]; ok {
		return val
	}
	return VZRequirement{}
}
