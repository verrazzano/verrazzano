// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package reconcile

import (
	"context"

	vzappclusters "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	clustersapi "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/pkg/constants"
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
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// vzStateUninstallStart is the state where Verrazzano is starting the uninstall flow
	vzStateUninstallStart uninstallState = "vzStateUninstallStart"

	// vzStateUninstallRancherLocal is the state where the Rancher local cluster is being uninstalled
	vzStateUninstallRancherLocal uninstallState = "vzStateUninstallRancherLocal"

	// vzStateUninstallMC is the state where the multi-cluster resources are being uninstalled
	vzStateUninstallMC uninstallState = "vzStateUninstallMC"

	// vzStateUninstallComponents is the state where the components are being uninstalled
	vzStateUninstallComponents uninstallState = "vzStateUninstallComponents"

	// vzStateUninstallCleanup is the state where the final cleanup is performed for a full uninstall
	vzStateUninstallCleanup uninstallState = "vzStateUninstallCleanup"

	// vzStateUninstallDone is the state when uninstall is done
	vzStateUninstallDone uninstallState = "vzStateUninstallDone"

	// vzStateUninstallEnd is the terminal state
	vzStateUninstallEnd uninstallState = "vzStateUninstallEnd"
)

type cmCleanupFuncType func(log vzlog.VerrazzanoLogger, cli client.Client, namespace string) error

var cmCleanupFunc cmCleanupFuncType = cmissuer.UninstallCleanup

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

// uninstallState identifies the state of a Verrazzano uninstall operation
type uninstallState string

// UninstallTracker has the Uninstall context for the Verrazzano Uninstall
// This tracker keeps an in-memory Uninstall state for Verrazzano and the components that
// are being Uninstall.
type UninstallTracker struct {
	vzState uninstallState
	gen     int64
	compMap map[string]*componentTrackerContext
}

// UninstallTrackerMap has a map of UninstallTrackers, one entry per Verrazzano CR resource generation
var UninstallTrackerMap = make(map[string]*UninstallTracker)

// reconcileUninstall will Uninstall a Verrazzano installation
func (r *Reconciler) reconcileUninstall(log vzlog.VerrazzanoLogger, cr *installv1alpha1.Verrazzano) (ctrl.Result, error) {
	log.Oncef("Uninstalling Verrazzano %s/%s", cr.Namespace, cr.Name)
	rancherProvisioned, err := rancher.IsClusterProvisionedByRancher()
	if err != nil {
		return ctrl.Result{}, err
	}
	tracker := getUninstallTracker(cr)
	done := false
	for !done {
		switch tracker.vzState {
		case vzStateUninstallStart:
			tracker.vzState = vzStateUninstallRancherLocal

		case vzStateUninstallRancherLocal:
			// Don't remove Rancher local namespace if cluster was provisioned by Rancher (for example RKE2).  Removing
			// will cause cluster corruption.
			if rancherProvisioned {
				tracker.vzState = vzStateUninstallMC
				continue
			}
			// If Rancher is installed, then delete local cluster
			found, comp := registry.FindComponent(rancher.ComponentName)
			if !found {
				tracker.vzState = vzStateUninstallMC
				continue
			}
			spiCtx, err := spi.NewContext(log, r.Client, cr, nil, r.DryRun)
			if err != nil {
				return newRequeueWithDelay(), err
			}
			compContext := spiCtx.Init(rancher.ComponentName).Operation(vzconst.UninstallOperation)
			installed, err := comp.IsInstalled(compContext)
			if err != nil {
				return newRequeueWithDelay(), err
			}
			if !installed {
				tracker.vzState = vzStateUninstallMC
				continue
			}
			rancher.DeleteLocalCluster(log, r.Client)
			tracker.vzState = vzStateUninstallMC

		case vzStateUninstallMC:
			spiCtx, err := spi.NewContext(log, r.Client, cr, nil, r.DryRun)
			if err != nil {
				return ctrl.Result{}, err
			}
			if err := r.deleteMCResources(spiCtx); err != nil {
				return ctrl.Result{}, err
			}
			tracker.vzState = vzStateUninstallComponents

		case vzStateUninstallComponents:
			log.Once("Uninstalling all Verrazzano components")
			res, err := r.uninstallComponents(log, cr, tracker)
			if err != nil || res.Requeue {
				return res, err
			}
			tracker.vzState = vzStateUninstallCleanup

		case vzStateUninstallCleanup:
			log.Once("Performing system-wide post-uninstall cleanup")
			spiCtx, err := spi.NewContext(log, r.Client, cr, nil, r.DryRun)
			if err != nil {
				return ctrl.Result{}, err
			}
			result, err := r.uninstallCleanup(spiCtx, rancherProvisioned)
			if err != nil || !result.IsZero() {
				return result, err
			}
			tracker.vzState = vzStateUninstallDone
		case vzStateUninstallDone:
			log.Once("Successfully uninstalled all Verrazzano components")
			tracker.vzState = vzStateUninstallEnd

		case vzStateUninstallEnd:
			done = true
		}
	}
	// Uninstall done, no need to requeue
	return ctrl.Result{}, nil
}

