// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/result"
	vzappclusters "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	clustersapi "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/pkg/k8s/resource"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/rancher"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// preUninstall does all the global preUninstall
func (r Reconciler) preUninstall(log vzlog.VerrazzanoLogger, actualCR *vzv1alpha1.Verrazzano, effectiveCR *vzv1alpha1.Verrazzano) result.Result {
	if res := r.preUninstallRancher(log, actualCR, effectiveCR); res.ShouldRequeue() {
		return res
	}

	if res := r.preUninstallMC(log, actualCR, effectiveCR); res.ShouldRequeue() {
		return res
	}

	return result.NewResult()
}

// doUninstall performs the verrazzano uninstall by deleting modules
// Return a requeue true until all modules are gone
func (r Reconciler) doUninstall(log vzlog.VerrazzanoLogger, actualCR *vzv1alpha1.Verrazzano, effectiveCR *vzv1alpha1.Verrazzano) result.Result {
	// Delete modules that are enabled and update status
	// Don't block status update of component if delete failed
	res := r.deleteModules(log, effectiveCR)
	if res.ShouldRequeue() {
		return result.NewResultShortRequeueDelay()
	}

	return result.NewResult()
}

// postUninstall does all the global postUninstall
func (r Reconciler) postUninstall(log vzlog.VerrazzanoLogger, actualCR *vzv1alpha1.Verrazzano, effectiveCR *vzv1alpha1.Verrazzano) result.Result {

	return result.NewResult()
}

// preUninstallRancherLocal does Rancher pre-uninstall
func (r Reconciler) preUninstallRancher(log vzlog.VerrazzanoLogger, actualCR *vzv1alpha1.Verrazzano, effectiveCR *vzv1alpha1.Verrazzano) result.Result {
	rancherProvisioned, err := rancher.IsClusterProvisionedByRancher()
	if err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}

	// Don't remove Rancher local namespace if cluster was provisioned by Rancher (for example RKE2).  Removing
	// will cause cluster corruption.
	if rancherProvisioned {
		return result.NewResult()
	}
	// If Rancher is installed, then delete local cluster
	found, comp := registry.FindComponent(rancher.ComponentName)
	if !found {
		return result.NewResult()
	}

	spiCtx, err := spi.NewContext(log, r.Client, actualCR, nil, r.DryRun)
	if err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}
	compContext := spiCtx.Init(rancher.ComponentName).Operation(vzconst.UninstallOperation)
	installed, err := comp.IsInstalled(compContext)
	if err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}
	if !installed {
		return result.NewResult()
	}
	rancher.DeleteLocalCluster(log, r.Client)

	if err := r.deleteMCResources(spiCtx); err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}
	return result.NewResult()
}

// preUninstallMC does MC pre-uninstall
func (r Reconciler) preUninstallMC(log vzlog.VerrazzanoLogger, actualCR *vzv1alpha1.Verrazzano, effectiveCR *vzv1alpha1.Verrazzano) result.Result {
	spiCtx, err := spi.NewContext(log, r.Client, actualCR, nil, r.DryRun)
	if err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}
	if err := r.deleteMCResources(spiCtx); err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}

	return result.NewResult()
}

// Delete multicluster related resources
func (r *Reconciler) deleteMCResources(ctx spi.ComponentContext) error {
	// Check if this is not managed cluster
	managed, err := r.isManagedCluster(ctx.Log())
	if err != nil {
		return err
	}

	projects := vzappclusters.VerrazzanoProjectList{}
	if err := r.Client.List(context.TODO(), &projects, &client.ListOptions{Namespace: vzconst.VerrazzanoMultiClusterNamespace}); err != nil && !meta.IsNoMatchError(err) {
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
	if err := r.Client.List(context.TODO(), &vmcList, &client.ListOptions{}); err != nil && !meta.IsNoMatchError(err) {
		return ctx.Log().ErrorfNewErr("Failed listing VMCs: %v", err)
	}

	for i, vmc := range vmcList.Items {
		// Delete the VMC ServiceAccount (since managed cluster role bindings associated to it should now be deleted)
		vmcSA := corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{Namespace: vmc.Namespace, Name: vmc.Spec.ServiceAccount},
		}
		if err := r.Client.Delete(context.TODO(), &vmcSA); err != nil {
			return ctx.Log().ErrorfNewErr("Failed to delete VMC service account %s/%s, %v", vmc.Namespace, vmc.Spec.ServiceAccount, err)
		}
		if err := r.Client.Delete(context.TODO(), &vmcList.Items[i]); err != nil {
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
		if err := r.deleteSecret(ctx.Log(), vzconst.VerrazzanoSystemNamespace, vzconst.MCRegistrationSecret); err != nil {
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
		if err := r.Client.List(context.TODO(), &rbList, &client.ListOptions{Namespace: projectNSTemplate.Metadata.Name}); err != nil {
			return err
		}
		for i, rb := range rbList.Items {
			if rb.RoleRef.Name == "verrazzano-managed-cluster" {
				if err := r.Client.Delete(context.TODO(), &rbList.Items[i]); err != nil {
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
	if err := r.Client.Get(context.TODO(), secretNsn, &secret); err != nil {
		if errors.IsNotFound(err) {
			log.Once("Determined that this is not a managed cluster")
			return false, nil
		}
		return false, log.ErrorfNewErr("Failed to fetch the multicluster secret %s/%s, %v", vzconst.VerrazzanoSystemNamespace, vzconst.MCAgentSecret, err)
	}
	return true, nil
}

// uninstallCleanup Perform the final cleanup of shared resources, etc not tracked by individual component uninstalls
func (r *Reconciler) uninstallCleanup(ctx spi.ComponentContext, rancherProvisioned bool) result.Result {
	if err := r.deleteIstioCARootCert(ctx); err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}

	if err := r.nodeExporterCleanup(ctx.Log()); err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}

	// Run Rancher Post Uninstall explicitly to delete any remaining Rancher resources; this may be needed in case
	// the uninstall was interrupted during uninstall, or if the cluster is a managed cluster where Rancher is not
	// installed explicitly.
	if !rancherProvisioned {
		if err := r.runRancherPostUninstall(ctx); err != nil {
			return result.NewResultShortRequeueDelayWithError(err)
		}
	}
	return r.deleteNamespaces(ctx, rancherProvisioned)
}

// deleteIstioCARootCert deletes the Istio root cert ConfigMap that gets distributed across the cluster
func (r *Reconciler) deleteIstioCARootCert(ctx spi.ComponentContext) error {
	namespaces := corev1.NamespaceList{}
	err := ctx.Client().List(context.TODO(), &namespaces)
	if err != nil {
		return ctx.Log().ErrorfNewErr("Failed to list the cluster namespaces: %v", err)
	}

	for _, ns := range namespaces.Items {
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

func (r *Reconciler) runRancherPostUninstall(ctx spi.ComponentContext) error {
	// Look up the Rancher component and call PostUninstall explicitly, without checking if it's installed;
	// this is to catch any lingering managed cluster resources
	if found, comp := registry.FindComponent(rancher.ComponentName); found {
		err := comp.PostUninstall(ctx.Init(rancher.ComponentName).Operation(vzconst.UninstallOperation))
		if err != nil {
			ctx.Log().Once("Waiting for Rancher post-uninstall cleanup to be done")
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
