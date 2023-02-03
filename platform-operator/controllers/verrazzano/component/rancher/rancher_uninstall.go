// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/verrazzano/verrazzano/pkg/k8sutil"

	"github.com/verrazzano/verrazzano/pkg/constants"
	ctrlerrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
	"github.com/verrazzano/verrazzano/pkg/k8s/resource"
	vzstring "github.com/verrazzano/verrazzano/pkg/string"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/monitor"
	admv1 "k8s.io/api/admissionregistration/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	webhookName                = "rancher.cattle.io"
	controllerCMName           = "cattle-controllers"
	lockCMName                 = "rancher-controller-lock"
	rancherSysNS               = "management.cattle.io/system-namespace"
	rancherCleanupJobYaml      = "/verrazzano/platform-operator/manifests/rancher-cleanup/rancher-cleanup.yaml"
	rancherCleanupJobName      = "cleanup-job"
	rancherCleanupJobNamespace = "kube-system"
)

var rancherSystemNS = []string{
	"cattle-system",
	"cattle-alerting",
	"cattle-logging",
	"cattle-pipeline",
	"cattle-prometheus",
	"cattle-global-data",
	"cattle-istio",
	"cattle-global-nt",
	"security-scan",
	"cattle-fleet-clusters-system",
	"cattle-fleet-system",
	"cattle-fleet-local-system",
	"tigera-operator",
	"cattle-impersonation-system",
	"rancher-operator-system",
	"cattle-csp-adapter-system",
	"fleet-default",
	"fleet-local",
	"local",
}

// create func vars for unit tests
type forkPostUninstallFuncSig func(ctx spi.ComponentContext, monitor monitor.BackgroundProcessMonitor) error

var forkPostUninstallFunc forkPostUninstallFuncSig = forkPostUninstall

type postUninstallFuncSig func(ctx spi.ComponentContext) error

var postUninstallFunc postUninstallFuncSig = invokeRancherSystemToolAndCleanup

// postUninstall - Rancher component post-uninstall
//
// This uses the rancher-cleanup tool for uninstall. Launch the uninstall operation in a goroutine and requeue to check back later.
// On subsequent callbacks, we check the status of the goroutine with the 'monitor' object, and postUninstall
// returns or requeues accordingly.
func postUninstall(ctx spi.ComponentContext, monitor monitor.BackgroundProcessMonitor) error {
	if monitor.IsRunning() {
		// Check the result
		succeeded, err := monitor.CheckResult()
		if err != nil {
			// Not finished yet, requeue
			ctx.Log().Progress("Component Rancher waiting to finish post-uninstall in the background")
			return err
		}
		// reset on success or failure
		monitor.Reset()
		// If it's not finished running, requeue
		if succeeded {
			return nil
		}
		// if we were unsuccessful, reset and drop through to try again
		ctx.Log().Debug("Error during Rancher post-uninstall, retrying")
	}

	return forkPostUninstallFunc(ctx, monitor)
}

// forkPostUninstall - fork uninstall install of Rancher
func forkPostUninstall(ctx spi.ComponentContext, monitor monitor.BackgroundProcessMonitor) error {
	ctx.Log().Debugf("Creating background post-uninstall goroutine for Rancher")

	monitor.Run(
		func() error {
			return postUninstallFunc(ctx)
		},
	)

	return ctrlerrors.RetryableError{Source: ComponentName}
}

