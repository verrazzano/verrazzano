// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricsutils

import (
	"context"

	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/app/v1alpha1"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// HandleDefaultMetricsTemplate handles pre-Verrazzano 1.4 metrics bindings that use the default
// metrics template, by creating/updating a service monitor that does the same work as the default
// template.
func HandleDefaultMetricsTemplate(ctx context.Context, client k8sclient.Client, metricsBinding *vzapi.MetricsBinding, log vzlog.VerrazzanoLogger) error {
	log.Infof("Default metrics template used by metrics binding %s/%s, service monitor time!", metricsBinding.Namespace, metricsBinding.Name)
	// serviceMonitor, err := r.createOrUpdateServiceMonitor(metricsBinding) // update because it may exist from prior reconcile where we failed to delete metricsbinding
	// if err != nil {
	// 	log.Errorf("Failed to create/update ServiceMonitor for MetricsBinding: %v", err)
	// 	return err
	// }
	return nil
	// if create service monitor succeeded, our conversion of legacy MetricsBinding is
	// done. Keep the MetricsBinding in the custom metrics template use case so we know this is a "legacy" app, update it with the
	// additionalScrapeConfig config map instead of the promConfigMap name
	// metricsBinding.Spec.ServiceMonitor = serviceMonitor.Name
	// err := a.Client.Update(ctx, metricsBinding)
	// if err != nil {
	// 	log.Errorf("Failed to update MetricsBinding with service monitor information: %v", err)
	// 	return admission.Errored(http.StatusInternalServerError, err)
	// }
}

// HandleCustomMetricsTemplate handles pre-Verrazzano 1.4 metrics bindings that use a custom
// metrics template, by updating the additionalScrapeConfigs secret for the Prometheus CR to collect
// metrics as specified by the custom template. TODO should this also handle user-specified configmap?
func HandleCustomMetricsTemplate(ctx context.Context, client k8sclient.Client, metricsBinding *vzapi.MetricsBinding, log vzlog.VerrazzanoLogger) error {
	log.Infof("Custom metrics template used by metrics binding %s/%s, edit additionalScrapeConfigs", metricsBinding.Namespace, metricsBinding.Name)
	return nil
}

// func (r *Reconciler) createOrUpdateServiceMonitor(metricsBinding vzapi.MetricsBinding) (*ServiceMonitor, error) {
//
// }
