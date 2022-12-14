// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	"errors"
	"fmt"
	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
)

// entryTracker is a Set like construct to track if a value was seen already
type entryTracker struct {
	set map[string]bool
}

func newTracker() entryTracker {
	return entryTracker{
		set: map[string]bool{},
	}
}

// add an item to the set. If it's already present, return an error.
func (e entryTracker) add(entry string) error {
	if _, exists := e.set[entry]; exists {
		return fmt.Errorf("%s already exists", entry)
	}
	e.set[entry] = true
	return nil
}

// validateNoDuplicatedConfiguration rejects any updates that contain duplicated argument names:
// Node group names or InstallArg names.
func validateNoDuplicatedConfiguration(vz *v1beta1.Verrazzano) error {
	if vz.Spec.Components.OpenSearch == nil {
		return nil
	}
	opensearch := vz.Spec.Components.OpenSearch

	return validateNoDuplicateNodeGroups(opensearch)

}

// validateNoDuplicateNodeGroups rejects Nodes with duplicated group names
func validateNoDuplicateNodeGroups(opensearch *v1beta1.OpenSearchComponent) error {
	tracker := newTracker()
	for _, group := range opensearch.Nodes {
		if err := tracker.add(group.Name); err != nil {
			return fmt.Errorf("OpenSearch node group name is duplicated or invalid: %v", err)
		}
	}
	numberNodes, totalNode := GetNodeRoleCounts(opensearch)

	if totalNode > int32(1) {
		if numberNodes[vmov1.MasterRole] < 3 {
			return errors.New("Number of master nodes should be at least 3")
		}
		if numberNodes[vmov1.DataRole] < 2 {
			return errors.New("Number of data nodes should be at least 2")
		}
		if numberNodes[vmov1.IngestRole] < 1 {
			return errors.New("Number of ingest nodes should be at least 1")
		}
	}
	return nil
}

func GetNodeRoleCounts(opensearch *v1beta1.OpenSearchComponent) (map[vmov1.NodeRole]int32, int32) {
	numberNodes := make(map[vmov1.NodeRole]int32)
	totalNodes := int32(0)
	for _, group := range opensearch.Nodes {
		for _, role := range group.Roles {
			numberNodes[role] += group.Replicas
		}
		totalNodes += group.Replicas
	}
	return numberNodes, totalNodes
}