// invokeRancherSystemToolAndCleanup - responsible for the actual deletion of resources
// This calls the rancher-cleanup tool.
func invokeRancherSystemToolAndCleanup(ctx spi.ComponentContext) error {
	// Create the rancher-cleanup job if it does not already exist
	job := &batchv1.Job{}
	err := ctx.Client().Get(context.TODO(), types.NamespacedName{Namespace: rancherCleanupJobNamespace, Name: rancherCleanupJobName}, job)
	if err != nil {
		if errors.IsNotFound(err) {
			// Create the rancher-cleanup job
			if err = k8sutil.NewYAMLApplier(ctx.Client(), "").ApplyF(rancherCleanupJobYaml); err != nil {
				return ctx.Log().ErrorfNewErr("Failed applying Yaml to create job %s/%s for component %s: %v", rancherCleanupJobNamespace, rancherCleanupJobName, ComponentName, err)
			}
			return ctx.Log().ErrorNewErr("Component %s waiting for job %s/%s to start", ComponentName, rancherCleanupJobNamespace, rancherCleanupJobName)
		} else {
			return err
		}
	}

	// Re-queue if the job has not completed
	for _, condition := range job.Status.Conditions {
		if condition.Type != batchv1.JobComplete {
			return ctx.Log().ErrorNewErr("Component %s waiting for job %s/%s to complete", ComponentName, job.Namespace, job.Name)
		}
	}

	// Remove the Rancher webhooks
	err = deleteWebhooks(ctx)
	if err != nil {
		return err
	}

	// Delete the Rancher resources that need to be matched by a string
	err = deleteMatchingResources(ctx)
	if err != nil {
		return err
	}

	// Delete the remaining Rancher ConfigMaps
	err = resource.Resource{
		Name:      controllerCMName,
		Namespace: constants.KubeSystem,
		Client:    ctx.Client(),
		Object:    &corev1.ConfigMap{},
		Log:       ctx.Log(),
	}.Delete()
	if err != nil {
		return err
	}
	err = resource.Resource{
		Name:      lockCMName,
		Namespace: constants.KubeSystem,
		Client:    ctx.Client(),
		Object:    &corev1.ConfigMap{},
		Log:       ctx.Log(),
	}.Delete()
	if err != nil {
		return err
	}

	crds := getCRDList(ctx)

	// Remove any Rancher custom resources that remain
	removeCRs(ctx, crds)

	// Remove any Rancher CRD finalizers that may be causing CRD deletion to hang
	removeCRDFinalizers(ctx, crds)

	return nil
}

// deleteWebhooks takes care of deleting the Webhook resources from Rancher
func deleteWebhooks(ctx spi.ComponentContext) error {
	vwcNames := []string{webhookName, "validating-webhook-configuration"}
	mwcNames := []string{webhookName, "mutating-webhook-configuration"}

	for _, name := range vwcNames {
		err := resource.Resource{
			Name:   name,
			Client: ctx.Client(),
			Object: &admv1.ValidatingWebhookConfiguration{},
			Log:    ctx.Log(),
		}.Delete()
		if err != nil {
			return err
		}
	}

	for _, name := range mwcNames {
		err := resource.Resource{
			Name:   name,
			Client: ctx.Client(),
			Object: &admv1.MutatingWebhookConfiguration{},
			Log:    ctx.Log(),
		}.Delete()
		if err != nil {
			return err
		}
	}
	return nil
}

// getCRDList returns the list of CRDs in the cluster
func getCRDList(ctx spi.ComponentContext) *v1.CustomResourceDefinitionList {
	crds := &v1.CustomResourceDefinitionList{}
	err := ctx.Client().List(context.TODO(), crds)
	if err != nil {
		ctx.Log().Errorf("Failed to list CRDs during uninstall: %v", err)
	}

	return crds
}

// removeCRs deletes any remaining Rancher cattle.io custom resources
func removeCRs(ctx spi.ComponentContext, crds *v1.CustomResourceDefinitionList) {
	ctx.Log().Oncef("Removing Rancher custom resources")
	for _, crd := range crds.Items {
		if strings.HasSuffix(crd.Name, ".cattle.io") {
			for _, version := range crd.Spec.Versions {
				rancherCRs := unstructured.UnstructuredList{}
				rancherCRs.SetAPIVersion(fmt.Sprintf("%s/%s", crd.Spec.Group, version.Name))
				rancherCRs.SetKind(crd.Spec.Names.Kind)
				err := ctx.Client().List(context.TODO(), &rancherCRs)
				if err != nil {
					ctx.Log().Errorf("Failed to list CustomResource %s during uninstall: %v", rancherCRs.GetKind(), err)
					continue
				}

				for _, rancherCR := range rancherCRs.Items {
					cr := rancherCR
					resource.Resource{
						Namespace: cr.GetNamespace(),
						Name:      cr.GetName(),
						Client:    ctx.Client(),
						Object:    &cr,
						Log:       ctx.Log(),
					}.RemoveFinalizersAndDelete()

				}
			}
		}
	}
}

