// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricsbinding

import (
	"context"
	"fmt"
	promoperapi "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/verrazzano/verrazzano/application-operator/internal/metrics"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	k8scorev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/rand"
	k8scontroller "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"

	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/app/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	workloadSourceLabel = "__meta_kubernetes_pod_label_app_verrazzano_io_workload"
)

// reconcileBindingCreateOrUpdate completes the reconcile process for an object that is being created or updated
func (r *Reconciler) reconcileBindingCreateOrUpdate(ctx context.Context, metricsBinding *vzapi.MetricsBinding, log vzlog.VerrazzanoLogger) (k8scontroller.Result, error) {
	log.Debugw("Reconcile for created or updated object", "resource", metricsBinding.GetName())

	// Requeue with a delay to account for situations where the scrape config
	// has changed but without the MetricsBinding changing.
	var seconds = rand.IntnRange(45, 90)
	var requeueDuration = time.Duration(seconds) * time.Second

	// Handle the case where the workload uses the default metrics template - in this case, we will
	// delete the metrics binding if processing succeeds, since this is a one-time conversion of
	// legacy apps using default metrics template, to ServiceMonitor. If it's not using VMI config map,
	// we treat it like custom metrics setup
	if isLegacyDefaultMetricsBinding(metricsBinding) {
		if err := r.handleDefaultMetricsTemplate(ctx, metricsBinding, log); err != nil {
			return k8scontroller.Result{Requeue: true}, err
		}
		if err := r.deleteMetricsBinding(metricsBinding, log); err != nil {
			return k8scontroller.Result{Requeue: true}, err
		}
		// Requeue with a delay to account for situations where the scrape config
		// has changed but without the MetricsBinding changing.
		return reconcile.Result{Requeue: true, RequeueAfter: requeueDuration}, nil
	}

	// Update the MetricsBinding to add workload as owner ref
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, metricsBinding, func() error {
		return r.updateMetricsBinding(metricsBinding, log)
	})
	if err != nil {
		return k8scontroller.Result{Requeue: true}, err
	}

	// Handle the case where the workloaded uses a custom metrics template
	if err = r.handleCustomMetricsTemplate(ctx, metricsBinding, log); err != nil {
		return k8scontroller.Result{Requeue: true}, err
	}

	// Requeue with a delay to account for situations where the scrape config
	// has changed but without the MetricsBinding changing.
	return reconcile.Result{Requeue: true, RequeueAfter: requeueDuration}, nil
}

// handleDefaultMetricsTemplate handles pre-Verrazzano 1.4 metrics bindings that use the default
// metrics template, by creating/updating a service monitor that does the same work as the default template
func (r *Reconciler) handleDefaultMetricsTemplate(ctx context.Context, metricsBinding *vzapi.MetricsBinding, log vzlog.VerrazzanoLogger) error {
	log.Infof("Default metrics template used by metrics binding %s/%s, service monitor time!", metricsBinding.Namespace, metricsBinding.Name)

	// Create the Service monitor from information gathered from the Metrics Binding
	scrapeInfo, err := r.createScrapeInfo(ctx, metricsBinding, log)
	if err != nil {
		return err
	}
	serviceMonitor := promoperapi.ServiceMonitor{}
	serviceMonitor.SetName(metricsBinding.Name)
	serviceMonitor.SetNamespace(metricsBinding.Namespace)
	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, &serviceMonitor, func() error {
		return metrics.PopulateServiceMonitor(scrapeInfo, &serviceMonitor, log)
	})
	if err != nil {
		return log.ErrorfNewErr("Failed to create or update the service monitor for the Metrics Binding %s/%s: %v", metricsBinding.Namespace, metricsBinding.Name, err)
	}
	return nil
}

// handleCustomMetricsTemplate handles pre-Verrazzano 1.4 metrics bindings that use a custom
// metrics template, by updating the additionalScrapeConfigs secret for the Prometheus CR to collect
// metrics as specified by the custom template.
func (r *Reconciler) handleCustomMetricsTemplate(ctx context.Context, metricsBinding *vzapi.MetricsBinding, log vzlog.VerrazzanoLogger) error {
	log.Infof("Custom metrics template used by metrics binding %s/%s, edit additionalScrapeConfigs", metricsBinding.Namespace, metricsBinding.Name)
	var configMap = getPromConfigMap(metricsBinding)
	if configMap != nil {

	}
	// If the Secret exists, delete the existing config from the Secret
	secret, key := getPromConfigSecret(metricsBinding)
	if err := r.deletePrometheusConfigSecret(ctx, metricsBinding, secret, key, log); err != nil {
	}

	return nil
}

