// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"
	moduleapi "github.com/verrazzano/verrazzano-modules/module-operator/apis/platform/v1alpha1"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/result"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// deleteModules deletes all the modules, optionally only deleting ones that disabled
func (r Reconciler) deleteModules(log vzlog.VerrazzanoLogger, effectiveCR *vzapi.Verrazzano) result.Result {
	var reterr error
	var deletedCount int
	var moduleCount int

	// If deletion timestamp is non-zero then the VZ CR got deleted
	fullUninstall := !effectiveCR.GetDeletionTimestamp().IsZero()

	// Delete all modules.  Loop through all the components once even if error occurs.
	for _, comp := range registry.GetComponents() {
		if !comp.ShouldUseModule() {
			continue
		}

		// If not full uninstall then only disabled components should be uninstalled
		if !fullUninstall && comp.IsEnabled(effectiveCR) {
			continue
		}
		moduleCount++
		module := moduleapi.Module{ObjectMeta: metav1.ObjectMeta{
			Name:      comp.Name(),
			Namespace: vzconst.VerrazzanoInstallNamespace,
		}}

		// Delete all the configuration secrets that were referenced by the module
		res := r.deleteConfigSecrets(log, module.Namespace, comp.Name())
		if res.ShouldRequeue() {
			return res
		}

		// Delete all the configuration configmaps that were referenced by the module
		res = r.deleteConfigMaps(log, module.Namespace, comp.Name())
		if res.ShouldRequeue() {
			return res
		}

		err := r.Client.Delete(context.TODO(), &module, &client.DeleteOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				deletedCount++
				continue
			}
			if !errors.IsConflict(err) {
				log.Progressf("Failed to delete Component %s, retrying: %v", comp.Name(), err)
			}
			reterr = err
			continue
		}
	}
	if deletedCount != moduleCount {
		return result.NewResultShortRequeueDelay()
	}

	// return last error found so that we retry
	if reterr != nil {
		return result.NewResultShortRequeueDelayWithError(reterr)
	}
	// All modules have been deleted and the Module CRs are gone
	return result.NewResult()
}

// deleteConfigSecrets deletes all the module config secrets
func (r Reconciler) deleteConfigSecrets(log vzlog.VerrazzanoLogger, namespace string, moduleName string) result.Result {
	secretList := &corev1.SecretList{}
	req, _ := labels.NewRequirement(vzconst.VerrazzanoModuleOwnerLabel, selection.Equals, []string{moduleName})
	selector := labels.NewSelector().Add(*req)
	if err := r.Client.List(context.TODO(), secretList, &client.ListOptions{Namespace: namespace, LabelSelector: selector}); err != nil {
		log.Infof("Failed getting secrets in %s namespace, retrying: %v", namespace, err)
	}

	for i, s := range secretList.Items {
		err := r.Client.Delete(context.TODO(), &secretList.Items[i])
		if err != nil {
			if errors.IsNotFound(err) {
				continue
			}
			log.Errorf("Failed deleting secret %s/%s, retrying: %v", namespace, s.Name, err)
		}
	}
	return result.NewResult()
}

// deleteConfigMaps deletes all the module config maps
func (r Reconciler) deleteConfigMaps(log vzlog.VerrazzanoLogger, namespace string, moduleName string) result.Result {
	configMapList := &corev1.ConfigMapList{}
	req, _ := labels.NewRequirement(vzconst.VerrazzanoModuleOwnerLabel, selection.Equals, []string{moduleName})
	selector := labels.NewSelector().Add(*req)
	if err := r.Client.List(context.TODO(), configMapList, &client.ListOptions{Namespace: namespace, LabelSelector: selector}); err != nil {
		log.Infof("Failed getting configMaps in %s namespace, retrying: %v", namespace, err)
	}

	for i, s := range configMapList.Items {
		err := r.Client.Delete(context.TODO(), &configMapList.Items[i])
		if err != nil {
			if errors.IsNotFound(err) {
				continue
			}
			log.Errorf("Failed deleting configMap %s/%s, retrying: %v", namespace, s.Name, err)
		}
	}
	return result.NewResult()
}