// getUninstallTracker gets the Uninstall tracker for Verrazzano
func getUninstallTracker(cr *installv1alpha1.Verrazzano) *UninstallTracker {
	key := getTrackerKey(cr)
	vuc, ok := UninstallTrackerMap[key]
	// If the entry is missing or the generation is different create a new entry
	if !ok || vuc.gen != cr.Generation {
		vuc = &UninstallTracker{
			vzState: vzStateUninstallStart,
			gen:     cr.Generation,
			compMap: make(map[string]*componentTrackerContext),
		}
		UninstallTrackerMap[key] = vuc
	}
	return vuc
}

// DeleteUninstallTracker deletes the Uninstall tracker for the Verrazzano resource
// This needs to be called when uninstall is completely done
func DeleteUninstallTracker(cr *installv1alpha1.Verrazzano) {
	key := getTrackerKey(cr)
	_, ok := UninstallTrackerMap[key]
	if ok {
		delete(UninstallTrackerMap, key)
	}
}

// Delete multicluster related resources
func (r *Reconciler) deleteMCResources(ctx spi.ComponentContext) error {
	// Check if this is not managed cluster
	managed, err := r.isManagedCluster(ctx.Log())
	if err != nil {
		return err
	}

	projects := vzappclusters.VerrazzanoProjectList{}
	if err := r.List(context.TODO(), &projects, &client.ListOptions{Namespace: vzconst.VerrazzanoMultiClusterNamespace}); err != nil && !meta.IsNoMatchError(err) {
		return ctx.Log().ErrorfNewErr("Failed listing MC projects: %v", err)
	}
	// Delete MC rolebindings for each project
	for _, p := range projects.Items {
		if err := r.deleteManagedClusterRoleBindings(p, ctx.Log()); err != nil {
			return err
		}
	}

	ctx.Log().Oncef("Deleting all VMC resources")
	vmcList := clustersapi.VerrazzanoManagedClusterList{}
	if err := r.List(context.TODO(), &vmcList, &client.ListOptions{}); err != nil && !meta.IsNoMatchError(err) {
		return ctx.Log().ErrorfNewErr("Failed listing VMCs: %v", err)
	}

	for i, vmc := range vmcList.Items {
		// Delete the VMC ServiceAccount (since managed cluster role bindings associated to it should now be deleted)
		vmcSA := corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{Namespace: vmc.Namespace, Name: vmc.Spec.ServiceAccount},
		}
		if err := r.Delete(context.TODO(), &vmcSA); err != nil {
			return ctx.Log().ErrorfNewErr("Failed to delete VMC service account %s/%s, %v", vmc.Namespace, vmc.Spec.ServiceAccount, err)
		}
		if err := r.Delete(context.TODO(), &vmcList.Items[i]); err != nil {
			return ctx.Log().ErrorfNewErr("Failed to delete VMC %s/%s, %v", vmc.Namespace, vmc.Name, err)
		}
	}

	// Delete VMC namespace only if there are no projects
	if len(projects.Items) == 0 {
		ctx.Log().Oncef("Deleting %s namespace", vzconst.VerrazzanoMultiClusterNamespace)
		if err := r.deleteNamespace(context.TODO(), ctx.Log(), vzconst.VerrazzanoMultiClusterNamespace); err != nil {
			return err
		}
	}

	// Delete secrets on managed cluster.  Don't delete MC agent secret until the end since it tells us this is MC install
	if managed {
		if err := r.deleteSecret(ctx.Log(), vzconst.VerrazzanoSystemNamespace, vzconst.MCRegistrationSecret); err != nil {
			return err
		}
		if err := r.deleteSecret(ctx.Log(), vzconst.VerrazzanoSystemNamespace, mcElasticSearchScrt); err != nil {
			return err
		}
		if err := r.deleteSecret(ctx.Log(), vzconst.VerrazzanoSystemNamespace, vzconst.MCAgentSecret); err != nil {
			return err
		}
	}

	return nil
}