func (r *Reconciler) createScrapeInfo(ctx context.Context, metricsBinding *vzapi.MetricsBinding, log vzlog.VerrazzanoLogger) (metrics.ScrapeInfo, error) {
	var scrapeInfo metrics.ScrapeInfo

	// Get the workload object from the Metrics Binding to populate the Service Monitor
	workload := metricsBinding.Spec.Workload
	workloadObject := unstructured.Unstructured{}
	workloadObject.SetKind(workload.TypeMeta.Kind)
	workloadObject.SetAPIVersion(workload.TypeMeta.APIVersion)
	workloadName := types.NamespacedName{Namespace: metricsBinding.Namespace, Name: workload.Name}
	err := r.Client.Get(ctx, workloadName, &workloadObject)
	if err != nil {
		return scrapeInfo, log.ErrorfNewErr("Failed to get the workload %s from the Metrics Binding %s/%s: %v", workload.Name, metricsBinding.Namespace, metricsBinding.Name, err)
	}

	// Get the namespace for the Metrics Binding to check if Istio is enabled
	workloadNamespace := k8scorev1.Namespace{}
	err = r.Client.Get(context.TODO(), k8sclient.ObjectKey{Name: metricsBinding.GetNamespace()}, &workloadNamespace)
	if err != nil {
		return scrapeInfo, log.ErrorfNewErr("Failed to get metrics binding namespace %s: %v", metricsBinding.GetName(), err)
	}

	// Verify if Istio is enabled from the Namespace annotations
	value, ok := workloadNamespace.Labels[constants.LabelIstioInjection]
	istioEnabled := ok && value == "enabled"
	scrapeInfo.IstioEnabled = &istioEnabled

	// Match the Verrazzano workload application labels that get applied by the Metrics Binding labeler
	value, ok = workloadObject.GetLabels()[constants.MetricsWorkloadLabel]
	if !ok {
		return scrapeInfo, log.ErrorfNewErr("Failed to find the annotation %s on the target workload", constants.MetricsWorkloadLabel)
	}
	scrapeInfo.KeepLabels = map[string]string{workloadSourceLabel: value}

	// Add a port to the Service Monitor endpoints
	scrapeInfo.Ports = 1

	return scrapeInfo, nil
}

// updateMetricsBinding updates the Metrics Binding Owner Reference from the target workload and adds a Finalizer
func (r *Reconciler) updateMetricsBinding(metricsBinding *vzapi.MetricsBinding, log vzlog.VerrazzanoLogger) error {
	// Add the finalizer
	controllerutil.AddFinalizer(metricsBinding, finalizerName)

	// Retrieve the workload object from the MetricsBinding
	workloadObject, err := r.getWorkloadObject(metricsBinding)
	if err != nil {
		return log.ErrorfNewErr("Failed to get the Workload from the MetricsBinding %s: %v", metricsBinding.Spec.Workload.Name, err)
	}

	// Return error if UID is not found
	if len(workloadObject.GetUID()) == 0 {
		err = fmt.Errorf("could not get UID from workload resource: %s, %s", workloadObject.GetKind(), workloadObject.GetName())
		return log.ErrorfNewErr("Failed to find UID for workload %s: %v", workloadObject.GetName(), err)
	}

	// Set the owner reference for the MetricsBinding so that it gets deleted with the workload
	trueValue := true
	metricsBinding.SetOwnerReferences([]k8smetav1.OwnerReference{
		{
			Name:               workloadObject.GetName(),
			APIVersion:         workloadObject.GetAPIVersion(),
			Kind:               workloadObject.GetKind(),
			UID:                workloadObject.GetUID(),
			Controller:         &trueValue,
			BlockOwnerDeletion: &trueValue,
		},
	})
	return nil
}

// getMetricsTemplate returns the MetricsTemplate given in the MetricsBinding
func (r *Reconciler) getMetricsTemplate(ctx context.Context, metricsBinding *vzapi.MetricsBinding, log vzlog.VerrazzanoLogger) (*vzapi.MetricsTemplate, error) {
	template := vzapi.MetricsTemplate{
		TypeMeta: k8smetav1.TypeMeta{
			Kind:       vzconst.MetricsTemplateKind,
			APIVersion: vzconst.MetricsTemplateAPIVersion,
		},
	}

	templateSpec := metricsBinding.Spec.MetricsTemplate
	namespacedName := types.NamespacedName{Name: templateSpec.Name, Namespace: templateSpec.Namespace}
	err := r.Client.Get(ctx, namespacedName, &template)
	if err != nil {
		newErr := fmt.Errorf("Failed to get the MetricsTemplate %s: %v", templateSpec.Name, err)
		return nil, log.ErrorfNewErr(newErr.Error())
	}
	return &template, nil
}

// deleteMetricsBinding deletes the Metrics Binding object from the cluster
func (r *Reconciler) deleteMetricsBinding(metricsBinding *vzapi.MetricsBinding, log vzlog.VerrazzanoLogger) error {
	err := r.Delete(context.Background(), metricsBinding)
	if err != nil {
		return log.ErrorfNewErr("Failed to delete the Metrics Binding %s%s from the cluster: %v", metricsBinding.Namespace, metricsBinding.Name, err)
	}
	return err
}
