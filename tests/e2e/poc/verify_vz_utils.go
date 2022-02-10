// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package poc

import (
	"context"
	"fmt"

	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
)

func doesNamespaceExist(client clientset.Interface, name string) (bool, error) {
	namespace, err := client.CoreV1().Namespaces().Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to get namespace %s: %v", name, err))
		return false, err
	}

	return namespace != nil && len(namespace.Name) > 0, nil
}
