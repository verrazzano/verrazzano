// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"

	"go.uber.org/zap"
	k8sapps "k8s.io/api/apps/v1"
	k8score "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component"
)

// The max upgrade failures for a given upgrade attempt is 2
const failedUpgradeLimit = 2

// Reconcile upgrade will upgrade the components as required
func (r *Reconciler) reconcileUpgrade(log *zap.SugaredLogger, req ctrl.Request, cr *installv1alpha1.Verrazzano) (ctrl.Result, error) {
	// Upgrade version was validated in webhook, see ValidateVersion
	targetVersion := cr.Spec.Version

	// Only allow upgrade to retry a certain amount of times during any upgrade attempt.
	if upgradeFailureCount(cr.Status, cr.Generation) > failedUpgradeLimit {
		log.Info("Upgrade failure limit reached, upgrade will not be attempted")
		return ctrl.Result{}, nil
	}

	// Only write the upgrade started message once
	if !isLastCondition(cr.Status, installv1alpha1.UpgradeStarted) {
		err := r.updateStatus(log, cr, fmt.Sprintf("Verrazzano upgrade to version %s in progress", cr.Spec.Version),
			installv1alpha1.UpgradeStarted)
		// Always requeue to get a fresh copy of status and avoid potential conflict
		return ctrl.Result{Requeue: true, RequeueAfter: 1}, err
	}

	// Loop through all of the Verrazzano components and upgrade each one sequentially
	for _, comp := range component.GetComponents() {
		err := comp.Upgrade(log, r, cr.Namespace, r.DryRun)
		if err != nil {
			log.Errorf("Error upgrading component %s: %v", comp.Name(), err)
			msg := fmt.Sprintf("Error upgrading component %s - %s\".  Error is %s", comp.Name(),
				fmtGeneration(cr.Generation), err.Error())
			err := r.updateStatus(log, cr, msg, installv1alpha1.UpgradeFailed)
			return ctrl.Result{}, err
		}
	}

	// Invoke the global post upgrade function after all components are upgraded.
	err := postUpgradeFunc(log, r)
	if err != nil {
		return ctrl.Result{}, err
	}

	msg := fmt.Sprintf("Verrazzano upgraded to version %s successfully", cr.Spec.Version)
	log.Info(msg)
	cr.Status.Version = targetVersion
	err = r.updateStatus(log, cr, msg, installv1alpha1.UpgradeComplete)

	return ctrl.Result{}, err
}

// Return true if verrazzano is installed
func isInstalled(st installv1alpha1.VerrazzanoStatus) bool {
	for _, cond := range st.Conditions {
		if cond.Type == installv1alpha1.InstallComplete {
			return true
		}
	}
	return false
}

// Return true if the last condition matches the condition type
func isLastCondition(st installv1alpha1.VerrazzanoStatus, conditionType installv1alpha1.ConditionType) bool {
	l := len(st.Conditions)
	if l == 0 {
		return false
	}
	return st.Conditions[l-1].Type == conditionType
}

// Get the number of times an upgrade failed for the specified
// generation, meaning the last time the CR spec was modified by the user
// This is needed as a circuit-breaker for upgrade and other operations where
// limit the retries on a given generation of the CR.
func upgradeFailureCount(st installv1alpha1.VerrazzanoStatus, generation int64) int {
	var c int
	for _, cond := range st.Conditions {
		// Look for an upgrade failed condition where the message contains the CR generation that
		// is currently being processed, then increment the count. If the generation is not found then
		// the condition is from a previous user upgrade request and we can ignore it.
		if cond.Type == installv1alpha1.UpgradeFailed &&
			strings.Contains(cond.Message, fmtGeneration(generation)) {
			c++
		}
	}
	return c
}

func fmtGeneration(gen int64) string {
	s := strconv.FormatInt(gen, 10)
	return "generation:" + s
}

func postUpgradeFunc(log *zap.SugaredLogger, client clipkg.Client) error {
	err := patchPrometheusDeployment(log, client)
	if err != nil {
		return err
	}
	return nil
}

func patchPrometheusDeployment(log *zap.SugaredLogger, client clipkg.Client) error {
	// If Prometheus isn't deployed don't do anything.
	promKey := clipkg.ObjectKey{Namespace: "verrazzano-system", Name: "vmi-system-prometheus-0"}
	promObj := k8sapps.Deployment{}
	err := client.Get(context.Background(), promKey, &promObj)
	if errors.IsNotFound(err) {
		log.Debugf("No Prometheus deployment found. Skip patching.")
		return nil
	}
	if err != nil {
		log.Errorf("Failed to fetch Prometheus deployment: %s", err)
		return err
	}

	// If Keycloak isn't deployed configure Prometheus to avoid the Istio sidecar for metrics scraping.
	kcKey := clipkg.ObjectKey{Namespace: "keycloak", Name: "keycloak"}
	kcObj := k8sapps.StatefulSet{}
	err = client.Get(context.Background(), kcKey, &kcObj)
	if errors.IsNotFound(err) {
		// Set the Istio annotation on Prometheus to exclude all IP addresses.
		promObj.Spec.Template.Annotations["traffic.sidecar.istio.io/excludeOutboundIPRanges"] = "0.0.0.0/0"
		err = client.Update(context.TODO(), &promObj)
		if err != nil {
			log.Errorf("Failed to update Istio annotations of Prometheus deployment: %s", err)
			return err
		}
		return nil
	}
	if err != nil {
		log.Errorf("Failed to fetch Keycloak statefulset: %s", err)
		return err
	}

	// Set the Istio annotation on Prometheus to exclude Keycloak HTTP Service IP address.
	// The includeOutboundIPRanges implies all others are excluded.
	svcKey := clipkg.ObjectKey{Namespace: "keycloak", Name: "keycloak-http"}
	svcObj := k8score.Service{}
	err = client.Get(context.Background(), svcKey, &svcObj)
	if errors.IsNotFound(err) {
		log.Errorf("Failed to find HTTP Service for Keycloak: %s", err)
		return err
	}
	promObj.Spec.Template.Annotations["traffic.sidecar.istio.io/includeOutboundIPRanges"] = fmt.Sprintf("%s/32", svcObj.Spec.ClusterIP)
	err = client.Update(context.TODO(), &promObj)
	if err != nil {
		log.Errorf("Failed to update Istio annotations of Prometheus deployment: %s", err)
		return err
	}
	return nil
}
