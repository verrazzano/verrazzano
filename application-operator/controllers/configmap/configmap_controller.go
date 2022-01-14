// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package configmap

import (
	"context"
	"github.com/go-logr/logr"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8scontroller "sigs.k8s.io/controller-runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Reconciler reconciles a ConfigMap object
type Reconciler struct {
	k8sclient.Client
	Log    logr.Logger
	Scheme *k8sruntime.Scheme
}

// SetupWithManager creates controller for the ConfigMap
func (r *Reconciler) SetupWithManager(mgr k8scontroller.Manager) error {
	return k8scontroller.NewControllerManagedBy(mgr).For(&corev1.ConfigMap{}).Complete(r)
}

// Reconcile manages the ConfigMap for the metrics binding webhook
func (r *Reconciler) Reconcile(req reconcile.Request) (reconcile.Result, error) {
	ctx := context.Background()

	// fetch the ConfigMap
	configMap := corev1.ConfigMap{}
	if err := r.Client.Get(ctx, req.NamespacedName, &configMap); err != nil {
		return reconcile.Result{}, k8sclient.IgnoreNotFound(err)
	}

	// Do nothing if it is the wrong ConfigMap
	if configMap.GetName() != ConfigMapName || configMap.GetNamespace() != constants.VerrazzanoSystemNamespace {
		return reconcile.Result{}, nil
	}

	r.Log.Info("Reconciling ConfigMap", "resource", req.Name)

	// fetch the Webhook
	mwc := admissionv1.MutatingWebhookConfiguration{}
	mwcName := types.NamespacedName{Name: mutatingWebhookConfigName}
	if err := r.Client.Get(ctx, mwcName, &mwc); err != nil {
		return reconcile.Result{}, k8sclient.IgnoreNotFound(err)
	}

	// Reconcile based on the status of the deletion timestamp
	if configMap.GetDeletionTimestamp().IsZero() {
		return r.reconcileConfigMapCreateOrUpdate(ctx, &configMap, &mwc)
	}
	return r.reconcileConfigMapDelete(ctx, &configMap, &mwc)
}

func (r *Reconciler) reconcileConfigMapCreateOrUpdate(ctx context.Context, configMap *corev1.ConfigMap, mwc *admissionv1.MutatingWebhookConfiguration) (reconcile.Result, error) {
	r.Log.Info("Updating the MutatingWebhookConfiguration to the ConfigMap values", "resource", configMap.GetName())

	// Update the Webhook
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, mwc, func() error {
		r.updateMutatingWebhookConfiguration(configMap, mwc)
		return nil
	})
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}

	// Update the ConfigMap
	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, configMap, func() error {
		controllerutil.AddFinalizer(configMap, finalizerName)
		return nil
	})
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}

	return reconcile.Result{}, nil
}

func (r *Reconciler) reconcileConfigMapDelete(ctx context.Context, configMap *corev1.ConfigMap, mwc *admissionv1.MutatingWebhookConfiguration) (reconcile.Result, error) {
	r.Log.Info("Resetting the MutatingWebhookConfiguration and Deleting the ConfigMap", "resource", configMap.GetName())

	// Reset the Webhook
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, mwc, func() error {
		r.resetMutatingWebhookConfiguration(mwc)
		return nil
	})
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}

	// Delete the ConfigMap
	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, configMap, func() error {
		controllerutil.RemoveFinalizer(configMap, finalizerName)
		return nil
	})
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}

	return reconcile.Result{}, nil
}

// updateMutatingWebhookConfiguration updates the workload resources allowed by the webhook
func (r *Reconciler) updateMutatingWebhookConfiguration(configMap *corev1.ConfigMap, mwc *admissionv1.MutatingWebhookConfiguration) {
	// Get the data from the ConfigMap and from the Webhook

	newWorkloadData := configMap.Data[resourceIdentifier]
	webhook := getWorkloadWebhook(mwc)
	if webhook == nil || len(webhook.Rules) < 1 {
		return
	}
	webhookWorkloads := webhook.Rules[0].Resources

	// Format the new Workload resources
	webhookWorkloads = formatWorkloadResources(newWorkloadData, webhookWorkloads)

	// Update the Webhook rule resource list
	webhook.Rules[0].Resources = webhookWorkloads
}

func (r *Reconciler) resetMutatingWebhookConfiguration(mwc *admissionv1.MutatingWebhookConfiguration) {
	// Reset the webhook to the default resource list
	webhook := getWorkloadWebhook(mwc)
	webhook.Rules[0].Resources = defaultResourceList
}
