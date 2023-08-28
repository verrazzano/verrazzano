// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ociocne

import (
	vmcv1alpha1 "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	vzstring "github.com/verrazzano/verrazzano/pkg/string"
)

func isMissingFinalizer(q *vmcv1alpha1.OCNEOCIQuickCreate) bool {
	return !vzstring.SliceContainsString(q.GetFinalizers(), finalizerKey)
}

func shouldProvision(q *vmcv1alpha1.OCNEOCIQuickCreate) bool {
	return q.Status.QuickCreateStatus.Phase == ""
}