func removeCRDFinalizers(ctx spi.ComponentContext, crds *v1.CustomResourceDefinitionList) {
	var rancherDeletedCRDs []v1.CustomResourceDefinition
	for _, crd := range crds.Items {
		if strings.HasSuffix(crd.Name, ".cattle.io") && crd.DeletionTimestamp != nil && !crd.DeletionTimestamp.IsZero() {
			rancherDeletedCRDs = append(rancherDeletedCRDs, crd)
		}
	}

	for _, crd := range rancherDeletedCRDs {
		ctx.Log().Infof("Removing finalizers from deleted Rancher CRD %s", crd.Name)
		err := resource.Resource{
			Name:   crd.Name,
			Client: ctx.Client(),
			Object: &v1.CustomResourceDefinition{},
			Log:    ctx.Log(),
		}.RemoveFinalizers()
		if err != nil {
			// not treated as a failure
			ctx.Log().Errorf("Failed to remove finalizer from Rancher CRD %s: %v", crd.Name, err)
		}
	}
}

// deleteMatchingResources delete the Rancher objects that need to match a string: ClusterRole, ClusterRoleBinding, RoleBinding, PersistentVolumes
func deleteMatchingResources(ctx spi.ComponentContext) error {
	// list of matching prefixes for cluster roles, clusterRole bindings
	roleMatch := []string{
		"cattle.io",
		"app:rancher",
		"rancher-webhook",
		"fleetworkspace-",
		"fleet-",
		"gitjob",
		"cattle-",
		"pod-impersonation-helm-op-",
		"cattle-unauthenticated",
		"default-admin-",
		"proxy-",
	}

	// Delete the Rancher Cluster Roles
	crList := rbacv1.ClusterRoleList{}
	err := ctx.Client().List(context.TODO(), &crList)
	if err != nil {
		return ctx.Log().ErrorfNewErr("Failed to list the ClusterRoles: %v", err)
	}
	for i := range crList.Items {
		err = deleteMatchingObject(ctx, roleMatch, &crList.Items[i])
		if err != nil {
			return err
		}
	}

	// Delete the Rancher Cluster Role Bindings
	crbList := rbacv1.ClusterRoleBindingList{}
	err = ctx.Client().List(context.TODO(), &crbList)
	if err != nil {
		return ctx.Log().ErrorfNewErr("Failed to list the ClusterRoleBindings: %v", err)
	}
	for i := range crbList.Items {
		err = deleteMatchingObject(ctx, roleMatch, &crbList.Items[i])
		if err != nil {
			return err
		}
	}

	// Delete the Rancher Role Bindings
	rblist := rbacv1.RoleBindingList{}
	err = ctx.Client().List(context.TODO(), &rblist)
	if err != nil {
		return ctx.Log().ErrorfNewErr("Failed to list the RoleBindings: %v", err)
	}
	for i := range rblist.Items {
		err = deleteMatchingObject(ctx, []string{"^rb-"}, &rblist.Items[i])
		if err != nil {
			return err
		}
	}

	// Delete the Rancher Persistent Volumes
	pvList := corev1.PersistentVolumeList{}
	err = ctx.Client().List(context.TODO(), &pvList)
	if err != nil {
		return ctx.Log().ErrorfNewErr("Failed to list the PersistentVolumes: %v", err)
	}
	for i := range pvList.Items {
		err = deleteMatchingObject(ctx, []string{"pvc-", "ocid1.volume"}, &pvList.Items[i])
		if err != nil {
			return err
		}
	}
	return nil
}

// deleteMatchingObjects is a helper function that deletes objects in an object list that match any of the object names using regex
func deleteMatchingObject(ctx spi.ComponentContext, matches []string, obj client.Object) error {
	objectMatch, err := regexp.MatchString(strings.Join(matches, "|"), obj.GetName())
	if err != nil {
		return ctx.Log().ErrorfNewErr("Failed to verify that the %s %s is from Rancher: %v", obj.GetObjectKind().GroupVersionKind().String(), obj.GetName(), err)
	}
	if objectMatch {
		err = resource.Resource{
			Name:      obj.GetName(),
			Namespace: obj.GetNamespace(),
			Client:    ctx.Client(),
			Object:    obj,
			Log:       ctx.Log(),
		}.Delete()
		if err != nil {
			return err
		}
	}
	return nil
}

// isRancherNamespace determines whether the namespace given is a Rancher ns
func isRancherNamespace(ns *corev1.Namespace) bool {
	if vzstring.SliceContainsString(rancherSystemNS, ns.Name) {
		return true
	}
	if ns.Annotations == nil {
		return false
	}
	if val, ok := ns.Annotations[rancherSysNS]; ok && val == "true" {
		return true
	}
	return false
}
