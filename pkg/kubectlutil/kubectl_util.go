// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package kubectlutil

import (
	"fmt"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubectl/pkg/util"
)

// SetLastAppliedConfigurationAnnotation applies the kubectl.kubernetes.io/last-applied-configuration annotation
// in order to calculate correct 3-way merges between object configuration file/configuration file,
// live object configuration/live configuration and declarative configuration writer/declarative writer
// (e.g. vz cli install or upgrade)
func SetLastAppliedConfigurationAnnotation(obj runtime.Object) error {
	err := util.CreateOrUpdateAnnotation(true, obj, unstructured.UnstructuredJSONScheme)
	if err != nil {
		return fmt.Errorf("error while applying %s annotation on the "+
			"object: %v", v1.LastAppliedConfigAnnotation, err)
	}
	return nil
}
