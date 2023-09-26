// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package custom

import (
	"context"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/result"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/k8s/resource"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	cmcontroller "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager/certmanager"
	cmissuer "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager/issuer"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/rancher"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	componentspi "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// old node-exporter constants replaced with prometheus-operator node-exporter
const (
	monitoringNamespace = "monitoring"
	nodeExporterName    = "node-exporter"
	mcElasticSearchScrt = "verrazzano-cluster-elasticsearch"
	istioRootCertName   = "istio-ca-root-cert"
)

// sharedNamespaces The set of namespaces shared by multiple components; managed separately apart from individual components
var sharedNamespaces = []string{
	vzconst.VerrazzanoMonitoringNamespace,
	constants.CertManagerNamespace,
	constants.VerrazzanoSystemNamespace,
	vzconst.KeycloakNamespace,
	monitoringNamespace,
}

// DeleteNamespace deletes a namespace
func DeleteNamespace(cli client.Client, log vzlog.VerrazzanoLogger, namespace string) error {
	err := resource.Resource{
		Name:   namespace,
		Client: cli,
		Object: &corev1.Namespace{},
		Log:    log,
	}.RemoveFinalizersAndDelete()
	if err != nil {
		log.ErrorfThrottled("Error during namespace deletion: %v", err)
		return err
	}
	return nil
}

// DeleteNamespaces deletes up all component namespaces plus any namespaces shared by multiple components
// - returns an error or a requeue with delay result
func DeleteNamespaces(componentCtx componentspi.ComponentContext, rancherProvisioned bool) result.Result {
	log := componentCtx.Log()
	// check on whether cluster is OCNE container driver provisioned
	ocneContainerDriverProvisioned, err := rancher.IsClusterProvisionedByOCNEContainerDriver()
	if err != nil {
		return result.NewResult()
	}
	// Load a set of all component namespaces plus shared namespaces
	nsSet := make(map[string]bool)
	for _, comp := range registry.GetComponents() {
		// Don't delete the rancher component namespace if cluster was provisioned by Rancher.
		if (rancherProvisioned || ocneContainerDriverProvisioned) && comp.Namespace() == rancher.ComponentNamespace {
			continue
		}
		if comp.Namespace() == cmcontroller.ComponentNamespace && !vzcr.IsCertManagerEnabled(componentCtx.EffectiveCR()) {
			log.Oncef("Cert-Manager not enabled, skip namespace cleanup")
			continue
		}
		nsSet[comp.Namespace()] = true
	}
	for i, ns := range sharedNamespaces {
		if ns == cmcontroller.ComponentNamespace && !vzcr.IsCertManagerEnabled(componentCtx.EffectiveCR()) {
			log.Oncef("Cert-Manager not enabled, skip namespace cleanup")
			continue
		}
		nsSet[sharedNamespaces[i]] = true
	}

	// Delete all the namespaces
	for ns := range nsSet {
		// Clean up any remaining CM resources in Verrazzano-managed namespaces
		if err := cmissuer.UninstallCleanup(componentCtx.Log(), componentCtx.Client(), ns); err != nil {
			return result.NewResultShortRequeueDelayWithError(err)
		}
		err := resource.Resource{
			Name:   ns,
			Client: componentCtx.Client(),
			Object: &corev1.Namespace{},
			Log:    log,
		}.RemoveFinalizersAndDelete()
		if err != nil {
			componentCtx.Log().Errorf("Error during namespace deletion: %v", err)
			return result.NewResultShortRequeueDelayWithError(err)
		}
	}

	// Wait for all the namespaces to be deleted
	waiting := false
	for ns := range nsSet {
		err := componentCtx.Client().Get(context.TODO(), types.NamespacedName{Name: ns}, &corev1.Namespace{})
		if err != nil {
			if errors.IsNotFound(err) {
				continue
			}
			return result.NewResultShortRequeueDelayWithError(err)
		}
		waiting = true
		log.Oncef("Waiting for namespace %s to terminate", ns)
	}
	if waiting {
		log.Oncef("Namespace terminations still in progress")
		return result.NewResultShortRequeueDelay()
	}
	log.Once("Namespaces terminated successfully")
	return result.NewResult()
}

// MergeMaps Merge one map into another, creating new one if necessary; returns the updated map and true if it was modified
func MergeMaps(to map[string]string, from map[string]string) (map[string]string, bool) {
	mergedMap := to
	if mergedMap == nil {
		mergedMap = make(map[string]string)
	}
	var updated bool
	for k, v := range from {
		if existingVal, ok := mergedMap[k]; !ok {
			mergedMap[k] = v
			updated = true
		} else {
			// check to see if the value changed and, if it has, treat as an update
			if v != existingVal {
				mergedMap[k] = v
				updated = true
			}
		}
	}
	return mergedMap, updated
}
