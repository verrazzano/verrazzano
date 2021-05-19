// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// isLocalClusterAdminCluster determines if the local cluster is the admin cluster.
func isLocalClusterAdminCluster(c client.Client) bool {
	s := core.Secret{}
	k := client.ObjectKey{Name: "verrazzano-cluster-registration", Namespace: constants.VerrazzanoSystemNamespace}
	err := c.Get(context.TODO(), k, &s)
	if err != nil && errors.IsNotFound(err) {
		return true
	}
	return false
}

// validateTargetClustersExist determines if all of the target clusters of the project have
// corresponding managed cluster resources.
func validateTargetClustersExist(c client.Client, p Placement) error {
	for _, cluster := range p.Clusters {
		key := client.ObjectKey{Name: cluster.Name, Namespace: constants.VerrazzanoMultiClusterNamespace}
		vmc := v1alpha1.VerrazzanoManagedCluster{}
		err := c.Get(context.TODO(), key, &vmc)
		if err != nil {
			return fmt.Errorf("target managed cluster %s is not registered: %v", cluster.Name, err)
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

