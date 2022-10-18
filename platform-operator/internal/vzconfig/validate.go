// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vzconfig

import (
	"fmt"
	"k8s.io/api/rbac/v1"
)

// ValidateRoleBindingSubject - Validates the requested subject content, used to validate the Verrazzano CR security customizations
// - refactored from the install_config code
func ValidateRoleBindingSubject(subject v1.Subject, name string) error {
	if len(subject.Name) < 1 {
		err := fmt.Errorf("no name for %s", name)
		return err
	}
	if subject.Kind != "User" && subject.Kind != "Group" && subject.Kind != "ServiceAccount" {
		err := fmt.Errorf("invalid kind '%s' for %s", subject.Kind, name)
		return err
	}
	if (subject.Kind == "User" || subject.Kind == "Group") && len(subject.APIGroup) > 0 && subject.APIGroup != "rbac.authorization.k8s.io" {
		err := fmt.Errorf("invalid apiGroup '%s' for %s", subject.APIGroup, name)
		return err
	}
	if subject.Kind == "ServiceAccount" && (len(subject.APIGroup) > 0 || subject.APIGroup != "") {
		err := fmt.Errorf("invalid apiGroup '%s' for %s", subject.APIGroup, name)
		return err
	}
	if subject.Kind == "ServiceAccount" && len(subject.Namespace) < 1 {
		err := fmt.Errorf("no namespace for ServiceAccount in %s", name)
		return err
	}
	return nil
}
