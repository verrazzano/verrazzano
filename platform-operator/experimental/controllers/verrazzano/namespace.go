// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/result"
	"github.com/verrazzano/verrazzano/pkg/k8s/resource"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	cmcontroller "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager/certmanager"
	cmissuer "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager/issuer"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/rancher"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/experimental/controllers/verrazzano/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// systemNamespaceLabels the verrazzano-system namespace labels required
var systemNamespaceLabels = map[string]string{
	"istio-injection":         "enabled",
	"verrazzano.io/namespace": vzconst.VerrazzanoSystemNamespace,
}

// createVerrazzanoSystemNamespace creates the Verrazzano system namespace if it does not already exist
func (r Reconciler) createVerrazzanoSystemNamespace(ctx context.Context, cr *installv1alpha1.Verrazzano, log vzlog.VerrazzanoLogger) error {
	// remove injection label if disabled
	istio := cr.Spec.Components.Istio
	if istio != nil && !istio.IsInjectionEnabled() {
		log.Infof("Disabling istio sidecar injection for Verrazzano system components")
		systemNamespaceLabels["istio-injection"] = "disabled"
	}
	log.Debugf("Verrazzano system namespace labels: %v", systemNamespaceLabels)

	// First check if VZ system namespace exists. If not, create it.
	var vzSystemNS corev1.Namespace
	err := r.Client.Get(ctx, types.NamespacedName{Name: vzconst.VerrazzanoSystemNamespace}, &vzSystemNS)
	if err != nil {
		log.Debugf("Creating Verrazzano system namespace")
		if !errors.IsNotFound(err) {
			log.Errorf("Failed to get namespace %s: %v", vzconst.VerrazzanoSystemNamespace, err)
			return err
		}
		vzSystemNS.Name = vzconst.VerrazzanoSystemNamespace
		vzSystemNS.Labels, _ = util.MergeMaps(nil, systemNamespaceLabels)
		log.Oncef("Creating Verrazzano system namespace. Labels: %v", vzSystemNS.Labels)
		if err := r.Client.Create(ctx, &vzSystemNS); err != nil {
			log.Errorf("Failed to create namespace %s: %v", vzconst.VerrazzanoSystemNamespace, err)
			return err
		}
		return nil
	}

	// Namespace exists, see if we need to add the label
	log.Oncef("Updating Verrazzano system namespace")
	var updated bool
	vzSystemNS.Labels, updated = util.MergeMaps(vzSystemNS.Labels, systemNamespaceLabels)
	if !updated {
		return nil
	}
	if err := r.Client.Update(ctx, &vzSystemNS); err != nil {
		log.Errorf("Failed to update namespace %s: %v", vzconst.VerrazzanoSystemNamespace, err)
		return err
	}
	return nil
}

// deleteNamespace deletes a namespace
func (r *Reconciler) deleteNamespace(ctx context.Context, log vzlog.VerrazzanoLogger, namespace string) error {
	ns := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace, // required by the controller Delete call
		},
	}
	err := r.Client.Delete(ctx, &ns, &client.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		log.Errorf("Failed deleting namespace %s: %v", ns.Name, err)
		return err
	}
	return nil
}

// deleteNamespaces deletes up all component namespaces plus any namespaces shared by multiple components
// - returns an error or a requeue with delay result
func (r *Reconciler) deleteNamespaces(ctx spi.ComponentContext, rancherProvisioned bool) result.Result {
	log := ctx.Log()
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
		if comp.Namespace() == cmcontroller.ComponentNamespace && !vzcr.IsCertManagerEnabled(ctx.EffectiveCR()) {
			log.Oncef("Cert-Manager not enabled, skip namespace cleanup")
			continue
		}
		nsSet[comp.Namespace()] = true
	}
	for i, ns := range sharedNamespaces {
		if ns == cmcontroller.ComponentNamespace && !vzcr.IsCertManagerEnabled(ctx.EffectiveCR()) {
			log.Oncef("Cert-Manager not enabled, skip namespace cleanup")
			continue
		}
		nsSet[sharedNamespaces[i]] = true
	}

	// Delete all the namespaces
	for ns := range nsSet {
		// Clean up any remaining CM resources in Verrazzano-managed namespaces
		if err := cmissuer.UninstallCleanup(ctx.Log(), ctx.Client(), ns); err != nil {
			return result.NewResultShortRequeueDelayWithError(err)
		}
		err := resource.Resource{
			Name:   ns,
			Client: r.Client,
			Object: &corev1.Namespace{},
			Log:    log,
		}.RemoveFinalizersAndDelete()
		if err != nil {
			ctx.Log().Errorf("Error during namespace deletion: %v", err)
			return result.NewResultShortRequeueDelayWithError(err)
		}
	}

	// Wait for all the namespaces to be deleted
	waiting := false
	for ns := range nsSet {
		err := r.Client.Get(context.TODO(), types.NamespacedName{Name: ns}, &corev1.Namespace{})
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
