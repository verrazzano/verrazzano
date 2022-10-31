// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	"github.com/verrazzano/verrazzano/application-operator/constants"
	platformopclusters "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (s *Syncer) syncMCAgentDelete(namespace string) error {
	vmcName := client.ObjectKey{Name: s.ManagedClusterName, Namespace: constants.VerrazzanoMultiClusterNamespace}
	vmc := platformopclusters.VerrazzanoManagedCluster{}
	err := s.AdminClient.Get(s.Context, vmcName, &vmc)
	if !apierrors.IsNotFound(err) {
		s.Log.Debugf("VMC still found on the managed cluster, skipping the deletion process")
		return nil
	}

	// TODO: delete all resources on the cluster
	// Delete the manifest secrets (agent, registration)
	// Delete the Rancher

}
