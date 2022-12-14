// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"context"
	"fmt"
	osexec "os/exec"
	"regexp"
	"strings"

	ctrlerrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/os"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/verrazzano/verrazzano/pkg/k8s/resource"
	vzstring "github.com/verrazzano/verrazzano/pkg/string"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	admv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var rancherSystemTool = "/usr/local/bin/system-tools"

const (
	webhookName      = "rancher.cattle.io"
	controllerCMName = "cattle-controllers"
	lockCMName       = "rancher-controller-lock"
	rancherSysNS     = "management.cattle.io/system-namespace"
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

type forkPostUninstallFuncSig func(ctx spi.ComponentContext, monitor postUninstallMonitor) error
var forkPostUninstallFunc forkPostUninstallFuncSig = forkPostUninstall

type rancherUninstallToolFuncSig func(log vzlog.VerrazzanoLogger, nsName string) ([]byte, error)
var rancherUninstallToolFunc rancherUninstallToolFuncSig = invokeRancherSystemTool

// postUninstallRoutineParams - Used to pass args to the postUninstall goroutine
type postUninstallRoutineParams struct {
	ctx spi.ComponentContext
}

// uninstallMonitor - Represents a monitor object used by the component to monitor a background goroutine used for running
// rancher uninstall operations asynchronously.
type postUninstallMonitor interface {
	// checkResult - Checks for a result from the install goroutine; returns either the result of the operation, or an error indicating
	// the install is still in progress
	checkResult() (bool, error)
	// reset - Resets the monitor and closes any open channels
	reset()
	// isRunning - returns true of the monitor/goroutine are active
	isRunning() bool
	// run - Run the install with the specified args
	run(args postUninstallRoutineParams)
}

type postUninstallMonitorType struct {
	running  bool
	resultCh chan bool
	inputCh  chan postUninstallRoutineParams
}

// checkResult - checks for a result from the goroutine
// - returns false and a retry error if it's still running, or the result from the channel and nil if an answer was received
func (m *postUninstallMonitorType) checkResult() (bool, error) {
	select {
	case result := <-m.resultCh:
		return result, nil
	default:
		return false, ctrlerrors.RetryableError{Source: ComponentName}
	}
}

// reset - reset the monitor and close the channel
func (m *postUninstallMonitorType) reset() {
	m.running = false
	close(m.resultCh)
	close(m.inputCh)
}

// isRunning - returns true of the monitor/goroutine are active
func (m *postUninstallMonitorType) isRunning() bool {
	return m.running
}

// postUninstall removes the objects after the Helm uninstall process finishes
func postUninstall(ctx spi.ComponentContext, monitor postUninstallMonitor) error {
	if monitor.isRunning() {
		// Check the result
		succeeded, err := monitor.checkResult()
		if err != nil {
			// Not finished yet, requeue
			ctx.Log().Progress("Component Rancher waiting to finish post-uninstall in the background")
			return err
		}
		// reset on success or failure
		monitor.reset()
		// If it's not finished running, requeue
		if succeeded {
			return nil
		}
		// if we were unsuccessful, reset and drop through to try again
		ctx.Log().Debug("Error during rancher post-uninstall, retrying")
	}

	return forkPostUninstallFunc(ctx, monitor)
}

func forkPostUninstall(ctx spi.ComponentContext, monitor postUninstallMonitor) error {
	ctx.Log().Debugf("Creating background post-uninstall goroutine for Rancher")

	monitor.run(
		postUninstallRoutineParams{
			ctx: ctx,
		},
	)

	return ctrlerrors.RetryableError{Source: ComponentName}
}

func (m *postUninstallMonitorType) run(args postUninstallRoutineParams) {
	m.running = true
	m.resultCh = make(chan bool, 2)
	m.inputCh = make(chan postUninstallRoutineParams, 2)

	go func(inputCh chan postUninstallRoutineParams, outputCh chan bool) {
		// The function will execute once, sending true on success, false on failure to the channel reader
		// Read inputs
		args := <-inputCh
		ctx := args.ctx

		// List all the namespaces that need to be cleaned from Rancher components
		nsList := corev1.NamespaceList{}
		err := ctx.Client().List(context.TODO(), &nsList)
		if err != nil {
			ctx.Log().ErrorfNewErr("Failed to list the Rancher namespaces: %v", err)
			outputCh <- false
			return
		}

		// For Rancher namespaces, run the system tools uninstaller
		for i, ns := range nsList.Items {
			if isRancherNamespace(&nsList.Items[i]) {
				stdErr, err := invokeRancherSystemTool(ctx.Log(), ns.Name)
				if err != nil {
					ctx.Log().ErrorNewErr("Failed to run system tools for Rancher deletion: %s: %v", stdErr, err)
					outputCh <- false
					return
				}
			}
		}

		// // FIXME: maybe don't run these in the background
		// // Remove the Rancher webhooks
		// err = deleteWebhooks(ctx)
		// if err != nil {
		// 	outputCh <- false
		// 	return
		// }

		// // Delete the Rancher resources that need to be matched by a string
		// err = deleteMatchingResources(ctx)
		// if err != nil {
		// 	outputCh <- false
		// 	return
		// }

		// // Delete the remaining Rancher ConfigMaps
		// err = resource.Resource{
		// 	Name:      controllerCMName,
		// 	Namespace: constants.KubeSystem,
		// 	Client:    ctx.Client(),
		// 	Object:    &corev1.ConfigMap{},
		// 	Log:       ctx.Log(),
		// }.Delete()
		// if err != nil {
		// 	outputCh <- false
		// 	return
		// }
		// err = resource.Resource{
		// 	Name:      lockCMName,
		// 	Namespace: constants.KubeSystem,
		// 	Client:    ctx.Client(),
		// 	Object:    &corev1.ConfigMap{},
		// 	Log:       ctx.Log(),
		// }.Delete()
		// if err != nil {
		// 	outputCh <- false
		// 	return
		// }

		// crds := getCRDList(ctx)

		// // Remove any Rancher custom resources that remain
		// removeCRs(ctx, crds)

		// // Remove any Rancher CRD finalizers that may be causing CRD deletion to hang
		// removeCRDFinalizers(ctx, crds)

		outputCh <- true
	}(m.inputCh, m.resultCh)

	m.inputCh <- args
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

// setRancherSystemTool sets the Rancher system tool to an arbitrary command
func setRancherSystemTool(cmd string) {
	rancherSystemTool = cmd
}

func invokeRancherSystemTool(log vzlog.VerrazzanoLogger, nsName string) (stdErr []byte, err error) {
	log.Infof("Running the Rancher uninstall system tool for namespace %s", nsName)
	args := []string{"remove", "-c", "/home/verrazzano/kubeconfig", "--namespace", nsName, "--force"}
	cmd := osexec.Command(rancherSystemTool, args...) //nolint:gosec //#nosec G204
	_, stdErr, err = os.DefaultRunner{}.Run(cmd)
	return
}
