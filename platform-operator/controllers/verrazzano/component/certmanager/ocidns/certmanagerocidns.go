// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ocidns

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"k8s.io/apimachinery/pkg/types"
)

const (
	ocidnsDeploymentName = "cert-manager-ocidns-provider"
)

// isCertManagerReady checks the state of the expected cert-manager deployments and returns true if they are in a ready state
func isCertManagerOciDNSReady(context spi.ComponentContext) bool {
	deployments := []types.NamespacedName{}
	if !vzcr.IsOCIDNSEnabled(context.EffectiveCR()) {
		context.Log().Oncef("OCI DNS is not enabled, skipping ready check")
		return true
	}
	deployments = append(deployments, types.NamespacedName{Name: ocidnsDeploymentName, Namespace: ComponentNamespace})
	prefix := fmt.Sprintf("Component %s", context.GetComponent())
	return ready.DeploymentsAreReady(context.Log(), context.Client(), deployments, 1, prefix)
}
