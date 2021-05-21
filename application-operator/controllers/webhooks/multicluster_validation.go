// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	"fmt"
	clusters "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	core "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// isLocalClusterManagedCluster determines if the local cluster is a registered managed cluster.
func isLocalClusterManagedCluster(c client.Client) bool {
	s := core.Secret{}
	k := client.ObjectKey{Name: constants.MCRegistrationSecret, Namespace: constants.VerrazzanoSystemNamespace}
	err := c.Get(context.TODO(), k, &s)
	if err == nil {
		return true
	}
	return false
}

// validateTargetClustersExist determines if all of the target clusters of the project have
// corresponding managed cluster resources.  The results are only valid when this
// is executed against an admin cluster.
func validateTargetClustersExist(c client.Client, p clusters.Placement) error {
	for _, cluster := range p.Clusters {
		targetClusterName := cluster.Name
		// If the target cluster name is local then assume it is valid.
		if targetClusterName != constants.DefaultClusterName {
			key := client.ObjectKey{Name: targetClusterName, Namespace: constants.VerrazzanoMultiClusterNamespace}
			// Need to use unstructured here to avoid a dependency on the platform operator
			vmc := unstructured.Unstructured{}
			vmc.SetGroupVersionKind(schema.GroupVersionKind{
				Group:   "clusters.verrazzano.io",
				Version: "v1alpha1",
				Kind:    "VerrazzanoManagedCluster",
			})
			vmc.SetNamespace(constants.VerrazzanoMultiClusterNamespace)
			vmc.SetName(targetClusterName)
			err := c.Get(context.TODO(), key, &vmc)
			if err != nil {
				return fmt.Errorf("target managed cluster %s is not registered: %v", cluster.Name, err)
			}
		}
	}
	return nil
}

// translateErrorToResponse translates an error to an admission.Response
func translateErrorToResponse(err error) admission.Response {
	if err == nil {
		return admission.Allowed("")
	}
	return admission.Denied(err.Error())
}
