// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"context"
	"regexp"

	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/k8s/resource"
	"github.com/verrazzano/verrazzano/pkg/os"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	admv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	osexec "os/exec"
)

var rancherSystemTool = "/usr/local/bin/system-tools"

const (
	webhookName      = "rancher.cattle.io"
	controllerCMName = "cattle-controllers"
	lockCMName       = "rancher-controller-lock"
)

// postUninstall removes the objects after the Helm uninstall process finishes
func postUninstall(ctx spi.ComponentContext) error {
	ctx.Log().Oncef("Running the Rancher uninstall system tool")

	// List all the namespaces that need to be cleaned from Rancher components
	nsList := corev1.NamespaceList{}
	err := ctx.Client().List(context.TODO(), &nsList)
	if err != nil {
		return ctx.Log().ErrorfNewErr("Failed to list the Rancher namespaces: %v", err)
	}

	// For Rancher namespaces, run the system tools uninstaller
	for _, ns := range nsList.Items {
		matches, err := regexp.MatchString("^cattle-|^local|^p-|^user-|^fleet|^rancher", ns.Name)
		if err != nil {
			return ctx.Log().ErrorfNewErr("Failed to verify that namespace %s is a Rancher namespace: %v", ns.Name, err)
		}
		if matches {
			args := []string{"remove", "-c", "/home/verrazzano/kubeconfig", "--namespace", ns.Name, "--force"}
			cmd := osexec.Command(rancherSystemTool, args...) //nolint:gosec //#nosec G204
			_, stdErr, err := os.DefaultRunner{}.Run(cmd)
			if err != nil {
				return ctx.Log().ErrorNewErr("Failed to run system tools for Rancher deletion: %s: %v", stdErr, err)
			}
		}
	}

	// Remove the Rancher webhooks
	err = resource.Resource{
		Name:   webhookName,
		Client: ctx.Client(),
		Object: &admv1.ValidatingWebhookConfiguration{},
		Log:    ctx.Log(),
	}.Delete()
	if err != nil {
		return err
	}
	err = resource.Resource{
		Name:   webhookName,
		Client: ctx.Client(),
		Object: &admv1.MutatingWebhookConfiguration{},
		Log:    ctx.Log(),
	}.Delete()
	if err != nil {
		return err
	}

	// Delete the Rancher ClusterRoles and ClusterRoleBindings
	err = deleteRoleResources(ctx)
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

	return nil
}

// deleteRoleResources delete the Rancher role objects: ClusterRole, ClusterRoleBinding
func deleteRoleResources(ctx spi.ComponentContext) error {
	clusterRoleMatch := "cattle.io|app:rancher|rancher-webhook|fleetworkspace-|fleet-|gitjob"

	// Get the lists for the CR and CRB
	crList := rbacv1.ClusterRoleList{}
	err := ctx.Client().List(context.TODO(), &crList)
	if err != nil {
		return ctx.Log().ErrorfNewErr("Failed to list the ClusterRoleBindings: %v", err)
	}
	crbList := rbacv1.ClusterRoleBindingList{}
	err = ctx.Client().List(context.TODO(), &crbList)
	if err != nil {
		return ctx.Log().ErrorfNewErr("Failed to list the ClusterRoleBindings: %v", err)
	}

	for i, cr := range crList.Items {
		matches, err := regexp.MatchString(clusterRoleMatch, cr.Name)
		if err != nil {
			return ctx.Log().ErrorfNewErr("Failed to verify that Cluster Role is from Rancher: %v", cr.Name, err)
		}
		if matches {
			err = resource.Resource{
				Name:      cr.Name,
				Namespace: cr.Namespace,
				Client:    ctx.Client(),
				Object:    &crList.Items[i],
				Log:       ctx.Log(),
			}.Delete()
			if err != nil {
				return err
			}
		}
	}

	for i, crb := range crbList.Items {
		matches, err := regexp.MatchString(clusterRoleMatch, crb.Name)
		if err != nil {
			return ctx.Log().ErrorfNewErr("Failed to verify that Cluster Role Binding is from Rancher: %v", crb.Name, err)
		}
		if matches {
			err = resource.Resource{
				Name:      crb.Name,
				Namespace: crb.Namespace,
				Client:    ctx.Client(),
				Object:    &crbList.Items[i],
				Log:       ctx.Log(),
			}.Delete()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// setRancherSystemTool sets the Rancher system tool to an arbitrary command
func setRancherSystemTool(cmd string) {
	rancherSystemTool = cmd
}
