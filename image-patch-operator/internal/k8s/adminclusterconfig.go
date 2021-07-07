// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package k8s

import (
	"context"
	"fmt"

	"github.com/verrazzano/verrazzano/platform-operator/constants"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

// GetAPIServerURL gets the external URL of the API server
func GetAPIServerURL(client clipkg.Client) (string, error) {
	// Get the configmap which has the info needed to build the URL
	var cm corev1.ConfigMap
	nsn := types.NamespacedName{
		Namespace: constants.VerrazzanoMultiClusterNamespace,
		Name:      constants.AdminClusterConfigMapName,
	}
	if err := client.Get(context.TODO(), nsn, &cm); err != nil {
		return "", fmt.Errorf("Failed to fetch configmap %s/%s, %v", nsn.Namespace, nsn.Name, err)
	}
	return cm.Data[constants.ServerDataKey], nil
}
