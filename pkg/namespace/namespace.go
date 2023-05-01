// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package namespace

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CheckIfVerrazzanoManagedNamespaceExists returns true if the namespace exists and has the verrazzano.io/namespace label
func CheckIfVerrazzanoManagedNamespaceExists(nsName string) (bool, error) {
	client, err := k8sutil.GetCoreV1Client()
	if err != nil {
		return false, fmt.Errorf("Failure creating corev1 client: %v", err)
	}

	namespace, err := client.Namespaces().Get(context.TODO(), nsName, metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return false, fmt.Errorf("Failure checking for namespace %s: %v", nsName, err)
	}
	if namespace == nil || namespace.Labels[constants.VerrazzanoManagedKey] == "" {
		return false, nil
	}
	return true, nil
}