// deleteManagedClusterRoleBindings deletes the managed cluster rolebindings from each namespace
// governed by the given project
func (r *Reconciler) deleteManagedClusterRoleBindings(project vzappclusters.VerrazzanoProject, _ vzlog.VerrazzanoLogger) error {
	for _, projectNSTemplate := range project.Spec.Template.Namespaces {
		rbList := rbacv1.RoleBindingList{}
		if err := r.List(context.TODO(), &rbList, &client.ListOptions{Namespace: projectNSTemplate.Metadata.Name}); err != nil {
			return err
		}
		for i, rb := range rbList.Items {
			if rb.RoleRef.Name == "verrazzano-managed-cluster" {
				if err := r.Delete(context.TODO(), &rbList.Items[i]); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// isManagedCluster returns true if this is a managed cluster
func (r *Reconciler) isManagedCluster(log vzlog.VerrazzanoLogger) (bool, error) {
	var secret corev1.Secret
	secretNsn := types.NamespacedName{
		Namespace: vzconst.VerrazzanoSystemNamespace,
		Name:      vzconst.MCAgentSecret,
	}

	// Get the MC agent secret and return if not found
	if err := r.Get(context.TODO(), secretNsn, &secret); err != nil {
		if errors.IsNotFound(err) {
			log.Once("Determined that this is not a managed cluster")
			return false, nil
		}
		return false, log.ErrorfNewErr("Failed to fetch the multicluster secret %s/%s, %v", vzconst.VerrazzanoSystemNamespace, vzconst.MCAgentSecret, err)
	}
	return true, nil
}

// uninstallCleanup Perform the final cleanup of shared resources, etc not tracked by individual component uninstalls
func (r *Reconciler) uninstallCleanup(ctx spi.ComponentContext, rancherProvisioned bool) (ctrl.Result, error) {
	if err := r.deleteIstioCARootCert(ctx); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.nodeExporterCleanup(ctx.Log()); err != nil {
		return ctrl.Result{}, err
	}

	// Run Rancher Post Uninstall explicitly to delete any remaining Rancher resources; this may be needed in case
	// the uninstall was interrupted during uninstall, or if the cluster is a managed cluster where Rancher is not
	// installed explicitly.
	if !rancherProvisioned {
		if err := r.runRancherPostUninstall(ctx); err != nil {
			return ctrl.Result{}, err
		}
	}
	return r.deleteNamespaces(ctx, rancherProvisioned)
}

func (r *Reconciler) runRancherPostUninstall(ctx spi.ComponentContext) error {
	// Look up the Rancher component and call PostUninstall explicitly, without checking if it's installed;
	// this is to catch any lingering managed cluster resources
	if found, comp := registry.FindComponent(rancher.ComponentName); found {
		err := comp.PostUninstall(ctx.Init(rancher.ComponentName).Operation(vzconst.UninstallOperation))
		if err != nil {
			ctx.Log().Progress("Waiting for Rancher post-uninstall cleanup to be done")
			return err
		}
	}
	return nil
}

// nodeExporterCleanup cleans up any resources from the old node-exporter that was
// replaced with the node-exporter from the prometheus-operator
func (r *Reconciler) nodeExporterCleanup(log vzlog.VerrazzanoLogger) error {
	err := resource.Resource{
		Name:   nodeExporterName,
		Client: r.Client,
		Object: &rbacv1.ClusterRoleBinding{},
		Log:    log,
	}.Delete()
	if err != nil {
		return err
	}
	err = resource.Resource{
		Name:   nodeExporterName,
		Client: r.Client,
		Object: &rbacv1.ClusterRole{},
		Log:    log,
	}.Delete()
	if err != nil {
		return err
	}

	return nil
}

// deleteSecret deletes a Kubernetes secret
func (r *Reconciler) deleteSecret(log vzlog.VerrazzanoLogger, namespace string, name string) error {
	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name},
	}
	log.Oncef("Deleting multicluster secret %s:%s", namespace, name)
	if err := r.Delete(context.TODO(), &secret); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return log.ErrorfNewErr("Failed to delete secret %s/%s, %v", namespace, name, err)
	}
	return nil
}

// deleteNamespaces deletes up all component namespaces plus any namespaces shared by multiple components
// - returns an error or a requeue with delay result
func (r *Reconciler) deleteNamespaces(ctx spi.ComponentContext, rancherProvisioned bool) (ctrl.Result, error) {
	log := ctx.Log()
	// Load a set of all component namespaces plus shared namespaces
	nsSet := make(map[string]bool)
	for _, comp := range registry.GetComponents() {
		// Don't delete the rancher component namespace if cluster was provisioned by Rancher.
		if rancherProvisioned && comp.Namespace() == rancher.ComponentNamespace {
			continue
		}
		if comp.Namespace() == cmcontroller.ComponentNamespace && !vzcr.IsCertManagerEnabled(ctx.EffectiveCR()) {
			log.Progressf("Cert-Manager not enabled, skip namespace cleanup")
			continue
		}
		nsSet[comp.Namespace()] = true
	}
	for i, ns := range sharedNamespaces {
		if ns == cmcontroller.ComponentNamespace && !vzcr.IsCertManagerEnabled(ctx.EffectiveCR()) {
			log.Progressf("Cert-Manager not enabled, skip namespace cleanup")
			continue
		}
		nsSet[sharedNamespaces[i]] = true
	}

	// Delete all the namespaces
	for ns := range nsSet {
		// Clean up any remaining CM resources in Verrazzano-managed namespaces
		if err := cmCleanupFunc(ctx.Log(), ctx.Client(), ns); err != nil {
			return newRequeueWithDelay(), err
		}
		log.Progressf("Deleting namespace %s", ns)
		err := resource.Resource{
			Name:   ns,
			Client: r.Client,
			Object: &corev1.Namespace{},
			Log:    log,
		}.RemoveFinalizersAndDelete()
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	// Wait for all the namespaces to be deleted
	waiting := false
	for ns := range nsSet {
		err := r.Get(context.TODO(), types.NamespacedName{Name: ns}, &corev1.Namespace{})
		if err != nil {
			if errors.IsNotFound(err) {
				continue
			}
			return ctrl.Result{}, err
		}
		waiting = true
		log.Progressf("Waiting for namespace %s to terminate", ns)
	}
	if waiting {
		log.Progressf("Namespace terminations still in progress")
		return newRequeueWithDelay(), nil
	}
	log.Once("Namespaces terminated successfully")
	return ctrl.Result{}, nil
}

// deleteIstioCARootCert deletes the Istio root cert ConfigMap that gets distributed across the cluster
func (r *Reconciler) deleteIstioCARootCert(ctx spi.ComponentContext) error {
	namespaces := corev1.NamespaceList{}
	err := ctx.Client().List(context.TODO(), &namespaces)
	if err != nil {
		return ctx.Log().ErrorfNewErr("Failed to list the cluster namespaces: %v", err)
	}

	for _, ns := range namespaces.Items {
		ctx.Log().Progressf("Deleting Istio root cert in namespace %s", ns.GetName())
		err := resource.Resource{
			Name:      istioRootCertName,
			Namespace: ns.GetName(),
			Client:    r.Client,
			Object:    &corev1.ConfigMap{},
			Log:       ctx.Log(),
		}.Delete()
		if err != nil {
			return err
		}
	}
	return nil
}
