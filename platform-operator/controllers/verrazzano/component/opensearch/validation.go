// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	"fmt"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
)

//entryTracker is a Set like construct to track if a value was seen already
type entryTracker struct {
	set map[string]bool
}

func newTracker() entryTracker {
	return entryTracker{
		set: map[string]bool{},
	}
}

//add an item to the set. If it's already present, return an error.
func (e entryTracker) add(entry string) error {
	if _, exists := e.set[entry]; exists {
		return fmt.Errorf("%s already exists", entry)
	}
	e.set[entry] = true
	return nil
}

//validateNoDuplicatedConfiguration rejects any updates that contain duplicated argument names:
// Node group names or InstallArg names.
func validateNoDuplicatedConfiguration(vz *vzapi.Verrazzano) error {
	if vz.Spec.Components.Elasticsearch == nil {
		return nil
	}
	opensearch := vz.Spec.Components.Elasticsearch
	if err := validateNoDuplicateArgs(opensearch); err != nil {
		return err
	}
	return validateNoDuplicateNodeGroups(opensearch)

}

//validateNoDuplicateArgs rejects InstallArgs with duplicated names
func validateNoDuplicateArgs(opensearch *vzapi.ElasticsearchComponent) error {
	tracker := newTracker()
	for _, arg := range opensearch.ESInstallArgs {
		if err := tracker.add(arg.Name); err != nil {
			return fmt.Errorf("duplicate OpenSearch install argument: %v", err)
		}
	}
	return nil
}

//validateNoDuplicateNodeGroups rejects Nodes with duplicated group names
func validateNoDuplicateNodeGroups(opensearch *vzapi.ElasticsearchComponent) error {
	tracker := newTracker()
	for _, group := range opensearch.Nodes {
		if err := tracker.add(group.Name); err != nil {
			return fmt.Errorf("OpenSearch node group name is duplicated or invalid: %v", err)
		}
	}
	return nil
}
