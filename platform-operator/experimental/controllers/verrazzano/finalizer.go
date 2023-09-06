// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/result"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/spi/controllerspi"
	vzappclusters "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	clustersapi "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzstring "github.com/verrazzano/verrazzano/pkg/string"
	vzv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/rancher"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/transform"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const finalizerName = "install.verrazzano.io"

// GetName returns the name of the finalizer
func (r Reconciler) GetName() string {
	return finalizerName
}

// PreRemoveFinalizer is called when the resource is being deleted, before the finalizer
// is removed.  Use this method to delete Kubernetes resources, etc.
func (r Reconciler) PreRemoveFinalizer(spictx controllerspi.ReconcileContext, u *unstructured.Unstructured) result.Result {
	// Convert the unstructured to a Verrazzano CR
	actualCR := &vzv1alpha1.Verrazzano{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, actualCR); err != nil {
		spictx.Log.ErrorfThrottled(err.Error())
		// This is a fatal error which should never happen, don't requeue
		return result.NewResult()
	}

	// Get the resource logger needed to log message using 'progress' and 'once' methods
	log, err := vzlog.EnsureResourceLogger(&vzlog.ResourceConfig{
		Name:           actualCR.Name,
		Namespace:      actualCR.Namespace,
		ID:             string(actualCR.UID),
		Generation:     actualCR.Generation,
		ControllerName: "verrazzano",
	})
	if err != nil {
		zap.S().Errorf("Failed to create controller logger for Verrazzano controller: %v", err)
	}

	r.updateStatusUninstalling(log, actualCR)

	// Get effective CR and set the effective CR status with the actual status
	// Note that the reconciler code only udpdate the status, which is why the effective
	// CR is passed.  If was ever to update the spec, then the actual CR would need to be used.
	effectiveCR, err := transform.GetEffectiveCR(actualCR)
	if err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}
	effectiveCR.Status = actualCR.Status

	// Do global pre-work
	if res := r.preWork(log, actualCR, effectiveCR); res.ShouldRequeue() {
		return res
	}

	// Do the actual install, update, and or upgrade.
	if res := r.doWork(log, actualCR, effectiveCR); res.ShouldRequeue() {
		return res
	}

	// Do global post-work
	if res := r.postWork(log, actualCR, effectiveCR); res.ShouldRequeue() {
		return res
	}

	// All done reconciling.  Add the completed condition to the status and set the state back to Ready.
	r.updateStatusUninstallComplete(actualCR)
	return result.NewResult()

	// All install related resources have been deleted, delete the finalizer so that the Verrazzano
	// resource can get removed from etcd.
	log.Oncef("Removing finalizer %s", finalizerName)
	actualCR.ObjectMeta.Finalizers = vzstring.RemoveStringFromSlice(actualCR.ObjectMeta.Finalizers, finalizerName)
	if err := r.Client.Update(context.TODO(), actualCR); err != nil {
		r.updateStatusUninstallComplete(actualCR)
		return result.NewResultShortRequeueDelayIfError(err)
	}

	// Always requeue, the legacy verrazzano controller will delete the finalizer and the VZ CR will go away.
	return result.NewResult()
}

// PostRemoveFinalizer is called after the finalizer is successfully removed.
// This method does garbage collection and other tasks that can never return an error
func (r Reconciler) PostRemoveFinalizer(spictx controllerspi.ReconcileContext, u *unstructured.Unstructured) {
	// Delete the tracker used for this CR
	//statemachine.DeleteTracker(u)
}

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
